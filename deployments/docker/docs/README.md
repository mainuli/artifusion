# Artifusion Docker Deployment

Complete Docker Compose stack for running Artifusion with all backend services.

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│ Clients (docker, npm, yarn, pnpm, mvn, gradle)                 │
└─────────────────────────────┬──────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Artifusion (Port 8080)                                          │
│ - GitHub PAT Authentication                                     │
│ - Protocol Detection (OCI, Maven, NPM)                          │
│ - Request Routing & URL Rewriting                              │
└───────┬─────────────────┬───────────────────┬──────────────────┘
        │                 │                   │
   ┌────▼─────┐     ┌─────▼─────┐      ┌────▼────────┐
   │OCI Ops   │     │Maven Ops  │      │ NPM Ops     │
   │          │     │           │      │             │
┌──▼────────┐ │  ┌──▼────────┐    ┌───▼─────────┐
│oci-registry│ │  │reposilite │    │ verdaccio   │
│Upstreams:  │ │  │Upstreams: │    │ Upstreams:  │
│- Local     │ │  │- GitHub   │    │ - GitHub    │
│- ghcr.io   │ │  │- Maven    │    │ - npmjs.org │
│- docker.io │ │  │  Central  │    │             │
│- quay.io   │ │  │- Google   │    │ Hosted +    │
│            │ │  │- Spring   │    │ Proxy       │
│ Caching    │ │  │           │    │             │
└────┬───────┘ │  └───────────┘    └─────────────┘
     │         │
┌────▼─────────▼┐
│  registry:2   │
│ (Push Target) │
│  Local Store  │
└───────────────┘
```

## Services

1. **artifusion** - Main reverse proxy with GitHub authentication
2. **oci-registry** - Pull-through cache for OCI/Docker images
3. **registry** - Docker Registry 2.0 for local image storage
4. **reposilite** - Maven repository manager
5. **verdaccio** - NPM package registry with proxying

## Quick Start

### 1. Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- GitHub Personal Access Token (PAT) for authentication
- GitHub organization (optional - only if you want to restrict access by org)

### 2. Configuration

Copy the example environment file:

```bash
cd deployments/docker
cp .env.example .env
```

Edit `.env` and configure:

```bash
# Optional: Your GitHub organization (leave empty to allow any valid GitHub user)
GITHUB_ORG=your-organization

# Optional: Restrict to specific teams (only checked if GITHUB_ORG is set)
GITHUB_TEAMS=platform-team,security-team

# Backend credentials
REGISTRY_PASSWORD=strong-password-here
REPOSILITE_READ_PASSWORD=readonly-password
REPOSILITE_WRITE_TOKEN=write-token-here
```

### 3. Start Services

```bash
# Start all services
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f artifusion
```

### 4. Test OCI/Docker Registry

```bash
# Option 1: Login with GitHub PAT
docker login localhost:8080
# Username: your-github-username
# Password: ghp_your_github_personal_access_token

# Option 2: Login with GitHub Actions token (from CI)
docker login localhost:8080
# Username: github-actions
# Password: ghs_token_from_workflow

# Pull an image (will cascade through upstreams)
docker pull localhost:8080/myorg.io/nginx:latest

# Tag and push a local image
docker tag my-app:latest localhost:8080/myorg.io/my-app:latest
docker push localhost:8080/myorg.io/my-app:latest

# Pull your pushed image (served from local registry)
docker pull localhost:8080/myorg.io/my-app:latest
```

**Supported Token Types:**
- Classic PAT: `ghp_[a-zA-Z0-9]{36}`
- Fine-grained PAT: `github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}`
- GitHub Actions: `ghs_[a-zA-Z0-9]{36}`

Invalid token formats are rejected immediately (<1ms) without API calls.

### 5. Test NPM Registry

```bash
# Configure npm client
npm config set registry http://localhost:8080/npm/

# Set authentication (your GitHub PAT)
npm config set //localhost:8080/npm/:_authToken ghp_your_github_personal_access_token

# Or add to .npmrc
echo "//localhost:8080/npm/:_authToken=ghp_your_token_here" >> ~/.npmrc

# Install a package from npmjs.org (proxied)
npm install express

# Install scoped package from GitHub Packages
npm install @myorg/my-package

# Publish a package
npm publish

# Works with yarn and pnpm too
yarn add lodash
pnpm add react
```

**Supported Package Managers:**
- npm 6.x, 7.x, 8.x, 9.x, 10.x
- yarn 1.x, 2.x, 3.x, 4.x
- pnpm 6.x, 7.x, 8.x, 9.x

### 6. Test Maven Repository

Configure Maven `~/.m2/settings.xml`:

```xml
<settings>
  <servers>
    <server>
      <id>artifusion</id>
      <username>your-github-username</username>
      <password>ghp_your_github_personal_access_token</password>
    </server>
  </servers>
