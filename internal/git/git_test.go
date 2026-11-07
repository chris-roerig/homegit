package git

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantType    string
		wantRepo    string
		expectError bool
	}{
		{
			name:        "upload-pack with single quotes",
			input:       "git-upload-pack 'my-repo.git'",
			wantType:    "upload-pack",
			wantRepo:    "my-repo.git",
			expectError: false,
		},
		{
			name:        "receive-pack with double quotes",
			input:       "git-receive-pack \"my-repo.git\"",
			wantType:    "receive-pack",
			wantRepo:    "my-repo.git",
			expectError: false,
		},
		{
			name:        "upload-pack without quotes",
			input:       "git-upload-pack my-repo.git",
			wantType:    "upload-pack",
			wantRepo:    "my-repo.git",
			expectError: false,
		},
		{
			name:        "invalid command",
			input:       "git-invalid my-repo.git",
			expectError: true,
		},
		{
			name:        "missing repo path",
			input:       "git-upload-pack",
			expectError: true,
		},
		{
			name:        "list command",
			input:       ListCommand,
			wantType:    "list",
			wantRepo:    "",
			expectError: false,
		},
		{
			name:        "list command ignores extra args",
			input:       ListCommand + " extra ignored",
			wantType:    "list",
			wantRepo:    "",
			expectError: false,
		},
		{
			name:        "empty command",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCommand(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cmd.Type != tt.wantType {
				t.Errorf("Expected type %s, got %s", tt.wantType, cmd.Type)
			}

			if cmd.RepoPath != tt.wantRepo {
				t.Errorf("Expected repo %s, got %s", tt.wantRepo, cmd.RepoPath)
			}
		})
	}
}

func TestExecuteList(t *testing.T) {
	reposDir := t.TempDir()

	// Create a mix of bare repos, a non-.git dir, and a file.
	for _, name := range []string{"alpha.git", "beta.git"} {
		if err := os.MkdirAll(filepath.Join(reposDir, name), 0755); err != nil {
			t.Fatalf("failed to create repo dir %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(reposDir, "not-a-repo"), 0755); err != nil {
		t.Fatalf("failed to create non-repo dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reposDir, "stray.git"), []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create stray file: %v", err)
	}

	cmd := &Command{Type: "list"}
	var out bytes.Buffer
	if err := cmd.Execute(reposDir, nil, &out, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := []string{}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line != "" {
			got = append(got, line)
		}
	}
	sort.Strings(got)

	want := []string{"alpha.git", "beta.git"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected %v, got %v", want, got)
		}
	}
}

func TestExecuteListMissingDir(t *testing.T) {
	// A non-existent repos directory should list nothing, not error.
	cmd := &Command{Type: "list"}
	var out bytes.Buffer
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if err := cmd.Execute(missing, nil, &out, &out); err != nil {
		t.Fatalf("unexpected error for missing dir: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("expected empty output, got %q", out.String())
	}
}
