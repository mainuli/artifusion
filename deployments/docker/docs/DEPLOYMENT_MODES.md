# Artifusion Deployment Modes

Artifusion supports two deployment modes for backend service authentication. Choose the mode that best fits your security requirements and infrastructure.

## Deployment Modes

### 1. No Backend Authentication (Recommended for Internal Networks)

**File:** `docker-compose.no-auth.yml`

Backend services (Docker Registry, Reposilite, Verdaccio) run **without authentication**, relying entirely on Artifusion for access control.

**Pros:**
- Simpler setup with fewer credentials to manage
- Artifusion handles all authentication via GitHub PAT
- Backend services are isolated on private Docker network
- Easier to troubleshoot and debug

**Cons:**
- If Artifusion is compromised, backend services are exposed
- Not suitable if backend services are directly accessible from outside

**Security Model:**
```
Client → Artifusion (GitHub Auth) → Backend Services (No Auth)
```

**When to Use:**
- Backend services are on a private Docker network
- Only Artifusion exposes port 8080 externally
- You trust your Docker network isolation
- Simplified credential management is preferred

**Start Command:**
```bash
docker-compose -f docker-compose.no-auth.yml up -d
```

---

### 2. With Backend Authentication (Defense in Depth)

**File:** `docker-compose.with-auth.yml`

Backend services have **their own authentication** enabled, providing an additional security layer.

**Pros:**
- Defense in depth - multiple authentication layers
- Backend services are protected even if Artifusion is bypassed
- Suitable for environments where backend ports might be exposed
- Better audit trail with per-service authentication logs

**Cons:**
- More complex setup with multiple credentials
- Requires managing htpasswd files and tokens
- Slightly more overhead for authentication

**Security Model:**
```
Client → Artifusion (GitHub Auth) → Backend Services (Token/htpasswd Auth)
```

**When to Use:**
- Backend services might be exposed directly (debugging ports)
- Compliance requires defense in depth
- You want independent authentication per service
- Running in untrusted or multi-tenant environments

**Start Command:**
```bash
docker-compose -f docker-compose.with-auth.yml up -d
```

---

## Configuration Comparison

| Feature | No Auth | With Auth |
|---------|---------|-----------|
| **Docker Registry** | No authentication | htpasswd authentication |
| **Reposilite** | No tokens required | Token-based auth (admin/read/write) |
| **Verdaccio** | Open access | htpasswd authentication |
| **Artifusion** | GitHub PAT required | GitHub PAT required |
| **Setup Complexity** | Simple | Moderate |
| **Credentials to Manage** | 1 (GitHub PAT) | 4+ (GitHub PAT + backend creds) |
| **Security Layers** | 1 | 2 |
| **Best For** | Internal networks | Public/exposed deployments |

---

## Quick Start

### Option 1: No Backend Authentication

```bash
cd deployments/docker

# Copy environment file
cp .env.example .env

# Edit .env - only GitHub settings needed
nano .env

# Start services
docker-compose -f docker-compose.no-auth.yml up -d

# Check health
docker-compose -f docker-compose.no-auth.yml ps
```

**Required Environment Variables:**
```bash
# GitHub authentication
GITHUB_ORG=your-organization          # Optional: restrict to org
GITHUB_TEAMS=team1,team2              # Optional: restrict to teams
AUTH_CACHE_TTL=30m                    # GitHub auth cache duration

# GitHub Packages token (for proxying)
GITHUB_PACKAGES_TOKEN=ghp_xxx

# Optional: Logging
LOG_LEVEL=info
LOG_FORMAT=console
```

---

### Option 2: With Backend Authentication

```bash
cd deployments/docker

# Copy environment file
cp .env.example .env

# Edit .env - GitHub + backend credentials
nano .env

# Create Docker Registry htpasswd file
mkdir -p config/registry-auth
docker run --rm --entrypoint htpasswd httpd:2 -Bbn artifusion "your-password" > config/registry-auth/htpasswd

# Start services
docker-compose -f docker-compose.with-auth.yml up -d

# Check health
docker-compose -f docker-compose.with-auth.yml ps
```

**Required Environment Variables:**
```bash
# GitHub authentication
GITHUB_ORG=your-organization
GITHUB_TEAMS=team1,team2
AUTH_CACHE_TTL=30m
GITHUB_PACKAGES_TOKEN=ghp_xxx

# Docker Registry credentials
DOCKER_REGISTRY_USERNAME=artifusion
DOCKER_REGISTRY_PASSWORD=strong-password-here

# Reposilite tokens
REPOSILITE_ADMIN_TOKEN=admin-token-here
REPOSILITE_WRITE_TOKEN=write-token-here
REPOSILITE_READ_TOKEN=read-token-here

# Verdaccio credentials
VERDACCIO_USERNAME=artifusion
VERDACCIO_PASSWORD=another-strong-password

# Logging
LOG_LEVEL=info
LOG_FORMAT=console
```

