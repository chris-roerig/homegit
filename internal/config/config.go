package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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
