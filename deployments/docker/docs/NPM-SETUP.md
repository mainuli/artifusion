# NPM/Verdaccio Setup Guide

This guide explains how to use the NPM registry functionality in Artifusion with Verdaccio.

## Overview

Verdaccio is configured as the NPM backend for Artifusion with the following capabilities:

- **Hosted Repository**: Publish private packages
- **GitHub Packages Mirror**: Proxy scoped packages from GitHub Packages NPM
- **npmjs.org Mirror**: Proxy public packages from the official NPM registry
- **Smart Routing**: Scoped packages try GitHub first, public packages try npmjs first

## Architecture

```
NPM Client (npm/yarn/pnpm)
    │
    │ Auth: GitHub PAT
    │
    ▼
Artifusion :8080/npm/
    │
    │ GitHub PAT validation
    │ URL rewriting
    │
    ▼
Verdaccio :4873
    │
    ├─── Hosted packages (local storage)
    │
    ├─── GitHub Packages NPM
    │    (for @org/package)
    │
    └─── npmjs.org
         (for public packages)
```

## Verdaccio Configuration

### Storage
- **Path**: `/verdaccio/storage` (Docker volume)
- **Persistence**: `verdaccio-storage` volume

### Uplinks (Upstream Registries)

#### 1. GitHub Packages NPM
```yaml
github:
  url: https://npm.pkg.github.com/
  timeout: 30s
  max_age: 30m
```

**Authentication**: Set `GITHUB_PACKAGES_TOKEN` in `.env` file

#### 2. npmjs.org
```yaml
npmjs:
  url: https://registry.npmjs.org/
  timeout: 30s
  max_age: 2h
```

**No authentication required** (public registry)

### Package Routing

#### Scoped Packages (`@org/package`)
```yaml
'@*/*':
  access: $all
  publish: $all
  unpublish: $all
  proxy: github npmjs  # Try GitHub first, then npmjs
```

**Use case**: Organization packages published to GitHub Packages

#### Private Packages (`private-*`)
```yaml
'private-*':
  access: $all
  publish: $all
  unpublish: $all
  # No proxy - hosted only
```

**Use case**: Internal packages not published to external registries

#### Public Packages (everything else)
```yaml
'**':
  access: $all
  publish: $all
  unpublish: $all
  proxy: npmjs github  # Try npmjs first for performance
```

**Use case**: Standard public NPM packages (express, react, lodash, etc.)

## Setup Instructions

### 1. Prerequisites

- Artifusion running with NPM support enabled
- GitHub Personal Access Token with `read:packages` scope (if using GitHub Packages)

### 2. Environment Configuration

Edit `deployments/docker/.env`:

```bash
# Required: Your GitHub organization (or leave empty)
GITHUB_ORG=my-organization

# Required: GitHub PAT for proxy and client auth
GITHUB_PACKAGES_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Optional: Verdaccio log level
VERDACCIO_LOG_LEVEL=info
```

### 3. Start Services

```bash
cd deployments/docker
docker-compose up -d verdaccio artifusion
```

### 4. Verify Service Health

```bash
# Check Verdaccio
curl http://localhost:4873/-/ping
# Expected: {"status":"ok"}

# Check Artifusion NPM endpoint
curl http://localhost:8080/npm/-/ping
# Expected: {"status":"ok"}
```

## Client Configuration

### npm

```bash
# Set registry
npm config set registry http://localhost:8080/npm/

# Set authentication
npm config set //localhost:8080/npm/:_authToken ghp_your_github_token_here

# Verify configuration
npm config get registry
npm config get //localhost:8080/npm/:_authToken
```

### yarn

```bash
# .yarnrc.yml (Yarn 2+)
npmRegistryServer: "http://localhost:8080/npm/"
npmAuthToken: "ghp_your_github_token_here"

# .yarnrc (Yarn 1.x)
registry "http://localhost:8080/npm/"
"//localhost:8080/npm/:_authToken" "ghp_your_github_token_here"
```

### pnpm

```bash
# Set registry
pnpm config set registry http://localhost:8080/npm/

# Set authentication
pnpm config set //localhost:8080/npm/:_authToken ghp_your_github_token_here
```

### Project-specific (.npmrc)

Create `.npmrc` in your project root:

```ini
registry=http://localhost:8080/npm/
//localhost:8080/npm/:_authToken=ghp_your_github_token_here
```

