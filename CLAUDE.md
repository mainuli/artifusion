# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

**Artifusion** is a production-ready, multi-protocol artifact reverse proxy with centralized GitHub authentication. It acts as a unified gateway for OCI/Docker, Maven, and NPM repositories with high-concurrency support, circuit breakers, and comprehensive observability.

**Architecture**: Middleware-based HTTP proxy with protocol detection chain and fault-tolerant backend routing.

**Status**: Production Ready - 112 tests, zero critical issues, comprehensive metrics.

## Quick Start

```bash
# Build and test
make build         # Build binary (creates bin/artifusion)
make test          # Run all tests with race detection
make lint          # Run linters
make ci            # Run complete CI pipeline (lint + test + build)

# Run locally
cp config/config.example.yaml config/config.yaml
make run

# Docker
make docker-build
cd deployments/docker && docker-compose up -d

# Health checks
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

## Architecture

### Request Flow

```
Client Request
    ↓
[Middleware Stack]
  RequestID → SecurityHeaders → Recovery → Logging →
  Timeout → ConcurrencyLimiter → RateLimiter
    ↓
[Protocol Detection]
  OCI/Docker, Maven, NPM (host or path-based)
    ↓
[Protocol Handler]
  GitHub PAT Auth → Backend Routing → Circuit Breaker →
  Proxying → Response Rewriting
    ↓
[Upstream Backends]
  OCI: GHCR, Docker Hub, local registry (cascading)
  Maven: Reposilite 3 (GitHub Packages, Maven Central, etc.)
  NPM: Verdaccio (npmjs.org cache)
