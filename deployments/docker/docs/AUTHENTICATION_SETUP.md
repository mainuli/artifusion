# Backend Authentication Setup Guide

Complete guide for securing Docker Registry and Reposilite with proper authentication.

---

## Table of Contents

- [Overview](#overview)
- [Docker Registry Authentication](#docker-registry-authentication)
- [Reposilite Authentication](#reposilite-authentication)
- [Quick Setup](#quick-setup)
- [Manual Setup](#manual-setup)
- [Testing Authentication](#testing-authentication)
- [Troubleshooting](#troubleshooting)

---

## Overview

Artifusion supports authentication for backend services to prevent unauthorized access:

```
Client (with GitHub PAT)
    ↓
Artifusion (validates GitHub PAT)
    ↓
Backend Services (with backend credentials)
├── Docker Registry (htpasswd)
└── Reposilite (bearer tokens)
```

**Security Benefits:**
- ✅ Prevent unauthorized direct access to backend services
- ✅ Separate client credentials (GitHub PAT) from backend credentials
- ✅ Token-based authentication for better security
- ✅ Credential rotation without client disruption

---

## Docker Registry Authentication

### How It Works

1. **htpasswd-based authentication** using bcrypt password hashing
2. **Artifusion authenticates** to registry with username/password
3. **Clients never see** registry credentials (proxied through Artifusion)

### Files Involved

```
deployments/docker/
├── config/
│   └── registry-auth/
│       ├── htpasswd         # Generated htpasswd file (bcrypt)
│       └── README.md        # Documentation
├── scripts/
│   └── generate-registry-password.sh  # Helper script
└── .env                     # Credentials stored here
```

---

## Reposilite Authentication

### How It Works

1. **Token-based authentication** with generated random tokens
2. **Three token types:**
   - `REPOSILITE_ADMIN_TOKEN` - Full access (management)
   - `REPOSILITE_WRITE_TOKEN` - Deploy artifacts (used by Artifusion)
   - `REPOSILITE_READ_TOKEN` - Pull dependencies (used by Artifusion)
3. **Configured via environment variables** in Reposilite CDN config

### Files Involved

```
deployments/docker/
├── config/
│   └── reposilite.cdn/
│       └── configuration.cdn  # Token definitions
├── scripts/
│   └── generate-reposilite-tokens.sh  # Helper script
└── .env                        # Tokens stored here
```

---

## Quick Setup

### One-Command Setup (Recommended)

```bash
cd deployments/docker

# 1. Generate all credentials
./scripts/generate-registry-password.sh
./scripts/generate-reposilite-tokens.sh

# 2. Review and save credentials
cat .env | grep -E "REGISTRY_|REPOSILITE_"

# 3. Start services
docker-compose up -d

# 4. Verify authentication
./scripts/test-authentication.sh  # (if exists)
```

---

## Manual Setup

### Docker Registry Setup

#### Step 1: Generate htpasswd File

**Option A: Using the script (Recommended)**

```bash
cd deployments/docker
./scripts/generate-registry-password.sh

# Or specify a custom username
./scripts/generate-registry-password.sh myuser
```

**Option B: Manual generation**

```bash
# Install htpasswd (if not already installed)
# Ubuntu/Debian: sudo apt-get install apache2-utils
# macOS:         brew install httpd
# Alpine:        apk add apache2-utils

# Generate htpasswd file
mkdir -p deployments/docker/config/registry-auth
htpasswd -Bbn artifusion YOUR_PASSWORD > deployments/docker/config/registry-auth/htpasswd

# Verify file contents
cat deployments/docker/config/registry-auth/htpasswd
# Should output: artifusion:$2y$05$...
```

#### Step 2: Update .env File

```bash
# Edit .env file
nano deployments/docker/.env

# Add credentials
REGISTRY_USERNAME=artifusion
REGISTRY_PASSWORD=YOUR_PASSWORD  # Same password used in htpasswd
```

#### Step 3: Update Artifusion Config

Verify `deployments/docker/config/artifusion.yaml` has:

```yaml
protocols:
  oci:
    pushBackend:
      auth:
        type: basic
        username: ${REGISTRY_USERNAME}  # Uses .env variable
        password: ${REGISTRY_PASSWORD}  # Uses .env variable
```

#### Step 4: Restart Registry

```bash
docker-compose restart registry

# Verify authentication is enabled
docker-compose logs registry | grep -i auth
```

---

### Reposilite Setup

#### Step 1: Generate Tokens

**Option A: Using the script (Recommended)**

```bash
cd deployments/docker
./scripts/generate-reposilite-tokens.sh

# Tokens will be displayed and optionally written to .env
```

**Option B: Manual generation**

```bash
# Generate three secure tokens (64 hex characters each)
openssl rand -hex 32  # Admin token
openssl rand -hex 32  # Write token
openssl rand -hex 32  # Read token
```

#### Step 2: Update .env File

```bash
# Edit .env file
nano deployments/docker/.env

# Add tokens
REPOSILITE_ADMIN_TOKEN=your_admin_token_here
REPOSILITE_WRITE_TOKEN=your_write_token_here
REPOSILITE_READ_TOKEN=your_read_token_here

# GitHub Packages token (optional, for GitHub Packages mirror)
GITHUB_PACKAGES_TOKEN=ghp_your_github_token_here
```

#### Step 3: Verify Configuration

Check `deployments/docker/config/reposilite.cdn/configuration.cdn`:

```cdn
# Access Tokens section
token ${REPOSILITE_ADMIN_TOKEN} {
  name admin
  routes *
  permissions m
}

token ${REPOSILITE_WRITE_TOKEN} {
  name artifusion-write
  routes /releases/* /snapshots/*
  permissions rw
}

token ${REPOSILITE_READ_TOKEN} {
  name artifusion-read
  routes *
  permissions r
}
```

Verify `deployments/docker/config/artifusion.yaml` has:

```yaml
protocols:
  maven:
    backend:
      auth:
        type: bearer
        token: ${REPOSILITE_WRITE_TOKEN}  # Uses .env variable
```

#### Step 4: Restart Reposilite

```bash
docker-compose restart reposilite

# Verify tokens are loaded
docker-compose logs reposilite | grep -i token
```

---

## Testing Authentication

### Test Docker Registry Authentication

```bash
# From Artifusion proxy (should work with GitHub PAT)
docker login localhost:8080 -u YOUR_GITHUB_USERNAME -p YOUR_GITHUB_PAT

# Direct to registry (should require registry credentials)
docker login localhost:5000 -u artifusion -p YOUR_REGISTRY_PASSWORD

# Test push through Artifusion
docker pull alpine:latest
docker tag alpine:latest localhost:8080/myorg/alpine:test
docker push localhost:8080/myorg/alpine:test  # Uses GitHub PAT

# Test pull through Artifusion
docker pull localhost:8080/myorg/alpine:test  # Uses GitHub PAT
```

### Test Reposilite Authentication

```bash
# Test direct access (should require token)
curl -H "Authorization: Bearer $REPOSILITE_READ_TOKEN" \
  http://localhost:8081/api/maven/details/releases

# Test through Artifusion (uses GitHub PAT)
curl -H "Authorization: Bearer $GITHUB_PAT" \
  http://localhost:8080/releases/

# Test Maven deploy (through Artifusion)
cd your-maven-project
mvn clean deploy  # Uses GitHub PAT from settings.xml
```

---

## Troubleshooting

### Docker Registry Issues

#### Problem: Authentication failed (401 Unauthorized)

**Symptom:**
```
docker push localhost:8080/myimage:tag
unauthorized: authentication required
```

**Solutions:**

1. Verify htpasswd file exists:
   ```bash
   ls -lh deployments/docker/config/registry-auth/htpasswd
   cat deployments/docker/config/registry-auth/htpasswd
   ```

2. Verify registry container has access:
   ```bash
   docker-compose exec registry ls -lh /auth/
   docker-compose exec registry cat /auth/htpasswd
   ```

3. Check registry logs:
   ```bash
   docker-compose logs registry | grep -i auth
   ```

4. Verify .env credentials match htpasswd:
   ```bash
   grep REGISTRY_ deployments/docker/.env
   ```

#### Problem: htpasswd file not found

**Symptom:**
```
configuration error: open /auth/htpasswd: no such file or directory
```

**Solutions:**

1. Generate htpasswd file:
   ```bash
   cd deployments/docker
   ./scripts/generate-registry-password.sh
   ```

2. Verify file exists:
   ```bash
   ls -lh deployments/docker/config/registry-auth/htpasswd
   ```

3. Restart registry:
   ```bash
   docker-compose restart registry
   ```

---

### Reposilite Issues

#### Problem: Invalid token (401 Unauthorized)

**Symptom:**
```
mvn deploy
[ERROR] Failed to deploy artifacts: Could not transfer artifact: Unauthorized (401)
```

**Solutions:**

1. Verify tokens are set in .env:
   ```bash
   grep REPOSILITE_ deployments/docker/.env
   ```

2. Verify tokens match in configuration.cdn:
   ```bash
   grep -A 3 "token \${REPOSILITE" deployments/docker/config/reposilite.cdn/configuration.cdn
   ```

3. Check Reposilite logs:
   ```bash
   docker-compose logs reposilite | grep -i auth
   docker-compose logs reposilite | grep -i token
   ```

4. Restart Reposilite:
   ```bash
   docker-compose restart reposilite
   ```

#### Problem: Token variables not expanded

**Symptom:**
```
Configuration error: ${REPOSILITE_ADMIN_TOKEN} is not a valid token
```

**Solutions:**

1. Verify environment variables are passed to container:
   ```bash
   docker-compose config | grep REPOSILITE
   ```

2. Verify .env file is in correct location:
   ```bash
   ls -lh deployments/docker/.env
   ```

3. Restart with fresh environment:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

---

## Security Best Practices

### Credential Storage

1. **Never commit credentials to version control**
   ```bash
   # Verify .env is in .gitignore
   git check-ignore .env
   # Should output: .env
   ```

2. **Use strong passwords and tokens**
   ```bash
   # Passwords: minimum 24 characters
   openssl rand -base64 24

   # Tokens: 32 bytes (64 hex characters)
   openssl rand -hex 32
   ```

3. **Rotate credentials regularly**
   ```bash
   # Re-generate all credentials
   ./scripts/generate-registry-password.sh
   ./scripts/generate-reposilite-tokens.sh

   # Restart services
   docker-compose restart registry reposilite
   ```

### Access Control

1. **Separate credentials by environment**
   ```
   .env.development   # Development credentials
   .env.staging       # Staging credentials
   .env.production    # Production credentials (in secrets manager)
   ```

2. **Limit token permissions**
   - Use `READ_TOKEN` for pull-only operations
   - Use `WRITE_TOKEN` only for Artifusion backend
   - Keep `ADMIN_TOKEN` separate and secure

3. **Monitor access logs**
   ```bash
   # Watch for authentication failures
   docker-compose logs -f | grep "401\|403"

   # Check registry access logs
   docker-compose logs registry | grep "authentication"
   ```

---

## Credential Rotation

### Rotate Registry Password

```bash
# 1. Generate new htpasswd file
cd deployments/docker
./scripts/generate-registry-password.sh

# 2. Update .env with new password
nano .env

# 3. Update Artifusion config if needed
# (only if username changed)

# 4. Restart services
docker-compose restart registry artifusion

# 5. Test authentication
docker login localhost:5000 -u artifusion -p NEW_PASSWORD
```

### Rotate Reposilite Tokens

```bash
# 1. Generate new tokens
cd deployments/docker
./scripts/generate-reposilite-tokens.sh

# 2. Update .env with new tokens
nano .env

# 3. Restart services
docker-compose restart reposilite artifusion

# 4. Test authentication
curl -H "Authorization: Bearer $NEW_READ_TOKEN" \
  http://localhost:8081/api/maven/details/releases
```

---

## Useful Commands

```bash
# View all credentials (use carefully!)
grep -E "REGISTRY_|REPOSILITE_|GITHUB_PACKAGES" deployments/docker/.env

# Validate htpasswd file
htpasswd -v deployments/docker/config/registry-auth/htpasswd artifusion

# Check which services have authentication enabled
docker-compose ps
docker-compose logs registry reposilite | grep -i auth

# Test registry authentication directly
curl -v -u artifusion:PASSWORD http://localhost:5000/v2/_catalog

# Test Reposilite authentication directly
curl -v -H "Authorization: Bearer TOKEN" \
  http://localhost:8081/api/maven/details/releases

# Monitor authentication attempts
docker-compose logs -f --tail=100 | grep -i "auth\|401\|403"
```

---

## Production Deployment

For production deployments, consider:

1. **Use secrets management** (HashiCorp Vault, AWS Secrets Manager, etc.)
2. **Enable TLS/HTTPS** for encrypted communication
3. **Set up log aggregation** to monitor authentication attempts
4. **Implement automated credential rotation**
5. **Use read-only tokens** where possible
6. **Enable audit logging** for compliance

---

**Status**: ✅ Production Ready | **Security**: Enhanced | **Auth**: htpasswd + Bearer Tokens