**⚠️ Important**: Add `.npmrc` to `.gitignore` to avoid committing tokens!

## Usage Examples

### Install Public Packages

```bash
# From npmjs.org (proxied and cached)
npm install express
npm install react react-dom
npm install lodash

# Works with all package managers
yarn add axios
pnpm add typescript
```

### Install Scoped Packages from GitHub

```bash
# From GitHub Packages (proxied and cached)
npm install @myorg/shared-components
npm install @myorg/api-client

# Falls back to npmjs if not found in GitHub
npm install @types/node
npm install @babel/core
```

### Publish Private Packages

```bash
# Initialize package
npm init --scope=@myorg

# Edit package.json
{
  "name": "@myorg/my-package",
  "version": "1.0.0",
  "publishConfig": {
    "registry": "http://localhost:8080/npm/"
  }
}

# Publish
npm publish
```

### Publish Unscoped Private Packages

```bash
# Name with 'private-' prefix (hosted only, not proxied)
{
  "name": "private-internal-utils",
  "version": "1.0.0"
}

npm publish
```

### Install Private Published Packages

```bash
# From your hosted registry
npm install @myorg/my-package
npm install private-internal-utils
```

## Authentication Flow

### Client → Artifusion
1. Client sends request with Authorization header: `Bearer ghp_...`
2. Artifusion validates GitHub PAT
3. Artifusion checks organization membership (if configured)
4. If valid, request is forwarded to Verdaccio

### Verdaccio → GitHub Packages
1. Verdaccio receives request from Artifusion
2. If package not in cache, Verdaccio proxies to upstream
3. For GitHub Packages, uses `GITHUB_PACKAGES_TOKEN` from environment
4. Response is cached and returned to Artifusion
5. Artifusion rewrites URLs and returns to client

## Troubleshooting

### Issue: Cannot install packages

**Symptoms**:
```bash
npm install express
# Error: 401 Unauthorized
```

**Solutions**:
1. Verify GitHub PAT is valid:
   ```bash
   curl -H "Authorization: Bearer ghp_xxx" https://api.github.com/user
   ```

2. Check authentication is configured:
   ```bash
   npm config get //localhost:8080/npm/:_authToken
   ```

3. Check Artifusion logs:
   ```bash
   docker-compose logs artifusion | grep npm
   ```

### Issue: Cannot install GitHub Packages scoped packages

**Symptoms**:
```bash
npm install @myorg/package
# Error: 404 Not Found
```

**Solutions**:
1. Verify `GITHUB_PACKAGES_TOKEN` is set:
   ```bash
   docker-compose exec verdaccio env | grep NPM_TOKEN
   ```

2. Verify package exists:
   ```bash
   curl -H "Authorization: Bearer ghp_xxx" \
     https://npm.pkg.github.com/@myorg/package
   ```

3. Check Verdaccio uplink config:
   ```bash
   docker-compose exec verdaccio cat /verdaccio/conf/config.yaml | grep -A 5 github
   ```

4. Check Verdaccio logs:
   ```bash
   docker-compose logs verdaccio | grep github
   ```

### Issue: Cannot publish packages

**Symptoms**:
```bash
npm publish
# Error: 403 Forbidden
```

**Solutions**:
1. Verify GitHub PAT has write access to org
2. Check package name doesn't conflict with existing npmjs package
3. Use scoped name: `@myorg/package` instead of `package`
4. Check Verdaccio logs:
   ```bash
   docker-compose logs verdaccio | grep publish
   ```

### Issue: Slow package installation

**Symptoms**: First install is slow, subsequent installs still slow

**Solutions**:
1. Check cache is enabled in `verdaccio.yaml`:
   ```yaml
   uplinks:
     npmjs:
       max_age: 2h  # Should not be 0s
   ```

2. Check cache volume exists:
   ```bash
   docker volume inspect artifusion_verdaccio-storage
   ```

3. Check disk space:
   ```bash
   df -h
   ```

### Issue: URL rewriting not working

**Symptoms**: Package metadata contains wrong URLs

**Solutions**:
1. Check `external_url` in `config/artifusion.yaml`
2. Verify NPM handler is rewriting URLs:
   ```bash
   curl -H "Authorization: Bearer ghp_xxx" \
     http://localhost:8080/npm/express | jq '.dist.tarball'
   # Should contain: http://localhost:8080/npm/...
   ```