</settings>
```

Configure project `pom.xml`:

```xml
<distributionManagement>
  <repository>
    <id>artifusion</id>
    <url>http://localhost:8080/</url>
  </repository>
</distributionManagement>

<repositories>
  <repository>
    <id>artifusion</id>
    <url>http://localhost:8080/</url>
  </repository>
</repositories>
```

Deploy artifact:

```bash
mvn clean deploy
```

## Volume Management

### Backup Volumes

```bash
# Backup Docker registry
docker run --rm -v artifusion_registry-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/registry-backup.tar.gz -C /data .

# Backup Maven repository
docker run --rm -v artifusion_reposilite-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/reposilite-backup.tar.gz -C /data .

# Backup NPM registry
docker run --rm -v artifusion_verdaccio-storage:/data -v $(pwd):/backup alpine \
  tar czf /backup/verdaccio-backup.tar.gz -C /data .
```

### Restore Volumes

```bash
# Restore NPM packages
docker run --rm -v artifusion_verdaccio-storage:/data -v $(pwd):/backup alpine \
  tar xzf /backup/verdaccio-backup.tar.gz -C /data
```

## Configuration Files

### artifusion.yaml

Main Artifusion configuration. Key settings:

- **GitHub authentication**: Organization and team requirements
- **OCI backends**: Pull backends all point to oci-registry with namespace injection
- **Maven backends**: Reposilite read/write endpoints
- **NPM backend**: Verdaccio with GitHub Packages and npmjs.org proxying
- **Logging**: Level and format configuration

See `config/artifusion.yaml` for full configuration.

### verdaccio.yaml

NPM registry configuration for Verdaccio:

- **Uplinks**: GitHub Packages NPM registry and npmjs.org
- **Storage**: Local package storage for published packages
- **Packages**: Access rules and proxy order
  - Scoped packages (`@org/package`): Try GitHub first, then npmjs
  - Public packages: Try npmjs first for better performance
  - Private packages: Hosted only, no proxying
- **Authentication**: Disabled (handled by Artifusion)

**Proxy Order:**
```yaml
# Scoped packages (e.g., @myorg/package)
'@*/*':
  proxy: github npmjs  # GitHub first

# Public packages
'**':
  proxy: npmjs github  # npmjs first for performance
```

See `config/verdaccio.yaml` for full configuration.

### oci-registry-upstream.yaml

Configures upstream registries for oci-registry:

- **local.registry** → registry:5000 (checked first, no caching)
- **ghcr.io** → GitHub Container Registry (14-day cache)
- **docker.io** → Docker Hub (14-day cache)
- **quay.io** → Quay.io (14-day cache)

Add more upstreams by editing this file and restarting oci-registry.

### Private Registry Authentication

To pull from private upstream registries, set `UPSTREAM_CREDENTIALS`:

```bash
export UPSTREAM_CREDENTIALS='{
  "ghcr.io": {
    "username": "github-user",
    "password": "ghp_xxxxxxxxxxxx"
  },
  "docker.io": {
    "username": "dockerhub-user",
    "password": "dckr_pat_xxxxxxxxxxxx"
  }
}'
```

## Request Flow Examples

### Docker Pull Flow

```
Client: docker pull localhost:8080/myorg.io/nginx:latest
  ↓
Artifusion: Authenticate GitHub PAT → Validate org membership
  ↓
Try Backend #1: local.registry
  Rewrite: /v2/myorg.io/nginx/... → /v2/local.registry/nginx/...
  oci-registry → registry:5000
  Result: 404 (not found locally)
  ↓
Try Backend #2: ghcr.io
  Rewrite: /v2/myorg.io/nginx/... → /v2/ghcr.io/nginx/...
  oci-registry → https://ghcr.io
  Result: 404 (not in GHCR)
  ↓
Try Backend #3: docker.io
  Rewrite: /v2/nginx/... → /v2/docker.io/library/nginx/...
  oci-registry → https://registry-1.docker.io
  Result: 200 OK ✓ (cached for 14 days)
  ↓
Return to client
```

### Docker Push Flow

```
Client: docker push localhost:8080/myorg.io/my-app:latest
  ↓
Artifusion: Authenticate GitHub PAT → Validate org membership
  ↓
Detect write operation (POST /blobs/uploads)
  ↓
Route directly to: registry:5000 (bypass oci-registry)
  ↓
Store in local registry
  ↓
Return 201 Created
```

## Management

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f artifusion
docker-compose logs -f oci-registry
```

### Restart Services

```bash
# Restart all
docker-compose restart

# Restart specific service
docker-compose restart artifusion
```

### Update Configuration

After editing configuration files:

