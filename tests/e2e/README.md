# Artifusion End-to-End Test Suite

Comprehensive end-to-end tests for all Artifusion protocols (OCI/Docker, Maven, NPM) covering pull/push operations and caching verification.

## Overview

This test suite validates:
- **OCI/Docker**: Pull-through caching, image push/pull, caching performance
- **Maven**: Dependency caching, artifact deployment, consumption
- **NPM**: Package caching, publishing, installation

## Prerequisites

### Required Tools
- **Docker**: For OCI/Docker tests
- **Maven**: For Maven repository tests
- **Node.js/npm**: For NPM registry tests
- **curl**: For API testing

Install on macOS:
```bash
brew install docker maven node
```

### Artifusion Deployment
Ensure Artifusion is running and accessible:
```bash
# Check deployment
kubectl get pods -n artifusion

# Verify health
curl http://artifacts.lvh.me/health
```

### GitHub Credentials
You need a GitHub Personal Access Token (PAT) for authentication:

1. Create PAT: https://github.com/settings/tokens
2. Required scopes: `read:packages`, `read:user`
3. Set environment variables:
   ```bash
   export GITHUB_USERNAME=your-github-username
   export GITHUB_TOKEN=ghp_your_token_here
   export GITHUB_EMAIL=your-email@example.com  # Optional for NPM
   ```

## Running Tests

### Run All Tests
```bash
# Run complete test suite
./run-all-tests.sh
```

This will:
1. Check Artifusion health
2. Verify prerequisites
3. Run all protocol tests sequentially
4. Display comprehensive summary

### Run Individual Protocol Tests

**OCI/Docker Tests**:
```bash
cd docker
./test-oci.sh
```

**Maven Tests**:
```bash
cd maven
./test-maven.sh
```

**NPM Tests**:
```bash
cd npm
./test-npm.sh
```

## Test Coverage

### OCI/Docker Tests (`docker/test-oci.sh`)
1. **Login**: Authenticate with Artifusion using GitHub PAT
2. **Cache Miss**: Pull alpine:latest from Docker Hub (first time)
3. **Cache Hit**: Pull same image again (should be faster)
4. **Custom Image**: Build, tag, and push custom image
5. **Pull Custom**: Pull the custom image back
6. **Run Image**: Verify image works correctly
7. **Metrics**: Check Prometheus metrics

**Expected Results**:
- Cache hit should be faster than cache miss
- Custom image push/pull should succeed
- Image should run with correct output

### Maven Tests (`maven/test-maven.sh`)
1. **Configuration**: Create Maven settings.xml with Artifusion
2. **Project Setup**: Create test Maven project
3. **Cache Miss**: Download commons-io dependency (first time)
4. **Cache Hit**: Download same dependency again (should be faster)
5. **Deploy**: Create and deploy custom artifact
6. **Consume**: Use deployed artifact in another project
7. **Metrics**: Check Prometheus metrics

**Expected Results**:
- Dependencies download through Artifusion
- Cache hit should be faster
- Artifact deployment and consumption should succeed

### NPM Tests (`npm/test-npm.sh`)
1. **Configuration**: Configure npm with Artifusion registry
2. **Connection**: Test registry connection
3. **Cache Miss**: Install lodash from npmjs.org (first time)
4. **Cache Hit**: Install lodash again (should be faster)
5. **Publish**: Create and publish custom package
6. **Install**: Install the published package
7. **Usage**: Verify package works correctly
8. **Metrics**: Check Prometheus metrics

**Expected Results**:
- Packages install through Artifusion
- Cache hit should be faster
- Package publish/install should succeed (if backend supports it)

## Environment Variables

### Required
- `GITHUB_USERNAME`: Your GitHub username
- `GITHUB_TOKEN`: GitHub Personal Access Token (ghp_xxx)

### Optional
- `ARTIFUSION_HOST`: Artifusion hostname (default: artifacts.lvh.me)
- `GITHUB_EMAIL`: Email for npm authentication (default: test@example.com)

Example:
```bash
export GITHUB_USERNAME=myusername
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
export GITHUB_EMAIL=me@example.com
export ARTIFUSION_HOST=artifacts.lvh.me

./run-all-tests.sh
```

## Test Output

