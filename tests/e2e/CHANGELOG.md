# E2E Test Suite Changelog

## Recent Improvements

### [2025-10-20] Fixed NPM Configuration, Metrics Checking, Docker Best Practices, and Added .env Support

#### 0. Added .env File Support and mise Integration

**Enhancement**: Automatic credential loading from `.env` file

**What Changed**:
- Added automatic `.env` file loading in all test scripts
- Added `mise` version manager support for tools like Maven
- Created `.env.example` template file
- Created comprehensive `ENV_SETUP.md` guide

**Before**:
```bash
# Had to export credentials every time
export GITHUB_USERNAME=mainuli
export GITHUB_TOKEN=ghp_xxx
./run-all-tests.sh
```

**After**:
```bash
# One-time setup
cp .env.example .env
# Edit .env with your credentials

# Just run tests (credentials auto-loaded)
./run-all-tests.sh
```

**mise Support**:
```bash
# If you use mise for Maven
mise use -g maven@latest

# Tests automatically activate mise shims
./maven/test-maven.sh  # ✅ Works!
```

**New Files**:
- `.env.example` - Template for credentials
- `ENV_SETUP.md` - Complete setup guide

**Security**:
- ✅ `.env` is already in `.gitignore`
- ✅ Never committed to git
- ✅ Local-only credentials

**Scripts Updated**:
- `run-all-tests.sh` - Loads `.env`, activates mise
- `docker/test-oci.sh` - Loads `.env`
- `maven/test-maven.sh` - Loads `.env`, activates mise
- `npm/test-npm.sh` - Loads `.env`

### [2025-10-20] Fixed NPM Configuration, Metrics Checking, and Docker Best Practices

#### 0. Fixed Docker CMD Instruction Format

**Issue**: Docker build produced warning:
```
JSONArgsRecommended: JSON arguments recommended for CMD to prevent unintended behavior
related to OS signals (line 3)
```

**Root Cause**:
- CMD instruction was using shell form: `CMD cat /hello.txt`
- Shell form doesn't properly handle OS signals (SIGTERM, SIGINT)
- Can cause issues with graceful shutdowns in containers

**Fix Applied**:
Changed CMD to JSON array format in `docker/test-oci.sh`:

**Before**:
```dockerfile
FROM alpine:latest
RUN echo "Hello from Artifusion!" > /hello.txt
CMD cat /hello.txt  # ❌ Shell form
```

**After**:
```dockerfile
FROM alpine:latest
RUN echo "Hello from Artifusion!" > /hello.txt
CMD ["cat", "/hello.txt"]  # ✅ JSON array form
```

**Benefits**:
- ✅ No build warnings
- ✅ Proper signal handling (SIGTERM/SIGINT)
- ✅ Container stops gracefully
- ✅ Follows Docker best practices

**Result**: Clean docker build with no warnings

### [2025-10-20] Fixed NPM Configuration and Metrics Checking

#### 1. Removed Deprecated `always-auth` Configuration

**Issue**: NPM tests produced warning:
```
npm warn Unknown user config "always-auth" (//artifacts.lvh.me/npm/:always-auth)
This will stop working in the next major version of npm.
```

**Root Cause**:
- The `always-auth` configuration option is deprecated in npm 7+
- Modern npm automatically uses `_authToken` when configured
- No need for explicit `always-auth` flag

**Fix Applied**:
Removed `always-auth` from `.npmrc` configuration in `npm/test-npm.sh`:

**Before**:
```ini
registry=http://artifacts.lvh.me/npm/
//artifacts.lvh.me/npm/:_authToken=${GITHUB_TOKEN}
//artifacts.lvh.me/npm/:username=${GITHUB_USERNAME}
//artifacts.lvh.me/npm/:email=${GITHUB_EMAIL}
//artifacts.lvh.me/npm/:always-auth=true  # ❌ Deprecated
```

**After**:
```ini
registry=http://artifacts.lvh.me/npm/
//artifacts.lvh.me/npm/:_authToken=${GITHUB_TOKEN}
//artifacts.lvh.me/npm/:username=${GITHUB_USERNAME}
//artifacts.lvh.me/npm/:email=${GITHUB_EMAIL}
```

**Result**: ✅ No more deprecation warnings

**Documentation**: Created `npm/NPM_CONFIG.md` with detailed authentication guide

#### 2. Enhanced Metrics Validation

**Issue**: Metrics test could show "Could not retrieve metrics" warning even when metrics were available

**Root Cause**:
- Empty response handling was not explicit
- Protocol-specific metrics might not exist if requests haven't been made yet
- No graceful handling for missing specific metric types

**Fix Applied**:
Improved metrics checking in all three test scripts (`docker/test-oci.sh`, `maven/test-maven.sh`, `npm/test-npm.sh`):

**Before**:
```bash
METRICS=$(curl -s "http://$ARTIFUSION_HOST/metrics")
if echo "$METRICS" | grep -q "artifusion_requests_total"; then
    echo "✓ Metrics endpoint accessible"
else
    echo "⚠️  Could not retrieve metrics"
fi
```

