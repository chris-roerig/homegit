package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/chris-roerig/homegit/internal/config"
)

func Config() error {
	home := config.GetHomeDir()
	configPath := filepath.Join(home, ".homegit", "config")

	// Ensure config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := config.Default()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
	}

	// Get editor from environment, fallback to vi
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Open config in editor
	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Println("\nConfig updated. Restart homegit for changes to take effect:")
	fmt.Println("  homegit restart")

	return nil
}
