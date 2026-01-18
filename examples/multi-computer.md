# Multi-Computer Setup

This guide shows how to use homegit across multiple computers on your network.

## Scenario

- **Computer A**: Running homegit server
- **Computer B**: Client pushing/pulling code
- **Computer C**: Another client

## Setup on Computer A (Server)

1. **Install and start homegit**:
   ```bash
   make install
   # Server starts automatically after install
   ```

2. **Find your IP address**:
   ```bash
   homegit help
   ```
   Look for "Local IP" in the output (e.g., `10.0.0.45`)

3. **Verify server is running**:
   ```bash
   homegit status
   ```

## Setup on Computer B (Client)

### Option 1: Push Existing Project

1. **Navigate to your project**:
   ```bash
   cd ~/my-project
   ```

2. **Add homegit server as remote** (use Computer A's IP):
   ```bash
   git remote add origin ssh://10.0.0.45:2222/my-project.git
   ```

3. **Push your code**:
   ```bash
   git push -u origin main
   ```

### Option 2: Clone Existing Repository

```bash
git clone ssh://10.0.0.45:2222/my-project.git
cd my-project
# Work on your code
git add .
git commit -m "Changes from Computer B"
git push
```

## Setup on Computer C (Another Client)

1. **Clone the repository**:
   ```bash
   git clone ssh://10.0.0.45:2222/my-project.git
   ```

2. **Work and sync**:
   ```bash
   cd my-project
   
   # Pull latest changes
   git pull
   
   # Make changes
   git add .
   git commit -m "Changes from Computer C"
   git push
   ```

## Workflow

### On Computer B
```bash
cd my-project
git pull                    # Get latest changes
# ... make changes ...
git add .
git commit -m "Update feature"
git push                    # Push to Computer A
```

### On Computer C
```bash
cd my-project
git pull                    # Get changes from Computer B
# ... make changes ...
git add .
git commit -m "Fix bug"
git push                    # Push to Computer A
```

### On Computer A (Server)
```bash
# View all repositories
homegit list

# Clone a repository locally if needed
homegit clone my-project

# View server activity
homegit logs --follow
```

## Network Configuration

### Firewall (if needed)

Allow port 2222 on Computer A:

**macOS**:
```bash
# Usually not needed on home networks
```

**Linux (ufw)**:
```bash
sudo ufw allow 2222/tcp
```

**Windows**:
```powershell
New-NetFirewallRule -DisplayName "homegit" -Direction Inbound -LocalPort 2222 -Protocol TCP -Action Allow
```

### Using Different Port

Edit `~/.homegit/config` on Computer A:
```json
{
  "port": 3333,
  ...
}
```

Then restart:
```bash
homegit restart
```

Clients use the new port:
```bash
git remote add origin ssh://10.0.0.45:3333/my-project.git
```

## Troubleshooting

### Can't connect from Computer B/C

1. **Check server is running** (on Computer A):
   ```bash
   homegit status
   ```

2. **Verify IP address** (on Computer A):
   ```bash
   homegit help
   ```

3. **Test connectivity** (from Computer B/C):
   ```bash
   nc -zv 10.0.0.45 2222
   ```

4. **Check firewall** (on Computer A)

### SSH Host Key Warning

First time connecting, you'll see:
```
Warning: Permanently added '[10.0.0.45]:2222' (ED25519) to the list of known hosts.
```

This is normal and only happens once per client computer.

## Tips

- All computers should be on the same network
- Computer A must remain on and running homegit
- Use static IP or hostname for Computer A to avoid changing remotes
- Consider using mDNS hostname (e.g., `computer-a.local`) instead of IP
