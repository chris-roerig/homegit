package cmd

import (
	"bufio"
	"fmt"
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
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		choice, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || choice < 1 || choice > len(repos) {
			return fmt.Errorf("invalid selection")
		}

		repoName = repos[choice-1]
	}

	if !strings.HasSuffix(repoName, ".git") {
		repoName = repoName + ".git"
	}

	repoPath := filepath.Join(cfg.ReposDir, repoName)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository not found: %s", repoName)
	}

	targetDir := strings.TrimSuffix(repoName, ".git")

	cmd := exec.Command("git", "clone", fmt.Sprintf("ssh://%s:%d/%s", cfg.ServerHost, cfg.Port, repoName), targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func getRepoList(cfg *config.Config) ([]string, error) {
	// If server_host is localhost, read local directory
	if cfg.ServerHost == "localhost" || cfg.ServerHost == "127.0.0.1" {
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
	// Use SSH to list repos on remote server
	cmd := exec.Command("ssh", "-p", strconv.Itoa(cfg.Port), cfg.ServerHost, "ls", "-1", cfg.ReposDir)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list remote repos: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	repos := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasSuffix(line, ".git") {
			repos = append(repos, line)
		}
	}

	return repos, nil
}