3. Check Artifusion logs for rewriting:
   ```bash
   docker-compose logs artifusion | grep rewrite
   ```

## Monitoring

### Package Cache Statistics

```bash
# Check Verdaccio storage size
docker-compose exec verdaccio du -sh /verdaccio/storage
```

### View Cached Packages

```bash
# List all cached packages
docker-compose exec verdaccio ls -lh /verdaccio/storage
```

### Metrics

```bash
# Artifusion Prometheus metrics
curl http://localhost:8080/metrics | grep npm

# Example metrics:
# artifusion_requests_total{protocol="npm"}
# artifusion_backend_latency_seconds{protocol="npm",backend="verdaccio"}
# artifusion_backend_errors_total{protocol="npm"}
```

## Maintenance

### Clear NPM Cache

```bash
# Stop Verdaccio
docker-compose stop verdaccio

# Remove cache (keeps configuration)
docker-compose exec verdaccio rm -rf /verdaccio/storage/*

# Or remove entire volume (WARNING: deletes all published packages)
docker volume rm artifusion_verdaccio-storage

# Restart
docker-compose up -d verdaccio
```

### Backup Published Packages

```bash
# Backup Verdaccio storage
docker run --rm \
  -v artifusion_verdaccio-storage:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/verdaccio-backup-$(date +%Y%m%d).tar.gz -C /data .

# Verify backup
tar tzf verdaccio-backup-*.tar.gz | head
```

### Restore Published Packages

```bash
# Stop Verdaccio
docker-compose stop verdaccio

# Restore from backup
docker run --rm \
  -v artifusion_verdaccio-storage:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/verdaccio-backup-20250101.tar.gz -C /data

# Restart
docker-compose up -d verdaccio
```

### Update Verdaccio

```bash
# Pull latest image
docker-compose pull verdaccio

# Recreate container
docker-compose up -d verdaccio

# Check version
docker-compose exec verdaccio npm --version
```

## Security Best Practices

1. **Token Management**
   - Use fine-grained GitHub PATs with minimal scopes
   - Rotate tokens regularly
   - Never commit tokens to git

2. **Network Security**
   - Keep Verdaccio on private network (behind Artifusion)
   - Only expose Artifusion port 8080
   - Use HTTPS in production (TLS termination at reverse proxy)

3. **Package Security**
   - Review dependencies before publishing
   - Use `npm audit` to check for vulnerabilities
   - Enable 2FA on npmjs.org account

4. **Access Control**
   - Use GitHub organization membership for access control
   - Configure `GITHUB_ORG` in `.env` to restrict access
   - Use GitHub teams for fine-grained control

## Production Recommendations

1. **Use HTTPS**: Deploy Artifusion behind nginx/Traefik with TLS
2. **Persistent Storage**: Use reliable volume drivers (not NFS for performance)
3. **Resource Limits**: Set memory/CPU limits in docker-compose.yml
4. **Monitoring**: Export metrics to Prometheus/Grafana
5. **Logging**: Ship logs to centralized logging system
6. **Backups**: Automate regular backups of `verdaccio-storage` volume
7. **High Availability**: Run multiple Verdaccio replicas with shared storage

## Advanced Configuration

### Custom Uplinks

Add more upstream registries in `config/verdaccio.yaml`:

```yaml
uplinks:
  # ... existing uplinks ...

  # Company private registry
  company-npm:
    url: https://npm.company.com/
    timeout: 30s
    max_age: 1h
    auth:
      type: bearer
      token: $COMPANY_NPM_TOKEN

packages:
  # Route company packages to company registry
  '@company/*':
    proxy: company-npm npmjs
```

### Disable Web UI

For production, disable the web UI:

```yaml
web:
  enable: false
```

### Custom Package Filters

```yaml
packages:
  # Block specific packages
  'malicious-package':
    access: $authenticated
    publish: $authenticated

  # Allow only specific users for sensitive packages
  '@company/secrets':
    access: $authenticated
    publish: $authenticated
```

## Support

- **Issues**: https://github.com/mainuli/artifusion/issues
- **Verdaccio Docs**: https://verdaccio.org/docs/
- **NPM CLI Docs**: https://docs.npmjs.com/cli/
