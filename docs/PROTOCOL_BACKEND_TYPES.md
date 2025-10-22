# Protocol-Specific Backend Configuration Types

## Summary

This document describes the protocol-specific backend configuration types implemented for Artifusion. The refactoring eliminates the generic `BackendConfig` type in favor of protocol-specific types that better represent each protocol's needs.

## Architecture Decision: Option A - Separate Types

After careful consideration, we implemented **Option A: Separate Protocol-Specific Types** for the following reasons:

### Advantages

1. **Type Safety**: Each protocol has its own strongly-typed configuration
2. **Clarity**: Clear separation of concerns - OCI, Maven, and NPM each have their own types
3. **Validation**: Protocol-specific validation logic without complex conditionals
4. **Evolution**: Easy to add protocol-specific fields without affecting other protocols
5. **Go Idioms**: Favors composition over inheritance ("clear is better than clever")
6. **Mapstructure Compatibility**: Works seamlessly with Viper's configuration unmarshaling

### Trade-offs

- **Minor Duplication**: Common fields (URL, timeouts, connection pooling) are duplicated across types
- This is acceptable because:
  - Each type remains simple and self-contained
  - Future protocol-specific fields won't pollute other types
  - Validation and defaults are cleanly separated

## Type Definitions

### OCIBackendConfig

For OCI/Docker registries with namespace routing and path rewriting:

```go
type OCIBackendConfig struct {
    // Common fields
    Name string
    URL  string
    Auth *AuthConfig

    // OCI-specific fields
    UpstreamNamespace string            // e.g., "ghcr.io", "docker.io"
    PathRewrite       PathRewriteConfig // Add library/ prefix for official images
    Scope             []string          // Org-based routing for GHCR

    // HTTP client pool settings
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    IdleConnTimeout     time.Duration
    DialTimeout         time.Duration
    RequestTimeout      time.Duration

    // Circuit breaker settings
    CircuitBreaker CircuitBreakerConfig
}
```

### MavenBackendConfig

For Maven repositories (Reposilite):

```go
type MavenBackendConfig struct {
    // Common fields
    Name string
    URL  string
    Auth *AuthConfig

    // HTTP client pool settings
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    IdleConnTimeout     time.Duration
    DialTimeout         time.Duration
    RequestTimeout      time.Duration

    // Circuit breaker settings
    CircuitBreaker CircuitBreakerConfig
}
```

**Note**: No Maven-specific fields currently. Reposilite manages read/write permissions internally based on authentication credentials.

### NPMBackendConfig

For NPM registries (Verdaccio):

```go
type NPMBackendConfig struct {
    // Common fields
    Name string
    URL  string
    Auth *AuthConfig  // Supports bearer tokens (preemptive)

    // HTTP client pool settings
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    IdleConnTimeout     time.Duration
    DialTimeout         time.Duration
    RequestTimeout      time.Duration

    // Circuit breaker settings
    CircuitBreaker CircuitBreakerConfig
}
```

**Note**: No NPM-specific fields currently, but the type is ready for future enhancements.

## Configuration Changes

### OCI Configuration

**Removed**: `priority` field - array index order determines cascade priority

**Before:**
```yaml
pullBackends:
  - name: ghcr
    priority: 2
  - name: local
    priority: 1
```

**After:**
```yaml
pullBackends:
  # Array order = cascade priority (first = highest)
  - name: local
  - name: ghcr
```

### Maven Configuration

**Consolidated**: `readBackend` and `writeBackend` → single `backend`

**Before:**
```yaml
maven:
  readBackend:
    url: http://reposilite:8080
    auth:
      username: readonly
  writeBackend:
    url: http://reposilite:8080
    auth:
      username: admin
```

**After:**
```yaml
maven:
  backend:
    url: http://reposilite:8080
    auth:
      username: admin  # Reposilite manages permissions
```

### NPM Configuration (New)

```yaml
npm:
  enabled: true
  backend:
    url: http://verdaccio:4873
    auth:
      type: bearer
      token: ${VERDACCIO_TOKEN}
```

## Implementation Details

### Validation

Each type has its own `Validate()` method in `internal/config/validation.go`:

- `OCIBackendConfig.Validate()` - Validates OCI-specific fields
- `MavenBackendConfig.Validate()` - Validates Maven backend requirements
- `NPMBackendConfig.Validate()` - Validates NPM backend requirements

### Default Values

Type-specific default value setters in `internal/config/config.go`:

- `setOCIBackendDefaults(*OCIBackendConfig)`
- `setMavenBackendDefaults(*MavenBackendConfig)`
- `setNPMBackendDefaults(*NPMBackendConfig)`

### Protocol Configs

Updated in `internal/config/config.go`:

```go
type OCIConfig struct {
    PullBackends []OCIBackendConfig  // Changed from []BackendConfig
    PushBackend  OCIBackendConfig    // Changed from BackendConfig
}

type MavenConfig struct {
    Backend MavenBackendConfig  // Changed from ReadBackend/WriteBackend
}

type NPMConfig struct {
    Backend NPMBackendConfig  // New
}
```