---

## Switching Between Modes

You can easily switch between deployment modes:

### Switch to No Auth
```bash
# Stop current deployment
docker-compose down

# Start no-auth mode
docker-compose -f docker-compose.no-auth.yml up -d
```

### Switch to With Auth
```bash
# Stop current deployment
docker-compose down

# Setup credentials first (see above)
# Then start with-auth mode
docker-compose -f docker-compose.with-auth.yml up -d
```

**Note:** Volumes persist between modes, so your data (images, packages, artifacts) remains intact.

---

## Configuration Files

Each deployment mode uses different configuration files:

### No Auth Mode
- `config/verdaccio-no-auth.yaml` - Verdaccio config without auth
- `config/reposilite-no-auth.cdn/configuration.cdn` - Reposilite config without tokens

### With Auth Mode
- `config/verdaccio.yaml` - Verdaccio config with htpasswd
- `config/reposilite.cdn/configuration.cdn` - Reposilite config with tokens
- `config/registry-auth/htpasswd` - Docker Registry credentials (must be created manually)

### Shared Configs (both modes)
- `config/artifusion.yaml` - Main Artifusion configuration
- `config/oci-registry-upstream.yaml` - OCI upstream registries

---

## Authentication Setup Guide

### For With-Auth Mode

#### 1. Docker Registry htpasswd

Create the htpasswd file:

```bash
mkdir -p config/registry-auth

# Add a user
docker run --rm --entrypoint htpasswd httpd:2 -Bbn artifusion "your-password" > config/registry-auth/htpasswd

# Add more users
docker run --rm --entrypoint htpasswd httpd:2 -Bbn another-user "another-password" >> config/registry-auth/htpasswd
```

#### 2. Reposilite Tokens

Generate secure tokens:

```bash
# Generate tokens (use a password generator or openssl)
ADMIN_TOKEN=$(openssl rand -hex 32)
WRITE_TOKEN=$(openssl rand -hex 32)
READ_TOKEN=$(openssl rand -hex 32)

# Add to .env
echo "REPOSILITE_ADMIN_TOKEN=$ADMIN_TOKEN" >> .env
echo "REPOSILITE_WRITE_TOKEN=$WRITE_TOKEN" >> .env
echo "REPOSILITE_READ_TOKEN=$READ_TOKEN" >> .env
```

#### 3. Verdaccio Password

Verdaccio uses htpasswd internally. The password is set via environment variable and managed automatically by the container.

```bash
# Add to .env
echo "VERDACCIO_USERNAME=artifusion" >> .env
echo "VERDACCIO_PASSWORD=$(openssl rand -base64 32)" >> .env
```

---

## Testing Your Deployment

### Test No-Auth Mode

```bash
# Test Docker login (only GitHub PAT needed)
docker login localhost:8080
Username: your-github-username
Password: ghp_your_github_token

# Test NPM (only GitHub PAT needed)
npm config set //localhost:8080/npm/:_authToken ghp_your_github_token

# Test Maven (only GitHub PAT needed)
# In settings.xml:
<username>your-github-username</username>
<password>ghp_your_github_token</password>
```

### Test With-Auth Mode

Same as no-auth mode from the client perspective! Artifusion handles the backend authentication transparently.

```bash
# Clients still use GitHub PAT
docker login localhost:8080
Username: your-github-username
Password: ghp_your_github_token

# Artifusion authenticates to backends using configured credentials
# This happens automatically - clients don't need to know backend credentials
```

---

## Troubleshooting

### No-Auth Mode Issues

**Problem:** Backend services rejecting requests

**Solution:**
1. Check that backend services are using no-auth configs:
   ```bash
   docker-compose -f docker-compose.no-auth.yml exec verdaccio cat /verdaccio/conf/config.yaml | grep -A 5 auth
   ```
2. Verify services are on the same Docker network:
   ```bash
   docker network inspect artifusion
   ```

### With-Auth Mode Issues

**Problem:** Artifusion can't authenticate to backends

**Solution:**
1. Verify environment variables are set:
   ```bash
   docker-compose -f docker-compose.with-auth.yml config
   ```
2. Check htpasswd file exists and is mounted:
   ```bash
   docker-compose -f docker-compose.with-auth.yml exec registry cat /auth/htpasswd
   ```
