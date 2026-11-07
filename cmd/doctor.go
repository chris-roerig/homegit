package cmd

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chris-roerig/homegit/internal/config"
	"github.com/chris-roerig/homegit/internal/daemon"
)

// checkStatus is the outcome of a single doctor check.
type checkStatus int

const (
	statusOK checkStatus = iota
	statusWarn
	statusFail
)

func (s checkStatus) symbol() string {
	switch s {
	case statusOK:
		return "✓"
	case statusWarn:
		return "⚠"
	default:
		return "✗"
	}
}

// checkResult captures the outcome of a check and an optional fix.
type checkResult struct {
	name    string
	status  checkStatus
	message string
	// fix, when non-nil, attempts to remediate the issue and returns a
	// human-readable description of what it did (or an error).
	fix func() (string, error)
}

// Doctor inspects the configuration and environment for common problems. With
// fix set, it attempts to remediate issues that have a known safe fix.
func Doctor(cfg *config.Config, fix bool) error {
	results := runDoctorChecks(cfg)

	var failures, warnings int
	for _, r := range results {
		fmt.Printf("%s %s: %s\n", r.status.symbol(), r.name, r.message)

		if r.status != statusOK && fix && r.fix != nil {
			desc, err := r.fix()
			if err != nil {
				fmt.Printf("    ↳ fix failed: %v\n", err)
			} else {
				fmt.Printf("    ↳ fixed: %s\n", desc)
				continue // treat as resolved for the summary
			}
		}

		switch r.status {
		case statusFail:
			failures++
		case statusWarn:
			warnings++
		}
	}

	fmt.Println()
	if failures == 0 && warnings == 0 {
		fmt.Printf("%s no problems detected\n", versionLabel())
		return nil
	}

	if !fix {
		fmt.Printf("Found %d issue(s) and %d warning(s). Re-run with --fix to attempt repairs.\n", failures, warnings)
	} else {
		fmt.Printf("Remaining: %d issue(s), %d warning(s).\n", failures, warnings)
	}
	return nil
}

// runDoctorChecks performs all checks and returns their results in order.
func runDoctorChecks(cfg *config.Config) []checkResult {
	var results []checkResult

	// 1. Client host is well-formed (when a remote client is configured).
	if cfg.Client.Host != "" {
		if normalized, err := config.NormalizeServerHost(cfg.Client.Host); err != nil {
			results = append(results, checkResult{
				name:    "client host",
				status:  statusFail,
				message: fmt.Sprintf("configured client host %q is invalid: %v", cfg.Client.Host, err),
			})
		} else if normalized != cfg.Client.Host {
			host := normalized
			results = append(results, checkResult{
				name:    "client host",
				status:  statusWarn,
				message: fmt.Sprintf("client host %q should be normalized to %q", cfg.Client.Host, host),
				fix: func() (string, error) {
					cfg.Client.Host = host
					if err := cfg.Save(); err != nil {
						return "", err
					}
					return fmt.Sprintf("set client host to %q", host), nil
				},
			})
		} else {
			results = append(results, checkResult{
				name:    "client host",
				status:  statusOK,
				message: fmt.Sprintf("%q is well-formed", cfg.Client.Host),
			})
		}
	}

	// 2. Server repos directory exists.
	results = append(results, dirCheck("repos directory", cfg.Server.ReposDir))

	// 3. Backup directory exists.
	results = append(results, dirCheck("backup directory", cfg.BackupDir))

	// 4. Host key presence (informational; generated on first serve).
	if _, err := os.Stat(cfg.Server.HostKey); err != nil {
		results = append(results, checkResult{
			name:    "host key",
			status:  statusWarn,
			message: fmt.Sprintf("host key not found at %s (it will be generated on first 'serve'/'start')", cfg.Server.HostKey),
		})
	} else {
		results = append(results, checkResult{
			name:    "host key",
			status:  statusOK,
			message: "present",
		})
	}

	// 5. Stale PID file (PID recorded but process not running).
	results = append(results, stalePIDCheck(cfg))

	// 6. Reachability of the configured target.
	results = append(results, reachabilityCheck(cfg))

	return results
}

// dirCheck verifies a directory exists, offering to create it.
func dirCheck(name, path string) checkResult {
	if path == "" {
		return checkResult{name: name, status: statusWarn, message: "not configured"}
	}
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return checkResult{name: name, status: statusFail, message: fmt.Sprintf("%s exists but is not a directory", path)}
		}
		return checkResult{name: name, status: statusOK, message: path}
	}
	if os.IsNotExist(err) {
		return checkResult{
			name:    name,
			status:  statusWarn,
			message: fmt.Sprintf("%s does not exist", path),
			fix: func() (string, error) {
				if err := os.MkdirAll(path, 0755); err != nil {
					return "", err
				}
				return "created " + path, nil
			},
		}
	}
	return checkResult{name: name, status: statusFail, message: fmt.Sprintf("cannot access %s: %v", path, err)}
}

// stalePIDCheck detects a PID file that references a process that is no longer
// running (e.g. after a crash), which blocks 'start'.
func stalePIDCheck(cfg *config.Config) checkResult {
	path := cfg.Server.PIDFile
	if _, err := os.Stat(path); err != nil {
		// No PID file is normal when the server isn't running.
		return checkResult{name: "pid file", status: statusOK, message: "no stale PID file"}
	}
	// A PID file exists; if the daemon isn't actually running, it's stale.
	if daemon.IsRunning(cfg) {
		return checkResult{name: "pid file", status: statusOK, message: fmt.Sprintf("server running (PID: %d)", daemon.PID(cfg))}
	}
	return checkResult{
		name:    "pid file",
		status:  statusWarn,
		message: fmt.Sprintf("stale PID file at %s (server not running)", path),
		fix: func() (string, error) {
			if err := os.Remove(path); err != nil {
				return "", err
			}
			return "removed stale PID file", nil
		},
	}
}

// reachabilityCheck verifies the configured target is reachable over TCP. For a
// remote client that's the client host:port; otherwise it's the local server
// port (only meaningful when the server is expected to be running).
func reachabilityCheck(cfg *config.Config) checkResult {
	if cfg.IsRemoteClient() {
		addr := net.JoinHostPort(cfg.ClientHost(), strconv.Itoa(cfg.ClientPort()))
		conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
		if err != nil {
			return checkResult{
				name:    "server reachability",
				status:  statusFail,
				message: fmt.Sprintf("cannot reach %s: %s", addr, cleanDialError(err)),
			}
		}
		_ = conn.Close()
		return checkResult{name: "server reachability", status: statusOK, message: fmt.Sprintf("%s is reachable", addr)}
	}

	// Local server: only report reachability if the daemon claims to be running.
	if !daemon.IsRunning(cfg) {
		return checkResult{
			name:    "server reachability",
			status:  statusOK,
			message: "local server not running (start it with 'homegit start')",
		}
	}
	addr := net.JoinHostPort("localhost", strconv.Itoa(cfg.Server.Port))
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return checkResult{
			name:    "server reachability",
			status:  statusFail,
			message: fmt.Sprintf("server reports running but %s is not accepting connections: %s", addr, cleanDialError(err)),
		}
	}
	_ = conn.Close()
	return checkResult{name: "server reachability", status: statusOK, message: fmt.Sprintf("%s is reachable", addr)}
}

// cleanDialError trims the verbose prefix from a net dial error for display.
func cleanDialError(err error) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		return msg[i+2:]
	}
	return msg
}
