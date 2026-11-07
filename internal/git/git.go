package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	repoCreateMutex sync.Mutex
)

type Command struct {
	Type     string // "upload-pack", "receive-pack", or "list"
	RepoPath string
}

// ListCommand is the SSH command clients send to request a repository
// listing from the server. It is read-only and takes no arguments, so the
// server uses its own configured repos directory as the authoritative source.
const ListCommand = "homegit-list-packs"

func ParseCommand(cmdStr string) (*Command, error) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	// The list command takes no arguments.
	if parts[0] == ListCommand {
		return &Command{Type: "list"}, nil
	}

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid git command: %s", cmdStr)
	}

	var cmdType string
	switch parts[0] {
	case "git-upload-pack":
		cmdType = "upload-pack"
	case "git-receive-pack":
		cmdType = "receive-pack"
	default:
		return nil, fmt.Errorf("unsupported command: %s", parts[0])
	}

	repoPath := strings.Trim(parts[1], "'\"")
	return &Command{Type: cmdType, RepoPath: repoPath}, nil
}

func (c *Command) Execute(reposDir string, stdin io.Reader, stdout, stderr io.Writer) error {
	// The list command is read-only and takes no repository path; it lists
	// the bare repositories in the server's configured repos directory.
	if c.Type == "list" {
		return listRepos(reposDir, stdout)
	}

	// Clean and validate repository path to prevent directory traversal
	// Remove leading slash if present (Git sends paths like /repo.git)
	repoPath := strings.TrimPrefix(c.RepoPath, "/")
	cleanPath := filepath.Clean(repoPath)

	// Reject paths with .. or absolute paths after cleaning
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid repository path (contains ..): %s", c.RepoPath)
	}
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("invalid repository path (absolute): %s -> %s", c.RepoPath, cleanPath)
	}

	fullPath := filepath.Join(reposDir, cleanPath)

	// Ensure the resolved path is still within repos directory
	absReposDir, err := filepath.Abs(reposDir)
	if err != nil {
		return fmt.Errorf("failed to resolve repos directory: %w", err)
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}
	if !strings.HasPrefix(absFullPath, absReposDir) {
		return fmt.Errorf("repository path outside repos directory: %s", c.RepoPath)
	}

	if c.Type == "receive-pack" {
		if err := ensureRepo(fullPath); err != nil {
			return err
		}
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("repository not found: %s", c.RepoPath)
	}

	cmd := exec.Command("git-"+c.Type, fullPath)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

// listRepos writes the names of all bare repositories (directories ending in
// ".git") in reposDir to w, one per line. The listing is derived entirely from
// the server's own directory, so no client-supplied path is involved.
func listRepos(reposDir string, w io.Writer) error {
	entries, err := os.ReadDir(reposDir)
	if err != nil {
		// A missing repos directory simply means no repositories yet.
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read repos directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".git") {
			if _, err := fmt.Fprintln(w, entry.Name()); err != nil {
				return fmt.Errorf("failed to write repository list: %w", err)
			}
		}
	}

	return nil
}

func ensureRepo(path string) error {
	// Lock to prevent race condition when multiple pushes try to create same repo
	repoCreateMutex.Lock()
	defer repoCreateMutex.Unlock()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		cmd := exec.Command("git", "init", "--bare", path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to init repository: %w", err)
		}
		// Set default branch to main
		cmd = exec.Command("git", "-C", path, "symbolic-ref", "HEAD", "refs/heads/main")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set default branch: %w", err)
		}
	}
	return nil
}