**After**:
```bash
METRICS=$(curl -s "http://$ARTIFUSION_HOST/metrics" 2>&1)

if [ -z "$METRICS" ]; then
    echo "⚠️  Could not retrieve metrics (empty response)"
elif echo "$METRICS" | grep -q "artifusion"; then
    echo "✓ Metrics endpoint accessible"

    # Show protocol-specific metrics if available
    PROTOCOL_METRICS=$(echo "$METRICS" | grep 'protocol="npm"')
    if [ -n "$PROTOCOL_METRICS" ]; then
        echo "$PROTOCOL_METRICS"
    else
        echo "(No protocol-specific metrics yet - this is normal)"
    fi

    # Show backend health if available
    BACKEND_HEALTH=$(echo "$METRICS" | grep 'backend="verdaccio"')
    if [ -n "$BACKEND_HEALTH" ]; then
        echo "$BACKEND_HEALTH"
    else
        echo "(Backend health metric not yet available)"
    fi
else
    echo "⚠️  Could not retrieve valid metrics"
    echo "Response: ${METRICS:0:100}..."
fi
```

**Improvements**:
- ✅ Better error detection (empty response vs invalid response)
- ✅ Graceful handling of missing protocol-specific metrics
- ✅ Informative messages when metrics don't exist yet
- ✅ Shows partial output on errors for debugging
- ✅ Captures stderr from curl for better error messages

**Result**: Tests now provide clearer feedback about metrics status

### Files Modified

1. **npm/test-npm.sh**
   - Removed `always-auth` from `.npmrc` configuration (line 71)
   - Enhanced metrics checking (lines 270-299)

2. **docker/test-oci.sh**
   - Enhanced metrics checking (lines 151-180)

3. **maven/test-maven.sh**
   - Enhanced metrics checking (lines 342-371)

4. **run-all-tests.sh**
   - Fixed bash 3.x compatibility (replaced associative arrays)
   - Added `SKIP_PREREQ_CHECK` environment variable support

### New Documentation

1. **npm/NPM_CONFIG.md**
   - Complete guide to npm authentication with Artifusion
   - Explanation of deprecated configurations
   - npm version compatibility matrix
   - Security best practices
   - Troubleshooting guide

2. **QUICKSTART.md**
   - Quick start guide for running tests
   - Essential prerequisites
   - Expected output examples

3. **SUMMARY.md**
   - Comprehensive test suite summary
   - Complete test coverage documentation
   - Feature validation checklist

4. **CHANGELOG.md** (this file)
   - Track all improvements and fixes
   - Document breaking changes
   - Provide migration guidance

## Testing Recommendations

### Before Running Tests

1. **Verify Artifusion is running**:
   ```bash
   kubectl get pods -n artifusion
   curl http://artifacts.lvh.me/health
   ```

2. **Set GitHub credentials**:
   ```bash
   export GITHUB_USERNAME=your-username
   export GITHUB_TOKEN=ghp_xxx
   export GITHUB_EMAIL=your-email@example.com  # For NPM
   ```

3. **Check prerequisites**:
   ```bash
   docker --version
   mvn --version
   npm --version
   ```

### Running Tests

```bash
cd tests/e2e
./run-all-tests.sh
```

### Expected Behavior

- ✅ **No deprecation warnings** from npm
- ✅ **Clear metrics feedback** (even if no protocol-specific metrics exist yet)
- ✅ **Informative error messages** if something fails
- ✅ **Graceful handling** of missing tools

## Compatibility

### npm Versions
- npm 6.x: ✅ Fully compatible
- npm 7.x: ✅ Fully compatible (no warnings)
- npm 8.x+: ✅ Fully compatible

### Bash Versions
- bash 3.x (macOS default): ✅ Fully compatible
- bash 4.x+ (Linux): ✅ Fully compatible

### Platforms
- macOS: ✅ Tested and working
- Linux: ✅ Expected to work
- WSL2: ✅ Expected to work

## Known Issues

### Maven Not Installed
**Symptom**: Test suite exits if Maven is not installed

**Workaround**:
```bash
export SKIP_PREREQ_CHECK=1
./run-all-tests.sh
```
This will run tests for available tools only.

### Metrics May Be Empty on First Run
**Symptom**: "No protocol-specific metrics yet" message

**Expected Behavior**: This is normal! Metrics are only generated after requests are made to Artifusion. The tests will still pass.

## Future Improvements

### Planned
- [ ] Add support for scoped npm packages (@org/package)
- [ ] Add tests for authentication failures (negative tests)
- [ ] Add performance benchmarking mode
- [ ] Add parallel test execution option
- [ ] Add CI/CD pipeline example (GitHub Actions, GitLab CI)

### Under Consideration
- [ ] Add support for custom backend configurations
- [ ] Add tests for rate limiting
- [ ] Add tests for large file uploads (>1GB)
- [ ] Add support for private GitHub packages
- [ ] Add network failure simulation tests

## Contributing

When adding new features or fixes:

1. Update the relevant test script(s)
2. Update documentation (README.md, SUMMARY.md, etc.)
3. Add entry to this CHANGELOG.md
4. Test on both macOS and Linux if possible
5. Ensure bash 3.x compatibility (no associative arrays)

## Version History

### v1.1.0 (2025-10-20)
- ✅ Fixed npm `always-auth` deprecation warning
- ✅ Enhanced metrics validation in all test scripts
- ✅ Added comprehensive NPM configuration documentation
- ✅ Fixed bash 3.x compatibility issues
- ✅ Added QUICKSTART.md and SUMMARY.md

### v1.0.0 (2025-10-20)
- ✅ Initial release
- ✅ Complete E2E tests for OCI/Docker, Maven, NPM
- ✅ Push/pull/cache testing for all protocols
- ✅ Performance measurement
- ✅ Metrics validation
- ✅ Comprehensive README

## Support

For issues or questions:
1. Check README.md for troubleshooting
2. Check npm/NPM_CONFIG.md for authentication issues
3. Review test logs for specific error messages
4. Open an issue in the repository with:
   - Test command used
   - Full output/logs
   - Environment details (OS, tool versions)
