package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chris-roerig/homegit/internal/config"
)

func Remove(cfg *config.Config, repoName string) error {
	// If no repo name provided, show interactive menu
	if repoName == "" {
		return removeInteractive(cfg)
	}

	if !strings.HasSuffix(repoName, ".git") {
		repoName = repoName + ".git"
	}

	repoPath := filepath.Join(cfg.ReposDir, repoName)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository not found: %s", repoName)
	}

	fmt.Printf("Remove repository '%s'? (y/N): ", repoName)
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" {
		fmt.Println("Cancelled")
		return nil
	}

	if err := os.RemoveAll(repoPath); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	fmt.Printf("Repository '%s' removed\n", repoName)
	return nil
}

func removeInteractive(cfg *config.Config) error {
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

	fmt.Println("Select a repository to remove:")
	for i, repo := range repos {
		fmt.Printf("  %d) %s\n", i+1, repo)
	}
	fmt.Printf("  0) Cancel\n\n")
	fmt.Print("Enter number: ")

	var choice int
	_, err = fmt.Scanln(&choice)
	if err != nil || choice < 0 || choice > len(repos) {
		fmt.Println("Invalid selection")
		return nil
	}

	if choice == 0 {
		fmt.Println("Cancelled")
		return nil
	}

	selectedRepo := repos[choice-1]
	repoPath := filepath.Join(cfg.ReposDir, selectedRepo)

	fmt.Printf("\nRemove repository '%s'? (y/N): ", selectedRepo)
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" {
		fmt.Println("Cancelled")
		return nil
	}

	if err := os.RemoveAll(repoPath); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	fmt.Printf("Repository '%s' removed\n", selectedRepo)
	return nil
}
