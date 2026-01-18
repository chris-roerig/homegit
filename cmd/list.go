package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chris-roerig/homegit/internal/config"
)

func List(cfg *config.Config) error {
	// If server_host is localhost, read local directory
	if cfg.ServerHost == "localhost" || cfg.ServerHost == "127.0.0.1" {
		return listLocal(cfg)
	}
	
	// Otherwise, list from remote server via SSH
	return listRemote(cfg)
}

func listLocal(cfg *config.Config) error {
	if err := os.MkdirAll(cfg.ReposDir, 0755); err != nil {
		return fmt.Errorf("failed to create repos directory: %w", err)
	}

	entries, err := os.ReadDir(cfg.ReposDir)
	if err != nil {
		return fmt.Errorf("failed to read repos directory: %w", err)
	}

	repos := []string{}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".git") {
			repos = append(repos, entry.Name())
		}
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found")
		return nil
	}

	fmt.Printf("Repositories in %s:\n", cfg.ReposDir)
	for _, repo := range repos {
		fullPath := filepath.Join(cfg.ReposDir, repo)
		fmt.Printf("  %s\n    Path: %s\n", repo, fullPath)
	}

	return nil
}

func listRemote(cfg *config.Config) error {
	// Use SSH to list repos on remote server
	cmd := exec.Command("ssh", "-p", strconv.Itoa(cfg.Port), cfg.ServerHost, "ls", "-1", cfg.ReposDir)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list remote repos: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	repos := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasSuffix(line, ".git") {
			repos = append(repos, line)
		}
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found")
		return nil
	}

	fmt.Printf("Repositories on %s:\n", cfg.ServerHost)
	for _, repo := range repos {
		fmt.Printf("  %s\n", repo)
	}

	return nil
}
