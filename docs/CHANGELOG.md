# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-18

### Added
- Initial release
- Embedded SSH server with no authentication
- Auto-create repositories on first push
- Daemon mode with PID file management
- Commands: `serve`, `start`, `stop`, `restart`, `status`
- Repository management: `list`, `clone`, `remove`
- Server logs: `logs` command with tail and follow options
- Configuration file at `~/.homegit/config`
- Default branch configuration (defaults to `main`)
- Help command with network examples and local IP detection
- Makefile with `install` target for easy setup
- Cross-platform support (macOS, Linux, Windows)

### Features
- Single binary with no external dependencies (except Git)
- Port configuration (default: 2222)
- Automatic repository creation on push
- Repository listing with full paths
- Local clone shortcut command
- Server log viewing with tail options
- Confirmation prompts for destructive operations

[1.0.0]: https://github.com/chris-roerig/homegit/releases/tag/v1.0.0
