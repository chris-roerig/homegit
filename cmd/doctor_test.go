package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chris-roerig/homegit/internal/config"
)

func baseCfg(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Server.ReposDir = filepath.Join(dir, "repos")
	cfg.Server.HostKey = filepath.Join(dir, "host_key")
	cfg.Server.PIDFile = filepath.Join(dir, "homegit.pid")
	cfg.Server.Port = 2222
	cfg.BackupDir = filepath.Join(dir, "backups")
	return cfg
}

func findResult(results []checkResult, name string) *checkResult {
	for i := range results {
		if results[i].name == name {
			return &results[i]
		}
	}
	return nil
}

func TestDoctorMissingDirsHaveFix(t *testing.T) {
	cfg := baseCfg(t)
	results := runDoctorChecks(cfg)

	repos := findResult(results, "repos directory")
	if repos == nil || repos.status != statusWarn || repos.fix == nil {
		t.Fatalf("expected repos directory warn with fix, got %+v", repos)
	}
	// Applying the fix should create the directory.
	if _, err := repos.fix(); err != nil {
		t.Fatalf("repos fix failed: %v", err)
	}
	if info, err := os.Stat(cfg.Server.ReposDir); err != nil || !info.IsDir() {
		t.Errorf("repos dir should exist after fix")
	}
}

func TestDoctorStalePID(t *testing.T) {
	cfg := baseCfg(t)
	// Write a PID for a process that does not exist.
	if err := os.WriteFile(cfg.Server.PIDFile, []byte("999999"), 0644); err != nil {
		t.Fatal(err)
	}
	results := runDoctorChecks(cfg)
	pid := findResult(results, "pid file")
	if pid == nil || pid.status != statusWarn || pid.fix == nil {
		t.Fatalf("expected stale pid warn with fix, got %+v", pid)
	}
	if _, err := pid.fix(); err != nil {
		t.Fatalf("pid fix failed: %v", err)
	}
	if _, err := os.Stat(cfg.Server.PIDFile); !os.IsNotExist(err) {
		t.Errorf("stale PID file should be removed after fix")
	}
}

func TestDoctorInvalidClientHost(t *testing.T) {
	cfg := baseCfg(t)
	cfg.Client.Host = "bad host" // contains a space -> invalid
	results := runDoctorChecks(cfg)
	ch := findResult(results, "client host")
	if ch == nil || ch.status != statusFail {
		t.Fatalf("expected client host fail, got %+v", ch)
	}
}

func TestDoctorHealthyLocal(t *testing.T) {
	cfg := baseCfg(t)
	// Create everything a healthy local server needs.
	os.MkdirAll(cfg.Server.ReposDir, 0755)
	os.MkdirAll(cfg.BackupDir, 0755)
	os.WriteFile(cfg.Server.HostKey, []byte("key"), 0600)

	results := runDoctorChecks(cfg)
	for _, r := range results {
		if r.status == statusFail {
			t.Errorf("unexpected failure in healthy local config: %s: %s", r.name, r.message)
		}
	}
}
