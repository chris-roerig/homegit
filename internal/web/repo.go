package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	gitTimeout     = 15 * time.Second
	maxTreeEntries = 20000
)

var (
	errNotFound = errors.New("not found")
	errEmpty    = errors.New("repository is empty")
)

// Entry is one object in a repository's tree at HEAD.
type Entry struct {
	Path string
	Type string // "blob", "tree", or "commit" (submodule)
	Mode string
	Size int64
}

func (e Entry) Name() string { return path.Base(e.Path) }

// Tree is the full tree of a repository at HEAD.
type Tree struct {
	Entries   []Entry
	byPath    map[string]Entry
	Truncated bool
}

// Lookup returns the entry at p. The empty path is the root directory.
func (t *Tree) Lookup(p string) (Entry, bool) {
	if p == "" {
		return Entry{Type: "tree"}, true
	}
	e, ok := t.byPath[p]
	return e, ok
}

// Children returns the direct children of directory dir (dirs first, then
// files, each sorted by name).
func (t *Tree) Children(dir string) []Entry {
	var out []Entry
	for _, e := range t.Entries {
		if path.Dir(e.Path) == dirOrDot(dir) {
			out = append(out, e)
		}
	}
	sortEntries(out)
	return out
}

func dirOrDot(d string) string {
	if d == "" {
		return "."
	}
	return d
}

func sortEntries(es []Entry) {
	sort.SliceStable(es, func(i, j int) bool {
		di, dj := es[i].Type == "tree", es[j].Type == "tree"
		if di != dj {
			return di
		}
		return strings.ToLower(es[i].Name()) < strings.ToLower(es[j].Name())
	})
}

// Store reads bare repositories under a directory.
type Store struct {
	ReposDir string
}

// ListRepos returns repo display names (directory names minus ".git").
func (s *Store) ListRepos() ([]string, error) {
	entries, err := os.ReadDir(s.ReposDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".git") {
			names = append(names, strings.TrimSuffix(e.Name(), ".git"))
		}
	}
	sort.Strings(names)
	return names, nil
}

// repoDir maps a display name to its directory, rejecting anything that is
// not a plain name of an existing repository directly under ReposDir.
func (s *Store) repoDir(name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
		return "", errNotFound
	}
	dir := filepath.Join(s.ReposDir, name+".git")
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return "", errNotFound
	}
	return dir, nil
}

func gitCmd(ctx context.Context, dir string, args ...string) *exec.Cmd {
	full := append([]string{"--git-dir=" + dir, "--no-pager"}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	return cmd
}

// Branch returns the short name of the branch HEAD points to.
func (s *Store) Branch(name string) (string, error) {
	dir, err := s.repoDir(name)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	out, err := gitCmd(ctx, dir, "symbolic-ref", "--short", "HEAD").Output()
	if err != nil {
		return "HEAD", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// Tree loads the full tree at HEAD.
func (s *Store) Tree(name string) (*Tree, error) {
	dir, err := s.repoDir(name)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	if err := gitCmd(ctx, dir, "rev-parse", "--verify", "--quiet", "HEAD^{commit}").Run(); err != nil {
		return nil, errEmpty
	}
	out, err := gitCmd(ctx, dir, "ls-tree", "-r", "-t", "-l", "-z", "HEAD").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree: %w", err)
	}
	t := &Tree{byPath: map[string]Entry{}}
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		// "<mode> <type> <sha> <size>\t<path>"
		tab := bytes.IndexByte(rec, '\t')
		if tab < 0 {
			continue
		}
		f := strings.Fields(string(rec[:tab]))
		if len(f) < 4 {
			continue
		}
		e := Entry{Mode: f[0], Type: f[1], Path: string(rec[tab+1:])}
		if f[3] != "-" {
			e.Size, _ = strconv.ParseInt(f[3], 10, 64)
		}
		if len(t.Entries) >= maxTreeEntries {
			t.Truncated = true
			break
		}
		t.Entries = append(t.Entries, e)
		t.byPath[e.Path] = e
	}
	return t, nil
}

// Blob returns up to limit bytes of the file at p (HEAD). The path must come
// from a validated Tree entry.
func (s *Store) Blob(name, p string, limit int64) ([]byte, error) {
	dir, err := s.repoDir(name)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := gitCmd(ctx, dir, "cat-file", "blob", "HEAD:"+p)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	data, rerr := io.ReadAll(io.LimitReader(stdout, limit))
	_, _ = io.Copy(io.Discard, stdout)
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return data, rerr
}

// StreamBlob copies the file at p (HEAD) to w.
func (s *Store) StreamBlob(name, p string, w io.Writer) error {
	dir, err := s.repoDir(name)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*gitTimeout)
	defer cancel()
	cmd := gitCmd(ctx, dir, "cat-file", "blob", "HEAD:"+p)
	cmd.Stdout = w
	return cmd.Run()
}

// cleanPath normalizes a URL path into a repo-relative path and rejects
// traversal attempts. The empty string denotes the repository root.
func cleanPath(p string) (string, bool) {
	p = strings.Trim(p, "/")
	if p == "" {
		return "", true
	}
	if strings.ContainsRune(p, 0) {
		return "", false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", false
		}
	}
	return p, true
}
