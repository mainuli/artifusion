# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Artifusion** is a production-ready, multi-protocol artifact reverse proxy with centralized GitHub authentication. It acts as a unified gateway for OCI/Docker, Maven, and NPM repositories with high-concurrency support, circuit breakers, and comprehensive observability.

**Architecture**: Middleware-based HTTP proxy with protocol detection chain and fault-tolerant backend routing.

**Status**: Production Ready (Grade A) - 112 tests, zero critical issues, comprehensive metrics.

## Build and Development Commands

### Essential Commands

```bash
# Build binary with version injection (creates bin/artifusion)
make build

# Run all tests with race detection
make test

# Generate HTML coverage report
make test-coverage
# View: open coverage.html

# Run linters (go vet, go fmt check)
make lint

# Format code
make fmt

# Clean build artifacts
make clean
```

### Running Locally

```bash
# Copy example configuration
cp config/config.example.yaml config/config.yaml
# Edit config.yaml with your settings (GitHub org, backend URLs, etc.)

# Run with local config
make run

# Or run in development mode
make run-dev
```

### Docker Commands

```bash
# Build Docker image (multi-stage, Chainguard-based)
make docker-build

# Start all services (Artifusion + backends: Verdaccio, Reposilite, Registry)
cd deployments/docker
docker-compose up -d

# Check health
curl http://localhost:8080/health
curl http://localhost:8080/metrics

# View logs
docker-compose logs -f artifusion

# Stop services
docker-compose down
```

### Testing Specific Packages

```bash
# Test specific package
go test -v ./internal/auth/...
go test -v ./internal/middleware/...
go test -v ./internal/handler/oci/...

# Test with coverage for specific package
go test -v -cover ./internal/auth/...

# Run benchmarks
make bench
```

### CI/Complete Build Pipeline

```bash
# Run complete CI pipeline (lint + test + build)
make ci

# Run all checks and build
make all
```

## High-Level Architecture

### Request Flow

```
Client Request
    ↓
[Middleware Stack - 7 layers in order]
  1. RequestID          - Generate unique request ID
  2. SecurityHeaders    - Add 8 security headers (HSTS, CSP, etc.)
  3. Recovery           - Panic recovery with stack traces
  4. Logging            - Structured request/response logging (zerolog)
  5. Timeout            - Request timeout enforcement (configurable)
  6. ConcurrencyLimiter - Max concurrent request limiting (semaphore)
  7. RateLimiter        - Global + per-user rate limiting (token bucket)
    ↓
[Protocol Detection Chain]
  - OCI/Docker Registry v2 (host or path-based detection)
  - Maven Repository (host or path-based detection)
  - NPM Registry (host or path-based detection)
    ↓
[Protocol Handler - per protocol]
  - Client Authentication (GitHub PAT validation)
  - Path/Namespace Rewriting
  - Backend Selection (cascading for OCI, single for Maven/NPM)
  - Circuit Breaker Execution (per-backend fault isolation)
  - Async Proxying with Connection Pooling
  - Response Rewriting (Location, WWW-Authenticate headers)
    ↓
[Upstream Backends]
  - OCI: GHCR, Docker Hub, Quay, local registry (cascading)
  - Maven: Reposilite (mirrors Maven Central, GitHub Packages, etc.)
  - NPM: Verdaccio (caches npmjs.org)
```

### Key Architectural Patterns

1. **Shared Authentication Layer** (`internal/auth/client_auth.go`)
   - All protocol handlers use common GitHub PAT validation
   - SHA256-hashed token caching (5min TTL, never stores plaintext)
   - Singleflight pattern prevents thundering herd on cache misses
   - Supports PATs (`ghp_*`), fine-grained PATs (`github_pat_*`), and GitHub Actions tokens (`ghs_*`)

2. **Shared Proxy Client** (`internal/proxy/client.go`)
   - Connection pooling (200 idle connections per host)
   - Integrated circuit breakers (per-backend fault tolerance)
   - Comprehensive metrics recording (latency histograms, error counters)
   - Timeout handling with context propagation

3. **Protocol Detection Chain** (`internal/detector/`)
   - Chain of Responsibility pattern
   - Supports both host-based and path-based routing
   - OCI: Detects by host match OR path prefix
   - Maven/NPM: Detects by path prefix (e.g., `/maven/`, `/npm/`)

