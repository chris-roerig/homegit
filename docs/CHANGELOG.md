# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-11-07

### Changed
- **Config is now organized into role-based sections.** Settings are split into
  a top-level area for shared, role-agnostic values (`default_branch`,
  `backup_dir`, and a reserved `name`) plus a `server` section (`port`,
  `repos_dir`, `host_key`, `pid_file`) used when hosting, and a `client` section
  (`host`, `port`) used when connecting to another homegit server. This removes
  the previous overloading of `port` and the `server_host == localhost`
  heuristic, making it clear which settings apply to hosting versus connecting.

### Added
- Existing flat (pre-1.1.0) configs are **migrated automatically** on first run.
  The original file is backed up alongside the new one (`config.bak`) and the
  server/client role is inferred from the previous `server_host` value. No
  manual action is required.
- Reserved an optional top-level `name` field (defaults to the machine's
  hostname) for future device-identification features.

[1.1.0]: https://github.com/chris-roerig/homegit/releases/tag/v1.1.0

## [1.0.5] - 2026-11-07

### Fixed
- `homegit list` and `homegit clone` against a remote server no longer fail with
  `unsupported command: ls`. The client now requests the repository listing
  through a dedicated read-only server command instead of running `ls` over SSH.
- `homegit clone <name>` against a remote server no longer reports
  "repository not found" when the repository exists on the server. The local
  existence check now only applies when the server is this machine.

### Added
- `server_host` values are normalized when the config is loaded and during
  `setup`: URL-style scheme prefixes (`http://`, `https://`, `ssh://`,
  `git://`), path components, trailing dots, and surrounding whitespace are
  stripped, and values containing spaces are rejected. This prevents cryptic
  SSH failures from malformed hosts.
- Remote `list`/`clone` operations now use an SSH connection timeout and
  translate common connection failures (connection refused, timed out, no route
  to host, unresolvable host, changed host key) into actionable messages.
- `homegit clone` now refuses to overwrite an existing destination directory
  and reports a clear error.

### Changed
- The interactive `clone` repository picker handles end-of-input and empty or
  out-of-range selections gracefully with clearer messages.

[1.0.5]: https://github.com/chris-roerig/homegit/releases/tag/v1.0.5

## [1.0.4] - 2026-01-19

### Changed
- Documentation improvements: added auto-start on boot instructions and a
  Homebrew installation link to the README; fixed spelling and formatting.

### Fixed
- Applied consistent Go formatting across the codebase.

[1.0.4]: https://github.com/chris-roerig/homegit/releases/tag/v1.0.4

## [1.0.3] - 2026-01-18

### Added
- Remote repository listing over SSH, so `homegit list` and the interactive
  `clone` picker can show repositories hosted on another machine.

### Changed
- Updated the README to document the new interactive features.

[1.0.3]: https://github.com/chris-roerig/homegit/releases/tag/v1.0.3

## [1.0.2] - 2026-01-18

### Added
- Interactive repository selection for the `clone` command: running
  `homegit clone` with no arguments lists available repositories and prompts
  for a choice.

[1.0.2]: https://github.com/chris-roerig/homegit/releases/tag/v1.0.2

## [1.0.1] - 2026-01-18

### Fixed
- Repository path validation now correctly handles the leading slash that Git
  includes in requested paths (e.g. `/repo.git`), preventing valid pushes and
  fetches from being rejected.

[1.0.1]: https://github.com/chris-roerig/homegit/releases/tag/v1.0.1

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
