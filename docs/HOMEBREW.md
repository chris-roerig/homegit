# Homebrew Tap Setup Guide

This guide shows how to create and publish your Homebrew tap for homegit.

## Step 1: Create the Tap Repository

1. **Create a new GitHub repository** named `homebrew-homegit`
   - Go to: https://github.com/new
   - Repository name: `homebrew-homegit`
   - Description: "Homebrew tap for homegit"
   - Public repository
   - Don't initialize with README

2. **Clone and setup the tap repo**:
   ```bash
   mkdir homebrew-homegit
   cd homebrew-homegit
   git init
   mkdir Formula
   cp /path/to/homegit/Formula/homegit.rb Formula/
   git add .
   git commit -m "Add homegit formula"
   git remote add origin https://github.com/chris-roerig/homebrew-homegit.git
   git branch -M main
   git push -u origin main
   ```

## Step 2: Create First Release of homegit

Before the formula works, you need to create a release:

1. **In the homegit repo**, create and push a tag:
   ```bash
   cd /path/to/homegit
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Wait for GitHub Actions** to build the release binaries

3. **Get the SHA256** of the release tarball:
   ```bash
   curl -sL https://github.com/chris-roerig/homegit/archive/refs/tags/v1.0.0.tar.gz | shasum -a 256
   ```

4. **Update the formula** with the SHA256:
   ```bash
   cd homebrew-homegit
   # Edit Formula/homegit.rb and add the sha256
   git add Formula/homegit.rb
   git commit -m "Add SHA256 for v1.0.0"
   git push
   ```

## Step 3: Test the Formula

```bash
# Install from your tap
brew install chris-roerig/homegit/homegit

# Test it works
homegit --version

# Uninstall for testing
brew uninstall homegit
```

## Step 4: Update README

Add installation instructions to your homegit README:

```markdown
### Homebrew (macOS/Linux)

```bash
brew install chris-roerig/homegit/homegit
```

Or tap first:
```bash
brew tap chris-roerig/homegit
brew install homegit
```
\`\`\`

## Updating the Formula

When you release a new version:

1. Create a new tag in homegit repo
2. Get the new SHA256
3. Update Formula/homegit.rb:
   ```ruby
   url "https://github.com/chris-roerig/homegit/archive/refs/tags/v1.1.0.tar.gz"
   sha256 "new-sha256-here"
   ```
4. Commit and push to homebrew-homegit

## Homebrew Service

Users can run homegit as a service:

```bash
# Start service
brew services start homegit

# Stop service
brew services stop homegit

# Restart service
brew services restart homegit
```

## Notes

- Formula is in `Formula/homegit.rb`
- Tap repo must be named `homebrew-*`
- Users install with: `brew install chris-roerig/homegit/homegit`
- After tapping: `brew install homegit`