4. **Response Rewriting** (`internal/handler/*/rewriter.go`)
   - Rewrites backend URLs to public-facing URLs
   - Handles `Location` and `WWW-Authenticate` headers
   - Public URL detection from `X-Forwarded-Host`, `X-Forwarded-Proto`, `Forwarded` headers
   - Critical for reverse proxy deployments

## Code Organization

### Package Structure

```
internal/
├── auth/                # GitHub authentication (8 files)
│   ├── client_auth.go   # SHARED auth layer used by all handlers
│   ├── github.go        # GitHub API client
│   ├── token_validator.go
│   └── cache.go         # SHA256-hashed token cache with TTL
│
├── config/              # Configuration management (4 files)
│   ├── config.go        # Config struct definitions
│   ├── loader.go        # YAML loading with Viper
│   └── validation.go    # Config validation rules
│
├── detector/            # Protocol detection (5 files)
│   ├── detector.go      # Chain pattern implementation
│   ├── oci.go          # Host/path-based OCI detection
│   ├── maven.go        # Maven detection
│   └── npm.go          # NPM detection
│
├── handler/             # Protocol handlers (20 files)
│   ├── handler.go       # Handler interface
│   ├── oci/            # Docker Registry v2 (6 files)
│   │   ├── handler.go   # Main handler
│   │   ├── routes.go    # Route definitions
│   │   ├── auth.go      # Authentication logic
│   │   ├── proxy.go     # Backend cascading and proxying
│   │   └── rewriter.go  # Response URL rewriting
│   ├── maven/          # Maven repository (6 files)
│   │   └── [same structure as oci/]
│   └── npm/            # NPM registry (6 files)
│       └── [same structure as oci/]
│
├── middleware/          # HTTP middleware (10 files)
│   ├── requestid.go     # Request ID generation
│   ├── security.go      # 8 security headers
│   ├── recovery.go      # Panic recovery
│   ├── logging.go       # Structured logging
│   ├── timeout.go       # Request timeout
│   ├── concurrency.go   # Concurrency limiter
│   └── ratelimit.go     # Rate limiting (global + per-user)
│
├── proxy/               # Shared proxy client (4 files)
│   ├── client.go        # SHARED HTTP client with pooling
│   ├── circuit_breaker.go # Per-backend circuit breakers
│   └── rewriter/
│       └── rewriter.go  # URL rewriting utilities
│
├── metrics/             # Prometheus metrics
│   └── metrics.go       # 15+ metric definitions
│
├── health/              # Health checks
│   └── health.go        # Liveness/readiness endpoints
│
├── logging/             # Logging setup
│   └── setup.go         # Zerolog configuration
│
├── errors/              # Error handling
│   └── errors.go        # Structured error types
│
├── constants/           # Shared constants
│   └── timeouts.go      # Default timeout values
│
└── utils/               # Utilities
    └── url.go           # URL manipulation helpers
```

### Important File Relationships

- **All protocol handlers** (`internal/handler/*/handler.go`) use:
  - `internal/auth/client_auth.go` for authentication
  - `internal/proxy/client.go` for proxying to backends
  - Their own `rewriter.go` for protocol-specific response rewriting

- **Middleware chain** is defined in `cmd/artifusion/main.go` in specific order (RequestID → Security → Recovery → Logging → Timeout → Concurrency → RateLimit)

- **Configuration** flows from `config/config.yaml` → `internal/config/loader.go` → validation → distributed to all handlers/middleware

## Configuration

### Two Routing Models

Artifusion supports two deployment models controlled by configuration:

**Model 1: Host-based routing** (different domains per protocol)
```yaml
protocols:
  oci:
    host: "docker.example.com"  # Set host for OCI routing
    # Note: OCI always uses /v2 path per OCI Distribution Spec (not configurable)
  maven:
    host: "maven.example.com"
    path_prefix: ""
```

**Model 2: Path-based routing** (shared domain, different paths)
```yaml
protocols:
  oci:
    host: ""               # Leave empty for path-based
    # Note: OCI always uses /v2 path per OCI Distribution Spec (not configurable)
  maven:
    host: ""
    path_prefix: "/maven"  # REQUIRED when host is empty
```

**Important**: Unlike Maven and NPM, the OCI protocol does **not** support custom `path_prefix` configuration. The OCI Distribution Specification mandates that all API requests use the `/v2` endpoint. Only the `host` field can be configured for host-based routing.

