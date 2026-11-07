package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chris-roerig/homegit/internal/config"
)

// normalizeRepoName converts a raw name (typically a parent folder name) into
// a clean repository name: lowercased, with runs of whitespace, hyphens, and
// dots collapsed to a single underscore, any other non-alphanumeric characters
// stripped, and duplicate/edge underscores removed. For example:
//
//	"Foo Bar"   -> "foo_bar"
//	"  My-App. " -> "my_app"
//
// It returns an empty string if nothing usable remains.
func normalizeRepoName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))

	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '\t' || r == '-' || r == '.' || r == '_':
			// Separators collapse to underscore (dedup handled below).
			b.WriteRune('_')
		default:
			// Drop any other character (punctuation, unicode, etc.).
		}
	}

	// Collapse consecutive underscores and trim leading/trailing ones.
	parts := strings.Split(b.String(), "_")
	cleaned := parts[:0]
	for _, p := range parts {
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, "_")
}

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
	repoName := normalizeRepoName(filepath.Base(cwd))
	if repoName == "" {
		return fmt.Errorf("could not derive a valid repository name from %q", filepath.Base(cwd))
	}

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
	remoteURL := fmt.Sprintf("ssh://%s:%d/%s.git", cfg.ClientHost(), cfg.ClientPort(), repoName)
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
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cfg.ClientHost(), cfg.ClientPort()), 2*time.Second)
	if err != nil {
		fmt.Println("⚠ Server not reachable. Start it with:")
		fmt.Printf("  homegit start    # Run on your homegit server (%s)\n", cfg.ClientHost())
	} else {
		if err := conn.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close connection: %v\n", err)
		}
		fmt.Println("✓ Server is running and reachable")
	}

	return nil
}
