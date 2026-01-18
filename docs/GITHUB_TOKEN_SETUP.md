# GitHub Token Setup for Automated Homebrew Updates

The release workflow automatically updates the Homebrew tap when you create a new release. To enable this, you need to create a GitHub Personal Access Token.

## Steps

1. **Create a Personal Access Token:**
   - Go to https://github.com/settings/tokens/new
   - Name: `Homebrew Tap Update`
   - Expiration: No expiration (or set to your preference)
   - Scopes: Check `repo` (Full control of private repositories)
   - Click "Generate token"
   - **Copy the token** (you won't see it again!)

2. **Add token to repository secrets:**
   - Go to https://github.com/chris-roerig/homegit/settings/secrets/actions
   - Click "New repository secret"
   - Name: `HOMEBREW_TAP_TOKEN`
   - Value: Paste the token you copied
   - Click "Add secret"

3. **Done!**
   - The release workflow will now automatically update the Homebrew tap
   - When you run `make release VERSION=v1.0.2`, GitHub Actions will:
     - Build binaries for all platforms
     - Create GitHub release
     - Calculate SHA256
     - Update homebrew-homegit formula
     - Commit and push to homebrew-homegit repo

## Without the Token

If you don't set up the token, the release workflow will still:
- Build binaries
- Create GitHub release
- But the Homebrew tap update step will fail (non-critical)

You can still update Homebrew manually with `make release` as before.
