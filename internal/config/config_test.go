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
