#!/usr/bin/env bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
ARTIFUSION_HOST="${ARTIFUSION_HOST:-artifacts.lvh.me}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load .env file if it exists
if [ -f "$SCRIPT_DIR/.env" ]; then
    echo "Loading credentials from .env file..."
    # shellcheck disable=SC1090
    source "$SCRIPT_DIR/.env"
fi

# Load mise shims if mise is installed (for version-managed tools)
if command -v mise &> /dev/null; then
    eval "$(mise activate bash --shims)" 2>/dev/null || true
fi

# Test results (using simple variables instead of associative arrays for bash 3.x compatibility)
OCI_RESULT=""
OCI_TIME=0
MAVEN_RESULT=""
MAVEN_TIME=0
NPM_RESULT=""
NPM_TIME=0

echo -e "${MAGENTA}"
cat << "EOF"
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║        ARTIFUSION E2E TEST SUITE                         ║
║        Multi-Protocol Artifact Proxy Tests               ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${CYAN}Testing: $ARTIFUSION_HOST${NC}"
echo ""

# Check if Artifusion is running
echo -e "${BLUE}Checking Artifusion health...${NC}"
if curl -sf "http://$ARTIFUSION_HOST/health" > /dev/null 2>&1; then
    HEALTH=$(curl -s "http://$ARTIFUSION_HOST/health")
    VERSION=$(echo "$HEALTH" | grep -o '"version":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Artifusion is healthy (version: $VERSION)${NC}"
else
    echo -e "${RED}✗ Artifusion is not responding at http://$ARTIFUSION_HOST${NC}"
    echo -e "${RED}  Please ensure Artifusion is running${NC}"
    echo ""
    echo "To start Artifusion locally:"
    echo "  kubectl get pods -n artifusion"
    exit 1
fi
echo ""

# Check for credentials
echo -e "${BLUE}Checking credentials...${NC}"
if [ -z "$GITHUB_USERNAME" ] || [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  GitHub credentials not set in environment${NC}"
    echo ""
    echo "You can either:"
    echo "  1. Set environment variables before running:"
    echo "     export GITHUB_USERNAME=your-username"
    echo "     export GITHUB_TOKEN=ghp_your_token"
    echo ""
    echo "  2. Enter them now (they will be used for this test run only)"
    echo ""

    read -p "Enter GitHub username (or press Enter to skip): " INPUT_USERNAME
    if [ -n "$INPUT_USERNAME" ]; then
        export GITHUB_USERNAME="$INPUT_USERNAME"
        read -sp "Enter GitHub token (ghp_...): " INPUT_TOKEN
        echo ""
        export GITHUB_TOKEN="$INPUT_TOKEN"
        echo -e "${GREEN}✓ Credentials set for this test run${NC}"
    else
        echo -e "${YELLOW}⚠️  Running tests without authentication${NC}"
        echo -e "${YELLOW}   Some tests may fail${NC}"
    fi
else
    echo -e "${GREEN}✓ Using GitHub credentials from environment${NC}"
    echo -e "  Username: $GITHUB_USERNAME"
fi
echo ""

# Check prerequisites
echo -e "${BLUE}Checking prerequisites...${NC}"

MISSING_TOOLS=()

if ! command -v docker &> /dev/null; then
    MISSING_TOOLS+=("docker")
fi

if ! command -v mvn &> /dev/null; then
    MISSING_TOOLS+=("maven")
fi

if ! command -v npm &> /dev/null; then
    MISSING_TOOLS+=("npm/node")
fi

if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Some tools are missing:${NC}"
    for tool in "${MISSING_TOOLS[@]}"; do
        echo -e "${YELLOW}   - $tool${NC}"
    done
    echo ""
    echo "Install with:"
    echo "  brew install docker maven node"
    echo ""
    if [ -z "$SKIP_PREREQ_CHECK" ]; then
        read -p "Continue with available tools? (y/N): " CONTINUE
        if [[ ! "$CONTINUE" =~ ^[Yy]$ ]]; then
            echo "Exiting..."
            exit 1
        fi
    else
        echo "Continuing with available tools (SKIP_PREREQ_CHECK set)..."
    fi
fi

echo -e "${GREEN}✓ Prerequisites checked${NC}"
echo ""

# Function to run a test
run_test() {
    local test_name=$1
    local test_script=$2
    local protocol=$3

    echo ""
    echo -e "${MAGENTA}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${MAGENTA}  Running $test_name Tests${NC}"
    echo -e "${MAGENTA}═══════════════════════════════════════════════════════════${NC}"
    echo ""

    START_TIME=$(date +%s)

    if bash "$test_script"; then
        END_TIME=$(date +%s)
        DURATION=$((END_TIME - START_TIME))
        case $protocol in
            oci)
                OCI_RESULT="PASS"
                OCI_TIME=$DURATION
                ;;
            maven)
                MAVEN_RESULT="PASS"
                MAVEN_TIME=$DURATION
                ;;
            npm)
                NPM_RESULT="PASS"
                NPM_TIME=$DURATION
                ;;
        esac
        echo -e "${GREEN}✓ $test_name tests passed (${DURATION}s)${NC}"
        return 0
    else
        END_TIME=$(date +%s)
        DURATION=$((END_TIME - START_TIME))
        case $protocol in
            oci)
                OCI_RESULT="FAIL"
                OCI_TIME=$DURATION
                ;;
            maven)
                MAVEN_RESULT="FAIL"
                MAVEN_TIME=$DURATION
                ;;
            npm)
                NPM_RESULT="FAIL"
                NPM_TIME=$DURATION
                ;;
        esac
        echo -e "${RED}✗ $test_name tests failed (${DURATION}s)${NC}"
        return 1
    fi
}

# Run tests
TOTAL_START=$(date +%s)

