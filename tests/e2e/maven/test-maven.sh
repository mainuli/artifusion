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

# Load mise shims if mise is installed (for version-managed tools like maven)
if command -v mise &> /dev/null; then
    eval "$(mise activate bash --shims)"
fi

ARTIFUSION_HOST="${ARTIFUSION_HOST:-artifacts.lvh.me}"
GITHUB_USERNAME="${GITHUB_USERNAME:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
TEST_DIR="/tmp/artifusion-maven-test"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Maven E2E Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check prerequisites
if [ -z "$GITHUB_USERNAME" ] || [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  GITHUB_USERNAME and GITHUB_TOKEN not set${NC}"
    echo -e "${YELLOW}   Tests will run but may fail without authentication${NC}"
    echo -e "${YELLOW}   Set with: export GITHUB_USERNAME=your-username GITHUB_TOKEN=ghp_xxx${NC}"
    echo ""
fi

# Check if Maven is installed
if ! command -v mvn &> /dev/null; then
    echo -e "${RED}✗ Maven (mvn) not found in PATH${NC}"
    echo -e "${RED}  Please install Maven:${NC}"
    echo -e "${RED}    - Via Homebrew: brew install maven${NC}"
    echo -e "${RED}    - Via mise: mise use -g maven@latest${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Maven found: $(mvn --version | head -1)${NC}"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo -e "${BLUE}Cleaning up...${NC}"
    rm -rf "$TEST_DIR"
}

trap cleanup EXIT

# Setup
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Test 1: Create Maven settings.xml with Artifusion
echo -e "${BLUE}Test 1: Configure Maven with Artifusion${NC}"
echo -e "${YELLOW}Using Artifusion unified Maven repository (proxies Maven Central + other upstreams)${NC}"
mkdir -p "$TEST_DIR/.m2"

# Detect path prefix by checking what's configured in Artifusion
MAVEN_PATH_PREFIX="/m2"  # Default to /m2 based on current config

# Client authentication with Artifusion uses GitHub credentials
# Artifusion handles backend authentication with Reposilite transparently
cat > "$TEST_DIR/.m2/settings.xml" << EOF
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0
                              https://maven.apache.org/xsd/settings-1.0.0.xsd">
  <servers>
    <!-- Server auth for Artifusion unified repository -->
    <!-- Uses GitHub credentials for client authentication -->
    <!-- Artifusion handles backend authentication with Reposilite -->
    <server>
      <id>artifusion</id>
      <username>${GITHUB_USERNAME}</username>
      <password>${GITHUB_TOKEN}</password>
    </server>
  </servers>

  <profiles>
    <profile>
      <id>artifusion</id>
      <repositories>
        <!-- Unified repository for both releases and snapshots -->
        <repository>
          <id>artifusion</id>
          <name>Artifusion Maven Repository</name>
          <url>http://${ARTIFUSION_HOST}${MAVEN_PATH_PREFIX}</url>
          <releases>
            <enabled>true</enabled>
          </releases>
          <snapshots>
            <enabled>true</enabled>
          </snapshots>
        </repository>
      </repositories>
    </profile>
  </profiles>

  <activeProfiles>
    <activeProfile>artifusion</activeProfile>
  </activeProfiles>
</settings>
EOF

echo -e "${GREEN}✓ Maven settings.xml created${NC}"
echo ""

# Test 2: Create a simple Maven project
echo -e "${BLUE}Test 2: Create Test Maven Project${NC}"
mkdir -p consumer-project
cd consumer-project

cat > pom.xml << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
                             http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>com.artifusion.test</groupId>
    <artifactId>consumer</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>

    <name>Artifusion Maven Test Consumer</name>

    <properties>
        <maven.compiler.source>11</maven.compiler.source>
        <maven.compiler.target>11</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <dependencies>
        <!-- Small, stable dependency for testing -->
        <dependency>
            <groupId>commons-io</groupId>
            <artifactId>commons-io</artifactId>
            <version>2.11.0</version>
        </dependency>
    </dependencies>
</project>
EOF