### Success Example
```
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║              ALL TESTS PASSED! ✓                         ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝

Protocol             Status     Time
--------             ------     ----
oci                  ✓ PASS     45s
maven                ✓ PASS     120s
npm                  ✓ PASS     35s

Total tests: 3
Passed: 3
Failed: 0
Total time: 200s
```

### Test Results
Each test will output:
- ✓ **Green**: Test passed
- ✗ **Red**: Test failed
- ⚠️ **Yellow**: Warning or skipped

## Troubleshooting

### Authentication Failures
```
Error: 401 Unauthorized
```
**Solution**: Check your GitHub token:
```bash
# Verify token is set
echo $GITHUB_TOKEN

# Test GitHub API access
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
```

### Connection Refused
```
Error: Connection refused
```
**Solution**: Check Artifusion is running:
```bash
kubectl get pods -n artifusion
kubectl get ingress -n artifusion
curl http://artifacts.lvh.me/health
```

### DNS Resolution Issues
```
Error: Could not resolve host
```
**Solution**: Check DNS for lvh.me:
```bash
ping artifacts.lvh.me

# If it fails, add to /etc/hosts
echo "127.0.0.1 artifacts.lvh.me" | sudo tee -a /etc/hosts
```

### Maven Build Failures
```
Error: Could not resolve dependencies
```
**Solution**: Check Maven settings and credentials:
```bash
# Verify settings.xml is correct
cat ~/.m2/settings.xml

# Test Artifusion Maven endpoint
curl -u "$GITHUB_USERNAME:$GITHUB_TOKEN" \
  http://artifacts.lvh.me/maven/releases/maven-metadata.xml
```

### NPM Publish Failures
```
Error: 404 Not Found (publishing)
```
**Solution**: This may be expected - Verdaccio backend may not have write access configured. The test will skip publish tests and continue with cache verification.

### Cache Performance
If cache hits are not faster:
- First run may not show improvement (caches are warming up)
- Small packages may not show significant difference
- Network variance can affect timing
- The important part is that artifacts ARE cached (check metrics)

## Viewing Metrics

After running tests, check Prometheus metrics:
```bash
curl http://artifacts.lvh.me/metrics | grep artifusion
```

**Key Metrics**:
- `artifusion_requests_total`: Total requests by protocol
- `artifusion_backend_requests_total`: Backend requests
- `artifusion_backend_latency_seconds`: Response times
- `artifusion_auth_cache_hits_total`: Auth cache efficiency

## Cleanup

Tests automatically clean up temporary files. To manually clean:
```bash
# Remove temporary test directories
rm -rf /tmp/artifusion-*-test

# Remove Docker test images
docker rmi artifacts.lvh.me/library/alpine:latest
docker rmi artifacts.lvh.me/testorg/hello-artifusion:latest

# Reset npm config
npm config delete registry
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup tools
        run: |
          # Docker, Maven, Node.js are pre-installed in ubuntu-latest

      - name: Run E2E Tests
        env:
          GITHUB_USERNAME: ${{ secrets.E2E_GITHUB_USERNAME }}
          GITHUB_TOKEN: ${{ secrets.E2E_GITHUB_TOKEN }}
          ARTIFUSION_HOST: artifacts.example.com
        run: |
          cd tests/e2e
          chmod +x run-all-tests.sh
          ./run-all-tests.sh
```

## Test Development

### Adding New Tests

1. Create test script in appropriate directory
2. Follow the existing pattern:
   - Use colors for output
   - Include cleanup trap
   - Test cache performance
   - Check metrics
3. Update this README
4. Add to `run-all-tests.sh` if needed

### Test Script Template
```bash
#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
ARTIFUSION_HOST="${ARTIFUSION_HOST:-artifacts.lvh.me}"

# Cleanup
cleanup() {
    echo "Cleaning up..."
    # Your cleanup code
}
trap cleanup EXIT

# Tests
echo -e "${BLUE}Test 1: Description${NC}"
# Test code
echo -e "${GREEN}✓ Test passed${NC}"
```

## Contributing

When adding new test coverage:
1. Ensure tests are idempotent (can run multiple times)
2. Include both success and failure scenarios
3. Add timing measurements for cache tests
4. Document expected behavior
5. Include troubleshooting tips

## License

Same as Artifusion project.