3. Verify tokens in Reposilite config:
   ```bash
   docker-compose -f docker-compose.with-auth.yml exec reposilite cat /app/configuration/configuration.cdn | grep token
   ```

**Problem:** Direct backend access fails

**Solution:** In with-auth mode, direct access to backend ports (8081, 4873) requires credentials. Access through Artifusion (port 8080) which handles backend authentication automatically.

---

## Migration Guide

### From No-Auth to With-Auth

1. **Stop services:**
   ```bash
   docker-compose -f docker-compose.no-auth.yml down
   ```

2. **Create authentication credentials:**
   ```bash
   # Docker Registry
   mkdir -p config/registry-auth
   docker run --rm --entrypoint htpasswd httpd:2 -Bbn artifusion "password" > config/registry-auth/htpasswd

   # Reposilite tokens
   echo "REPOSILITE_ADMIN_TOKEN=$(openssl rand -hex 32)" >> .env
   echo "REPOSILITE_WRITE_TOKEN=$(openssl rand -hex 32)" >> .env
   echo "REPOSILITE_READ_TOKEN=$(openssl rand -hex 32)" >> .env

   # Verdaccio
   echo "VERDACCIO_USERNAME=artifusion" >> .env
   echo "VERDACCIO_PASSWORD=$(openssl rand -base64 32)" >> .env
   ```

3. **Update Artifusion config:**
   Edit `config/artifusion.yaml` to include backend credentials (if needed)

4. **Start with auth:**
   ```bash
   docker-compose -f docker-compose.with-auth.yml up -d
   ```

5. **Verify:** Test that Artifusion can still pull/push to backends

### From With-Auth to No-Auth

1. **Stop services:**
   ```bash
   docker-compose -f docker-compose.with-auth.yml down
   ```

2. **Start no-auth mode:**
   ```bash
   docker-compose -f docker-compose.no-auth.yml up -d
   ```

3. **Cleanup (optional):**
   ```bash
   # Remove unused credentials from .env
   # Keep GITHUB_ORG, GITHUB_TEAMS, GITHUB_PACKAGES_TOKEN
   ```

**Note:** Volumes are preserved, so no data loss during migration.

---

## Best Practices

### For No-Auth Mode
1. Never expose backend service ports externally
2. Use Docker network isolation
3. Run Artifusion behind HTTPS (nginx/Traefik)
4. Monitor Artifusion logs for suspicious activity
5. Regularly rotate GitHub PATs

### For With-Auth Mode
1. Use strong passwords (32+ characters, random)
2. Store credentials in secrets manager (Vault, AWS Secrets Manager)
3. Rotate backend credentials regularly (monthly)
4. Enable audit logging on backend services
5. Use separate credentials for different access levels
6. Monitor both Artifusion and backend logs

### General
1. Keep Docker images updated
2. Use specific version tags (not `latest`)
3. Backup volumes regularly
4. Test disaster recovery procedures
5. Document your deployment choices

---

## Security Considerations

### No-Auth Mode
- **Threat Model:** Assumes Docker network is secure and Artifusion is the only entry point
- **Risk:** If Artifusion is compromised, backends are fully accessible
- **Mitigation:** Strong GitHub authentication, network isolation, HTTPS, regular security updates

### With-Auth Mode
- **Threat Model:** Defense in depth - even if Artifusion is bypassed, backends are protected
- **Risk:** More credentials to manage and potentially leak
- **Mitigation:** Secrets management, credential rotation, audit logging

### Recommendations by Environment

| Environment | Recommended Mode | Reasoning |
|-------------|------------------|-----------|
| Development | No-Auth | Simplicity, easy debugging |
| Internal Network | No-Auth | Network isolation sufficient |
| DMZ/Public | With-Auth | Defense in depth required |
| Multi-tenant | With-Auth | Isolation between tenants |
| Compliance | With-Auth | Audit requirements |

---

## Additional Resources

- [Main README](README.md) - General deployment guide
- [Authentication Setup](AUTHENTICATION_SETUP.md) - Detailed auth configuration
- [Testing Guide](TESTING.md) - Comprehensive testing procedures
- [Maven Setup](MAVEN_SETUP.md) - Maven-specific configuration
- [NPM Setup](NPM-SETUP.md) - NPM-specific configuration

---

## Support

If you encounter issues:
1. Check service logs: `docker-compose logs -f <service-name>`
2. Verify configuration files are mounted correctly
3. Check environment variables: `docker-compose config`
4. Review [Troubleshooting](README.md#troubleshooting) section
5. Open an issue with logs and configuration (sanitize credentials!)
