package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chris-roerig/homegit/internal/web"
)

func newGitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return dir
}

func runAddMeta(t *testing.T, dir, stdin string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := AddMeta(args, dir, strings.NewReader(stdin), &out)
	return out.String(), err
}

func readMeta(t *testing.T, dir string) *web.Meta {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, web.MetaFile))
	if err != nil {
		t.Fatal(err)
	}
	m, err := web.ParseMeta(data)
	if err != nil {
		t.Fatalf("written file does not parse: %v\n%s", err, data)
	}
	return m
}

func TestAddMetaFlags(t *testing.T) {
	dir := newGitDir(t)
	out, err := runAddMeta(t, dir, "", "--description", "A <tiny> tool", "--homepage", "https://example.com", "--topics", "Go, cli")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	m := readMeta(t, dir)
	if m.Description != "A <tiny> tool" || m.Homepage != "https://example.com" || strings.Join(m.Topics, ",") != "go,cli" {
		t.Errorf("unexpected meta: %+v", m)
	}
	data, _ := os.ReadFile(filepath.Join(dir, web.MetaFile))
	if strings.Contains(string(data), `\u003c`) {
		t.Errorf("HTML characters should not be escaped in the file:\n%s", data)
	}
	if !strings.Contains(out, "git add "+web.MetaFile) {
		t.Errorf("expected next-step hint, got:\n%s", out)
	}
}

func TestAddMetaPromptsAndRunsFromSubdir(t *testing.T) {
	dir := newGitDir(t)
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := runAddMeta(t, sub, "My project\n\ngo git-server\n"); err != nil {
		t.Fatal(err)
	}
	m := readMeta(t, dir) // written at the repo root, not the subdirectory
	if m.Description != "My project" || m.Homepage != "" || strings.Join(m.Topics, ",") != "go,git-server" {
		t.Errorf("unexpected meta: %+v", m)
	}
}

func TestAddMetaWarnsAboutDroppedValues(t *testing.T) {
	dir := newGitDir(t)
	out, err := runAddMeta(t, dir, "", "--description", "ok", "--homepage", "javascript:alert(1)", "--topics", "good,bad_topic")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ignored homepage") || !strings.Contains(out, "ignored topics bad_topic") {
		t.Errorf("missing warnings:\n%s", out)
	}
	m := readMeta(t, dir)
	if m.Homepage != "" || strings.Join(m.Topics, ",") != "good" {
		t.Errorf("unsafe values were written: %+v", m)
	}
}

func TestAddMetaExistingFile(t *testing.T) {
	dir := newGitDir(t)
	if _, err := runAddMeta(t, dir, "", "--description", "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := runAddMeta(t, dir, "", "--description", "second"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected already-exists error, got %v", err)
	}
	if readMeta(t, dir).Description != "first" {
		t.Error("existing file must not be modified without --force")
	}
	if _, err := runAddMeta(t, dir, "", "--force", "--description", "second"); err != nil {
		t.Fatal(err)
	}
	if readMeta(t, dir).Description != "second" {
		t.Error("--force should overwrite")
	}
}

func TestAddMetaErrors(t *testing.T) {
	if _, err := runAddMeta(t, t.TempDir(), "", "--description", "x"); err == nil || !strings.Contains(err.Error(), "not inside a git repository") {
		t.Errorf("expected not-a-repo error, got %v", err)
	}
	dir := newGitDir(t)
	if _, err := runAddMeta(t, dir, "\n\n\n"); err == nil || !strings.Contains(err.Error(), "nothing to write") {
		t.Errorf("expected nothing-to-write error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, web.MetaFile)); err == nil {
		t.Error("no file should be created when there is nothing to write")
	}
	if _, err := runAddMeta(t, dir, "", "stray"); err == nil {
		t.Error("positional arguments should be rejected")
	}
}
