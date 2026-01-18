package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chris-roerig/homegit/internal/config"
)

func Clone(cfg *config.Config, repoName string) error {
	if repoName == "" {
		return fmt.Errorf("repository name required")
	}

	if !strings.HasSuffix(repoName, ".git") {
		repoName = repoName + ".git"
	}

	repoPath := filepath.Join(cfg.ReposDir, repoName)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository not found: %s", repoName)
	}

	targetDir := strings.TrimSuffix(repoName, ".git")

	cmd := exec.Command("git", "clone", fmt.Sprintf("ssh://localhost:%d/%s", cfg.Port, repoName), targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
