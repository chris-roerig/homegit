package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chris-roerig/homegit/internal/config"
)

func Backup(cfg *config.Config, repoName string) error {
	// If no repo name provided, show interactive menu
	if repoName == "" {
		return backupInteractive(cfg)
	}

	if !strings.HasSuffix(repoName, ".git") {
		repoName = repoName + ".git"
	}

	repoPath := filepath.Join(cfg.ReposDir, repoName)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository not found: %s", repoName)
	}

	return createBackup(cfg, repoName, repoPath)
}

func backupInteractive(cfg *config.Config) error {
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

	fmt.Println("Select a repository to backup:")
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

	return createBackup(cfg, selectedRepo, repoPath)
}

func createBackup(cfg *config.Config, repoName, repoPath string) error {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(cfg.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	baseName := strings.TrimSuffix(repoName, ".git")
	backupFile := filepath.Join(cfg.BackupDir, fmt.Sprintf("%s-%s.tar.gz", baseName, timestamp))

	fmt.Printf("Creating backup of '%s'...\n", repoName)

	// Create tar.gz archive
	cmd := exec.Command("tar", "-czf", backupFile, "-C", cfg.ReposDir, repoName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Get file size
	info, err := os.Stat(backupFile)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}

	size := float64(info.Size()) / 1024 / 1024 // Convert to MB
	fmt.Printf("Backup created: %s (%.2f MB)\n", backupFile, size)

	return nil
}
