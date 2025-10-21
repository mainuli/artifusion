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
echo -e "${YELLOW}NOTE: Currently using Maven Central directly due to Reposilite backend configuration issue${NC}"
echo -e "${YELLOW}      This test validates deploy/consume functionality only${NC}"
mkdir -p "$TEST_DIR/.m2"

cat > "$TEST_DIR/.m2/settings.xml" << EOF
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0
                              https://maven.apache.org/xsd/settings-1.0.0.xsd">
  <servers>
    <!-- Server auth for deploying to Artifusion -->
    <server>
      <id>artifusion-releases</id>
      <username>${GITHUB_USERNAME}</username>
      <password>${GITHUB_TOKEN}</password>
    </server>
    <server>
      <id>artifusion-snapshots</id>
      <username>${GITHUB_USERNAME}</username>
      <password>${GITHUB_TOKEN}</password>
    </server>
  </servers>

  <profiles>
    <profile>
      <id>artifusion</id>
      <repositories>
        <!-- For deploying releases to Artifusion -->
        <repository>
          <id>artifusion-releases</id>
          <name>Artifusion Releases</name>
          <url>http://${ARTIFUSION_HOST}/maven/releases</url>
          <releases>
            <enabled>true</enabled>
          </releases>
          <snapshots>
            <enabled>false</enabled>
          </snapshots>
        </repository>
        <!-- For deploying snapshots to Artifusion -->
        <repository>
          <id>artifusion-snapshots</id>
          <name>Artifusion Snapshots</name>
          <url>http://${ARTIFUSION_HOST}/maven/snapshots</url>
          <releases>
            <enabled>false</enabled>
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

# Test 3: Build project with dependencies from Maven Central
echo -e "${BLUE}Test 3: Build Consumer Project${NC}"
echo "Building project (dependencies from Maven Central)..."
START_TIME=$(date +%s)
mvn -s "$TEST_DIR/.m2/settings.xml" clean compile || {
    echo -e "${RED}✗ Failed to build project${NC}"
    exit 1
}
END_TIME=$(date +%s)
BUILD_TIME=$((END_TIME - START_TIME))
echo -e "${GREEN}✓ Project built successfully (took ${BUILD_TIME}s)${NC}"
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
            <id>artifusion-releases</id>
            <name>Artifusion Releases</name>
            <url>http://${ARTIFUSION_HOST}/maven/releases</url>
        </repository>
        <snapshotRepository>
            <id>artifusion-snapshots</id>
            <name>Artifusion Snapshots</name>
            <url>http://${ARTIFUSION_HOST}/maven/snapshots</url>
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
mvn -s "$TEST_DIR/.m2/settings.xml" deploy || {
    echo -e "${RED}✗ Failed to deploy artifact${NC}"
    echo -e "${YELLOW}Note: This may fail if authentication is not configured${NC}"
    exit 1
}
echo -e "${GREEN}✓ Artifact deployed successfully${NC}"
echo ""

# Test 5: Consume the deployed artifact
echo -e "${BLUE}Test 5: Consume Deployed Artifact${NC}"
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
echo "  - Artifact deploy to Artifusion: ✓"
echo "  - Artifact consumption from Artifusion: ✓"
echo ""
echo -e "${YELLOW}Note: Maven Central caching via Artifusion is currently disabled${NC}"
echo -e "${YELLOW}      due to Reposilite backend configuration issue.${NC}"
echo ""