### Conversion Helpers

For backward compatibility, each type provides a `ToBackendConfig()` method:

```go
func (o *OCIBackendConfig) ToBackendConfig() *BackendConfig
func (m *MavenBackendConfig) ToBackendConfig() *BackendConfig
func (n *NPMBackendConfig) ToBackendConfig() *BackendConfig
```

These are used internally by the proxy layer which still uses the generic type.

## Modified Files

### Primary Files

1. `internal/config/config.go`
   - Added: `OCIBackendConfig`, `MavenBackendConfig`, `NPMBackendConfig`
   - Added: `NPMConfig`
   - Updated: `OCIConfig`, `MavenConfig` to use new types
   - Added: Protocol-specific default setters
   - Added: `ToBackendConfig()` conversion methods

2. `internal/config/validation.go`
   - Added: `OCIBackendConfig.Validate()`
   - Added: `MavenBackendConfig.Validate()`
   - Added: `NPMBackendConfig.Validate()`
   - Added: `NPMConfig.Validate()`
   - Updated: Protocol validation to include NPM

3. `internal/config/loader.go`
   - Added: `expandOCIBackendAuthEnvVars(*OCIBackendConfig)`
   - Added: `expandMavenBackendAuthEnvVars(*MavenBackendConfig)`
   - Added: `expandNPMBackendAuthEnvVars(*NPMBackendConfig)`
   - Updated: `expandEnvVars()` to use protocol-specific functions

### Handler Files

4. `internal/handler/oci/`
   - Updated all functions to use `*OCIBackendConfig` instead of `*BackendConfig`
   - Files: `routes.go`, `proxy.go`, `rewriter.go`, `auth.go`, `rewriter_test.go`

5. `internal/handler/maven/`
   - Updated to use single `Backend` field
   - Updated to use `*MavenBackendConfig`
   - Files: `routes.go`, `proxy.go`, `auth.go`

6. `internal/auth/client_auth.go`
   - Added: `InjectAuthCredentials(*AuthConfig)` - protocol-agnostic auth injection

### Main Application

7. `cmd/artifusion/main.go`
   - Updated Maven logging to use single `Backend` field

### Tests

8. `internal/config/validation_test.go`
   - Updated all tests to use new types

9. `internal/handler/oci/rewriter_test.go`
   - Updated to use `OCIBackendConfig`

### Configuration

10. `config/config.example.yaml`
    - Removed `priority` fields from OCI backends
    - Consolidated Maven `readBackend`/`writeBackend` → `backend`
    - Added NPM configuration example

## Benefits Realized

### 1. Type Safety

Each protocol's backend configuration is strongly typed:
- Compile-time validation prevents configuration errors
- IDE autocomplete shows only relevant fields
- No confusion about which fields apply to which protocol

### 2. Simplicity

**OCI**: Array order instead of explicit priority numbers
```yaml
# Old (complex)
pullBackends:
  - name: ghcr
    priority: 2
  - name: local
    priority: 1

# New (simple)
pullBackends:
  - name: local  # First = highest priority
  - name: ghcr
```

**Maven**: Single backend instead of separate read/write
```yaml
# Old (duplication)
readBackend:
  url: http://reposilite:8080
  auth: {username: readonly}
writeBackend:
  url: http://reposilite:8080
  auth: {username: admin}

# New (consolidated)
backend:
  url: http://reposilite:8080
  auth: {username: admin}
```

### 3. Maintainability

- Each protocol can evolve independently
- Adding NPM-specific fields won't affect OCI or Maven
- Clear validation and default logic per protocol
- No complex conditionals based on protocol type

### 4. Evolution

Ready for future enhancements:
- OCI: Could add registry-specific features (e.g., ORAS support)
- Maven: Could add repository-specific settings
- NPM: Could add npm-specific features (e.g., scope mapping)

## Backward Compatibility

The deprecated `BackendConfig` type is kept for backward compatibility:
- Marked with deprecation comments
- Conversion methods allow smooth migration
- Will be removed in a future major version

## Testing

All tests pass:
```
ok  	internal/auth
ok  	internal/config
ok  	internal/handler/oci
ok  	internal/middleware
```

## Migration Guide

See [CONFIG_MIGRATION.md](docs/CONFIG_MIGRATION.md) for detailed migration instructions.

## Future Work

1. **Remove deprecated BackendConfig**: In next major version
2. **Add NPM-specific fields**: As requirements emerge
3. **Enhance validation**: Add cross-field validation rules
4. **Performance**: Consider caching converted BackendConfig instances

## Conclusion

The protocol-specific backend types provide:
- ✅ Better type safety
- ✅ Clearer intent
- ✅ Simpler configuration
- ✅ Easier maintenance
- ✅ Ready for evolution

This design follows Go best practices: "Clear is better than clever" and "A little copying is better than a little dependency."
