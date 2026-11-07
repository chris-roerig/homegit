package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Port != 2222 {
		t.Errorf("Expected default port 2222, got %d", cfg.Port)
	}

	if cfg.DefaultBranch != "main" {
		t.Errorf("Expected default branch 'main', got %s", cfg.DefaultBranch)
	}

	if cfg.Daemon != false {
		t.Errorf("Expected daemon false, got %v", cfg.Daemon)
	}
}

func TestLoadAndSave(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	home := os.Getenv("HOME")
	defer os.Setenv("HOME", home)
	os.Setenv("HOME", tmpDir)

	// Load should create default config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify config file was created
	configPath := filepath.Join(tmpDir, ".homegit", "config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created at %s", configPath)
	}

	// Verify default values
	if cfg.Port != 2222 {
		t.Errorf("Expected port 2222, got %d", cfg.Port)
	}

	// Modify and save
	cfg.Port = 3333
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load again and verify changes persisted
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if cfg2.Port != 3333 {
		t.Errorf("Expected port 3333 after reload, got %d", cfg2.Port)
	}
}

func TestNormalizeServerHost(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "plain host", input: "example.local", want: "example.local"},
		{name: "plain ip", input: "10.0.0.142", want: "10.0.0.142"},
		{name: "localhost unchanged", input: "localhost", want: "localhost"},
		{name: "http scheme", input: "http://10.0.0.142", want: "10.0.0.142"},
		{name: "https scheme", input: "https://host.local", want: "host.local"},
		{name: "ssh scheme", input: "ssh://10.0.0.142", want: "10.0.0.142"},
		{name: "git scheme", input: "git://host", want: "host"},
		{name: "scheme case insensitive", input: "HTTP://Host", want: "Host"},
		{name: "trailing dot", input: "host.local.", want: "host.local"},
		{name: "surrounding whitespace", input: "  10.0.0.142  ", want: "10.0.0.142"},
		{name: "scheme with path", input: "http://10.0.0.142/repos", want: "10.0.0.142"},
		{name: "trailing slash", input: "host/", want: "host"},
		{name: "combo scheme ws dot", input: "  ssh://host.local.  ", want: "host.local"},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "interior space", input: "bad host", wantErr: true},
		{name: "scheme only", input: "http://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeServerHost(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizeServerHost(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizeServerHost(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeServerHost(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