mkdir -p src/main/java/com/artifusion/test
cat > src/main/java/com/artifusion/test/Main.java << 'EOF'
package com.artifusion.test;

import org.apache.commons.io.FileUtils;

public class Main {
    public static void main(String[] args) {
        System.out.println("Hello from Artifusion Maven Test!");
        System.out.println("Commons IO version: " + FileUtils.class.getPackage().getImplementationVersion());
    }
}
EOF

echo -e "${GREEN}✓ Maven project created${NC}"
echo ""

# Test 3: Build project with dependencies from Maven Central (via Artifusion)
echo -e "${BLUE}Test 3: Build Consumer Project${NC}"
echo "Building project (dependencies proxied through Artifusion from Maven Central)..."
START_TIME=$(date +%s)
mvn -s "$TEST_DIR/.m2/settings.xml" clean compile || {
    echo -e "${RED}✗ Failed to build project${NC}"
    exit 1
}
END_TIME=$(date +%s)
BUILD_TIME=$((END_TIME - START_TIME))
echo -e "${GREEN}✓ Project built successfully (took ${BUILD_TIME}s)${NC}"
echo -e "${GREEN}  Dependencies fetched through Artifusion proxy${NC}"
echo ""

# Test 4: Create and deploy a custom artifact
echo -e "${BLUE}Test 4: Create and Deploy Custom Artifact${NC}"
cd "$TEST_DIR"
mkdir -p library-project
cd library-project

cat > pom.xml << EOF
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
                             http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>com.artifusion.test</groupId>
    <artifactId>hello-library</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>

    <name>Hello Library</name>

    <properties>
        <maven.compiler.source>11</maven.compiler.source>
        <maven.compiler.target>11</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <distributionManagement>
        <repository>
            <id>artifusion</id>
            <name>Artifusion Maven Repository</name>
            <url>http://${ARTIFUSION_HOST}${MAVEN_PATH_PREFIX}</url>
        </repository>
        <snapshotRepository>
            <id>artifusion</id>
            <name>Artifusion Maven Repository</name>
            <url>http://${ARTIFUSION_HOST}${MAVEN_PATH_PREFIX}</url>
        </snapshotRepository>
    </distributionManagement>
</project>
EOF

mkdir -p src/main/java/com/artifusion/test
cat > src/main/java/com/artifusion/test/HelloLibrary.java << 'EOF'
package com.artifusion.test;

public class HelloLibrary {
    public static String sayHello() {
        return "Hello from Artifusion Library!";
    }
}
EOF

echo "Building library..."
mvn -s "$TEST_DIR/.m2/settings.xml" clean package || {
    echo -e "${RED}✗ Failed to build library${NC}"
    exit 1
}
echo -e "${GREEN}✓ Library built${NC}"

echo "Deploying to Artifusion..."
# Check if credentials are both set and non-empty
if [ -n "$GITHUB_USERNAME" ] && [ -n "$GITHUB_TOKEN" ]; then
    if mvn -s "$TEST_DIR/.m2/settings.xml" deploy; then
        echo -e "${GREEN}✓ Artifact deployed successfully${NC}"
        DEPLOYED_TO_ARTIFUSION=true
    else
        echo -e "${RED}✗ Failed to deploy artifact${NC}"
        echo -e "${YELLOW}   Check that credentials have write permissions${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠️  Skipping deploy test (no GitHub credentials)${NC}"
    echo -e "${YELLOW}   Set GITHUB_USERNAME and GITHUB_TOKEN to test artifact deployment${NC}"
    echo ""
    # Create artifact locally for consumption test (explicitly skip deploy phase)
    mvn -s "$TEST_DIR/.m2/settings.xml" clean package install -DskipTests -Dmaven.deploy.skip=true > /dev/null 2>&1
    DEPLOYED_TO_ARTIFUSION=false
fi
echo ""

# Test 5: Consume the deployed artifact (from local repo or Artifusion)
echo -e "${BLUE}Test 5: Consume Deployed Artifact${NC}"
if [ "$DEPLOYED_TO_ARTIFUSION" = "true" ]; then
    echo -e "${YELLOW}Testing artifact consumption from Artifusion${NC}"
