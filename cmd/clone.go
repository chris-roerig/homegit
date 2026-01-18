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
	if err := os.MkdirAll(cfg.ReposDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repos directory: %w", err)
	}

	entries, err := os.ReadDir(cfg.ReposDir)
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
