#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load .env file if it exists (one level up from script)
if [ -f "$SCRIPT_DIR/../.env" ]; then
    # shellcheck disable=SC1091
    source "$SCRIPT_DIR/../.env"
fi

ARTIFUSION_HOST="${ARTIFUSION_HOST:-artifacts.lvh.me}"
GITHUB_USERNAME="${GITHUB_USERNAME:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   OCI/Docker E2E Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check prerequisites
if [ -z "$GITHUB_USERNAME" ] || [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  GITHUB_USERNAME and GITHUB_TOKEN not set${NC}"
    echo -e "${YELLOW}   Tests will run but may fail without authentication${NC}"
    echo -e "${YELLOW}   Set with: export GITHUB_USERNAME=your-username GITHUB_TOKEN=ghp_xxx${NC}"
    echo ""
fi

# Cleanup function
cleanup() {
    echo ""
    echo -e "${BLUE}Cleaning up...${NC}"
    docker rmi "$ARTIFUSION_HOST/library/alpine:latest" 2>/dev/null || true
    docker rmi "$ARTIFUSION_HOST/testorg/hello-artifusion:latest" 2>/dev/null || true
    docker rmi hello-artifusion:latest 2>/dev/null || true
}

trap cleanup EXIT

# Test 1: Docker Login
echo -e "${BLUE}Test 1: Docker Login${NC}"
echo "Command: docker login $ARTIFUSION_HOST"
if [ -n "$GITHUB_USERNAME" ] && [ -n "$GITHUB_TOKEN" ]; then
    echo "$GITHUB_TOKEN" | docker login "$ARTIFUSION_HOST" -u "$GITHUB_USERNAME" --password-stdin
    echo -e "${GREEN}✓ Login successful${NC}"
else
    echo -e "${YELLOW}⚠️  Skipping login (credentials not provided)${NC}"
fi
echo ""

# Test 2: Pull from Docker Hub via cache (first time - cache miss)
echo -e "${BLUE}Test 2: Pull from Docker Hub via cache (Cache Miss)${NC}"
echo "Command: docker pull $ARTIFUSION_HOST/library/alpine:latest"
echo "This will cache the image from Docker Hub..."
START_TIME=$(date +%s)
docker pull "$ARTIFUSION_HOST/library/alpine:latest" || {
    echo -e "${RED}✗ Failed to pull image${NC}"
    exit 1
}
END_TIME=$(date +%s)
CACHE_MISS_TIME=$((END_TIME - START_TIME))
echo -e "${GREEN}✓ Image pulled successfully (took ${CACHE_MISS_TIME}s)${NC}"
echo ""

# Test 3: Pull same image again (cache hit)
echo -e "${BLUE}Test 3: Pull from cache (Cache Hit)${NC}"
echo "Command: docker pull $ARTIFUSION_HOST/library/alpine:latest"
echo "This should be faster as the image is cached..."
# Remove local image first to force pull from Artifusion
docker rmi "$ARTIFUSION_HOST/library/alpine:latest" || true
START_TIME=$(date +%s)
docker pull "$ARTIFUSION_HOST/library/alpine:latest" || {
    echo -e "${RED}✗ Failed to pull cached image${NC}"
    exit 1
}
END_TIME=$(date +%s)
CACHE_HIT_TIME=$((END_TIME - START_TIME))
echo -e "${GREEN}✓ Image pulled from cache (took ${CACHE_HIT_TIME}s)${NC}"

# Compare times
if [ "$CACHE_HIT_TIME" -lt "$CACHE_MISS_TIME" ]; then
    SPEEDUP=$((CACHE_MISS_TIME - CACHE_HIT_TIME))
    echo -e "${GREEN}✓ Cache hit was ${SPEEDUP}s faster!${NC}"
else
    echo -e "${YELLOW}⚠️  Cache hit was not faster (times can vary on first run)${NC}"
fi
echo ""

# Test 4: Create and push custom image
echo -e "${BLUE}Test 4: Create and Push Custom Image${NC}"
echo "Creating a simple test image..."

# Create a Dockerfile
mkdir -p /tmp/artifusion-docker-test
cat > /tmp/artifusion-docker-test/Dockerfile << 'EOF'
FROM alpine:latest
RUN echo "Hello from Artifusion!" > /hello.txt
CMD ["cat", "/hello.txt"]
EOF

cd /tmp/artifusion-docker-test
docker build -t hello-artifusion:latest . || {
    echo -e "${RED}✗ Failed to build image${NC}"
    exit 1
}
echo -e "${GREEN}✓ Image built${NC}"

# Tag for Artifusion
docker tag hello-artifusion:latest "$ARTIFUSION_HOST/testorg/hello-artifusion:latest"
echo -e "${GREEN}✓ Image tagged${NC}"

# Push to Artifusion
echo "Pushing to Artifusion hosted registry..."
docker push "$ARTIFUSION_HOST/testorg/hello-artifusion:latest" || {
    echo -e "${RED}✗ Failed to push image${NC}"
    exit 1
}
echo -e "${GREEN}✓ Image pushed successfully${NC}"
echo ""

# Test 5: Pull the custom image back
echo -e "${BLUE}Test 5: Pull Custom Image${NC}"
echo "Command: docker pull $ARTIFUSION_HOST/testorg/hello-artifusion:latest"

# Remove local images first
docker rmi "$ARTIFUSION_HOST/testorg/hello-artifusion:latest" || true
docker rmi hello-artifusion:latest || true

docker pull "$ARTIFUSION_HOST/testorg/hello-artifusion:latest" || {
    echo -e "${RED}✗ Failed to pull custom image${NC}"
    exit 1
}
echo -e "${GREEN}✓ Custom image pulled successfully${NC}"
echo ""

# Test 6: Run the custom image
echo -e "${BLUE}Test 6: Run Custom Image${NC}"
OUTPUT=$(docker run --rm "$ARTIFUSION_HOST/testorg/hello-artifusion:latest")
EXPECTED="Hello from Artifusion!"

if [ "$OUTPUT" = "$EXPECTED" ]; then
    echo -e "${GREEN}✓ Image output correct: $OUTPUT${NC}"
else
    echo -e "${RED}✗ Image output incorrect${NC}"
    echo -e "${RED}  Expected: $EXPECTED${NC}"
    echo -e "${RED}  Got: $OUTPUT${NC}"
    exit 1
fi
echo ""

# Test 7: Verify caching with metrics
echo -e "${BLUE}Test 7: Check Metrics${NC}"
echo "Querying Artifusion metrics..."
METRICS=$(curl -s "http://$ARTIFUSION_HOST/metrics" 2>&1)

if [ -z "$METRICS" ]; then
    echo -e "${YELLOW}⚠️  Could not retrieve metrics (empty response)${NC}"
elif echo "$METRICS" | grep -q "artifusion"; then
    echo -e "${GREEN}✓ Metrics endpoint accessible${NC}"
    echo ""
    echo "OCI-related metrics:"
    OCI_METRICS=$(echo "$METRICS" | grep 'protocol="oci"' | head -5)
    if [ -n "$OCI_METRICS" ]; then
        echo "$OCI_METRICS"
    else
        echo "  (No OCI-specific metrics yet - this is normal if OCI requests haven't been made)"
    fi
    echo ""
    echo "Backend Health:"
    BACKEND_HEALTH=$(echo "$METRICS" | grep 'artifusion_backend_health{' | grep -E 'local-hosted|ghcr-cache|dockerhub-cache')
    if [ -n "$BACKEND_HEALTH" ]; then
        echo "$BACKEND_HEALTH"
    else
        echo "  (Backend health metrics not yet available)"
    fi
else
    echo -e "${YELLOW}⚠️  Could not retrieve valid metrics${NC}"
    echo "  Response: ${METRICS:0:100}..."
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}   All OCI/Docker Tests Passed! ✓${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Summary:"
echo "  - Docker Hub pull via cache: ${CACHE_MISS_TIME}s (cache miss)"
echo "  - Second pull: ${CACHE_HIT_TIME}s (cache hit)"
echo "  - Custom image push: ✓"
echo "  - Custom image pull: ✓"
echo "  - Custom image run: ✓"
echo ""
