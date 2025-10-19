# Configuration Migration Guide: Protocol-Specific Backend Types

## Overview

This guide explains the migration from generic `BackendConfig` to protocol-specific backend configuration types (`OCIBackendConfig`, `MavenBackendConfig`, `NPMBackendConfig`).

## What Changed

### 1. Protocol-Specific Backend Types

**Before (Generic):**
```go
type BackendConfig struct {
    Name              string
    URL               string
    Priority          int               // OCI-only
    UpstreamNamespace string            // OCI-only
    PathRewrite       PathRewriteConfig // OCI-only
    Scope             []string          // OCI-only
    Auth              *AuthConfig
    // HTTP client pool settings
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    IdleConnTimeout     time.Duration
    DialTimeout         time.Duration
    RequestTimeout      time.Duration
    CircuitBreaker      CircuitBreakerConfig
}
```

**After (Protocol-Specific):**
```go
// OCI/Docker registries
type OCIBackendConfig struct {
    Name              string
    URL               string
    Auth              *AuthConfig
    // OCI-specific fields
    UpstreamNamespace string
    PathRewrite       PathRewriteConfig
    Scope             []string
    // HTTP client pool settings
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    IdleConnTimeout     time.Duration
    DialTimeout         time.Duration
    RequestTimeout      time.Duration
    CircuitBreaker      CircuitBreakerConfig
}

// Maven repositories (Reposilite)
type MavenBackendConfig struct {
    Name string
    URL  string
    Auth *AuthConfig
    // HTTP client pool settings
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    IdleConnTimeout     time.Duration
    DialTimeout         time.Duration
    RequestTimeout      time.Duration
    CircuitBreaker      CircuitBreakerConfig
}

// NPM registries (Verdaccio)
type NPMBackendConfig struct {
    Name string
    URL  string
    Auth *AuthConfig  // Supports bearer tokens (preemptive)
    // HTTP client pool settings
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    IdleConnTimeout     time.Duration
    DialTimeout         time.Duration
    RequestTimeout      time.Duration
    CircuitBreaker      CircuitBreakerConfig
}
```

### 2. OCI Configuration Changes

**Removed:** `priority` field - use array index order instead

**Before:**
```yaml
protocols:
  oci:
    pullBackends:
      - name: ghcr
        url: http://oci-registry:8080
        priority: 2
      - name: local
        url: http://registry:5000
        priority: 1
      - name: dockerhub
        url: http://oci-registry:8080
        priority: 3
```

**After (simpler):**
```yaml
protocols:
  oci:
    pullBackends:
      # Array order determines cascade priority (first = highest priority)
      - name: local
        url: http://registry:5000
      - name: ghcr
        url: http://oci-registry:8080
      - name: dockerhub
        url: http://oci-registry:8080
```

### 3. Maven Configuration Changes

**Consolidated:** `readBackend` and `writeBackend` → single `backend`

**Before:**
```yaml
protocols:
  maven:
    enabled: true
    proxyPublicURL: http://localhost:8080
    readBackend:
      name: maven-read
      url: http://reposilite:8080
      auth:
        type: basic
        username: readonly
        password: ${REPOSILITE_READ_PASSWORD}
    writeBackend:
      name: maven-write
      url: http://reposilite:8080
      auth:
        type: basic
        username: admin
        password: ${REPOSILITE_WRITE_TOKEN}
```

**After (simpler):**
```yaml
protocols:
  maven:
    enabled: true
    proxyPublicURL: http://localhost:8080
    backend:
      name: maven
      url: http://reposilite:8080
      auth:
        type: basic
        username: admin
        password: ${REPOSILITE_PASSWORD}
```

**Note:** Reposilite handles read/write permissions internally based on auth credentials. A single backend configuration is sufficient.

### 4. NPM Configuration (New)

**Before:** Not supported

**After:**
```yaml
protocols:
  npm:
    enabled: true
    proxyPublicURL: https://artifusion.example.com
    clientAuth:
      supportedSchemes: [bearer]
      realm: "Artifusion NPM Registry"
    backend:
      name: verdaccio
      url: http://verdaccio:4873
      auth:
        type: bearer
        token: ${VERDACCIO_TOKEN}
```

## Migration Steps

### Step 1: Update OCI Configuration

1. Remove `priority` fields from all `pullBackends`
2. Reorder backends in desired cascade priority (first = highest priority)

```diff
protocols:
  oci:
    pullBackends:
-      - name: ghcr
-        priority: 2
-      - name: local
-        priority: 1
-      - name: dockerhub
-        priority: 3
+      - name: local
+      - name: ghcr
+      - name: dockerhub
```

### Step 2: Update Maven Configuration

1. Replace `readBackend` and `writeBackend` with single `backend`
2. Use write-capable credentials (Reposilite manages permissions internally)

```diff
protocols:
  maven:
-    readBackend:
-      name: maven-read
-      url: http://reposilite:8080
-      auth:
-        type: basic
-        username: readonly
-        password: ${REPOSILITE_READ_PASSWORD}
-    writeBackend:
-      name: maven-write
+    backend:
+      name: maven
       url: http://reposilite:8080
       auth:
         type: basic
         username: admin
-        password: ${REPOSILITE_WRITE_TOKEN}
+        password: ${REPOSILITE_PASSWORD}
```

