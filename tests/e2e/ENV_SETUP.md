# Environment Setup Guide

## Quick Setup

```bash
cd tests/e2e

# Copy the example file
cp .env.example .env

# Edit with your credentials
nano .env  # or vim .env, or your favorite editor
```

Add your GitHub credentials:
```bash
export GITHUB_USERNAME="your-github-username"
export GITHUB_TOKEN="ghp_your_github_token_here"
```

That's it! Tests will now automatically load your credentials.

## Creating a GitHub Personal Access Token

1. **Go to GitHub Settings**:
   - Visit: https://github.com/settings/tokens
   - Click "Generate new token (classic)"

2. **Configure Token**:
   - **Note**: "Artifusion E2E Tests"
   - **Expiration**: 90 days (recommended)
   - **Scopes** (required):
     - ✅ `read:packages` - Read packages from GitHub Packages
     - ✅ `read:user` - Read user profile info

3. **Generate and Copy**:
   - Click "Generate token"
   - **IMPORTANT**: Copy the token immediately (you won't see it again!)
   - Token format: `ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

4. **Add to .env**:
   ```bash
   export GITHUB_USERNAME="your-username"
   export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
   ```

## .env File Format

The `.env` file should contain bash export statements:

```bash
# Good - uses export
export GITHUB_USERNAME="mainuli"
export GITHUB_TOKEN="ghp_xxx"

# Bad - missing export (won't work)
GITHUB_USERNAME="mainuli"
GITHUB_TOKEN="ghp_xxx"
```

## Security Best Practices

### ✅ DO:
- ✅ Keep `.env` file local (it's in `.gitignore`)
- ✅ Use fine-grained tokens when possible
- ✅ Set token expiration (90 days recommended)
- ✅ Rotate tokens regularly
- ✅ Revoke tokens immediately if compromised
- ✅ Use different tokens for different purposes

### ❌ DON'T:
- ❌ Commit `.env` to git
- ❌ Share tokens in chat/email
- ❌ Use tokens with more permissions than needed
- ❌ Leave tokens without expiration
- ❌ Reuse the same token everywhere

## Token Scopes Explained

### read:packages
Allows reading from:
- GitHub Container Registry (ghcr.io)
- GitHub Packages (Maven, npm)
- Private packages you have access to

### read:user
Allows reading:
- Your public profile information
- Organization memberships
- Used for authentication validation

## Using mise (Version Manager)

If you use `mise` to manage tools like Maven, the test scripts now automatically detect and activate mise shims:

```bash
# Install maven with mise
mise use -g maven@latest

# Run tests (mise activation is automatic)
./run-all-tests.sh
```

The scripts will automatically run:
```bash
eval "$(mise activate bash --shims)"
```

This makes Maven (and other mise-managed tools) available in the test environment.

## Environment Variables Reference

### Required
- `GITHUB_USERNAME` - Your GitHub username
- `GITHUB_TOKEN` - GitHub Personal Access Token (ghp_xxx)

### Optional
- `GITHUB_EMAIL` - Email for npm authentication (default: test@example.com)
- `ARTIFUSION_HOST` - Artifusion server hostname (default: artifacts.lvh.me)
- `SKIP_PREREQ_CHECK` - Skip prerequisite checks (set to 1)

### Example .env with all options:
```bash
# Required
export GITHUB_USERNAME="mainuli"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# Optional
export GITHUB_EMAIL="me@example.com"
export ARTIFUSION_HOST="artifacts.lvh.me"

# For skipping prerequisite checks
export SKIP_PREREQ_CHECK=1
```

## How .env Loading Works

### 1. Master Test Runner (`run-all-tests.sh`)
```bash
# Loads .env from same directory
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi
```

### 2. Individual Tests (`docker/test-oci.sh`, etc.)
```bash
# Loads .env from parent directory
if [ -f "$SCRIPT_DIR/../.env" ]; then
    source "$SCRIPT_DIR/../.env"
fi
```

This means:
- ✅ Running `./run-all-tests.sh` → loads `tests/e2e/.env`
- ✅ Running `./docker/test-oci.sh` → loads `tests/e2e/.env`
- ✅ Running `cd docker && ./test-oci.sh` → loads `tests/e2e/.env`

## Troubleshooting

### Credentials Not Loading

**Symptom**:
```
⚠️  GITHUB_USERNAME and GITHUB_TOKEN not set
```

**Solutions**:

1. **Check .env exists**:
   ```bash
   ls -la tests/e2e/.env
   ```
   If missing, copy from example:
   ```bash
   cp tests/e2e/.env.example tests/e2e/.env
   ```

2. **Check .env format**:
   ```bash
   cat tests/e2e/.env
   ```
   Should have `export` statements:
   ```bash
   export GITHUB_USERNAME="..."
   export GITHUB_TOKEN="..."
   ```

3. **Check file permissions**:
   ```bash
   chmod 600 tests/e2e/.env
   ```

### Maven Not Found (Using mise)

**Symptom**:
```
✗ Maven (mvn) not found in PATH
```

**Solutions**:

1. **Check mise is installed**:
   ```bash
   mise --version
   ```

2. **Install Maven with mise**:
   ```bash
   mise use -g maven@latest
   ```

3. **Verify Maven is available**:
   ```bash
   eval "$(mise activate bash --shims)"
   mvn --version
   ```

4. **Run test** (mise activation is automatic):
   ```bash
   ./maven/test-maven.sh
   ```

### Token Invalid/Expired

**Symptom**:
```
401 Unauthorized
```

**Solutions**:

1. **Test token**:
   ```bash
   curl -H "Authorization: token $GITHUB_TOKEN" \
     https://api.github.com/user
   ```

2. **Check token scopes**:
   ```bash
   curl -H "Authorization: token $GITHUB_TOKEN" \
     https://api.github.com/user \
     -I | grep X-OAuth-Scopes
   ```
   Should include: `read:packages, read:user`

3. **Generate new token**:
   - Revoke old: https://github.com/settings/tokens
   - Create new with correct scopes
   - Update `.env`

## CI/CD Usage

For CI/CD pipelines, use environment variables directly instead of `.env`:

### GitHub Actions
```yaml
- name: Run E2E Tests
  env:
    GITHUB_USERNAME: ${{ secrets.E2E_GITHUB_USERNAME }}
    GITHUB_TOKEN: ${{ secrets.E2E_GITHUB_TOKEN }}
  run: |
    cd tests/e2e
    ./run-all-tests.sh
```

### GitLab CI
```yaml
e2e-tests:
  script:
    - cd tests/e2e
    - ./run-all-tests.sh
  variables:
    GITHUB_USERNAME: $E2E_GITHUB_USERNAME
    GITHUB_TOKEN: $E2E_GITHUB_TOKEN
```

## Token Rotation

Rotate tokens every 90 days:

1. **Generate new token** with same scopes
2. **Test new token**:
   ```bash
   export GITHUB_TOKEN="ghp_new_token"
   curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
   ```
3. **Update .env** with new token
4. **Run tests** to verify:
   ```bash
   ./run-all-tests.sh
   ```
5. **Revoke old token** at https://github.com/settings/tokens

## Additional Resources

- GitHub Tokens: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token
- mise Documentation: https://mise.jdx.dev
- Test Suite README: `./README.md`
- NPM Configuration: `./npm/NPM_CONFIG.md`