### Critical Configuration Fields

```yaml
github:
  required_org: "myorg"        # Optional - leave empty to allow any valid GitHub user
  required_teams: []           # Optional - only checked if required_org is set
  auth_cache_ttl: 30m          # Reduces GitHub API calls by ~99%

server:
  max_concurrent_requests: 10000  # Semaphore limit

rate_limit:
  requests_per_sec: 1000.0     # Global rate limit
  per_user_requests: 100.0     # Per-user rate limit
```

### Environment Variable Overrides

Viper is configured with prefix `ARTIFUSION` and automatic environment variable binding. Dots in config keys are replaced with underscores.

**Special environment variables** (handled directly in main.go):
- `CONFIG_PATH` - Path to config.yaml (no prefix, default: looks in /etc/artifusion, ~/.artifusion, ./config, .)
- `ARTIFUSION_LOGGING_LEVEL` - Initial log level before config loads (debug, info, warn, error)
- `ARTIFUSION_LOGGING_FORMAT` - Initial log format before config loads (console, json)

**Config overrides** (via Viper's AutomaticEnv):
Any config field can be overridden with pattern: `ARTIFUSION_<SECTION>_<KEY>` where:
- All dots (`.`) in config paths are replaced with underscores (`_`)
- All keys are uppercased
- Existing underscores in config keys are preserved

Examples:
- `ARTIFUSION_GITHUB_REQUIRED_ORG` overrides `github.required_org`
- `ARTIFUSION_GITHUB_API_URL` overrides `github.api_url`
- `ARTIFUSION_GITHUB_AUTH_CACHE_TTL` overrides `github.auth_cache_ttl`
- `ARTIFUSION_SERVER_PORT` overrides `server.port`
- `ARTIFUSION_SERVER_MAX_CONCURRENT_REQUESTS` overrides `server.max_concurrent_requests`
- `ARTIFUSION_RATE_LIMIT_ENABLED` overrides `rate_limit.enabled`
- `ARTIFUSION_RATE_LIMIT_REQUESTS_PER_SEC` overrides `rate_limit.requests_per_sec`
- `ARTIFUSION_METRICS_ENABLED` overrides `metrics.enabled`
- `ARTIFUSION_PROTOCOLS_OCI_ENABLED` overrides `protocols.oci.enabled`

Note: Environment variables take precedence over config file values.

## Testing Philosophy

### Test Coverage

- **112 total tests** across 14 test files
- **Race detection enabled** for all tests (`-race` flag)
- **Table-driven tests** for comprehensive scenario coverage
- **No external test frameworks** - uses standard Go `testing` package only

### Running Tests

```bash
# All tests with race detection
make test

# Specific package tests
go test -v ./internal/auth/...
go test -v ./internal/middleware/...
go test -v ./internal/handler/oci/...

# Test coverage report
make test-coverage
open coverage.html

# CI pipeline (lint + test + build)
make ci
```

### Key Test Files

- `internal/auth/*_test.go` - Token validation, caching, GitHub API mocking
- `internal/config/validation_test.go` - All 44 config validation scenarios
- `internal/handler/oci/rewriter_test.go` - 41 URL rewriting test cases
- `internal/middleware/*_test.go` - Middleware behavior (timeout, rate limiting, etc.)
- `internal/proxy/client_test.go` - Circuit breaker and proxy client tests

## Critical Implementation Details

### Authentication Flow

1. Client provides GitHub PAT via Basic Auth or Bearer token
2. Token format validated via regex (blocks invalid tokens before GitHub API calls)
3. Token hashed with SHA256 for cache lookup
4. On cache miss: GitHub API validates token + org/team membership
5. Result cached for 5 minutes (configurable via `auth_cache_ttl`)
6. Singleflight prevents duplicate GitHub API calls during cache miss

**IMPORTANT**: Never store tokens in plaintext. Always use `internal/auth/cache.go` which stores SHA256 hashes only.

### Circuit Breaker Pattern

- **Per-backend circuit breakers** (`internal/proxy/circuit_breaker.go`)
- Uses `github.com/sony/gobreaker` library
- States: Closed (normal) → Open (failing) → Half-Open (testing recovery)
- Metrics: `artifusion_circuit_breaker_state` (0=closed, 1=open, 2=half-open)
- Auto-recovery after timeout period

**When modifying backend calls**: Always wrap in circuit breaker execution via `proxy.Client.ProxyRequestWithCircuitBreaker()`.

### OCI Cascading Backends

OCI pull requests cascade through backends in priority order:
1. Check local registry first
2. If 404, try next backend (GHCR)
3. Continue cascading until success or all backends exhausted
4. Return 404 only if all backends fail

**Code location**: `internal/handler/oci/proxy.go` - `cascadePullRequest()` function

### Response Rewriting

All protocol handlers rewrite backend responses to use public-facing URLs:

```go
// Example: Backend returns Location: http://registry:5000/v2/myimage/blobs/sha256:abc123
// Rewritten to:          Location: https://docker.example.com/v2/myimage/blobs/sha256:abc123
```

**Public URL detection order**:
1. Protocol's configured `publicURL` field (if set)
2. `X-Forwarded-Host` + `X-Forwarded-Proto` headers
3. `Forwarded` header (RFC 7239)
4. Fallback to request URL

**Code location**: Each protocol handler has `rewriter.go` with protocol-specific logic.

### Logging Standards

- Uses `github.com/rs/zerolog` for structured logging
- **Console format** (default): Human-readable with auto TTY detection
- **JSON format**: For log aggregation systems (ELK, Splunk, etc.)
- Always include `requestID` field for request correlation
- Log levels: DEBUG (internal details) → INFO (normal ops) → WARN (degraded) → ERROR (failures)

**When adding logs**:
```go
log.Info().
    Str("requestID", requestID).
    Str("protocol", "oci").
    Str("backend", backendName).
    Msg("Proxying to backend")
```

### Metrics Recording

All backend interactions must record metrics:

```go
// Required metrics for all backend calls:
metrics.RecordBackendLatency(backendName, method, duration)
metrics.RecordBackendHealth(backendName, isHealthy)
metrics.RecordBackendError(backendName, errorType, statusCode)
```

**Prometheus endpoint**: `http://localhost:8080/metrics`

**Key metrics**:
- `artifusion_requests_total` - Total requests by protocol/method/status
- `artifusion_backend_health` - Backend health (1=healthy, 0=unhealthy)
- `artifusion_backend_latency_seconds` - Latency histogram
- `artifusion_circuit_breaker_state` - Circuit breaker state
- `artifusion_rate_limit_exceeded_total` - Rate limit rejections

## Version Injection

The build system injects version information via ldflags:

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS=-ldflags "-w -s \
    -X main.version=$(VERSION) \
    -X main.gitCommit=$(GIT_COMMIT) \
    -X main.buildTime=$(BUILD_TIME)"
```

**Access in code** (`cmd/artifusion/main.go`):
```go
var (
    version   = "dev"
    gitCommit = "unknown"
    buildTime = "unknown"
)
```

## Common Development Patterns

### Adding a New Protocol Handler

1. Create package in `internal/handler/<protocol>/`
2. Implement `handler.Handler` interface
3. Create `routes.go`, `auth.go`, `proxy.go`, `rewriter.go`
4. Add protocol detector in `internal/detector/<protocol>.go`
5. Wire up in `cmd/artifusion/main.go` middleware chain
6. Add configuration struct in `internal/config/config.go`
7. Add validation in `internal/config/validation.go`
8. Write tests in `internal/handler/<protocol>/*_test.go`

### Adding a New Middleware

1. Create file in `internal/middleware/<name>.go`
2. Implement `func(http.Handler) http.Handler` signature
3. Add to middleware chain in `cmd/artifusion/main.go` (order matters!)
4. Add configuration if needed in `internal/config/config.go`
5. Write tests in `internal/middleware/<name>_test.go`

### Adding Metrics

1. Define metric in `internal/metrics/metrics.go`:
   ```go
   var myMetric = prometheus.NewCounterVec(
       prometheus.CounterOpts{
           Name: "artifusion_my_metric_total",
           Help: "Description of metric",
       },
       []string{"label1", "label2"},
   )
   ```
2. Register in `init()` function: `prometheus.MustRegister(myMetric)`
3. Record in code: `myMetric.WithLabelValues(val1, val2).Inc()`

## Deployment Considerations

### Docker Image

- **Multi-stage build** using Chainguard static base image
- **Non-root user** (UID 65532)
- **Minimal size** (~10MB final image)
- **Build**: `make docker-build`

### Health Checks

- **Liveness**: `GET /health` - Returns 200 if server is running
- **Readiness**: `GET /ready` - Returns 200 if server is ready to handle requests
- **Metrics**: `GET /metrics` - Prometheus metrics endpoint

### Reverse Proxy Setup

When deploying behind Nginx/Traefik, ensure these headers are set:
- `X-Forwarded-Host` (or `Forwarded` with host parameter)
- `X-Forwarded-Proto` (or `Forwarded` with proto parameter)
- `X-Forwarded-For` (client IP)

**Example Nginx**:
```nginx
proxy_set_header X-Forwarded-Host $host;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
```

### Performance Tuning

- **Connection pooling**: 200 idle connections per backend (configurable in config.yaml)
- **Max concurrent requests**: 10,000 default (configurable via `server.max_concurrent_requests`)
- **Rate limiting**: 1,000 req/sec global, 100 req/sec per-user (configurable)
- **Timeouts**: 60s read, 300s write (configurable via `server.*_timeout`)

### Scalability

- **Stateless architecture** - Safe to run multiple instances
- **Horizontal scaling** - Load balance across instances
- **Shared-nothing** - No inter-instance coordination needed
- **Cache is local** - Each instance has own auth cache (reduces GitHub API calls by ~99%)

## Troubleshooting

### High Latency
1. Check backend health: `curl http://localhost:8080/metrics | grep artifusion_backend_health`
2. Check circuit breaker state: `grep circuit_breaker_state`
3. Review backend latency histogram: `grep artifusion_backend_latency_seconds`

### Auth Failures
1. Verify GitHub PAT format (must match `ghp_*`, `github_pat_*`, or `ghs_*`)
2. Check org membership: `curl -H "Authorization: token $PAT" https://api.github.com/user/orgs`
3. Review auth cache metrics: `grep artifusion_auth_cache`
4. Check logs for auth errors with `requestID` for correlation

### Rate Limiting
1. Check current limits in config.yaml
2. Monitor `artifusion_rate_limit_exceeded_total` metric
3. Adjust `rate_limit.requests_per_sec` or `rate_limit.per_user_requests` if needed

### Circuit Breaker Open
1. Identify failing backend: `grep circuit_breaker_state | grep " 1"`
2. Check backend health directly
3. Review backend error metrics: `grep artifusion_backend_errors_total`
4. Circuit breaker auto-recovers after timeout - monitor transition to half-open (2) then closed (0)

## External Dependencies

### Core Libraries
- **github.com/go-chi/chi/v5** - HTTP router and middleware framework
- **github.com/rs/zerolog** - Structured logging
- **github.com/spf13/viper** - Configuration management
- **github.com/prometheus/client_golang** - Prometheus metrics
- **github.com/google/go-github/v58** - GitHub API client
- **golang.org/x/oauth2** - OAuth2 support for GitHub API
- **github.com/sony/gobreaker** - Circuit breaker implementation
- **golang.org/x/time/rate** - Rate limiting (token bucket)
- **github.com/patrickmn/go-cache** - In-memory cache with TTL
- **github.com/google/uuid** - UUID generation for request IDs
- **golang.org/x/sync/singleflight** - Duplicate request suppression

All dependencies are security-audited and actively maintained.

## Additional Documentation

- **Architecture**: `docs/architecture/FINAL_ARCHITECTURE_SUMMARY.md` - Comprehensive system design
- **Configuration**: `config/config.example.yaml` - Fully commented config reference
- **Protocol Details**: `docs/PROTOCOL_BACKEND_TYPES.md` - Backend type specifications
- **Deployment**: `deployments/docker/README.md` - Docker Compose deployment guide
- **Migration**: `docs/CONFIG_MIGRATION.md` - Configuration migration guide

## Development Workflow

1. **Make changes** to code
2. **Run tests**: `make test` (always run with race detection)
3. **Check formatting**: `make lint`
4. **Build binary**: `make build`
5. **Test locally**: `make run` (requires config.yaml)
6. **Run full CI**: `make ci` (lint + test + build)
7. **Docker testing**: `make docker-build && make docker-up`
8. **Check metrics**: `curl http://localhost:8080/metrics`
9. **View logs**: `docker-compose logs -f artifusion` or check console output
