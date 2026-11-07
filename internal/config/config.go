package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Port          int    `json:"port"`
	ServerHost    string `json:"server_host"`
	ReposDir      string `json:"repos_dir"`
	Daemon        bool   `json:"daemon"`
	HostKey       string `json:"host_key"`
	PIDFile       string `json:"pid_file"`
	DefaultBranch string `json:"default_branch"`
	BackupDir     string `json:"backup_dir"`
}

// serverHostSchemes are URL-style scheme prefixes that users sometimes paste
// into server_host but that break SSH (which expects a bare host or IP).
var serverHostSchemes = []string{"ssh://", "git://", "https://", "http://"}

// NormalizeServerHost cleans a user-supplied server_host value into a bare
// hostname or IP suitable for use as an SSH destination. It trims surrounding
// whitespace, strips URL-style scheme prefixes (e.g. "http://"), removes any
// trailing dots, and strips a trailing slash or path component. It returns an
// error if the result is empty or contains interior whitespace, which SSH
// cannot handle.
func NormalizeServerHost(host string) (string, error) {
	h := strings.TrimSpace(host)
	if h == "" {
		return "", fmt.Errorf("server_host is empty")
	}

	// Strip a known scheme prefix if present (case-insensitive).
	for _, scheme := range serverHostSchemes {
		if len(h) >= len(scheme) && strings.EqualFold(h[:len(scheme)], scheme) {
			h = h[len(scheme):]
			break
		}
	}

	// Drop any path component (e.g. "host/path" -> "host").
	if i := strings.IndexByte(h, '/'); i >= 0 {
		h = h[:i]
	}

	// Remove trailing dots (e.g. "host." -> "host").
	h = strings.TrimRight(h, ".")

	if h == "" {
		return "", fmt.Errorf("server_host %q has no host component", host)
	}
	if strings.ContainsAny(h, " \t") {
		return "", fmt.Errorf("invalid server_host %q: cannot contain spaces", host)
	}

	return h, nil
}

func getHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func GetHomeDir() string {
	return getHomeDir()
}

func Default() *Config {
	home := getHomeDir()
	baseDir := filepath.Join(home, ".homegit")
	return &Config{
		Port:          2222,
		ServerHost:    "localhost",
		ReposDir:      filepath.Join(baseDir, "repos"),
		Daemon:        false,
		HostKey:       filepath.Join(baseDir, "host_key"),
		PIDFile:       filepath.Join(baseDir, "homegit.pid"),
		DefaultBranch: "main",
		BackupDir:     filepath.Join(baseDir, "backups"),
	}
}

func Load() (*Config, error) {
	home := getHomeDir()
	configPath := filepath.Join(home, ".homegit", "config")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := Default()
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Normalize server_host so stale/malformed values (e.g. "http://host",
	// "host.", or stray whitespace) don't cause cryptic SSH failures later.
	normalized, err := NormalizeServerHost(cfg.ServerHost)
	if err != nil {
		return nil, fmt.Errorf("invalid server_host in config: %w", err)
	}
	cfg.ServerHost = normalized

	return cfg, nil
}

func (c *Config) Save() error {
	home := getHomeDir()
	baseDir := filepath.Join(home, ".homegit")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(baseDir, "config")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
