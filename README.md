# homegit

[![Build Status](https://github.com/chris-roerig/homegit/workflows/Build%20and%20Test/badge.svg)](https://github.com/chris-roerig/homegit/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/chris-roerig/homegit)](https://go.dev/)
[![License](https://img.shields.io/github/license/chris-roerig/homegit)](LICENSE)

**Your personal Git server. No cloud, no auth, no complexity.**

Stop pushing your side projects to GitHub. Stop paying for private repos. Stop worrying about rate limits. Run your own Git server in 30 seconds.

```bash
brew install chris-roerig/homegit/homegit
homegit start
cd my-project
homegit init
git add .
git commit -m "Initial commit"
git push -u origin main
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

## Quick Start

**Install:**
```bash
brew install chris-roerig/homegit/homegit
```

**First-time setup:**
```bash
homegit setup
# Choose where to store repos (this computer or connect to another)
```

**Initialize project:**
```bash
cd your-project
homegit init
git add .
git commit -m "Initial commit"
git push -u origin main
```

**Clone from another computer:**
```bash
git clone ssh://192.168.1.100:2222/your-project.git
```

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
homegit version    # Show version
homegit help       # Show help
```

## Configuration

Edit with `homegit config` or directly at `~/.homegit/config`:

```json
{
  "port": 2222,
  "server_host": "localhost",
  "repos_dir": "/Users/you/.homegit/repos",
  "default_branch": "main",
  "backup_dir": "/Users/you/.homegit/backups"
}
```

**Key settings:**
- `server_host` - Where repos are stored (localhost or another computer's IP/hostname)
- `port` - SSH server port (default: 2222)
- `repos_dir` - Repository storage location
- `default_branch` - Default branch for new repos (default: main)

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

Run on localhost or private networks only. Use firewall rules or VPN for additional security.

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

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md)

## License

MIT - See [LICENSE](LICENSE)