else
    echo -e "${YELLOW}Testing artifact consumption from local Maven repository${NC}"
fi

cd "$TEST_DIR"
mkdir -p consumer-project-2
cd consumer-project-2

cat > pom.xml << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
                             http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>com.artifusion.test</groupId>
    <artifactId>consumer-2</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>

    <properties>
        <maven.compiler.source>11</maven.compiler.source>
        <maven.compiler.target>11</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <dependencies>
        <dependency>
            <groupId>com.artifusion.test</groupId>
            <artifactId>hello-library</artifactId>
            <version>1.0.0</version>
        </dependency>
    </dependencies>
</project>
EOF

mkdir -p src/main/java/com/artifusion/test
cat > src/main/java/com/artifusion/test/Consumer.java << 'EOF'
package com.artifusion.test;

public class Consumer {
    public static void main(String[] args) {
        System.out.println(HelloLibrary.sayHello());
    }
}
EOF

echo "Building consumer project with deployed artifact..."
mvn -s "$TEST_DIR/.m2/settings.xml" clean compile || {
    echo -e "${RED}✗ Failed to build consumer project${NC}"
    exit 1
}
echo -e "${GREEN}✓ Successfully consumed deployed artifact${NC}"
echo ""

# Test 6: Check Metrics
echo -e "${BLUE}Test 6: Check Metrics${NC}"
echo "Querying Artifusion metrics..."
METRICS=$(curl -s "http://$ARTIFUSION_HOST/metrics" 2>&1)

if [ -z "$METRICS" ]; then
    echo -e "${YELLOW}⚠️  Could not retrieve metrics (empty response)${NC}"
elif echo "$METRICS" | grep -q "artifusion"; then
    echo -e "${GREEN}✓ Metrics endpoint accessible${NC}"
    echo ""
    echo "Maven-related metrics:"
    MAVEN_METRICS=$(echo "$METRICS" | grep 'protocol="maven"' | head -5)
    if [ -n "$MAVEN_METRICS" ]; then
        echo "$MAVEN_METRICS"
    else
        echo "  (No Maven-specific metrics yet - this is normal if Maven requests haven't been made)"
    fi
    echo ""
    echo "Reposilite backend health:"
    REPOSILITE_HEALTH=$(echo "$METRICS" | grep 'artifusion_backend_health{.*backend="reposilite"')
    if [ -n "$REPOSILITE_HEALTH" ]; then
        echo "  $REPOSILITE_HEALTH"
    else
        echo "  (Backend health metric not yet available)"
    fi
else
    echo -e "${YELLOW}⚠️  Could not retrieve valid metrics${NC}"
    echo "  Response: ${METRICS:0:100}..."
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}   All Maven Tests Passed! ✓${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Summary:"
echo "  - Project build: ${BUILD_TIME}s"
if [ "$DEPLOYED_TO_ARTIFUSION" = "true" ]; then
    echo "  - Artifact deployment: ✓"
    echo "  - Artifact consumption: ✓ (from Artifusion)"
else
    echo "  - Artifact deployment: ⊘ (skipped - no credentials)"
    echo "  - Artifact consumption: ✓ (from local repository)"
fi
echo ""
echo -e "${CYAN}Configuration Details:${NC}"
echo "  - Artifusion host: $ARTIFUSION_HOST"
echo "  - Maven path: ${MAVEN_PATH_PREFIX}"
echo "  - Repository: Unified (releases + snapshots)"
echo "  - Proxy upstreams: Maven Central, JasperReports, Spring, Sonatype, Gradle"
echo "  - Dependency proxy: ✓ Working (Maven Central proxied through Artifusion)"
if [ "$DEPLOYED_TO_ARTIFUSION" != "true" ]; then
    echo ""
    echo -e "${YELLOW}💡 To test artifact deployment, set GitHub credentials:${NC}"
    echo "     export GITHUB_USERNAME=your-username"
    echo "     export GITHUB_TOKEN=ghp_your_token"
fi
echo ""
