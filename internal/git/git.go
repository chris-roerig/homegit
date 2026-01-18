package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	repoCreateMutex sync.Mutex
)

type Command struct {
	Type     string // "upload-pack" or "receive-pack"
	RepoPath string
}

func ParseCommand(cmdStr string) (*Command, error) {
	parts := strings.Fields(cmdStr)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid git command: %s", cmdStr)
	}

	var cmdType string
	switch parts[0] {
	case "git-upload-pack":
		cmdType = "upload-pack"
	case "git-receive-pack":
		cmdType = "receive-pack"
	default:
		return nil, fmt.Errorf("unsupported command: %s", parts[0])
	}

	repoPath := strings.Trim(parts[1], "'\"")
	return &Command{Type: cmdType, RepoPath: repoPath}, nil
}

func (c *Command) Execute(reposDir string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Clean and validate repository path to prevent directory traversal
	cleanPath := filepath.Clean(c.RepoPath)
	if strings.Contains(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return fmt.Errorf("invalid repository path: %s", c.RepoPath)
	}

	fullPath := filepath.Join(reposDir, cleanPath)

	// Ensure the resolved path is still within repos directory
	absReposDir, err := filepath.Abs(reposDir)
	if err != nil {
		return fmt.Errorf("failed to resolve repos directory: %w", err)
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}
	if !strings.HasPrefix(absFullPath, absReposDir) {
		return fmt.Errorf("repository path outside repos directory: %s", c.RepoPath)
	}

	if c.Type == "receive-pack" {
		if err := ensureRepo(fullPath); err != nil {
			return err
		}
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("repository not found: %s", c.RepoPath)
	}

	cmd := exec.Command("git-"+c.Type, fullPath)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

func ensureRepo(path string) error {
	// Lock to prevent race condition when multiple pushes try to create same repo
	repoCreateMutex.Lock()
	defer repoCreateMutex.Unlock()
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		cmd := exec.Command("git", "init", "--bare", path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to init repository: %w", err)
		}
		// Set default branch to main
		cmd = exec.Command("git", "-C", path, "symbolic-ref", "HEAD", "refs/heads/main")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set default branch: %w", err)
		}
	}
	return nil
}