```

### Key Patterns

1. **Shared Authentication** (`internal/auth/client_auth.go`)
   - GitHub PAT validation with SHA256-hashed token caching (5min TTL)
   - Singleflight pattern prevents thundering herd
   - Supports `ghp_*`, `github_pat_*`, and `ghs_*` tokens

2. **Shared Proxy Client** (`internal/proxy/client.go`)
   - Connection pooling (200 idle connections per host)
   - Per-backend circuit breakers
   - Comprehensive metrics recording

3. **Protocol Detection** (`internal/detector/`)
   - Chain of Responsibility pattern
   - Host-based and path-based routing

4. **Response Rewriting** (`internal/handler/*/rewriter.go`)
   - Rewrites backend URLs to public URLs
   - Handles `Location` and `WWW-Authenticate` headers

## Code Organization

```
internal/
├── auth/                # GitHub PAT authentication
│   └── client_auth.go   # SHARED by all handlers
├── config/              # Configuration management (Viper)
├── detector/            # Protocol detection chain
├── handler/             # Protocol handlers
│   ├── oci/            # Docker Registry v2
│   ├── maven/          # Maven repository
│   └── npm/            # NPM registry
├── middleware/          # HTTP middleware (7 layers)
├── proxy/               # Shared HTTP client with circuit breakers
│   └── client.go        # SHARED by all handlers
├── metrics/             # Prometheus metrics
└── health/              # Health check endpoints
```

**Important relationships**:
- All handlers use `internal/auth/client_auth.go` for authentication
- All handlers use `internal/proxy/client.go` for backend proxying
- Middleware chain defined in `cmd/artifusion/main.go` (order matters!)

## Configuration

### Routing Models

**Host-based** (different domains per protocol):
```yaml
protocols:
  oci:
    host: "docker.example.com"  # OCI always uses /v2 path (not configurable)
  maven:
    host: "maven.example.com"
```

**Path-based** (shared domain):
```yaml
protocols:
  oci:
    host: ""  # OCI always uses /v2 path (not configurable)
  maven:
    host: ""
    path_prefix: "/maven"  # REQUIRED when host is empty
```

**Important**: OCI protocol does NOT support custom `path_prefix` - it always uses `/v2` per OCI Distribution Spec.

### Key Configuration Fields

```yaml
github:
  required_org: "myorg"        # Optional - leave empty to allow any GitHub user
  auth_cache_ttl: 30m          # Reduces GitHub API calls by ~99%

server:
  max_concurrent_requests: 10000

rate_limit:
  requests_per_sec: 1000.0     # Global
  per_user_requests: 100.0     # Per-user
```

### Environment Variables

All config fields can be overridden with pattern: `ARTIFUSION_<SECTION>_<KEY>`

Examples:
- `ARTIFUSION_GITHUB_REQUIRED_ORG` → `github.required_org`
- `ARTIFUSION_SERVER_PORT` → `server.port`
- `ARTIFUSION_METRICS_ENABLED` → `metrics.enabled`

Special variables (handled in main.go):
- `CONFIG_PATH` - Path to config.yaml
- `ARTIFUSION_LOGGING_LEVEL` - Log level (debug, info, warn, error)
- `ARTIFUSION_LOGGING_FORMAT` - Format (console, json)

## Critical Implementation Details

### Authentication Flow

1. Client provides GitHub PAT via Basic Auth or Bearer token
2. Token format validated via regex
3. Token hashed with SHA256 for cache lookup
4. On cache miss: GitHub API validates token + org/team membership
5. Result cached for 5 minutes
6. Singleflight prevents duplicate API calls

**IMPORTANT**: Never store tokens in plaintext. Always use `internal/auth/cache.go` which stores SHA256 hashes only.

### Circuit Breaker Pattern

- Per-backend circuit breakers in `internal/proxy/circuit_breaker.go`
- Uses `github.com/sony/gobreaker` library
- States: Closed (normal) → Open (failing) → Half-Open (testing)
- Always wrap backend calls in circuit breaker via `proxy.Client.ProxyRequestWithCircuitBreaker()`

### OCI Cascading Backends

OCI pull requests cascade through backends in priority order:
1. Check local registry first
2. If 404, try next backend
3. Continue until success or all exhausted

**Code location**: `internal/handler/oci/proxy.go:cascadePullRequest()`

### Logging Standards

Use `github.com/rs/zerolog` for structured logging:

```go
log.Info().
    Str("requestID", requestID).
    Str("protocol", "oci").
    Str("backend", backendName).
    Msg("Proxying to backend")
```

Always include `requestID` for request correlation.

### Metrics

Record metrics for all backend calls:

```go
metrics.RecordBackendLatency(backendName, method, duration)
metrics.RecordBackendHealth(backendName, isHealthy)
metrics.RecordBackendError(backendName, errorType, statusCode)
```

**Prometheus endpoint**: `http://localhost:8080/metrics`

## Maven Backend (Reposilite 3)

### Configuration Files

Reposilite 3 uses two configuration files:

1. **Local Config** (`configuration.cdn`) - Infrastructure settings
   - Network binding, ports, database, thread pools
   - `defaultFrontend: false` - Disable web UI (Artifusion handles auth)

2. **Shared Config** (`configuration.shared.json`) - Repository definitions
   - JSON object with domain keys: `authentication`, `statistics`, `web`, `maven`, `frontend`
   - **CRITICAL**: Must be single JSON object, NOT an array
   - Contains repository config with proxied upstreams

### Unified Repository Approach

Single repository `maven` handles both deployments and dependencies:

**Proxied Upstreams** (cascades in order):
1. GitHub Packages (if org configured with BASIC auth)
2. Maven Central
3. JasperReports JFrog
4. Spring Releases
5. Sonatype OSS Snapshots
6. Gradle Plugins

### Artifusion Routing

Artifusion strips `path_prefix` before forwarding, so backend URL must include repository name:

```yaml
maven:
  path_prefix: /maven
  backend:
    url: http://reposilite:8080/maven  # Must include repository name!
```

Request flow:
```
GET /maven/com/example/app/1.0.0/app-1.0.0.jar
→ Strip prefix: /com/example/app/1.0.0/app-1.0.0.jar
→ Append to backend: http://reposilite:8080/maven/com/example/app/1.0.0/app-1.0.0.jar
```

### Config Locations

- **Docker**: `deployments/docker/config/configuration.shared.json`
  - Replace `YOUR_ORG` in GitHub Packages URL
  - Set `GITHUB_PACKAGES_USERNAME` and `GITHUB_PACKAGES_TOKEN` env vars

- **Kubernetes**: `deployments/helm/artifusion/templates/reposilite/configmap.yaml`
  - Org automatically templated from Helm values
  - Credentials injected from secrets

## Backend Authentication

Backend auth is **separate** from client auth (GitHub PATs).

```
Client → [GitHub PAT] → Artifusion → [Backend Auth] → Backend
```

### Supported Types

**1. Basic Auth**:
```yaml
backend:
  auth:
    type: basic
    username: admin
    password: ${REPOSILITE_ADMIN_TOKEN}  # From environment variable
```

**2. Bearer Token**:
```yaml
backend:
  auth:
    type: bearer
    token: your-token
```

**3. Custom Header**:
```yaml
backend:
  auth:
    type: header
    header_name: X-API-Key
    header_value: your-key
```

### Helm Chart Auto-Generated Secrets

The Helm chart automatically generates secure random admin token for Reposilite:
- Generates 32-char token on first install
- Preserves token on upgrades
- Injects as `REPOSILITE_ADMIN_TOKEN` environment variable
- No manual configuration needed!

Retrieve token:
```bash
kubectl get secret artifusion-reposilite-admin -n <namespace> \
  -o jsonpath='{.data.admin-token}' | base64 -d
```

### When to Use

**Use backend auth when**:
- Backend requires authentication
- Backend exposed outside cluster
- Compliance requirements

**Skip when**:
- Backend on private network
- Using NetworkPolicy for access control

**Code location**: `internal/proxy/client.go:injectBackendAuth()`

## Testing

- 112 tests across 14 test files
- Race detection enabled (`-race` flag)
- Table-driven tests
- Standard Go `testing` package only

```bash
make test                    # All tests
make test-coverage           # HTML coverage report
go test -v ./internal/auth/  # Specific package
```

Key test files:
- `internal/auth/*_test.go` - Token validation, caching
- `internal/config/validation_test.go` - 44 config validation scenarios
- `internal/handler/oci/rewriter_test.go` - 41 URL rewriting cases

## Development Patterns

### Adding a Protocol Handler

1. Create package `internal/handler/<protocol>/`
2. Implement `handler.Handler` interface
3. Create `routes.go`, `auth.go`, `proxy.go`, `rewriter.go`
4. Add detector `internal/detector/<protocol>.go`
5. Wire up in `cmd/artifusion/main.go`
6. Add config struct in `internal/config/config.go`
7. Add validation in `internal/config/validation.go`
8. Write tests

### Adding Middleware

1. Create `internal/middleware/<name>.go`
2. Implement `func(http.Handler) http.Handler`
3. Add to chain in `cmd/artifusion/main.go` (order matters!)
4. Add config if needed
5. Write tests

### Adding Metrics

```go
// Define in internal/metrics/metrics.go
var myMetric = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "artifusion_my_metric_total",
        Help: "Description",
    },
    []string{"label1", "label2"},
)

// Register in init()
prometheus.MustRegister(myMetric)

// Record
myMetric.WithLabelValues(val1, val2).Inc()
```

## Deployment

### Docker

- Multi-stage build with Chainguard static base
- Non-root user (UID 65532)
- ~10MB final image

### Health Checks

- `/health` - Liveness
- `/ready` - Readiness
- `/metrics` - Prometheus metrics

### Reverse Proxy Headers

When behind Nginx/Traefik, ensure these headers:
- `X-Forwarded-Host`
- `X-Forwarded-Proto`
- `X-Forwarded-For`

### Performance

- Connection pooling: 200 idle connections/host
- Max concurrent requests: 10,000 (configurable)
- Rate limiting: 1,000 req/sec global, 100 per-user
- Timeouts: 60s read, 300s write

### Scalability

- Stateless architecture - safe to run multiple instances
- Horizontal scaling with load balancing
- No inter-instance coordination needed
- Local auth cache reduces GitHub API calls by ~99%

## Troubleshooting

### High Latency
```bash
curl http://localhost:8080/metrics | grep artifusion_backend_health
curl http://localhost:8080/metrics | grep circuit_breaker_state
```

### Auth Failures
- Verify PAT format: `ghp_*`, `github_pat_*`, or `ghs_*`
- Check org membership: `curl -H "Authorization: token $PAT" https://api.github.com/user/orgs`
- Review logs with `requestID` for correlation

### Rate Limiting
- Check `artifusion_rate_limit_exceeded_total` metric
- Adjust `rate_limit.requests_per_sec` or `rate_limit.per_user_requests`

### Circuit Breaker Open
- Identify failing backend: `grep circuit_breaker_state | grep " 1"`
- Check backend health directly
- Auto-recovers after timeout period

## Additional Documentation

- **Architecture**: `docs/architecture/FINAL_ARCHITECTURE_SUMMARY.md`
- **Configuration**: `config/config.example.yaml`
- **Protocol Details**: `docs/PROTOCOL_BACKEND_TYPES.md`
- **Deployment**: `deployments/docker/README.md`
- **Migration**: `docs/CONFIG_MIGRATION.md`