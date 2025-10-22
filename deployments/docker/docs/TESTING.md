# Artifusion Docker Compose Testing Guide

This guide will help you test Artifusion with Docker Compose using real backend services.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- A GitHub Personal Access Token (PAT) for testing
- GitHub organization membership (optional - only if using org-based restriction)

## Quick Start

### 1. Prepare Environment

```bash
# Navigate to docker directory
cd deployments/docker

# Copy environment template
cp .env.example .env

# Edit .env and set your GitHub organization (optional)
nano .env
```

**Optional .env settings:**
```bash
# If empty, any valid GitHub PAT is allowed
GITHUB_ORG=your-organization-name

# Only checked if GITHUB_ORG is set
# GITHUB_TEAMS=team1,team2
```

### 2. Start the Stack

```bash
# Build and start all services
docker-compose up --build

# Or run in background
docker-compose up --build -d

# View logs
docker-compose logs -f artifusion
```

### 3. Verify Services

```bash
# Check all services are healthy
docker-compose ps

# Expected output:
# NAME              STATUS
# artifusion        Up (healthy)
# oci-registry      Up (healthy)
# docker-registry   Up (healthy)
# reposilite         Up (healthy)
```

### 4. Test Health Endpoints

```bash
# Liveness check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Prometheus metrics
curl http://localhost:8080/metrics
```

## Testing OCI/Docker Protocol

### Test 1: Docker Version Check

```bash
# This should work without authentication
curl -i http://localhost:8080/v2/
```

**Expected response:**
```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="http://localhost:8080/v2/token",service="artifusion"
Docker-Distribution-Api-Version: registry/2.0
```

### Test 2: Authenticate with GitHub PAT

```bash
# Set your GitHub PAT
export GITHUB_PAT="ghp_your_github_token_here"

# Test authentication with Bearer token
curl -H "Authorization: Bearer $GITHUB_PAT" http://localhost:8080/v2/
```

**Expected response (200 OK):**
```json
{}
```

### Test 3: Pull a Public Docker Image

```bash
# Pull nginx from Docker Hub through Artifusion
# This tests: auth → oci-registry → docker.io cascade
docker login localhost:8080 --username anything --password $GITHUB_PAT

# Pull nginx (tests docker.io backend with library/ prefix)
docker pull localhost:8080/nginx:latest

# Tag and push to local registry
docker tag nginx:latest localhost:8080/myorg/nginx:test
docker push localhost:8080/myorg/nginx:test

# Pull back from local registry (tests local.registry backend)
docker rmi localhost:8080/myorg/nginx:test
docker pull localhost:8080/myorg/nginx:test
```

### Test 4: Test Cascading Backends

```bash
# Pull from Docker Hub (priority 2)
docker pull localhost:8080/alpine:latest

# Pull from local registry after push (priority 1)
docker tag alpine:latest localhost:8080/mytest/alpine:v1
docker push localhost:8080/mytest/alpine:v1
docker rmi localhost:8080/mytest/alpine:v1
docker pull localhost:8080/mytest/alpine:v1  # Should come from local
```

## Testing Maven Protocol

### Test 1: Basic Maven Authentication

```bash
# Test Maven authentication
curl -u "anything:$GITHUB_PAT" http://localhost:8080/maven-metadata.xml
```

### Test 2: Maven Download with Maven CLI

Create `~/.m2/settings.xml`:

```xml
<settings>
  <servers>
    <server>
      <id>artifusion</id>
      <username>anything</username>
      <password>YOUR_GITHUB_PAT_HERE</password>
    </server>
  </servers>

  <mirrors>
    <mirror>
      <id>artifusion</id>
      <url>http://localhost:8080</url>
      <mirrorOf>*</mirrorOf>
    </mirror>
  </mirrors>
</settings>
```

Test with Maven:

```bash
# Create test project
mkdir maven-test && cd maven-test

# Create pom.xml
cat > pom.xml << 'EOF'
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>test</artifactId>
  <version>1.0.0</version>

  <dependencies>
    <dependency>
      <groupId>junit</groupId>
      <artifactId>junit</artifactId>
      <version>4.13.2</version>
      <scope>test</scope>
    </dependency>
  </dependencies>
</project>
EOF

# Test download through Artifusion
mvn clean compile
```

### Test 3: Maven Deploy

```bash
# Add distributionManagement to pom.xml
cat >> pom.xml << 'EOF'
  <distributionManagement>
    <repository>
      <id>artifusion</id>
      <url>http://localhost:8080</url>
    </repository>
  </distributionManagement>
</project>
EOF

# Deploy artifact
mvn deploy
```

