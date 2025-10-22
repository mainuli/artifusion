# E2E Tests Quick Start

## Prerequisites

1. **Install tools**:
   ```bash
   # Via Homebrew
   brew install docker maven node

   # Or via mise (version manager)
   mise use -g maven@latest node@latest
   ```

2. **Setup credentials** (one-time):
   ```bash
   cd tests/e2e

   # Copy template
   cp .env.example .env

   # Edit with your credentials
   nano .env
   ```

   Get GitHub token from: https://github.com/settings/tokens
   - Required scopes: `read:packages`, `read:user`

   Your `.env` should look like:
   ```bash
   export GITHUB_USERNAME="your-username"
   export GITHUB_TOKEN="ghp_your_token_here"
   ```

3. **Ensure Artifusion is running**:
   ```bash
   kubectl get pods -n artifusion
   curl http://artifacts.lvh.me/health
   ```

## Run Tests

```bash
cd tests/e2e

# Credentials are automatically loaded from .env
./run-all-tests.sh
```

## What Gets Tested

### Docker (OCI Registry)
- ✅ Pull alpine from Docker Hub → **Cache it**
- ✅ Pull alpine again → **From cache (faster!)**
- ✅ Build custom image → **Push to Artifusion**
- ✅ Pull custom image → **Verify it works**

### Maven
- ✅ Download commons-io from Maven Central → **Cache it**
- ✅ Download again → **From cache (faster!)**
- ✅ Build custom JAR → **Deploy to Artifusion**
- ✅ Use custom JAR → **Verify compilation**

### NPM
- ✅ Install lodash from npmjs.org → **Cache it**
- ✅ Install again → **From cache (faster!)**
- ✅ Create custom package → **Publish to Artifusion**
- ✅ Install custom package → **Verify it works**

## Expected Result

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

## Troubleshooting

**Authentication failed?**
```bash
# Test your token
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
```

**Artifusion not responding?**
```bash
kubectl get pods -n artifusion
kubectl logs -n artifusion deployment/artifusion-artifusion
```

**Missing tools?**
```bash
# Skip prerequisite check
export SKIP_PREREQ_CHECK=1
./run-all-tests.sh
```

## Run Individual Tests

**Docker only**:
```bash
cd docker && ./test-oci.sh
```

**Maven only**:
```bash
cd maven && ./test-maven.sh
```

**NPM only**:
```bash
cd npm && ./test-npm.sh
```

## More Information

- Complete docs: `README.md`
- Test summary: `SUMMARY.md`
- Deployment: `../../deployments/helm/LOCAL_DEPLOYMENT.md`