# Test 1: OCI/Docker
if command -v docker &> /dev/null; then
    run_test "OCI/Docker" "$SCRIPT_DIR/docker/test-oci.sh" "oci"
else
    echo -e "${YELLOW}Skipping OCI/Docker tests (docker not installed)${NC}"
    OCI_RESULT="SKIP"
fi

# Test 2: Maven
if command -v mvn &> /dev/null; then
    run_test "Maven" "$SCRIPT_DIR/maven/test-maven.sh" "maven"
else
    echo -e "${YELLOW}Skipping Maven tests (maven not installed)${NC}"
    MAVEN_RESULT="SKIP"
fi

# Test 3: NPM
if command -v npm &> /dev/null; then
    run_test "NPM" "$SCRIPT_DIR/npm/test-npm.sh" "npm"
else
    echo -e "${YELLOW}Skipping NPM tests (npm not installed)${NC}"
    NPM_RESULT="SKIP"
fi

TOTAL_END=$(date +%s)
TOTAL_DURATION=$((TOTAL_END - TOTAL_START))

# Summary
echo ""
echo -e "${MAGENTA}═══════════════════════════════════════════════════════════${NC}"
echo -e "${MAGENTA}  TEST SUITE SUMMARY${NC}"
echo -e "${MAGENTA}═══════════════════════════════════════════════════════════${NC}"
echo ""

echo -e "${CYAN}Individual Test Results:${NC}"
echo ""

# Table header
printf "%-20s %-10s %-10s\n" "Protocol" "Status" "Time"
printf "%-20s %-10s %-10s\n" "--------" "------" "----"

PASSED=0
FAILED=0
SKIPPED=0

# Print OCI results
if [ "$OCI_RESULT" = "PASS" ]; then
    printf "%-20s ${GREEN}%-10s${NC} %-10s\n" "oci" "✓ PASS" "${OCI_TIME}s"
    ((PASSED++))
elif [ "$OCI_RESULT" = "FAIL" ]; then
    printf "%-20s ${RED}%-10s${NC} %-10s\n" "oci" "✗ FAIL" "${OCI_TIME}s"
    ((FAILED++))
else
    printf "%-20s ${YELLOW}%-10s${NC} %-10s\n" "oci" "⊘ SKIP" "N/A"
    ((SKIPPED++))
fi

# Print Maven results
if [ "$MAVEN_RESULT" = "PASS" ]; then
    printf "%-20s ${GREEN}%-10s${NC} %-10s\n" "maven" "✓ PASS" "${MAVEN_TIME}s"
    ((PASSED++))
elif [ "$MAVEN_RESULT" = "FAIL" ]; then
    printf "%-20s ${RED}%-10s${NC} %-10s\n" "maven" "✗ FAIL" "${MAVEN_TIME}s"
    ((FAILED++))
else
    printf "%-20s ${YELLOW}%-10s${NC} %-10s\n" "maven" "⊘ SKIP" "N/A"
    ((SKIPPED++))
fi

# Print NPM results
if [ "$NPM_RESULT" = "PASS" ]; then
    printf "%-20s ${GREEN}%-10s${NC} %-10s\n" "npm" "✓ PASS" "${NPM_TIME}s"
    ((PASSED++))
elif [ "$NPM_RESULT" = "FAIL" ]; then
    printf "%-20s ${RED}%-10s${NC} %-10s\n" "npm" "✗ FAIL" "${NPM_TIME}s"
    ((FAILED++))
else
    printf "%-20s ${YELLOW}%-10s${NC} %-10s\n" "npm" "⊘ SKIP" "N/A"
    ((SKIPPED++))
fi

echo ""
echo -e "${CYAN}Overall Statistics:${NC}"
echo "  Total tests: $((PASSED + FAILED + SKIPPED))"
echo -e "  ${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "  ${RED}Failed: $FAILED${NC}"
else
    echo -e "  Failed: 0"
fi
if [ $SKIPPED -gt 0 ]; then
    echo -e "  ${YELLOW}Skipped: $SKIPPED${NC}"
fi
echo "  Total time: ${TOTAL_DURATION}s"
echo ""

# Final status
if [ $FAILED -eq 0 ] && [ $PASSED -gt 0 ]; then
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                                                           ║${NC}"
    echo -e "${GREEN}║              ALL TESTS PASSED! ✓                         ║${NC}"
    echo -e "${GREEN}║                                                           ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Your Artifusion deployment is working correctly across all protocols!"
    echo ""
    echo "What was tested:"
    echo "  ✓ OCI/Docker: Pull caching, Push to registry, Image verification"
    echo "  ✓ Maven: Dependency caching, Artifact deployment, Consumption"
    echo "  ✓ NPM: Package caching, Publishing, Installation"
    echo ""
    exit 0
elif [ $PASSED -eq 0 ] && [ $SKIPPED -gt 0 ]; then
    echo -e "${YELLOW}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║                                                           ║${NC}"
    echo -e "${YELLOW}║              ALL TESTS SKIPPED                           ║${NC}"
    echo -e "${YELLOW}║                                                           ║${NC}"
    echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Please install the required tools and try again:"
    echo "  brew install docker maven node"
    echo ""
    exit 2
else
    echo -e "${RED}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║                                                           ║${NC}"
    echo -e "${RED}║              SOME TESTS FAILED                           ║${NC}"
    echo -e "${RED}║                                                           ║${NC}"
    echo -e "${RED}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Please review the test output above for details."
    echo ""
    echo "Common issues:"
    echo "  - Authentication: Set GITHUB_USERNAME and GITHUB_TOKEN"
    echo "  - Artifusion not running: kubectl get pods -n artifusion"
    echo "  - Network issues: Check ingress and DNS resolution"
    echo ""
    exit 1
fi
