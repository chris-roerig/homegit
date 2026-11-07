package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chris-roerig/homegit/internal/config"
)

func List(cfg *config.Config) error {
	// If the server is this machine, read local directory
	if isLocalServer(cfg) {
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
	repos, err := remoteRepoList(cfg)
	if err != nil {
		return err
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
