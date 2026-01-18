# Basic Workflow

This guide shows the basic workflow for using homegit.

## Setup

1. **Install homegit**:
   ```bash
   make install
   ```

2. **Start the server**:
   ```bash
   homegit start
   ```

3. **Check status**:
   ```bash
   homegit status
   ```

## Creating Your First Repository

1. **Navigate to your project**:
   ```bash
   cd ~/my-project
   ```

2. **Initialize Git** (if not already):
   ```bash
   git init -b main
   ```

3. **Add and commit your files**:
   ```bash
   git add .
   git commit -m "Initial commit"
   ```

4. **Add homegit as remote**:
   ```bash
   git remote add origin ssh://localhost:2222/my-project.git
   ```

5. **Push to homegit**:
   ```bash
   git push -u origin main
   ```

The repository is automatically created on first push!

## Working with Repositories

### List all repositories
```bash
homegit list
```

### Clone a repository
```bash
# Using homegit shortcut
homegit clone my-project

# Or using git directly
git clone ssh://localhost:2222/my-project.git
```

### Remove a repository
```bash
homegit remove my-project
```

### View server logs
```bash
# Last 50 lines (default)
homegit logs

# Last 10 lines
homegit logs --tail 10

# Follow logs in real-time
homegit logs --follow
```

## Managing the Server

### Start as daemon
```bash
homegit start
```

### Stop daemon
```bash
homegit stop
```

### Restart daemon
```bash
homegit restart
```

### Check status
```bash
homegit status
```

### Run in foreground (for debugging)
```bash
homegit serve
```

## Configuration

Edit `~/.homegit/config`:

```json
{
  "port": 2222,
  "repos_dir": "/Users/you/.homegit/repos",
  "daemon": false,
  "host_key": "/Users/you/.homegit/host_key",
  "pid_file": "/Users/you/.homegit/homegit.pid",
  "default_branch": "main"
}
```

After editing, restart the server:
```bash
homegit restart
```

## Tips

- Use `homegit help` to see all commands and network examples
- Server logs are at `~/.homegit/server.log`
- Repositories are stored in `~/.homegit/repos/`
- Each repository is a bare Git repository (`.git` directory only)
