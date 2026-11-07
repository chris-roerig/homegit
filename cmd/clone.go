package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chris-roerig/homegit/internal/config"
)

func Clone(cfg *config.Config, repoName string) error {
	// If no repo name provided, show interactive list
	if repoName == "" {
		repos, err := getRepoList(cfg)
		if err != nil {
			return err
		}

		if len(repos) == 0 {
			return fmt.Errorf("no repositories found")
		}

		// Show interactive menu
		fmt.Println("Available repositories:")
		for i, repo := range repos {
			fmt.Printf("  %d. %s\n", i+1, strings.TrimSuffix(repo, ".git"))
		}
		fmt.Print("\nSelect repository (number): ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		// Treat EOF (e.g. piped/empty stdin or Ctrl-D) with any partial input
		// still usable; only a hard read error is fatal.
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			return fmt.Errorf("no selection made; run 'homegit clone <name>' or choose a number from the list")
		}

		choice, convErr := strconv.Atoi(input)
		if convErr != nil || choice < 1 || choice > len(repos) {
			return fmt.Errorf("invalid selection %q: enter a number between 1 and %d", input, len(repos))
		}

		repoName = repos[choice-1]
	}

	if !strings.HasSuffix(repoName, ".git") {
		repoName = repoName + ".git"
	}

	// Only validate repository existence locally when the server is this
	// machine. For a remote server the repo lives on the server's disk, not in
	// our local ReposDir, so a local stat would wrongly report "not found".
	if isLocalServer(cfg) {
		repoPath := filepath.Join(cfg.ReposDir, repoName)
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			return fmt.Errorf("repository not found: %s", repoName)
		}
	}

	targetDir := strings.TrimSuffix(repoName, ".git")

	// Avoid clobbering an existing directory; git clone would fail with a less
	// obvious error.
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("destination path %q already exists", targetDir)
	}

	cmd := exec.Command("git", "clone", fmt.Sprintf("ssh://%s:%d/%s", cfg.ServerHost, cfg.Port, repoName), targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// isLocalServer reports whether the configured server is this machine.
func isLocalServer(cfg *config.Config) bool {
	return cfg.ServerHost == "localhost" || cfg.ServerHost == "127.0.0.1"
}

func getRepoList(cfg *config.Config) ([]string, error) {
	// If the server is this machine, read the local directory
	if isLocalServer(cfg) {
		return getLocalRepoList(cfg.ReposDir)
	}

	// Otherwise, get list from remote server via SSH
	return getRemoteRepoList(cfg)
}

func getLocalRepoList(reposDir string) ([]string, error) {
	if err := os.MkdirAll(reposDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repos directory: %w", err)
	}

	entries, err := os.ReadDir(reposDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read repos directory: %w", err)
	}

	repos := []string{}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".git") {
			repos = append(repos, entry.Name())
		}
	}

	return repos, nil
}

func getRemoteRepoList(cfg *config.Config) ([]string, error) {
	return remoteRepoList(cfg)
}
