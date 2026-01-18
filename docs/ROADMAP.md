# Roadmap

This document tracks known issues, planned improvements, and future features for homegit.

## Post-Release (v1.1.0)

### Security Improvements

- [ ] **Git Command Timeouts** (git/git.go:56)
  - Use `CommandContext` with timeout
  - Prevent malicious clients from hanging server
  - Suggested timeout: 5 minutes for large repos

- [ ] **Connection Limits** (ssh/server.go:47-50)
  - Add semaphore to limit concurrent connections
  - Prevent resource exhaustion
  - Suggested limit: 10 concurrent connections

- [ ] **Input Validation** (cmd/*.go)
  - Validate numeric input in interactive menus
  - Handle unexpected input gracefully
  - Add range checks for menu selections

### Reliability Improvements

- [ ] **PID File Race Condition** (daemon/daemon.go:58)
  - Use atomic file operations or file locks
  - Prevent race between check and remove
  - Consider using `flock` or similar

- [ ] **PID Validation** (daemon/daemon.go:90)
  - Validate PID is reasonable (> 0, < max)
  - Check process name matches expected binary
  - Handle stale PID files pointing to wrong process

- [ ] **PID Write Timing** (daemon/daemon.go:48)
  - Write PID file before starting process or handle cleanup
  - Prevent orphaned processes if PID write fails
  - Add rollback on failure

- [ ] **Error Backoff** (ssh/server.go:47-50)
  - Add exponential backoff on Accept() errors
  - Prevent tight loop on resource exhaustion
  - Log repeated errors

### Code Quality

- [ ] **Config Validation** (config.go:50)
  - Warn on unknown JSON fields
  - Validate port ranges (1024-65535)
  - Validate directory paths exist or can be created

- [ ] **Structured Logging**
  - Replace `fmt.Printf` with structured logger
  - Add log levels (debug, info, warn, error)
  - Make debugging easier

- [ ] **Metrics/Monitoring**
  - Add basic metrics (connections, repos, errors)
  - Optional Prometheus endpoint
  - Help diagnose issues

## Future Features (v2.0.0+)

### Enhancements

- [ ] **Repository Info Command**
  - Show branches, size, last commit
  - `homegit info <repo-name>`

- [ ] **Repository Rename**
  - Rename repositories safely
  - `homegit rename <old> <new>`

- [ ] **Batch Operations**
  - Backup all repositories
  - `homegit backup --all`

- [ ] **Web UI** (Optional)
  - Simple read-only web interface
  - Browse repositories and commits
  - View logs and status

- [ ] **Webhooks** (Optional)
  - Trigger actions on push
  - Integration with CI/CD
  - Notification support

### Platform Support

- [ ] **Windows Daemon Support**
  - Implement Windows service support
  - Replace Unix-specific syscalls
  - Use `golang.org/x/sys/windows`

- [ ] **Systemd Integration**
  - Provide systemd unit file
  - Better Linux integration
  - Auto-start on boot

- [ ] **Launchd Integration**
  - Provide launchd plist
  - Better macOS integration
  - Auto-start on login

## Known Limitations

These are documented limitations that are acceptable for the current scope:

- **No Authentication** - By design for personal use
- **No Rate Limiting** - Acceptable for trusted networks
- **No Web UI** - Command-line only
- **Requires System Git** - Not bundled
- **Requires tar for Backups** - Unix assumption
- **No Windows Daemon** - Foreground mode only on Windows

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for how to contribute to these improvements.

Issues marked with "good first issue" are suitable for new contributors.
