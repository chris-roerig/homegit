package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/chris-roerig/homegit/internal/config"
)

func Init() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current directory name for repo name
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	repoName := filepath.Base(cwd)

	// Check if already a git repo
	if _, err := os.Stat(".git"); err == nil {
		fmt.Println("Git repository already initialized")
	} else {
		// Initialize git repo with main as default branch
		cmd := exec.Command("git", "init", "-b", "main")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
		fmt.Println("Initialized git repository")
	}

	// Add remote
	remoteURL := fmt.Sprintf("ssh://%s:%d/%s.git", cfg.ServerHost, cfg.Port, repoName)
	cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Remote might already exist, try to set-url instead
		cmd = exec.Command("git", "remote", "set-url", "origin", remoteURL)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
		fmt.Println("Updated 'origin' remote")
	} else {
		fmt.Println("Added 'origin' remote")
	}

	// Print helpful message
	fmt.Printf("\n✓ Repository initialized: %s\n", repoName)
	fmt.Printf("✓ Remote added: %s\n\n", remoteURL)
	fmt.Println("Next steps:")
	fmt.Println("  1. Add files:       git add .")
	fmt.Printf("  2. Commit:          git commit -m \"Initial commit\"\n")
	fmt.Println("  3. Push to server:  git push -u origin main")

	// Check if server is reachable
	fmt.Println("\nChecking homegit server...")
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.Port), 2*time.Second)
	if err != nil {
		fmt.Println("⚠ Server not reachable. Start it with:")
		fmt.Printf("  homegit start    # Run on your homegit server (%s)\n", cfg.ServerHost)
	} else {
		if err := conn.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close connection: %v\n", err)
		}
		fmt.Println("✓ Server is running and reachable")
	}

	return nil
}