## Testing with GitHub Actions Tokens

### Test 1: Understanding GitHub Actions Tokens

GitHub Actions tokens (`ghs_` prefix) are installation tokens that:
- Expire after 1 hour
- Are scoped to the repository running the workflow
- Use the `/installation/repositories` API endpoint for validation
- Provide `github-actions[bot]` as the username

### Test 2: Simulate GitHub Actions Authentication

```bash
# Note: You would typically get this from a real GitHub Actions workflow
# For testing purposes, you can simulate with a GitHub App installation token

# Test with curl (using a real ghs_ token from a workflow)
export GHS_TOKEN="ghs_your_github_actions_token_here"

# Test authentication
curl -H "Authorization: Bearer $GHS_TOKEN" http://localhost:8080/v2/
```

### Test 3: Organization Validation

If `GITHUB_ORG` is configured in `.env`:
- Repository owner must match the configured organization
- Example: If `GITHUB_ORG=myorg`, token from `myorg/repo` succeeds, `other/repo` fails

If `GITHUB_ORG` is empty:
- Any valid GitHub Actions token is accepted

### Test 4: Docker Operations with GitHub Actions Token

```bash
# Login with GitHub Actions token
docker login localhost:8080 --username github-actions --password $GHS_TOKEN

# Pull image
docker pull localhost:8080/nginx:latest

# Push image
docker tag nginx:latest localhost:8080/myrepo/nginx:gha-test
docker push localhost:8080/myrepo/nginx:gha-test
```

### Test 5: Token Format Validation

Test that invalid tokens are rejected immediately:

```bash
# Invalid token (wrong format) - should fail instantly
curl -H "Authorization: Bearer invalid_token" http://localhost:8080/v2/

# Invalid ghs_ token (wrong length) - should fail instantly
curl -H "Authorization: Bearer ghs_tooshort" http://localhost:8080/v2/

# Check logs for preemptive rejection
docker-compose logs artifusion | grep "Invalid token format"
```

### Performance Notes

- GitHub Actions token validation uses `PerPage=1` for optimal performance
- Only fetches one repository to extract owner information
- Typical response time: <200ms (compared to ~500ms for PAT validation)

## Monitoring and Debugging

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f artifusion
docker-compose logs -f oci-registry
docker-compose logs -f registry
docker-compose logs -f reposilite
```

### Check Metrics

```bash
# Get all metrics
curl http://localhost:8080/metrics

# Filter specific metrics
curl http://localhost:8080/metrics | grep artifusion_requests_total
curl http://localhost:8080/metrics | grep artifusion_auth_cache
curl http://localhost:8080/metrics | grep artifusion_backend
```

### Inspect Containers

```bash
# Get container stats
docker stats

# Execute commands in container
docker-compose exec artifusion wget -qO- http://localhost:8080/health
docker-compose exec oci-registry wget -qO- http://localhost:8080/v2/
docker-compose exec registry wget -qO- http://localhost:5000/v2/
docker-compose exec reposilite wget -qO- http://localhost:8080/health
```

### Check Storage

```bash
# List volumes
docker volume ls | grep artifusion

# Inspect registry storage
docker volume inspect artifusion_registry-data
docker volume inspect artifusion_oci-cache
docker volume inspect artifusion_reposilite-data

# View registry contents
docker-compose exec registry ls -R /var/lib/registry/docker/registry/v2/repositories/
```

## Testing Scenarios

### Scenario 1: High Concurrency

```bash
# Install Apache Bench
# macOS: brew install ab
# Ubuntu: apt-get install apache2-utils

# Test 1000 concurrent requests
ab -n 10000 -c 1000 \
   -H "Authorization: Bearer $GITHUB_PAT" \
   http://localhost:8080/v2/
```

### Scenario 2: Rate Limiting

```bash
# Test global rate limit (1000 req/sec)
# This should trigger rate limiting
ab -n 50000 -c 2000 \
   -H "Authorization: Bearer $GITHUB_PAT" \
   http://localhost:8080/v2/

# Check metrics for rate limit rejections
curl http://localhost:8080/metrics | grep rate_limit_exceeded
```

### Scenario 3: Auth Cache Performance

```bash
# Warm up cache
for i in {1..100}; do
  curl -s -H "Authorization: Bearer $GITHUB_PAT" http://localhost:8080/v2/ > /dev/null
done

