# Docker Compose Deployment

Single optimized compose file with Reposilite backend authentication only.

## Quick Start

```bash
cp .env.example .env
echo "REPOSILITE_ADMIN_TOKEN=$(openssl rand -hex 16)" >> .env
docker-compose up -d
```

## Architecture

**Backend Authentication:**
- ✅ **Reposilite** - Token auth enabled (via REPOSILITE_ADMIN_TOKEN)
- ❌ **Docker Registry** - No auth (private network)
- ❌ **Verdaccio** - No auth (private network)

**Services:**
- `artifusion` (8080) - Reverse proxy with GitHub auth
- `reposilite` - Maven repository
- `verdaccio` - NPM registry
- `registry` - Docker registry
- `oci-registry` - OCI cache

## Configuration

**Required:**
- `REPOSILITE_ADMIN_TOKEN` - Backend auth token

**Optional:**
- `GITHUB_ORG` - Organization restriction
- `GITHUB_PACKAGES_TOKEN` - GitHub Packages proxy
- `LOG_LEVEL` - Logging (info/debug/warn/error)

## Common Commands

```bash
docker-compose up -d       # Start
docker-compose ps          # Status
docker-compose logs -f     # Logs
docker-compose down        # Stop
```

## Health Checks

```bash
curl http://localhost:8080/health  # Artifusion
curl http://localhost:5000/v2/     # Docker Registry
curl http://localhost:4873/-/ping  # Verdaccio
```
