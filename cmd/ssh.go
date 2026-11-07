package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/chris-roerig/homegit/internal/config"
	"github.com/chris-roerig/homegit/internal/git"
)

// sshConnectTimeoutSecs bounds how long we wait for the TCP/SSH connection to
// establish before failing. This keeps commands from hanging indefinitely
// against an unreachable host.
const sshConnectTimeoutSecs = 5

// remoteRepoList asks the homegit server for its repository listing over SSH
// and returns the repo names (entries ending in ".git"). It applies a
// connection timeout and translates common SSH failures into actionable
// messages instead of surfacing a bare "exit status N".
func remoteRepoList(cfg *config.Config) ([]string, error) {
	args := []string{
		"-p", strconv.Itoa(cfg.Port),
		"-o", "BatchMode=yes",
		// homegit is a no-auth server for trusted networks and uses an
		// auto-generated host key, so auto-accept a new host key rather than
		// failing a non-interactive connection on first contact. A *changed*
		// key still fails, which surfaces unexpected host identity changes.
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", fmt.Sprintf("ConnectTimeout=%d", sshConnectTimeoutSecs),
		cfg.ServerHost,
		git.ListCommand,
	}

	cmd := exec.Command("ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, classifySSHError(cfg, err, stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	repos := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasSuffix(line, ".git") {
			repos = append(repos, line)
		}
	}

	return repos, nil
}

// classifySSHError converts a failed ssh invocation into a user-facing error
// with guidance. stderr is the captured standard error from ssh.
func classifySSHError(cfg *config.Config, runErr error, stderr string) error {
	target := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.Port)
	s := strings.ToLower(stderr)
	trimmed := strings.TrimSpace(stderr)

	switch {
	case strings.Contains(s, "connection refused"):
		return fmt.Errorf("could not connect to %s: connection refused. Is homegit running on the server? Try 'homegit status' on the server, or 'homegit start'", target)
	case strings.Contains(s, "connection timed out") || strings.Contains(s, "operation timed out") || strings.Contains(s, "timed out"):
		return fmt.Errorf("could not connect to %s: connection timed out. Check that the server is powered on and reachable on your network", target)
	case strings.Contains(s, "no route to host"):
		return fmt.Errorf("could not reach %s: no route to host. Check the server_host address and your network connection", target)
	case strings.Contains(s, "could not resolve") || strings.Contains(s, "name or service not known") || strings.Contains(s, "nodename nor servname"):
		return fmt.Errorf("could not resolve host %q. Check server_host in your config (run 'homegit config')", cfg.ServerHost)
	case strings.Contains(s, "permission denied"):
		return fmt.Errorf("connection to %s was refused (permission denied). Verify the server configuration", target)
	case strings.Contains(s, "host key verification failed") || strings.Contains(s, "remote host identification has changed"):
		return fmt.Errorf("host key for %s has changed or could not be verified. If the server was reinstalled, remove its entry from ~/.ssh/known_hosts and try again", target)
	}

	// Fall back to including captured stderr when available so the user sees
	// the underlying cause rather than only an exit code.
	if trimmed != "" {
		return fmt.Errorf("failed to list remote repos from %s: %s", target, trimmed)
	}
	return fmt.Errorf("failed to list remote repos from %s: %w", target, runErr)
}
