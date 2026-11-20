package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newTestServer creates repos "demo" (with content) and "blank" (empty).
func newTestServer(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	reposDir := filepath.Join(root, "repos")
	work := filepath.Join(root, "work")
	git(t, root, "init", "-q", "-b", "main", work)
	write(t, filepath.Join(work, "README.md"), "# Demo\n\nSee [guide](docs/guide.md) and ![logo](img/logo.png).\n")
	write(t, filepath.Join(work, "docs", "guide.md"), "# Guide\n\n[up](../README.md)\n")
	write(t, filepath.Join(work, "src", "main.go"), "package main\n\n// <b>hi</b>\n")
	write(t, filepath.Join(work, "img", "logo.png"), "\x89PNG\x00\x01")
	write(t, filepath.Join(work, "bin.dat"), "a\x00b")
	write(t, filepath.Join(work, "x.svg"), `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	git(t, work, "add", ".")
	git(t, work, "commit", "-q", "-m", "init")
	if err := os.MkdirAll(reposDir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, root, "clone", "-q", "--bare", work, filepath.Join(reposDir, "demo.git"))
	git(t, root, "init", "-q", "--bare", "-b", "main", filepath.Join(reposDir, "blank.git"))
	s, err := New(reposDir, 2222)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func get(s *Server, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = "box.local:8222"
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func TestBrowse(t *testing.T) {
	s := newTestServer(t)
	tests := []struct {
		name   string
		url    string
		status int
		want   []string
		not    []string
	}{
		{"home lists repos", "/", 200, []string{`href="/demo/tree/"`, `href="/blank/tree/"`}, nil},
		{"repo redirects", "/demo", 302, nil, nil},
		{"root with readme", "/demo/tree/", 200,
			[]string{`<h1 id="demo">Demo</h1>`, `href="/demo/blob/docs/guide.md"`, `src="/demo/raw/img/logo.png"`,
				`ssh://box.local:2222/demo.git`, `class="branch">main`, "docs", "src"}, nil},
		{"subdir", "/demo/tree/src", 200, []string{`main.go`}, nil},
		{"text file escaped", "/demo/blob/src/main.go", 200, []string{"package main", "// &lt;b&gt;hi&lt;/b&gt;"}, []string{"<b>hi</b>"}},
		{"markdown file relative link", "/demo/blob/docs/guide.md", 200, []string{`href="/demo/blob/README.md"`}, nil},
		{"binary note", "/demo/blob/bin.dat", 200, []string{"binary"}, nil},
		{"image view", "/demo/blob/img/logo.png", 200, []string{`<img src="/demo/raw/img/logo.png"`}, nil},
		{"blob dir redirects", "/demo/blob/src", 302, nil, nil},
		{"tree file redirects", "/demo/tree/src/main.go", 302, nil, nil},
		{"empty repo", "/blank/tree/", 200, []string{"Empty repository", "ssh://box.local:2222/blank.git"}, nil},
		{"missing repo", "/nope/tree/", 404, nil, nil},
		{"missing path", "/demo/tree/zzz", 404, nil, nil},
		{"traversal", "/demo/blob/../../etc/passwd", 400, nil, nil},
		{"dotdot repo", "/../tree/", 404, nil, nil},
		{"encoded slash repo", "/a%2F..%2Fdemo/tree/", 404, nil, nil},
		{"unknown action", "/demo/zzz/", 404, nil, nil},
		{"static css", "/_static/style.css", 200, []string{"markdown-body"}, nil},
		{"raw", "/demo/raw/src/main.go", 200, []string{"package main"}, nil},
		{"raw missing", "/demo/raw/nope", 404, nil, nil},
		{"raw dir", "/demo/raw/src", 404, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(s, tt.url)
			body := rec.Body.String()
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d\n%s", rec.Code, tt.status, body)
			}
			for _, w := range tt.want {
				if !strings.Contains(body, w) {
					t.Errorf("missing %q in body:\n%s", w, body)
				}
			}
			for _, n := range tt.not {
				if strings.Contains(body, n) {
					t.Errorf("unexpected %q in body", n)
				}
			}
		})
	}
}

func TestSidebarOpensAncestors(t *testing.T) {
	s := newTestServer(t)
	body := get(s, "/demo/blob/src/main.go").Body.String()
	if !strings.Contains(body, `<details open><summary><a href="/demo/tree/src">src</a>`) {
		t.Errorf("src not open in sidebar:\n%s", body)
	}
	if strings.Contains(body, `<details open><summary><a href="/demo/tree/docs">`) {
		t.Errorf("docs should be closed")
	}
	if !strings.Contains(body, `class="active"`) {
		t.Errorf("current file not marked active")
	}
}

func TestRawSVGIsNeutralized(t *testing.T) {
	s := newTestServer(t)
	rec := get(s, "/demo/raw/x.svg")
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
		t.Errorf("raw responses need a sandbox CSP, got %q", csp)
	}
}

func TestReadOnly(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/demo/tree/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status = %d, want 405", rec.Code)
	}
}