# Check cache hit rate (should be >95%)
curl http://localhost:8080/metrics | grep auth_cache
```

### Scenario 4: Backend Failover

```bash
# Stop oci-registry to test error handling
docker-compose stop oci-registry

# Try pulling - should fail gracefully
docker pull localhost:8080/nginx:latest

# Restart oci-registry
docker-compose start oci-registry

# Should work again
docker pull localhost:8080/nginx:latest
```

## Cleanup

```bash
# Stop all services
docker-compose down

# Remove volumes (clean slate)
docker-compose down -v

# Remove images
docker-compose down -v --rmi all
```

## Troubleshooting

### Issue: Authentication Fails

**Symptoms:**
```
HTTP/1.1 401 Unauthorized
```

**Solutions:**
1. Verify GitHub PAT is valid:
   ```bash
   curl -H "Authorization: Bearer $GITHUB_PAT" https://api.github.com/user
   ```

2. If using org-based restriction, check organization membership:
   ```bash
   curl -H "Authorization: Bearer $GITHUB_PAT" \
     https://api.github.com/user/memberships/orgs
   ```

3. Check Artifusion logs:
   ```bash
   docker-compose logs artifusion | grep "authentication failed"
   ```

4. Verify GITHUB_ORG setting in `.env` (leave empty to allow any valid GitHub user)

### Issue: Cannot Pull Images

**Symptoms:**
```
Error response from daemon: pull access denied
```

**Solutions:**
1. Check docker login:
   ```bash
   docker logout localhost:8080
   docker login localhost:8080 --username anything --password $GITHUB_PAT
   ```

2. Verify oci-registry is healthy:
   ```bash
   docker-compose ps oci-registry
   curl http://localhost:8080/v2/ -H "Authorization: Bearer $GITHUB_PAT"
   ```

3. Check backend connectivity:
   ```bash
   docker-compose exec artifusion wget -qO- http://oci-registry:8080/v2/
   docker-compose exec artifusion wget -qO- http://registry:5000/v2/
   ```

### Issue: Slow Performance

**Symptoms:**
- Slow pull/push operations
- Timeouts

**Solutions:**
1. Check container resources:
   ```bash
   docker stats
   ```

2. Increase Docker resources (Docker Desktop → Settings → Resources)

3. Check network latency:
   ```bash
   docker-compose exec artifusion ping oci-registry
   docker-compose exec artifusion ping registry
   ```

4. Review connection pool settings in `config/artifusion.yaml`

### Issue: Reposilite Maven Errors

**Symptoms:**
- Maven deploy fails
- 401/403 errors

**Solutions:**
1. Check Reposilite is running:
   ```bash
   docker-compose logs reposilite
   curl http://localhost:8080/ (direct access, no auth)
   ```

2. Verify credentials in `.env`:
   ```bash
   cat .env | grep REPOSILITE
   ```

3. Check Maven settings.xml has correct credentials

## Advanced Testing

### Load Testing with k6

```javascript
// load-test.js
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  vus: 100,
  duration: '30s',
};

const GITHUB_PAT = __ENV.GITHUB_PAT;

export default function() {
  let res = http.get('http://localhost:8080/v2/', {
    headers: { 'Authorization': `Bearer ${GITHUB_PAT}` },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 100ms': (r) => r.timings.duration < 100,
  });
}
```

Run:
```bash
k6 run load-test.js
```

### Network Traffic Analysis

```bash
# Capture traffic
docker-compose exec artifusion tcpdump -i any -w /tmp/capture.pcap

# Analyze with wireshark
docker cp artifusion:/tmp/capture.pcap .
wireshark capture.pcap
```

## Success Criteria

✅ All health checks pass
✅ Authentication works with GitHub PAT (`ghp_`, `github_pat_`)
✅ Authentication works with GitHub Actions tokens (`ghs_`)
✅ Invalid token formats rejected immediately (<1ms)
✅ OCI/Docker pull works from Docker Hub
✅ OCI/Docker push works to local registry
✅ OCI/Docker pull from local registry works
✅ Maven dependency download works
✅ Maven artifact deploy works
✅ Metrics endpoint returns data
✅ Auth cache hit rate >95%
✅ No errors in logs under normal operation
✅ Graceful degradation when backends fail

## Next Steps

- **Production Deployment**: See main README for Kubernetes deployment
- **CI/CD Integration**: Add to your CI/CD pipeline
- **Monitoring**: Set up Prometheus + Grafana
- **Alerting**: Configure alerts for critical metrics