### Step 3: (Optional) Add NPM Support

If using NPM/Verdaccio, add the new NPM configuration:

```yaml
protocols:
  npm:
    enabled: true
    proxyPublicURL: https://artifusion.example.com
    clientAuth:
      supportedSchemes: [bearer]
      realm: "Artifusion NPM Registry"
    backend:
      name: verdaccio
      url: http://verdaccio:4873
      auth:
        type: bearer
        token: ${VERDACCIO_TOKEN}
```

## Complete Example Configuration

```yaml
# ===== Protocol Handlers =====
protocols:

  # ===== OCI/Docker Registry Protocol =====
  oci:
    enabled: true
    publicURL: ""
    clientAuth:
      supportedSchemes: [bearer, basic]
      realm: ""
      service: "artifusion"

    # Pull backends (cascade by array order)
    pullBackends:
      # 1. Local hosted registry (highest priority)
      - name: local-hosted
        url: http://registry:5000
        upstreamNamespace: ""
        pathRewrite:
          addLibraryPrefix: false
        maxIdleConns: 200
        maxIdleConnsPerHost: 100
        idleConnTimeout: 90s
        dialTimeout: 10s
        requestTimeout: 300s

      # 2. GitHub Container Registry
      - name: ghcr
        url: http://oci-registry:8080
        upstreamNamespace: ghcr.io
        scope: []  # Empty: use requiredOrg | ["*"]: all orgs | [org1, org2]: specific orgs
        pathRewrite:
          addLibraryPrefix: false
        maxIdleConns: 200
        maxIdleConnsPerHost: 100
        idleConnTimeout: 90s
        dialTimeout: 10s
        requestTimeout: 300s

      # 3. Docker Hub
      - name: dockerhub
        url: http://oci-registry:8080
        upstreamNamespace: docker.io
        pathRewrite:
          addLibraryPrefix: true
        maxIdleConns: 200
        maxIdleConnsPerHost: 100
        idleConnTimeout: 90s
        dialTimeout: 10s
        requestTimeout: 300s

    # Push backend
    pushBackend:
      name: push
      url: http://registry:5000
      maxIdleConns: 200
      maxIdleConnsPerHost: 100
      idleConnTimeout: 90s
      dialTimeout: 10s
      requestTimeout: 300s
      auth:
        type: basic
        username: artifusion
        password: ${REGISTRY_PASSWORD}

  # ===== Maven Repository Protocol =====
  maven:
    enabled: true
    proxyPublicURL: http://localhost:8080
    clientAuth:
      supportedSchemes: [basic, bearer]
      realm: "Artifusion Maven Repository"

    backend:
      name: maven
      url: http://reposilite:8080
      maxIdleConns: 200
      maxIdleConnsPerHost: 100
      idleConnTimeout: 90s
      dialTimeout: 10s
      requestTimeout: 300s
      auth:
        type: basic
        username: admin
        password: ${REPOSILITE_PASSWORD}

  # ===== NPM Registry Protocol =====
  npm:
    enabled: false
    proxyPublicURL: https://artifusion.example.com
    clientAuth:
      supportedSchemes: [bearer]
      realm: "Artifusion NPM Registry"

    backend:
      name: verdaccio
      url: http://verdaccio:4873
      maxIdleConns: 200
      maxIdleConnsPerHost: 100
      idleConnTimeout: 90s
      dialTimeout: 10s
      requestTimeout: 300s
      auth:
        type: bearer
        token: ${VERDACCIO_TOKEN}
```

## Benefits

### Type Safety
- Each protocol has its own type with only relevant fields
- Compile-time validation prevents configuration errors
- Clear intent in code

### Simplicity
- **OCI:** Array order instead of explicit priority numbers
- **Maven:** Single backend instead of separate read/write (Reposilite handles this internally)
- **NPM:** Clean, purpose-built configuration

### Maintainability
- Protocol-specific validation logic
- Easy to add protocol-specific features
- No field confusion across protocols

### Evolution
- Each protocol can evolve independently
- Adding new protocols doesn't affect existing ones
- Clean separation of concerns

## Backward Compatibility

The old `BackendConfig` type is marked as **DEPRECATED** but still exists for compatibility. The codebase uses conversion methods (`ToBackendConfig()`) internally where needed.

**Timeline:**
- **Current:** Both old and new types supported
- **Future:** Old type will be removed in a future major version

## Troubleshooting

### Build Errors

If you see errors like `ReadBackend undefined` or `WriteBackend undefined`:
- Update Maven configuration to use single `backend` field
- See Step 2 above

If you see errors about `Priority`:
- Remove all `priority` fields from OCI `pullBackends`
- Reorder array elements to desired priority
- See Step 1 above

### Runtime Issues

**OCI cascade not working:**
- Verify backends are in correct array order
- First backend = highest priority
- Check logs for backend selection

**Maven authentication failing:**
- Ensure `backend.auth` has write-capable credentials
- Reposilite manages read/write permissions internally

**NPM not working:**
- Verify `npm.enabled: true`
- Check `backend.auth.token` is valid
- Ensure Verdaccio is accessible at `backend.url`

## Questions?

For issues or questions about this migration, please:
1. Check the complete example configuration above
2. Review the troubleshooting section
3. Open an issue on GitHub with your configuration (redact sensitive values)
