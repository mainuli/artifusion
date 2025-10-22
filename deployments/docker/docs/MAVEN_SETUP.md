# Maven Repository Setup Guide

Complete guide for using Artifusion with Reposilite Maven repository manager.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Repository Types](#repository-types)
- [Quick Start](#quick-start)
- [Maven Client Configuration](#maven-client-configuration)
- [Deploying Artifacts](#deploying-artifacts)
- [Pulling Dependencies](#pulling-dependencies)
- [Troubleshooting](#troubleshooting)

---

## Architecture Overview

```
Maven Client (mvn deploy/install)
    ↓
GitHub PAT Authentication
    ↓
Artifusion Proxy (Port 8080)
    ↓
URL Rewriting & Auth Injection
    ↓
Reposilite (Port 8081)
    ↓
┌─────────────────┬──────────────────────┐
│ Hosted Repos    │ Mirror Repos         │
├─────────────────┼──────────────────────┤
│ /releases       │ /github-packages (1) │
│ /snapshots      │ /maven-central (2)   │
│                 │ /google (3)          │
│                 │ /spring-releases (4) │
│                 │ /gradle-plugins (5)  │
│                 │ /jcenter (6)         │
└─────────────────┴──────────────────────┘
```

**Key Features:**
- ✅ **Hosted repositories** for releases and snapshots
- ✅ **Mirror repositories** with cascade fallback
- ✅ **Artifact caching** for faster downloads
- ✅ **GitHub authentication** via Artifusion
- ✅ **URL rewriting** for transparent proxying
- ✅ **No Web UI** (API-only mode for security)

---

## Repository Types

### Hosted Repositories

**`/releases`** - Stable Release Artifacts
- **Immutable**: Cannot redeploy same version
- **Example versions**: `1.0.0`, `2.1.5`, `3.0.0-RC1`
- **Use for**: Production-ready releases

**`/snapshots`** - Development Snapshot Artifacts
- **Mutable**: Can redeploy same snapshot version
- **Example versions**: `1.0.0-SNAPSHOT`, `2.1.0-SNAPSHOT`
- **Use for**: Development and testing

### Mirror Repositories (Cascade Priority)

Reposilite searches mirror repositories in this order when artifacts are not found in hosted repositories:

1. **`/github-packages`** (HIGHEST PRIORITY)
   - GitHub Packages Maven repository
   - Requires GitHub PAT with `read:packages` scope
   - Scoped to your organization repositories

2. **`/maven-central`**
   - Maven Central Repository
   - Primary public repository
   - Most common Java libraries

3. **`/google`**
   - Google Maven Repository
   - Android libraries and Google dependencies

4. **`/spring-releases`**
   - Spring Framework Repository
   - Spring Boot, Spring Cloud, etc.

5. **`/gradle-plugins`**
   - Gradle Plugin Portal
   - Gradle plugins as Maven artifacts

6. **`/jcenter`** (OPTIONAL)
   - JCenter Repository
   - Legacy support (deprecated but still useful)

**Caching:** All proxied artifacts are cached locally for faster subsequent access.

---

## Quick Start

### 1. Start Services

```bash
cd deployments/docker

# Copy and configure environment variables
cp .env.example .env
# Edit .env with your tokens

# Start all services
docker-compose up -d

# Check health
docker-compose ps
curl http://localhost:8080/health
```

### 2. Generate Reposilite Tokens

```bash
# Generate secure tokens
openssl rand -hex 32  # Admin token
openssl rand -hex 32  # Write token
openssl rand -hex 32  # Read token

# Update .env file with generated tokens
nano .env
```

### 3. Verify Maven Setup

```bash
# Check Reposilite repositories
curl -u "$REPOSILITE_READ_TOKEN:" http://localhost:8081/api/maven/details/releases

# Test through Artifusion proxy
curl -H "Authorization: Bearer $GITHUB_PAT" http://localhost:8080/releases/
```

---

## Maven Client Configuration

### settings.xml

Configure Maven to use Artifusion as your repository manager:

```xml
<!-- ~/.m2/settings.xml -->
<settings>
  <servers>
    <!-- Credentials for deploying to releases -->
    <server>
      <id>artifusion-releases</id>
      <username>your-github-username</username>
      <password>ghp_your_github_personal_access_token</password>
    </server>

    <!-- Credentials for deploying to snapshots -->
    <server>
      <id>artifusion-snapshots</id>
      <username>your-github-username</username>
      <password>ghp_your_github_personal_access_token</password>
    </server>

    <!-- Credentials for pulling dependencies -->
    <server>
      <id>artifusion-central</id>
      <username>your-github-username</username>
      <password>ghp_your_github_personal_access_token</password>
    </server>
  </servers>

  <!-- Use Artifusion as the central mirror for all Maven repositories -->
  <mirrors>
    <mirror>
      <id>artifusion-mirror</id>
      <name>Artifusion Maven Proxy</name>
      <url>http://localhost:8080/maven-central</url>
      <mirrorOf>central</mirrorOf>
    </mirror>
  </mirrors>

  <profiles>
    <profile>
      <id>artifusion</id>
      <repositories>
        <repository>
          <id>artifusion-central</id>
          <url>http://localhost:8080/maven-central</url>
          <releases><enabled>true</enabled></releases>
          <snapshots><enabled>false</enabled></snapshots>
        </repository>
        <repository>
          <id>artifusion-snapshots-repo</id>
          <url>http://localhost:8080/snapshots</url>
          <releases><enabled>false</enabled></releases>
          <snapshots>
            <enabled>true</enabled>
            <updatePolicy>always</updatePolicy>
          </snapshots>
        </repository>
      </repositories>
    </profile>
  </profiles>

  <activeProfiles>
    <activeProfile>artifusion</activeProfile>
  </activeProfiles>
</settings>
```

### pom.xml

Configure your project to deploy artifacts to Artifusion:

```xml
<!-- pom.xml -->
<project>
  <groupId>com.example</groupId>
  <artifactId>my-library</artifactId>
  <version>1.0.0</version>  <!-- Or 1.0.0-SNAPSHOT -->

  <!-- Distribution management -->
  <distributionManagement>
    <repository>
      <id>artifusion-releases</id>
      <name>Artifusion Releases Repository</name>
      <url>http://localhost:8080/releases</url>
    </repository>
    <snapshotRepository>
      <id>artifusion-snapshots</id>
      <name>Artifusion Snapshots Repository</name>
      <url>http://localhost:8080/snapshots</url>
    </snapshotRepository>
  </distributionManagement>

  <!-- Optional: Explicitly configure repositories -->
  <repositories>
    <repository>
      <id>artifusion-central</id>
      <url>http://localhost:8080/maven-central</url>
      <releases><enabled>true</enabled></releases>
      <snapshots><enabled>false</enabled></snapshots>
    </repository>
  </repositories>
</project>
```

---

## Deploying Artifacts

### Deploy Release

```bash
# Set version to release (no -SNAPSHOT suffix)
mvn versions:set -DnewVersion=1.0.0

# Deploy to releases repository
mvn clean deploy

# Artifacts deployed to: http://localhost:8080/releases/com/example/my-library/1.0.0/
```

**Note:** Release artifacts are **immutable**. Attempting to redeploy the same version will fail.

### Deploy Snapshot

```bash
# Set version to snapshot (with -SNAPSHOT suffix)
mvn versions:set -DnewVersion=1.0.0-SNAPSHOT

# Deploy to snapshots repository
mvn clean deploy

# Artifacts deployed to: http://localhost:8080/snapshots/com/example/my-library/1.0.0-SNAPSHOT/
```

**Note:** Snapshot artifacts are **mutable**. You can redeploy the same snapshot version multiple times.

---

## Pulling Dependencies

### From Hosted Repositories

```xml
<dependency>
  <groupId>com.example</groupId>
  <artifactId>my-library</artifactId>
  <version>1.0.0</version>  <!-- From /releases -->
</dependency>

<dependency>
  <groupId>com.example</groupId>
  <artifactId>my-library</artifactId>
  <version>1.0.0-SNAPSHOT</version>  <!-- From /snapshots -->
</dependency>
```

```bash
mvn clean install
```

### From Mirror Repositories (Cascade)

When you request a public dependency, Reposilite searches in this order:

```xml
<dependency>
  <groupId>org.springframework.boot</groupId>
  <artifactId>spring-boot-starter-web</artifactId>
  <version>3.2.0</version>
</dependency>
```

**Search order:**
1. Check `/releases` (not found)
2. Check `/snapshots` (not found)
3. Check `/github-packages` (not found, requires org packages)
4. **Check `/maven-central`** ✅ **FOUND** → cache and return

**Subsequent requests:**
- Artifact is served from Reposilite cache (faster!)
- No upstream request needed

---

## GitHub Packages Integration

### Setup

1. **Generate GitHub PAT** with `read:packages` scope:
   ```
   https://github.com/settings/tokens/new
   ```

2. **Add to `.env`**:
   ```bash
   GITHUB_PACKAGES_TOKEN=ghp_your_token_here
   ```

3. **Restart services**:
   ```bash
   docker-compose restart reposilite
   ```

### Using GitHub Packages Artifacts

```xml
<dependency>
  <groupId>com.yourorg</groupId>
  <artifactId>your-package</artifactId>
  <version>1.0.0</version>
</dependency>
```

**Search flow:**
1. Check `/github-packages` (found if package exists in your org)
2. Artifact cached locally
3. Subsequent requests served from cache

---

## Troubleshooting

### Authentication Fails

**Symptom:** `401 Unauthorized` when deploying or pulling

**Solutions:**
1. Verify GitHub PAT is valid:
   ```bash
   curl -H "Authorization: Bearer ghp_your_token" https://api.github.com/user
   ```

2. Check Artifusion logs:
   ```bash
   docker-compose logs artifusion | grep -i auth
   ```

3. Verify token has correct scopes:
   - `read:user` for basic authentication
   - `read:org` if organization is required
   - `read:packages` for GitHub Packages

### Artifact Not Found

**Symptom:** `404 Not Found` when resolving dependency

**Solutions:**
1. Check repository availability:
   ```bash
   # Check Reposilite directly
   curl -u "$REPOSILITE_READ_TOKEN:" \
     http://localhost:8081/maven-central/com/example/mylib/1.0.0/mylib-1.0.0.pom
   ```

2. Check Reposilite logs for upstream errors:
   ```bash
   docker-compose logs reposilite | grep -i error
   ```

3. Verify artifact exists in upstream:
   ```bash
   curl -I https://repo.maven.apache.org/maven2/com/example/mylib/1.0.0/mylib-1.0.0.pom
   ```

### Cannot Redeploy Release

**Symptom:** `409 Conflict` when deploying release version

**Explanation:** Releases are immutable by design.

**Solutions:**
1. Increment version number:
   ```bash
   mvn versions:set -DnewVersion=1.0.1
   ```

2. Or use snapshots for development:
   ```bash
   mvn versions:set -DnewVersion=1.0.1-SNAPSHOT
   ```

### Slow Dependency Resolution

**Symptom:** First `mvn install` takes a long time

**Explanation:** Reposilite is fetching and caching artifacts from upstream repositories.

**Solutions:**
1. **This is normal behavior** - subsequent builds will be much faster
2. Check cache hit rate in logs:
   ```bash
   docker-compose logs reposilite | grep -i cache
   ```

3. Verify artifacts are being cached:
   ```bash
   docker-compose exec reposilite ls -lh /app/data/maven-central/
   ```

### GitHub Packages Access Denied

**Symptom:** `403 Forbidden` when accessing GitHub Packages artifacts

**Solutions:**
1. Verify token has `read:packages` scope
2. Check package visibility (must be accessible to your account/org)
3. Update `GITHUB_PACKAGES_TOKEN` in `.env`
4. Restart Reposilite:
   ```bash
   docker-compose restart reposilite
   ```

---

## Advanced Configuration

### Adjust Cache Timeouts

Edit `deployments/docker/config/reposilite.cdn/configuration.cdn`:

```cdn
repository maven-central {
  proxied {
    connectTimeout 5s     # Increase for slow networks
    readTimeout 30s       # Increase for large artifacts
  }
}
```

### Disable Specific Mirrors

Comment out unwanted repositories in `configuration.cdn`:

```cdn
# repository jcenter {
#   visibility public
#   proxied {
#     link https://jcenter.bintray.com/
#   }
# }
```

### Add Custom Mirror Repository

Add to `configuration.cdn`:

```cdn
repository my-custom-repo {
  visibility public
  redeployment false
  proxied {
    link https://my-custom-maven-repo.com/repository/
    store true
    storagePolicy PRIORITIZE_UPSTREAM_METADATA
    connectTimeout 3s
    readTimeout 15s
  }
  storageProvider fs my-custom-repo
}
```

---

## Useful Commands

```bash
# View Reposilite logs
docker-compose logs -f reposilite

# View Artifusion logs (Maven protocol)
docker-compose logs -f artifusion | grep maven

# Check disk usage
docker-compose exec reposilite du -sh /app/data/*

# Clear Maven local cache
rm -rf ~/.m2/repository

# Clear Reposilite cache for a specific repository
docker-compose exec reposilite rm -rf /app/data/maven-central/*

# Restart services
docker-compose restart artifusion reposilite

# View health status
curl http://localhost:8080/health
curl http://localhost:8080/metrics | grep maven
```

---

## Security Best Practices

1. **Never commit tokens to version control**
   - Use `.env` file (already in `.gitignore`)
   - Rotate tokens regularly

2. **Use environment-specific configuration**
   - Development: relaxed authentication
   - Production: strict GitHub org/team requirements

3. **Monitor access logs**
   ```bash
   docker-compose logs artifusion | grep "401\|403"
   ```

4. **Limit GitHub PAT scopes**
   - Use minimal required scopes
   - Create separate PATs for different purposes

5. **Enable HTTPS in production**
   - Deploy behind reverse proxy (Nginx, Traefik)
   - Use valid TLS certificates

---

## Support

- **Issues**: Report bugs at [GitHub Issues](https://github.com/mainuli/artifusion/issues)
- **Logs**: Check `docker-compose logs` for detailed error messages
- **Metrics**: Monitor Prometheus metrics at `http://localhost:8080/metrics`

---

**Status**: ✅ Production Ready | **Version**: 1.0.0 | **Protocol**: Maven 2/3
