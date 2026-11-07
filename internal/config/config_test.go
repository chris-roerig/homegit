package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Server.Port != 2222 {
		t.Errorf("Expected default server port 2222, got %d", cfg.Server.Port)
	}

	if cfg.DefaultBranch != "main" {
		t.Errorf("Expected default branch 'main', got %s", cfg.DefaultBranch)
	}

	if cfg.IsRemoteClient() {
		t.Errorf("Expected default config to not be a remote client")
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
	if cfg.Server.Port != 2222 {
		t.Errorf("Expected port 2222, got %d", cfg.Server.Port)
	}

	// Modify and save
	cfg.Server.Port = 3333
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load again and verify changes persisted
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if cfg2.Server.Port != 3333 {
		t.Errorf("Expected port 3333 after reload, got %d", cfg2.Server.Port)
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

// writeConfig writes raw config JSON into a temp HOME and returns the config path.
func writeConfig(t *testing.T, raw string) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	prevHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	dir := filepath.Join(tmpDir, ".homegit")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path, func() { os.Setenv("HOME", prevHome) }
}

func TestIsLegacyConfig(t *testing.T) {
	legacy := `{"port":2222,"server_host":"localhost","repos_dir":"/x"}`
	if !isLegacyConfig([]byte(legacy)) {
		t.Errorf("expected legacy format to be detected")
	}
	modern := `{"default_branch":"main","server":{"port":2222},"client":{"host":""}}`
	if isLegacyConfig([]byte(modern)) {
		t.Errorf("modern format should not be detected as legacy")
	}
	// A config with neither marker is not legacy.
	neither := `{"default_branch":"main"}`
	if isLegacyConfig([]byte(neither)) {
		t.Errorf("config without server_host should not be legacy")
	}
}

func TestMigrateLegacyServer(t *testing.T) {
	// Legacy server (localhost) config.
	raw := `{
		"port": 2223,
		"server_host": "localhost",
		"repos_dir": "/home/chris/.homegit/repos",
		"host_key": "/home/chris/.homegit/host_key",
		"pid_file": "/home/chris/.homegit/homegit.pid",
		"default_branch": "trunk",
		"backup_dir": "/home/chris/.homegit/backups"
	}`
	path, cleanup := writeConfig(t, raw)
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load (migrate) failed: %v", err)
	}

	if cfg.Server.Port != 2223 {
		t.Errorf("server port: got %d, want 2223", cfg.Server.Port)
	}
	if cfg.Server.ReposDir != "/home/chris/.homegit/repos" {
		t.Errorf("repos_dir not migrated: %q", cfg.Server.ReposDir)
	}
	if cfg.DefaultBranch != "trunk" {
		t.Errorf("default_branch not migrated: %q", cfg.DefaultBranch)
	}
	if cfg.IsRemoteClient() {
		t.Errorf("localhost legacy config should not be a remote client")
	}
	// Backup should have been written.
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Errorf("expected backup file at %s.bak: %v", path, err)
	}
	// Reloading should now see the modern format (no second migration).
	data, _ := os.ReadFile(path)
	if isLegacyConfig(data) {
		t.Errorf("config should have been rewritten in modern format")
	}
}

func TestMigrateLegacyClient(t *testing.T) {
	// Legacy client (remote) config, with a messy scheme-prefixed host.
	raw := `{
		"port": 2200,
		"server_host": "http://10.0.0.142",
		"repos_dir": "/x/repos",
		"default_branch": "main"
	}`
	_, cleanup := writeConfig(t, raw)
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load (migrate) failed: %v", err)
	}

	if !cfg.IsRemoteClient() {
		t.Errorf("remote legacy config should be a remote client")
	}
	if cfg.Client.Host != "10.0.0.142" {
		t.Errorf("client host not normalized/migrated: %q", cfg.Client.Host)
	}
	if cfg.Client.Port != 2200 {
		t.Errorf("client port: got %d, want 2200", cfg.Client.Port)
	}
	// Server section should still have sane defaults for optional local hosting.
	if cfg.Server.Port == 0 {
		t.Errorf("server port should default, got 0")
	}
}

func TestModernConfigNoMigration(t *testing.T) {
	raw := `{
		"default_branch": "main",
		"backup_dir": "/x/backups",
		"name": "house",
		"server": {"port": 2222, "repos_dir": "/x/repos", "host_key": "/x/hk", "pid_file": "/x/pid"},
		"client": {"host": "10.0.0.9", "port": 2222}
	}`
	path, cleanup := writeConfig(t, raw)
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Client.Host != "10.0.0.9" {
		t.Errorf("client host: %q", cfg.Client.Host)
	}
	if cfg.Name != "house" {
		t.Errorf("name: %q", cfg.Name)
	}
	// Modern config must NOT create a backup (no migration happened).
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Errorf("modern config should not produce a .bak file")
	}
}

func TestClientHelpers(t *testing.T) {
	// Client port falls back to server port when unset.
	cfg := &Config{}
	cfg.Server.Port = 2222
	cfg.Client.Port = 0
	if cfg.ClientPort() != 2222 {
		t.Errorf("ClientPort fallback: got %d, want 2222", cfg.ClientPort())
	}
	cfg.Client.Port = 2200
	if cfg.ClientPort() != 2200 {
		t.Errorf("ClientPort: got %d, want 2200", cfg.ClientPort())
	}
	// ClientHost defaults to localhost when unset.
	if cfg.ClientHost() != "localhost" {
		t.Errorf("ClientHost default: got %q, want localhost", cfg.ClientHost())
	}
	cfg.Client.Host = "host.local"
	if cfg.ClientHost() != "host.local" {
		t.Errorf("ClientHost: got %q", cfg.ClientHost())
	}
}