```bash
# Restart affected service
docker-compose restart artifusion

# Restart oci-registry after upstream changes
docker-compose restart oci-registry
```

### Clear Cache

```bash
# Clear oci-registry cache
docker-compose down
docker volume rm artifusion_oci-cache
docker-compose up -d
```

### Stop Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes all data)
docker-compose down -v
```

## Monitoring

### Health Checks

```bash
# Artifusion health
curl http://localhost:8080/health

# Artifusion readiness (includes backend checks)
curl http://localhost:8080/ready

# Prometheus metrics
curl http://localhost:8080/metrics
```

### Service Status

```bash
# Check all services
docker-compose ps

# Expected output:
# NAME              STATUS        PORTS
# artifusion        Up (healthy)  0.0.0.0:8080->8080/tcp
# oci-registry      Up (healthy)
# docker-registry   Up (healthy)
# reposilite         Up (healthy)
```

## Troubleshooting

### Authentication Failed

**Symptom:** `docker login` returns 401 Unauthorized

**Solutions:**
1. Verify GitHub PAT is valid: `curl -H "Authorization: Bearer ghp_xxx" https://api.github.com/user`
2. If using org restriction, check organization membership: `curl -H "Authorization: Bearer ghp_xxx" https://api.github.com/user/orgs`
3. Check Artifusion logs: `docker-compose logs artifusion | grep auth`
4. If using org restriction, verify GITHUB_ORG in `.env` matches your organization (or leave empty to allow any user)

### Image Pull 404

**Symptom:** `docker pull` returns 404 Not Found

**Solutions:**
1. Check all upstreams were tried: `docker-compose logs artifusion`
2. Verify image exists in one of the upstreams
3. Check oci-registry logs: `docker-compose logs oci-registry`
4. Test upstream directly: `docker pull ghcr.io/owner/image:tag`

### Image Push Failed

**Symptom:** `docker push` fails or hangs

**Solutions:**
1. Verify registry:2 is running: `docker-compose ps registry`
2. Check registry logs: `docker-compose logs registry`
3. Verify disk space: `df -h`
4. Check registry authentication in artifusion.yaml

### oci-registry Not Caching

**Symptom:** Every pull is slow, not using cache

**Solutions:**
1. Check cache volume: `docker volume inspect artifusion_oci-cache`
2. Verify invalidation times in `oci-registry-upstream.yaml` (should be 336h, not 0s)
3. Check oci-registry logs for cache hits: `docker-compose logs oci-registry | grep cache`

### Maven Deployment Failed

**Symptom:** `mvn deploy` returns 401 or connection refused

**Solutions:**
1. Verify reposilite is running: `docker-compose ps reposilite`
2. Check Maven settings.xml has correct GitHub PAT
3. Check artifusion logs: `docker-compose logs artifusion | grep maven`
4. Verify server ID matches in settings.xml and pom.xml

## Advanced Configuration

### Custom Upstream Registries

Add to `config/oci-registry-upstream.yaml`:

```yaml
- namespace: gcr.io
  host: gcr.io
  tls: true
  manifest_invalidation_time: 336h
  blob_invalidation_time: 336h
```

Add to `config/artifusion.yaml`:

```yaml
pullBackends:
  # ... existing backends ...
  - name: gcr
    url: http://oci-registry:8080
    priority: 5
    upstreamNamespace: gcr.io
    pathRewrite:
      stripPrefix: true
```

Restart services:

```bash
docker-compose restart oci-registry artifusion
```

### Enable Debug Logging

Edit `.env`:

```bash
LOG_LEVEL=debug
```

Restart:

```bash
docker-compose restart artifusion
```

### Persistent GitHub Token

For CI/CD, store token in Docker config:

```bash
echo "ghp_your_token" | docker login localhost:8080 -u github-user --password-stdin
```

## Production Considerations

### Security

1. **Use HTTPS**: Deploy behind nginx/Traefik with TLS termination
2. **Rotate Credentials**: Change REGISTRY_PASSWORD and REPOSILITE tokens regularly
3. **Network Isolation**: Use Docker networks to isolate backend services
4. **Secrets Management**: Use Docker secrets or external secret managers

### Performance

1. **Resource Limits**: Set memory/CPU limits in docker-compose.yml
2. **Cache Tuning**: Adjust invalidation times based on update frequency
3. **Volume Performance**: Use local volumes for production (not NFS)

### Backup

```bash
# Backup volumes
docker run --rm -v artifusion_registry-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/registry-backup.tar.gz -C /data .

docker run --rm -v artifusion_reposilite-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/reposilite-backup.tar.gz -C /data .
```

## Support

- **Issues**: https://github.com/yourorg/artifusion/issues
- **Documentation**: https://github.com/yourorg/artifusion/docs
- **Architecture**: See `.claude/ARCHITECTURE_PLAN.md`
