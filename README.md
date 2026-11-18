# homegit

[![Build Status](https://github.com/chris-roerig/homegit/workflows/Build%20and%20Test/badge.svg)](https://github.com/chris-roerig/homegit/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/chris-roerig/homegit)](https://go.dev/)
[![License](https://img.shields.io/github/license/chris-roerig/homegit)](LICENSE)

**Your personal Git server. No cloud, no auth, no complexity.**

Stop pushing your side projects to GitHub. Stop paying for private repos. Stop worrying about rate limits. Run your own Git server in 30 seconds.

## Quick Start

**Using [Homebrew](https://brew.sh)**

```bash
brew install chris-roerig/homegit/homegit
homegit setup
```

**Or install with the shell script (macOS/Linux)**

```bash
curl -fsSL https://raw.githubusercontent.com/chris-roerig/homegit/main/install.sh | sh
homegit setup
```

Prefer to inspect first? Download it, read it, then run it:

```bash
curl -fsSL https://raw.githubusercontent.com/chris-roerig/homegit/main/install.sh -o install.sh
less install.sh
sh install.sh
```

**Or clone this repo and run**

```bash
make install
```

**Once installed, find a repo and start using homegit**

```bash
cd my-project
homegit init
git add .
git commit -m "Initial commit"
git push -u origin main

# If using multiple computers, clone the project by running
homegit clone  # This will show you a list of repos to clone
```

That's it. Your code is now on your machine, accessible from anywhere on your network.

## Why homegit?

**For developers who want:**
- Private repos without GitHub/GitLab/Bitbucket
- Offline Git hosting for personal projects
- Network-accessible repos across home computers
- Zero-config Git server that just works
- No authentication overhead for trusted networks

**Not for:**
- Production environments
- Public-facing servers
- Multi-user teams with access control
- Anything requiring security/authentication

## Commands

```bash
homegit setup      # First-time configuration
homegit init       # Initialize git repo and add remote
homegit config     # Edit configuration
homegit start      # Start server
homegit stop       # Stop server
homegit status     # Check if running
homegit list       # List repositories (local or remote)
homegit clone      # Clone from server (interactive if no name given)
homegit backup     # Backup repository
homegit remove     # Remove repository
homegit logs       # View server logs
homegit doctor     # Diagnose and optionally fix issues (--fix)
homegit update     # Update homegit to the latest release
homegit version    # Show version
homegit help       # Show help
```

## Configuration

Edit with `homegit config` or directly at `~/.homegit/config`:

```json
{
  "default_branch": "main",
  "backup_dir": "/Users/you/.homegit/backups",
  "name": "my-machine",
  "server": {
    "port": 2222,
    "repos_dir": "/Users/you/.homegit/repos",
    "host_key": "/Users/you/.homegit/host_key",
    "pid_file": "/Users/you/.homegit/homegit.pid",
    "web_port": 8222,
    "web_enabled": true,
    "allowed_ips": []
  },
  "client": {
    "host": "",
    "port": 2222
  }
}
```

Config is organized by role:

- **Top level** - shared settings: `default_branch` (default branch for new repos), `backup_dir`, and `name` (identifies this device; reserved for future use).
- **`server`** - used when this machine *hosts* repos (`serve`/`start`/`stop`/`status`): `port` to listen on, `repos_dir`, `host_key`, `pid_file`, `web_port` (web UI port, default 8222), and `web_enabled` (set to `false` to never start the web UI), and `allowed_ips` (see [Security](#security)).
- **`client`** - used when connecting to *another* homegit server (`list`/`clone`): `host` and `port`. Leave `host` empty to operate against the local server.

> **Upgrading from an older version?** Your existing config is migrated to this
> format automatically on first run, and the original is saved as
> `~/.homegit/config.bak`. No manual changes are needed.

## Web UI

`homegit serve` (and `homegit start`) also serve a read-only repo browser at `http://<hostname>:8222`. It has a file tree sidebar, a file viewer, and rendered Markdown (including each directory's README). It shows the default branch (HEAD) only.

- Skip it for one run: `homegit serve --no-web`
- Turn it off permanently: set `server.web_enabled` to `false` (`homegit setup` asks), or change the port with `server.web_port`.
- Like the SSH server it has no authentication and listens on all interfaces, so anyone on your network can read your code.
- Markdown uses a small built-in renderer. Raw HTML in READMEs is limited to a safe set of tags.

## Auto-start on Boot

**Using Homebrew:**
```bash
brew services start homegit
```

This starts homegit now and automatically on every boot.

**Manage the service:**
```bash
brew services stop homegit     # Stop service
brew services restart homegit  # Restart service
brew services list             # List all services
```

## Security

**homegit has NO authentication.** Anyone who can reach the server can push/pull.

**Safe for:**
- Home networks with trusted devices
- Localhost development
- Offline use

**NOT safe for:**
- Public internet
- Untrusted networks
- Production code

This includes the web UI (port 8222), which exposes read access to every repo. Run on localhost or private networks only.

**Optionally restrict who can connect.** Set `server.allowed_ips` to a list of IPs or CIDR ranges, for example `["192.168.1.20", "192.168.1.0/24"]`. When the list is non-empty, both the git (SSH) server and the web UI refuse connections from any other address. Loopback (`127.0.0.1`, `::1`) is always allowed so local `homegit` commands keep working. An empty list (the default) allows everyone. Restart the server after editing it, and run `homegit doctor` to validate entries; `serve` refuses to start if an entry is invalid.

This is a network filter, not authentication. It checks the connecting address only, so it does not help behind a proxy or Docker port forwarding (all connections look like they come from the proxy), and it cannot stop a device on your network from spoofing an allowed address. Use firewall rules or VPN for additional security.

## How It Works

homegit wraps Git's built-in server capabilities:

1. Embedded SSH server (no system SSH config needed)
2. Bare repositories in `~/.homegit/repos`
3. Auto-creates repos on first push
4. Delegates to system `git-upload-pack` and `git-receive-pack`

It's a convenience layer for running Git over SSH locally, not a GitHub replacement.

## Troubleshooting

**Port in use:**
```bash
lsof -ti:2222 | xargs kill
# Or change port: homegit config
```

**Can't connect from another computer:**
```bash
homegit status          # Check server is running
homegit help            # Get your IP address
nc -zv 192.168.1.100 2222  # Test connectivity
```

**Repository not found:**
```bash
homegit list  # List all repos
# Ensure remote URL ends with .git
```

## Building

```bash
go build -o homegit
```

Cross-compile:
```bash
GOOS=linux GOARCH=amd64 go build -o homegit-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o homegit-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o homegit-windows-amd64.exe
```

## FAQ

**Q: Why not use `git daemon` or system SSH?**  
A: homegit is simpler - single binary, no system configuration, auto-creates repos.

**Q: Can I use this in production?**  
A: No. Use GitHub, GitLab, or Gitea for production.

**Q: Does it work on Windows?**  
A: Yes, but daemon mode doesn't work. Use `homegit serve` in foreground.

**Q: Can I mirror to GitHub?**  
A: Yes! Add both remotes and push to each.

**Q: Need more features?**  
A: Check out Gitea (full GitHub alternative) or GitLab (enterprise-grade).

## License

MIT - See [LICENSE](LICENSE)
