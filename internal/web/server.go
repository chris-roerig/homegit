// Package web implements homegit's read-only repository browser.
package web

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed templates/*.html static/*
var assets embed.FS

const (
	maxViewBytes = 1 << 20 // largest text file rendered inline
)

// Server serves the repository browser.
type Server struct {
	store   *Store
	sshPort int
	version string
	tmpl    *template.Template
	static  http.Handler
}

// New creates a Server that browses bare repositories in reposDir. sshPort is
// only used to display clone URLs. version, if non-empty, is shown in the
// page header.
func New(reposDir string, sshPort int, version string) (*Server, error) {
	t, err := template.New("").Funcs(template.FuncMap{
		"fmtSize": fmtSize,
	}).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, err
	}
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	return &Server{
		store:   &Store{ReposDir: reposDir},
		sshPort: sshPort,
		version: displayVersion(version),
		tmpl:    t,
		static:  http.StripPrefix("/_static/", http.FileServer(http.FS(sub))),
	}, nil
}

// ListenAndServe serves on addr until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(ctx, ln)
}

// Serve serves on ln until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ---- view models ----

type crumb struct {
	Name string
	URL  string
}

type node struct {
	Name     string
	URL      string
	IsDir    bool
	Open     bool
	Active   bool
	Children []*node
}

type listItem struct {
	Name  string
	URL   string
	IsDir bool
	Size  int64
}

type repoCard struct {
	Name        string
	CloneURL    string
	Description string
	Topics      []string
	Search      string // lowercased description and topics, for the client-side filter
}

type page struct {
	Title    string
	Version  string
	Kind     string // "home", "dir", "file", "error", "empty"
	Repo     string
	Branch   string
	CloneURL string
	Crumbs   []crumb
	Sidebar  []*node
	Truncate bool
	IsRoot   bool
	Meta     *Meta

	Repos []repoCard

	Items    []listItem
	Readme   template.HTML
	ReadmeAt string

	File     string
	Markdown template.HTML
	Lines    []string
	ImageURL string
	RawURL   string
	Note     string
	Size     int64

	Status  int
	Message string
}

// ---- routing ----

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "same-origin")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; script-src 'self'; img-src 'self' data: https: http:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.Set("Allow", "GET, HEAD")
		http.Error(w, "read-only", http.StatusMethodNotAllowed)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/_static/") {
		s.static.ServeHTTP(w, r)
		return
	}

	// Paths are decoded once by net/http; segments are validated below.
	segs := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 3)
	switch {
	case segs[0] == "":
		s.home(w, r)
	case len(segs) == 1 || (len(segs) == 2 && segs[1] == ""):
		http.Redirect(w, r, "/"+escSeg(segs[0])+"/tree/", http.StatusFound)
	default:
		rest := ""
		if len(segs) == 3 {
			rest = segs[2]
		}
		switch segs[1] {
		case "tree":
			s.tree(w, r, segs[0], rest)
		case "blob":
			s.blob(w, r, segs[0], rest)
		case "raw":
			s.raw(w, r, segs[0], rest)
		default:
			s.fail(w, r, segs[0], http.StatusNotFound, "Page not found")
		}
	}
}

func escSeg(s string) string { return url.PathEscape(s) }

func repoURL(repo, kind, p string) string {
	u := "/" + escSeg(repo) + "/" + kind + "/"
	if p == "" {
		return u
	}
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = escSeg(parts[i])
	}
	return u + strings.Join(parts, "/")
}

// ---- handlers ----

func (s *Server) render(w http.ResponseWriter, status int, p *page) {
	p.Version = s.version
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, "layout", p); err != nil {
		log.Printf("web: template error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, repo string, status int, msg string) {
	p := &page{Title: msg, Kind: "error", Status: status, Message: msg}
	if repo != "" {
		if _, err := s.store.repoDir(repo); err == nil {
			p.Repo = repo
			p.Crumbs = []crumb{{Name: repo, URL: repoURL(repo, "tree", "")}}
		}
	}
	s.render(w, status, p)
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	repos, err := s.store.ListRepos()
	if err != nil {
		log.Printf("web: list repos: %v", err)
		s.fail(w, r, "", http.StatusInternalServerError, "Could not read repositories")
		return
	}
	cards := make([]repoCard, len(repos))
	for i, name := range repos {
		cards[i] = repoCard{Name: name, CloneURL: s.cloneURL(r, name)}
		if m := s.store.ReadMeta(name); m != nil {
			cards[i].Description = m.Description
			cards[i].Topics = m.Topics
			cards[i].Search = strings.ToLower(m.Description + " " + strings.Join(m.Topics, " "))
		}
	}
	s.render(w, http.StatusOK, &page{Title: "Repositories", Kind: "home", Repos: cards})
}

// loadRepo validates the repo and path, loading the tree. On failure it
// writes the error response and returns ok=false.
func (s *Server) loadRepo(w http.ResponseWriter, r *http.Request, repo, rawPath string) (*page, *Tree, string, bool) {
	if _, err := s.store.repoDir(repo); err != nil {
		s.fail(w, r, "", http.StatusNotFound, "Repository not found")
		return nil, nil, "", false
	}
	p, ok := cleanPath(rawPath)
	if !ok {
		s.fail(w, r, repo, http.StatusBadRequest, "Invalid path")
		return nil, nil, "", false
	}
	branch, _ := s.store.Branch(repo)
	pg := &page{
		Title:    repo,
		Repo:     repo,
		Branch:   branch,
		CloneURL: s.cloneURL(r, repo),
	}
	tree, err := s.store.Tree(repo)
	if errors.Is(err, errEmpty) {
		pg.Kind = "empty"
		pg.Crumbs = []crumb{{Name: repo, URL: repoURL(repo, "tree", "")}}
		s.render(w, http.StatusOK, pg)
		return nil, nil, "", false
	}
	if err != nil {
		log.Printf("web: tree %s: %v", repo, err)
		s.fail(w, r, repo, http.StatusInternalServerError, "Could not read repository")
		return nil, nil, "", false
	}
	return pg, tree, p, true
}

func (s *Server) cloneURL(r *http.Request, repo string) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(r.Host); err == nil {
		host = h
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("ssh://%s:%d/%s.git", host, s.sshPort, repo)
}

func crumbs(repo, p, kind string) []crumb {
	cs := []crumb{{Name: repo, URL: repoURL(repo, "tree", "")}}
	if p == "" {
		return cs
	}
	parts := strings.Split(p, "/")
	for i, part := range parts {
		k := "tree"
		if i == len(parts)-1 {
			k = kind
		}
		cs = append(cs, crumb{Name: part, URL: repoURL(repo, k, strings.Join(parts[:i+1], "/"))})
	}
	return cs
}

func (s *Server) tree(w http.ResponseWriter, r *http.Request, repo, rawPath string) {
	pg, tree, p, ok := s.loadRepo(w, r, repo, rawPath)
	if !ok {
		return
	}
	e, found := tree.Lookup(p)
	if !found {
		s.fail(w, r, repo, http.StatusNotFound, "Path not found")
		return
	}
	if e.Type != "tree" {
		http.Redirect(w, r, repoURL(repo, "blob", p), http.StatusFound)
		return
	}

	pg.Kind = "dir"
	pg.Title = repo
	if p != "" {
		pg.Title = p + " · " + repo
	}
	pg.Crumbs = crumbs(repo, p, "tree")
	pg.Sidebar = buildSidebar(repo, tree, p, true)
	pg.Truncate = tree.Truncated
	if p == "" {
		pg.IsRoot = true
		pg.Meta = s.store.ReadMeta(repo)
	}

	children := tree.Children(p)
	var readme *Entry
	for i, c := range children {
		item := listItem{Name: c.Name(), IsDir: c.Type == "tree", Size: c.Size}
		switch c.Type {
		case "tree":
			item.URL = repoURL(repo, "tree", c.Path)
		case "blob":
			item.URL = repoURL(repo, "blob", c.Path)
		}
		pg.Items = append(pg.Items, item)
		if c.Type == "blob" && readme == nil && isReadme(c.Name()) {
			readme = &children[i]
		}
	}
	if readme != nil && readme.Size <= maxViewBytes {
		if data, err := s.store.Blob(repo, readme.Path, maxViewBytes); err == nil && utf8.Valid(data) {
			pg.ReadmeAt = readme.Name()
			pg.Readme = s.renderDoc(repo, readme.Path, readme.Name(), data)
		}
	}
	s.render(w, http.StatusOK, pg)
}

func isReadme(name string) bool {
	l := strings.ToLower(name)
	switch l {
	case "readme.md", "readme.markdown", "readme", "readme.txt":
		return true
	}
	return false
}

func isMarkdown(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".md", ".markdown", ".mdown":
		return true
	}
	return false
}

// renderDoc renders a README or markdown file; non-markdown readmes are shown
// as preformatted text.
func (s *Server) renderDoc(repo, filePath, name string, data []byte) template.HTML {
	if !isMarkdown(name) {
		return template.HTML("<pre>" + template.HTMLEscapeString(string(data)) + "</pre>")
	}
	dir := path.Dir(filePath)
	if dir == "." {
		dir = ""
	}
	resolve := func(u string, image bool) string {
		u = strings.TrimSpace(u)
		if strings.HasPrefix(u, "#") {
			return u
		}
		if strings.HasPrefix(u, "//") {
			return ""
		}
		if safe := safeURL(u); safe == "" || strings.Contains(safe, ":") && !isRelative(safe) {
			return safe
		}
		// Relative URL: resolve against the document's directory.
		ref, err := url.Parse(u)
		if err != nil {
			return ""
		}
		target := ref.Path
		if strings.HasPrefix(target, "/") {
			target = strings.TrimPrefix(path.Clean(target), "/")
		} else {
			target = path.Clean(path.Join(dir, target))
		}
		if target == "." {
			target = ""
		}
		if target == ".." || strings.HasPrefix(target, "../") {
			return ""
		}
		kind := "tree"
		if image {
			kind = "raw"
		} else if ref.Path != "" && !strings.HasSuffix(ref.Path, "/") {
			kind = "blob" // the blob handler redirects to tree for directories
		}
		out := repoURL(repo, kind, target)
		if ref.Fragment != "" {
			out += "#" + url.PathEscape(ref.Fragment)
		}
		return out
	}
	return template.HTML(RenderMarkdown(string(data), resolve))
}

// isRelative reports whether u has no scheme.
func isRelative(u string) bool {
	i := strings.IndexAny(u, ":/?#")
	return i < 0 || u[i] != ':'
}

func (s *Server) blob(w http.ResponseWriter, r *http.Request, repo, rawPath string) {
	pg, tree, p, ok := s.loadRepo(w, r, repo, rawPath)
	if !ok {
		return
	}
	e, found := tree.Lookup(p)
	if !found {
		s.fail(w, r, repo, http.StatusNotFound, "Path not found")
		return
	}
	if e.Type == "tree" {
		http.Redirect(w, r, repoURL(repo, "tree", p), http.StatusFound)
		return
	}
	if e.Type != "blob" {
		s.fail(w, r, repo, http.StatusNotFound, "Submodules are not browsable")
		return
	}

	pg.Kind = "file"
	pg.File = e.Name()
	pg.Title = p + " · " + repo
	pg.Crumbs = crumbs(repo, p, "blob")
	pg.Sidebar = buildSidebar(repo, tree, p, false)
	pg.Truncate = tree.Truncated
	pg.RawURL = repoURL(repo, "raw", p)
	pg.Size = e.Size

	switch {
	case isImage(e.Name()):
		pg.ImageURL = pg.RawURL
	case e.Size > maxViewBytes:
		pg.Note = "This file is too large to display."
	default:
		data, err := s.store.Blob(repo, p, maxViewBytes)
		if err != nil {
			log.Printf("web: blob %s/%s: %v", repo, p, err)
			s.fail(w, r, repo, http.StatusInternalServerError, "Could not read file")
			return
		}
		if bytes.IndexByte(data[:min(len(data), 8000)], 0) >= 0 || !utf8.Valid(data) {
			pg.Note = "This file is binary and cannot be displayed."
		} else if isMarkdown(e.Name()) {
			pg.Markdown = s.renderDoc(repo, p, e.Name(), data)
		} else {
			text := strings.TrimSuffix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
			pg.Lines = strings.Split(text, "\n")
			if len(data) == 0 {
				pg.Lines = nil
				pg.Note = "This file is empty."
			}
		}
	}
	s.render(w, http.StatusOK, pg)
}

func isImage(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".bmp":
		return true
	}
	return false
}

func (s *Server) raw(w http.ResponseWriter, r *http.Request, repo, rawPath string) {
	if _, err := s.store.repoDir(repo); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	p, ok := cleanPath(rawPath)
	if !ok || p == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	tree, err := s.store.Tree(repo)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	e, found := tree.Lookup(p)
	if !found || e.Type != "blob" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ct := mime.TypeByExtension(strings.ToLower(path.Ext(p)))
	switch {
	case ct == "" || strings.HasPrefix(ct, "text/html") || strings.Contains(ct, "javascript") || strings.Contains(ct, "xml") && !strings.Contains(ct, "svg"):
		ct = "text/plain; charset=utf-8"
	}
	h := w.Header()
	h.Set("Content-Type", ct)
	// Neutralize active content (e.g. SVG scripts) if opened directly.
	h.Set("Content-Security-Policy", "sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src data:")
	h.Set("Content-Length", fmt.Sprint(e.Size))
	if r.Method == http.MethodHead {
		return
	}
	if err := s.store.StreamBlob(repo, p, w); err != nil {
		log.Printf("web: raw %s/%s: %v", repo, p, err)
	}
}

// ---- sidebar ----

// buildSidebar builds the nested file tree. Ancestors of current are open;
// when current is a directory it is open too.
func buildSidebar(repo string, t *Tree, current string, currentIsDir bool) []*node {
	nodes := map[string]*node{}
	var roots []*node
	open := map[string]bool{}
	for p := current; p != "" && p != "."; p = path.Dir(p) {
		if p != current || currentIsDir {
			open[p] = true
		}
	}
	for _, e := range sortedForSidebar(t.Entries) {
		if e.Type == "commit" {
			continue
		}
		n := &node{Name: e.Name(), IsDir: e.Type == "tree", Active: e.Path == current}
		if n.IsDir {
			n.URL = repoURL(repo, "tree", e.Path)
			n.Open = open[e.Path]
		} else {
			n.URL = repoURL(repo, "blob", e.Path)
		}
		nodes[e.Path] = n
		parent := path.Dir(e.Path)
		if parent == "." {
			roots = append(roots, n)
		} else if pn, ok := nodes[parent]; ok {
			pn.Children = append(pn.Children, n)
		}
	}
	return roots
}

// sortedForSidebar orders entries so parents precede children and siblings are
// dirs-first then alphabetical.
func sortedForSidebar(es []Entry) []Entry {
	byDir := map[string][]Entry{}
	for _, e := range es {
		d := path.Dir(e.Path)
		byDir[d] = append(byDir[d], e)
	}
	var out []Entry
	var walk func(dir string)
	walk = func(dir string) {
		kids := byDir[dir]
		sortEntries(kids)
		for _, k := range kids {
			out = append(out, k)
			if k.Type == "tree" {
				walk(k.Path)
			}
		}
	}
	walk(".")
	return out
}

func fmtSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
}

// displayVersion formats v for the header as "v1.2.3". Empty stays empty.
func displayVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}
