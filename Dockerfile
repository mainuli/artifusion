# ===== Build Stage =====
# Using Chainguard Go 1.25 image (distroless)
FROM cgr.dev/chainguard/go:latest AS builder

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -X main.version=1.0.0" \
    -o artifusion \
    ./cmd/artifusion

# ===== Runtime Stage =====
# Using Chainguard static image - minimal distroless base for static binaries
# Includes ca-certificates, tzdata, and nonroot user (uid:65532, gid:65532)
FROM cgr.dev/chainguard/static:latest

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
