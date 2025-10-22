# Artifusion Environment Variables Reference

**Complete reference for all environment variables** used in Artifusion Docker deployment.

---

## Table of Contents

- [Overview](#overview)
- [Environment Variable Mapping](#environment-variable-mapping)
- [GitHub Authentication](#github-authentication)
- [Backend Services](#backend-services)
- [Server Configuration](#server-configuration)
- [Protocol Configuration](#protocol-configuration)
- [Logging](#logging)
- [Metrics](#metrics)
- [Rate Limiting](#rate-limiting)
- [Docker Compose Variables](#docker-compose-variables)
- [Quick Reference Table](#quick-reference-table)

---

## Overview

Artifusion uses **Viper** for configuration management with the following behavior:

- **Prefix**: All Artifusion env vars start with `ARTIFUSION_`
- **Mapping**: YAML keys use snake_case, env vars use UPPERCASE with underscores
  - Example: `server.port` → `ARTIFUSION_SERVER_PORT`
  - Example: `github.required_org` → `ARTIFUSION_GITHUB_REQUIRED_ORG`
- **Priority**: Environment variables override values in config files
- **Auto-loading**: All config keys can be overridden with env vars

---

## Environment Variable Mapping

### How Mapping Works

```yaml
# config.yaml
server:
  port: 8080
  external_url: "http://localhost:8080"

github:
  required_org: "myorg"
  required_teams: ["team1", "team2"]
```

Becomes:

```bash
ARTIFUSION_SERVER_PORT=8080
ARTIFUSION_SERVER_EXTERNAL_URL=http://localhost:8080
ARTIFUSION_GITHUB_REQUIRED_ORG=myorg
ARTIFUSION_GITHUB_REQUIRED_TEAMS=team1,team2
```

---

## GitHub Authentication

### Required Variables

#### `GITHUB_PACKAGES_TOKEN`
- **Purpose**: GitHub Personal Access Token for accessing GitHub Packages
- **Scopes Required**: `read:packages`
- **Used By**: Reposilite (Maven), OCI Registry (Docker)
- **Example**: `ghp_xxxxxxxxxxxxxxxxxxxx`
- **Note**: This is a **Docker Compose** variable, not an Artifusion variable

#### `ARTIFUSION_GITHUB_API_URL`
- **Purpose**: GitHub API endpoint URL
- **Default**: `https://api.github.com`
- **Example**: `https://api.github.com` (public), `https://github.enterprise.com/api/v3` (enterprise)
- **Config Key**: `github.api_url`

### Optional Restriction Variables

#### `ARTIFUSION_GITHUB_REQUIRED_ORG`
- **Purpose**: Restrict access to specific GitHub organization
- **Default**: `""` (allow any authenticated GitHub user)
- **Example**: `myorganization`
- **Config Key**: `github.required_org`

#### `ARTIFUSION_GITHUB_REQUIRED_TEAMS`
- **Purpose**: Restrict access to specific teams within organization
- **Format**: Comma-separated team slugs
- **Example**: `platform-team,security-team`
- **Note**: Only checked if `REQUIRED_ORG` is set
- **Config Key**: `github.required_teams`

### Cache Configuration

#### `ARTIFUSION_GITHUB_AUTH_CACHE_TTL`
- **Purpose**: Cache duration for authentication results
- **Default**: `30m`
- **Format**: Go duration string (`30m`, `1h`, `24h`)
- **Impact**: Reduces GitHub API calls by ~99%
- **Config Key**: `github.auth_cache_ttl`

#### `ARTIFUSION_GITHUB_RATE_LIMIT_BUFFER`
- **Purpose**: Warning threshold for GitHub API rate limit
- **Default**: `100`
- **Example**: Warn when remaining calls < 100
- **Config Key**: `github.rate_limit_buffer`

---

## Backend Services

### Docker Registry (With-Auth Mode)

#### `DOCKER_REGISTRY_USERNAME`
- **Purpose**: Username for Docker Registry authentication
- **Default**: `artifusion`
- **Used In**: `docker-compose.with-auth.yml`
- **Note**: Docker Compose variable, not Artifusion

#### `DOCKER_REGISTRY_PASSWORD`
- **Purpose**: Password for Docker Registry htpasswd auth
- **Required**: Only in with-auth mode
- **Generation**: `htpasswd -Bbn artifusion YOUR_PASSWORD`
- **Note**: Docker Compose variable, not Artifusion

### Reposilite (Maven) (With-Auth Mode)

#### `REPOSILITE_ADMIN_TOKEN`
- **Purpose**: Admin token for Reposilite management
- **Permissions**: Full access (management operations)
- **Generation**: `openssl rand -hex 32`
- **Note**: Docker Compose variable, not Artifusion

#### `REPOSILITE_WRITE_TOKEN`
- **Purpose**: Write token for deploying artifacts
- **Permissions**: Read/Write on `/releases/*` and `/snapshots/*`
- **Used By**: Artifusion backend authentication
- **Generation**: `openssl rand -hex 32`
- **Note**: Docker Compose variable, not Artifusion

#### `REPOSILITE_READ_TOKEN`
- **Purpose**: Read token for pulling dependencies
- **Permissions**: Read-only on all repositories
- **Used By**: Artifusion backend authentication
- **Generation**: `openssl rand -hex 32`
- **Note**: Docker Compose variable, not Artifusion

### Verdaccio (NPM) (With-Auth Mode)

#### `VERDACCIO_USERNAME`
- **Purpose**: Username for Verdaccio authentication
- **Default**: `artifusion`
- **Note**: Docker Compose variable, not Artifusion

#### `VERDACCIO_PASSWORD`
- **Purpose**: Password for Verdaccio authentication
- **Required**: Only in with-auth mode
- **Note**: Docker Compose variable, not Artifusion

---

## Server Configuration

### Core Settings

#### `ARTIFUSION_SERVER_PORT`
- **Purpose**: HTTP server listen port
- **Default**: `8080`
- **Example**: `8080`, `3000`
- **Config Key**: `server.port`

#### `ARTIFUSION_SERVER_EXTERNAL_URL`
- **Purpose**: External URL for reverse proxy deployments
- **Default**: Auto-detected from request headers
- **Example**: `https://artifusion.example.com`
- **Used For**: URL rewriting in Location headers, metadata files
- **Config Key**: `server.external_url`

### Timeouts

#### `ARTIFUSION_SERVER_READ_TIMEOUT`
- **Purpose**: Maximum duration for reading request
- **Default**: `60s`
- **Format**: Go duration (`60s`, `5m`)
- **Config Key**: `server.read_timeout`

#### `ARTIFUSION_SERVER_WRITE_TIMEOUT`
- **Purpose**: Maximum duration for writing response
- **Default**: `300s` (5 minutes)
- **Note**: Long timeout for large uploads
- **Config Key**: `server.write_timeout`

#### `ARTIFUSION_SERVER_IDLE_TIMEOUT`
- **Purpose**: Keep-alive idle timeout
- **Default**: `120s`
- **Config Key**: `server.idle_timeout`

#### `ARTIFUSION_SERVER_SHUTDOWN_TIMEOUT`
- **Purpose**: Graceful shutdown timeout
- **Default**: `30s`
- **Config Key**: `server.shutdown_timeout`

### Performance

#### `ARTIFUSION_SERVER_MAX_CONCURRENT_REQUESTS`
- **Purpose**: Maximum concurrent requests allowed
- **Default**: `10000`
- **Config Key**: `server.max_concurrent_requests`

#### `ARTIFUSION_SERVER_MAX_HEADER_BYTES`
- **Purpose**: Maximum size of request headers
- **Default**: `1048576` (1MB)
- **Config Key**: `server.max_header_bytes`

#### `ARTIFUSION_SERVER_READ_BUFFER_SIZE`
- **Purpose**: Size of read buffer
- **Default**: `4096` (4KB)
- **Config Key**: `server.read_buffer_size`

#### `ARTIFUSION_SERVER_WRITE_BUFFER_SIZE`
- **Purpose**: Size of write buffer
- **Default**: `4096` (4KB)
- **Config Key**: `server.write_buffer_size`

---

## Protocol Configuration

### Maven Protocol

#### `ARTIFUSION_PROTOCOLS_MAVEN_ENABLED`
- **Purpose**: Enable/disable Maven protocol handler
- **Default**: `true`
- **Values**: `true`, `false`
- **Config Key**: `protocols.maven.enabled`

#### `ARTIFUSION_PROTOCOLS_MAVEN_PATH_PREFIX`
- **Purpose**: URL path prefix for Maven requests
- **Default**: `/maven`
- **Example**: All Maven requests start with `/maven/*`
- **Config Key**: `protocols.maven.path_prefix`

#### `ARTIFUSION_PROTOCOLS_MAVEN_BACKEND_URL`
- **Purpose**: Reposilite backend URL
- **Default**: `http://reposilite:8080`
- **Example**: `http://reposilite:8080`, `http://maven.internal:8081`
- **Config Key**: `protocols.maven.backend.url`

### NPM Protocol

#### `ARTIFUSION_PROTOCOLS_NPM_ENABLED`
- **Purpose**: Enable/disable NPM protocol handler
- **Default**: `true`
- **Values**: `true`, `false`
- **Config Key**: `protocols.npm.enabled`

#### `ARTIFUSION_PROTOCOLS_NPM_PATH_PREFIX`
- **Purpose**: URL path prefix for NPM requests
- **Default**: `/npm`
- **Example**: All NPM requests start with `/npm/*`
- **Config Key**: `protocols.npm.path_prefix`

#### `ARTIFUSION_PROTOCOLS_NPM_BACKEND_URL`
- **Purpose**: Verdaccio backend URL
- **Default**: `http://verdaccio:4873`
- **Example**: `http://verdaccio:4873`, `http://npm.internal:4873`
- **Config Key**: `protocols.npm.backend.url`

### OCI/Docker Protocol

#### `ARTIFUSION_PROTOCOLS_OCI_ENABLED`
- **Purpose**: Enable/disable OCI/Docker protocol handler
- **Default**: `true`
- **Values**: `true`, `false`
- **Config Key**: `protocols.oci.enabled`

#### `ARTIFUSION_PROTOCOLS_OCI_PUSHBACKEND_URL`
- **Purpose**: Docker Registry push backend URL
- **Default**: `http://registry:5000`
- **Example**: `http://registry:5000`, `http://docker.internal:5000`
- **Config Key**: `protocols.oci.pushBackend.url`

---

## Logging

### Format and Level

#### `ARTIFUSION_LOGGING_LEVEL`
- **Purpose**: Log level
- **Default**: `info`
- **Values**: `debug`, `info`, `warn`, `error`
- **Config Key**: `logging.level`

#### `ARTIFUSION_LOGGING_FORMAT`
- **Purpose**: Log output format
- **Default**: `console`
- **Values**:
  - `console` - Human-readable with auto-detected TTY colors
  - `json` - Structured JSON for log aggregation
- **Config Key**: `logging.format`

### Debug Options

#### `ARTIFUSION_LOGGING_INCLUDE_HEADERS`
- **Purpose**: Include HTTP headers in logs
- **Default**: `false`
- **Warning**: May log sensitive data
- **Config Key**: `logging.include_headers`

#### `ARTIFUSION_LOGGING_INCLUDE_BODY`
- **Purpose**: Include request/response bodies in logs
- **Default**: `false`
- **Warning**: May log sensitive data (tokens, passwords)
- **Config Key**: `logging.include_body`

---

## Metrics

### Prometheus Configuration

#### `ARTIFUSION_METRICS_ENABLED`
- **Purpose**: Enable/disable Prometheus metrics endpoint
- **Default**: `true`
- **Config Key**: `metrics.enabled`

#### `ARTIFUSION_METRICS_PATH`
- **Purpose**: Metrics endpoint path
- **Default**: `/metrics`
- **Example**: Access at `http://localhost:8080/metrics`
- **Config Key**: `metrics.path`

---

## Rate Limiting

### Global Rate Limiting

#### `ARTIFUSION_RATE_LIMIT_ENABLED`
- **Purpose**: Enable global rate limiting
- **Default**: `true`
- **Config Key**: `rate_limit.enabled`

#### `ARTIFUSION_RATE_LIMIT_REQUESTS_PER_SEC`
- **Purpose**: Requests per second (global)
- **Default**: `1000.0`
- **Example**: `1000.0`, `5000.0`
- **Config Key**: `rate_limit.requests_per_sec`

#### `ARTIFUSION_RATE_LIMIT_BURST`
- **Purpose**: Burst size for global rate limit
- **Default**: `2000`
- **Example**: `2000`, `10000`
- **Config Key**: `rate_limit.burst`

### Per-User Rate Limiting

#### `ARTIFUSION_RATE_LIMIT_PER_USER_ENABLED`
- **Purpose**: Enable per-user rate limiting
- **Default**: `true`
- **Config Key**: `rate_limit.per_user_enabled`

#### `ARTIFUSION_RATE_LIMIT_PER_USER_REQUESTS`
- **Purpose**: Requests per second (per user)
- **Default**: `100.0`
- **Example**: `100.0`, `500.0`
- **Config Key**: `rate_limit.per_user_requests`

#### `ARTIFUSION_RATE_LIMIT_PER_USER_BURST`
- **Purpose**: Burst size for per-user rate limit
- **Default**: `200`
- **Example**: `200`, `1000`
- **Config Key**: `rate_limit.per_user_burst`

---

## Docker Compose Variables

These variables are used **only by Docker Compose**, not by Artifusion itself:

### Deployment Mode

#### `GITHUB_ORG`
- **Purpose**: Pass to Artifusion config
- **Maps To**: `ARTIFUSION_GITHUB_REQUIRED_ORG`

#### `GITHUB_TEAMS`
- **Purpose**: Pass to Artifusion config
- **Maps To**: `ARTIFUSION_GITHUB_REQUIRED_TEAMS`

#### `AUTH_CACHE_TTL`
- **Purpose**: Pass to Artifusion config
- **Maps To**: `ARTIFUSION_GITHUB_AUTH_CACHE_TTL`

### Logging

#### `LOG_LEVEL`
- **Purpose**: Pass to Artifusion config
- **Maps To**: `ARTIFUSION_LOGGING_LEVEL`

#### `LOG_FORMAT`
- **Purpose**: Pass to Artifusion config
- **Maps To**: `ARTIFUSION_LOGGING_FORMAT`

#### `VERDACCIO_LOG_LEVEL`
- **Purpose**: Verdaccio log level
- **Values**: `http`, `debug`, `info`, `warn`, `error`
- **Default**: `info`

---

## Quick Reference Table

| Config Key | Environment Variable | Default | Type |
|------------|---------------------|---------|------|
| `server.port` | `ARTIFUSION_SERVER_PORT` | `8080` | int |
| `server.external_url` | `ARTIFUSION_SERVER_EXTERNAL_URL` | Auto-detected | string |
| `server.read_timeout` | `ARTIFUSION_SERVER_READ_TIMEOUT` | `60s` | duration |
| `server.write_timeout` | `ARTIFUSION_SERVER_WRITE_TIMEOUT` | `300s` | duration |
| `server.idle_timeout` | `ARTIFUSION_SERVER_IDLE_TIMEOUT` | `120s` | duration |
| `server.shutdown_timeout` | `ARTIFUSION_SERVER_SHUTDOWN_TIMEOUT` | `30s` | duration |
| `server.max_header_bytes` | `ARTIFUSION_SERVER_MAX_HEADER_BYTES` | `1048576` | int |
| `server.read_buffer_size` | `ARTIFUSION_SERVER_READ_BUFFER_SIZE` | `4096` | int |
| `server.write_buffer_size` | `ARTIFUSION_SERVER_WRITE_BUFFER_SIZE` | `4096` | int |
| `server.max_concurrent_requests` | `ARTIFUSION_SERVER_MAX_CONCURRENT_REQUESTS` | `10000` | int |
| `github.api_url` | `ARTIFUSION_GITHUB_API_URL` | `https://api.github.com` | string |
| `github.required_org` | `ARTIFUSION_GITHUB_REQUIRED_ORG` | `""` | string |
| `github.required_teams` | `ARTIFUSION_GITHUB_REQUIRED_TEAMS` | `[]` | []string |
| `github.auth_cache_ttl` | `ARTIFUSION_GITHUB_AUTH_CACHE_TTL` | `30m` | duration |
| `github.rate_limit_buffer` | `ARTIFUSION_GITHUB_RATE_LIMIT_BUFFER` | `100` | int |
| `protocols.maven.enabled` | `ARTIFUSION_PROTOCOLS_MAVEN_ENABLED` | `true` | bool |
| `protocols.maven.path_prefix` | `ARTIFUSION_PROTOCOLS_MAVEN_PATH_PREFIX` | `/maven` | string |
| `protocols.maven.backend.url` | `ARTIFUSION_PROTOCOLS_MAVEN_BACKEND_URL` | `http://reposilite:8080` | string |
| `protocols.npm.enabled` | `ARTIFUSION_PROTOCOLS_NPM_ENABLED` | `true` | bool |
| `protocols.npm.path_prefix` | `ARTIFUSION_PROTOCOLS_NPM_PATH_PREFIX` | `/npm` | string |
| `protocols.npm.backend.url` | `ARTIFUSION_PROTOCOLS_NPM_BACKEND_URL` | `http://verdaccio:4873` | string |
| `protocols.oci.enabled` | `ARTIFUSION_PROTOCOLS_OCI_ENABLED` | `true` | bool |
| `protocols.oci.pushBackend.url` | `ARTIFUSION_PROTOCOLS_OCI_PUSHBACKEND_URL` | `http://registry:5000` | string |
| `logging.level` | `ARTIFUSION_LOGGING_LEVEL` | `info` | string |
| `logging.format` | `ARTIFUSION_LOGGING_FORMAT` | `console` | string |
| `logging.include_headers` | `ARTIFUSION_LOGGING_INCLUDE_HEADERS` | `false` | bool |
| `logging.include_body` | `ARTIFUSION_LOGGING_INCLUDE_BODY` | `false` | bool |
| `metrics.enabled` | `ARTIFUSION_METRICS_ENABLED` | `true` | bool |
| `metrics.path` | `ARTIFUSION_METRICS_PATH` | `/metrics` | string |
| `rate_limit.enabled` | `ARTIFUSION_RATE_LIMIT_ENABLED` | `true` | bool |
| `rate_limit.requests_per_sec` | `ARTIFUSION_RATE_LIMIT_REQUESTS_PER_SEC` | `1000.0` | float64 |
| `rate_limit.burst` | `ARTIFUSION_RATE_LIMIT_BURST` | `2000` | int |
| `rate_limit.per_user_enabled` | `ARTIFUSION_RATE_LIMIT_PER_USER_ENABLED` | `true` | bool |
| `rate_limit.per_user_requests` | `ARTIFUSION_RATE_LIMIT_PER_USER_REQUESTS` | `100.0` | float64 |
| `rate_limit.per_user_burst` | `ARTIFUSION_RATE_LIMIT_PER_USER_BURST` | `200` | int |

### Backend Service Variables (Docker Compose Only)

| Variable | Purpose | Used In | Type |
|----------|---------|---------|------|
| `GITHUB_PACKAGES_TOKEN` | GitHub Packages access | Reposilite, OCI Registry | Docker Compose |
| `DOCKER_REGISTRY_USERNAME` | Registry auth username | Docker Registry | Docker Compose (with-auth) |
| `DOCKER_REGISTRY_PASSWORD` | Registry auth password | Docker Registry | Docker Compose (with-auth) |
| `REPOSILITE_ADMIN_TOKEN` | Reposilite admin token | Reposilite | Docker Compose (with-auth) |
| `REPOSILITE_WRITE_TOKEN` | Reposilite write token | Reposilite, Artifusion | Docker Compose (with-auth) |
| `REPOSILITE_READ_TOKEN` | Reposilite read token | Reposilite, Artifusion | Docker Compose (with-auth) |
| `VERDACCIO_USERNAME` | Verdaccio auth username | Verdaccio | Docker Compose (with-auth) |
| `VERDACCIO_PASSWORD` | Verdaccio auth password | Verdaccio | Docker Compose (with-auth) |

---

## Examples

### Development Setup

```bash
# .env
GITHUB_PACKAGES_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
LOG_LEVEL=debug
LOG_FORMAT=console
```

### Production Setup

```bash
# .env (or secrets manager)
GITHUB_PACKAGES_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
GITHUB_ORG=myorganization
GITHUB_TEAMS=platform-team,security-team

# With-auth mode
DOCKER_REGISTRY_USERNAME=artifusion
DOCKER_REGISTRY_PASSWORD=$(openssl rand -base64 24)
REPOSILITE_ADMIN_TOKEN=$(openssl rand -hex 32)
REPOSILITE_WRITE_TOKEN=$(openssl rand -hex 32)
REPOSILITE_READ_TOKEN=$(openssl rand -hex 32)
VERDACCIO_USERNAME=artifusion
VERDACCIO_PASSWORD=$(openssl rand -base64 24)

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Override Config with Environment

```bash
# Override port
export ARTIFUSION_SERVER_PORT=3000

# Override external URL
export ARTIFUSION_SERVER_EXTERNAL_URL=https://artifusion.company.com

# Override Maven path prefix
export ARTIFUSION_PROTOCOLS_MAVEN_PATH_PREFIX=/m2

# Override rate limiting
export ARTIFUSION_RATE_LIMIT_REQUESTS_PER_SEC=5000.0
export ARTIFUSION_RATE_LIMIT_BURST=10000

# Start Artifusion (env vars override config file)
./artifusion --config config.yaml
```

---

## See Also

- [Quick Start Guide](QUICKSTART.md)
- [Authentication Setup](AUTHENTICATION_SETUP.md)
- [Deployment Modes](DEPLOYMENT_MODES.md)
- [Configuration Examples](../config/)

---

**Note**: All `ARTIFUSION_*` variables directly map to the YAML configuration structure. Any config value can be overridden using the corresponding environment variable.
