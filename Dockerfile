# ===== Build Stage =====
# Using Chainguard Go 1.25 image (distroless)
FROM cgr.dev/chainguard/go:latest AS builder

# Build arguments for version information
# These are injected into the binary at build time via ldflags
# Usage: docker build --build-arg VERSION=1.2.3 --build-arg GIT_COMMIT=$(git rev-parse HEAD) .
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Chainguard images include ca-certificates and tzdata by default
# Set Go toolchain to auto (allows using newer Go versions)
ENV GOTOOLCHAIN=auto

WORKDIR /build

# Copy go mod files for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with optimizations for Go 1.25
# Using CGO_ENABLED=0 for fully static binary
# Version information is injected via ldflags from build args
# TARGETARCH is automatically set by Docker buildx based on the platform being built
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s \
      -X main.version=${VERSION} \
      -X main.gitCommit=${GIT_COMMIT} \
      -X main.buildTime=${BUILD_TIME}" \
    -o artifusion \
    ./cmd/artifusion

# ===== Runtime Stage =====
# Using Chainguard static image - minimal distroless base for static binaries
# Includes ca-certificates, tzdata, and nonroot user (uid:65532, gid:65532)
FROM cgr.dev/chainguard/static:latest

# Pass build args to runtime stage for labels
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# OCI Image Specification labels
# These can be inspected with: docker inspect <image>
LABEL org.opencontainers.image.title="Artifusion"
LABEL org.opencontainers.image.description="Multi-protocol reverse proxy with GitHub authentication for artifact registries (OCI, Maven, NPM)"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"
LABEL org.opencontainers.image.source="https://github.com/mainuli/artifusion"
LABEL org.opencontainers.image.licenses="MIT"

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/artifusion .

# Copy example config (can be overridden with volume mount)
COPY --from=builder --chown=nonroot:nonroot /build/config/config.example.yaml /etc/artifusion/config.yaml

# Chainguard static image includes 'nonroot' user (uid:65532)
# No need to create users or install packages - distroless image
USER nonroot

# Health check note: Chainguard static image is minimal (no wget/curl)
# Health checks should be performed by orchestrator (Docker/K8s) via HTTP probe
# Example K8s liveness probe: httpGet path:/health port:8080

# Expose port
EXPOSE 8080

# Run binary
ENTRYPOINT ["./artifusion"]
CMD ["--config", "/etc/artifusion/config.yaml"]
