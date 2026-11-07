package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	// Top-level, role-agnostic settings shared by server and client roles.
	DefaultBranch string `json:"default_branch"`
	BackupDir     string `json:"backup_dir"`
	// Name identifies this homegit device. Reserved for future use (device
	// identification in client output); defaults to the hostname. Not yet
	// exchanged between instances.
	Name string `json:"name,omitempty"`

	// Server describes this machine acting as a host ("me as a host"). Used by
	// serve/start/stop/status/restart.
	Server ServerConfig `json:"server"`
	// Client describes this machine's upstream ("where I pull/push from"). Used
	// by list/clone/init. When Host is empty, client commands operate against
	// the local server.
	Client ClientConfig `json:"client"`
}

// ServerConfig holds settings used when this machine hosts repositories.
type ServerConfig struct {
	Port     int    `json:"port"`
	ReposDir string `json:"repos_dir"`
	HostKey  string `json:"host_key"`
	PIDFile  string `json:"pid_file"`
}

// ClientConfig holds settings used when connecting to a (possibly remote)
// homegit server.
type ClientConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// IsRemoteClient reports whether the client is configured to talk to a remote
// server (a non-empty, non-local host). When false, client commands should
// operate against the local server.
func (c *Config) IsRemoteClient() bool {
	h := c.Client.Host
	return h != "" && h != "localhost" && h != "127.0.0.1"
}

// ClientPort returns the port client commands should connect to, falling back
// to the server's listen port when the client port is unset (e.g. a pure
// local-server setup).
func (c *Config) ClientPort() int {
	if c.Client.Port != 0 {
		return c.Client.Port
	}
	return c.Server.Port
}

// ClientHost returns the host client commands should connect to, defaulting to
// localhost when no remote client host is configured.
func (c *Config) ClientHost() string {
	if c.Client.Host != "" {
		return c.Client.Host
	}
	return "localhost"
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
	hostname, _ := os.Hostname()
	return &Config{
		DefaultBranch: "main",
		BackupDir:     filepath.Join(baseDir, "backups"),
		Name:          hostname,
		Server: ServerConfig{
			Port:     2222,
			ReposDir: filepath.Join(baseDir, "repos"),
			HostKey:  filepath.Join(baseDir, "host_key"),
			PIDFile:  filepath.Join(baseDir, "homegit.pid"),
		},
		Client: ClientConfig{
			// Empty host => operate against the local server by default.
			Host: "",
			Port: 2222,
		},
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

	// Detect whether this is the legacy flat format (pre-1.1.0). The legacy
	// format has a top-level "server_host" scalar and no "server" object.
	if isLegacyConfig(data) {
		cfg, err := migrateLegacyConfig(data)
		if err != nil {
			return nil, err
		}
		// Persist the migrated config, backing up the original first so the
		// user can recover it if needed.
		if err := backupConfigFile(configPath); err != nil {
			return nil, fmt.Errorf("failed to back up config before migration: %w", err)
		}
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to write migrated config: %w", err)
		}
		return cfg, nil
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Normalize the client host so stale/malformed values (e.g. "http://host",
	// "host.", or stray whitespace) don't cause cryptic SSH failures later.
	if cfg.Client.Host != "" {
		normalized, err := NormalizeServerHost(cfg.Client.Host)
		if err != nil {
			return nil, fmt.Errorf("invalid client host in config: %w", err)
		}
		cfg.Client.Host = normalized
	}

	return cfg, nil
}

// legacyConfig mirrors the flat pre-1.1.0 config schema. It is used only for
// migration to the current sectioned schema.
type legacyConfig struct {
	Port          int    `json:"port"`
	ServerHost    string `json:"server_host"`
	ReposDir      string `json:"repos_dir"`
	HostKey       string `json:"host_key"`
	PIDFile       string `json:"pid_file"`
	DefaultBranch string `json:"default_branch"`
	BackupDir     string `json:"backup_dir"`
}

// isLegacyConfig reports whether raw config JSON is in the legacy flat format.
// The legacy format is identified by the presence of a top-level "server_host"
// key and the absence of a "server" object.
func isLegacyConfig(data []byte) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		// If it doesn't parse as an object we can't migrate it; let the normal
		// path surface the error.
		return false
	}
	_, hasServerSection := probe["server"]
	_, hasServerHost := probe["server_host"]
	return hasServerHost && !hasServerSection
}

// migrateLegacyConfig converts a legacy flat config into the current sectioned
// schema. Role inference matches the old behavior: server_host of localhost/
// 127.0.0.1 means this machine is the host; any other value means it is a
// client pointing at that remote.
func migrateLegacyConfig(data []byte) (*Config, error) {
	var legacy legacyConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("failed to parse legacy config: %w", err)
	}

	// Start from defaults so any field the legacy config omitted is populated.
	cfg := Default()

	if legacy.DefaultBranch != "" {
		cfg.DefaultBranch = legacy.DefaultBranch
	}
	if legacy.BackupDir != "" {
		cfg.BackupDir = legacy.BackupDir
	}
	if legacy.ReposDir != "" {
		cfg.Server.ReposDir = legacy.ReposDir
	}
	if legacy.HostKey != "" {
		cfg.Server.HostKey = legacy.HostKey
	}
	if legacy.PIDFile != "" {
		cfg.Server.PIDFile = legacy.PIDFile
	}

	host := strings.TrimSpace(legacy.ServerHost)
	if host == "" || host == "localhost" || host == "127.0.0.1" {
		// This machine is the server.
		if legacy.Port != 0 {
			cfg.Server.Port = legacy.Port
		}
		cfg.Client.Host = ""
		cfg.Client.Port = cfg.Server.Port
	} else {
		// This machine is a client pointing at a remote server. Normalize the
		// host so a stale "http://" style value is cleaned up during migration.
		normalized, err := NormalizeServerHost(host)
		if err != nil {
			// Fall back to the raw host rather than failing the migration; the
			// user can fix it later (e.g. via a future config/doctor command).
			normalized = host
		}
		cfg.Client.Host = normalized
		if legacy.Port != 0 {
			cfg.Client.Port = legacy.Port
		}
		// Keep server defaults so the machine can still host locally if desired.
	}

	return cfg, nil
}

// backupConfigFile copies the config at path to "<path>.bak" before it is
// overwritten by a migration.
func backupConfigFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path+".bak", data, 0644)
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
