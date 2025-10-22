# Artifusion Quick Start Guide

Get Artifusion running in under 5 minutes.

## Choose Your Deployment Mode

Artifusion supports two deployment modes:

- **No Backend Auth (Default)** - Simpler setup, recommended for internal networks
- **With Backend Auth** - Defense in depth, recommended for public deployments

[Learn more about deployment modes →](DEPLOYMENT_MODES.md)

---

## Quick Start: No Backend Authentication (Recommended)

**Best for:** Internal networks, development, simple setups

### 1. Setup Environment

```bash
cd deployments/docker
cp .env.example .env
```

Edit `.env` and set at minimum:

```bash
GITHUB_PACKAGES_TOKEN=ghp_your_github_token_here
```

Optional organization/team restrictions:
```bash
GITHUB_ORG=your-organization
GITHUB_TEAMS=team1,team2
```

### 2. Start Services

```bash
docker-compose up -d
```

That's it! The default `docker-compose.yml` points to the no-auth configuration.

### 3. Verify Health

```bash
docker-compose ps
curl http://localhost:8080/health
```

### 4. Test Docker Registry

```bash
# Login with your GitHub credentials
docker login localhost:8080
Username: your-github-username
Password: ghp_your_github_token

# Pull an image (will cascade through upstreams)
docker pull localhost:8080/nginx:latest

# Tag and push a local image
docker tag my-app:latest localhost:8080/my-app:latest
docker push localhost:8080/my-app:latest
```

### 5. Test NPM Registry

```bash
# Configure npm
npm config set registry http://localhost:8080/npm/
npm config set //localhost:8080/npm/:_authToken ghp_your_github_token

# Install a package
npm install express
```

### 6. Test Maven Repository

Add to `~/.m2/settings.xml`:

```xml
<server>
  <id>artifusion</id>
  <username>your-github-username</username>
  <password>ghp_your_github_token</password>
</server>
```

Add to `pom.xml`:

```xml
<repository>
  <id>artifusion</id>
  <url>http://localhost:8080/</url>
</repository>
```

---

## Quick Start: With Backend Authentication

**Best for:** Public deployments, compliance requirements, defense in depth

### 1. Setup Environment

```bash
cd deployments/docker
cp .env.example .env
```

Edit `.env` and configure all required variables:

```bash
# GitHub authentication
GITHUB_PACKAGES_TOKEN=ghp_your_token_here
GITHUB_ORG=your-organization  # Optional

# Backend credentials
DOCKER_REGISTRY_USERNAME=artifusion
DOCKER_REGISTRY_PASSWORD=your-strong-password

REPOSILITE_ADMIN_TOKEN=$(openssl rand -hex 32)
REPOSILITE_WRITE_TOKEN=$(openssl rand -hex 32)
REPOSILITE_READ_TOKEN=$(openssl rand -hex 32)

VERDACCIO_USERNAME=artifusion
VERDACCIO_PASSWORD=your-strong-password
```

### 2. Create Docker Registry Auth File

```bash
mkdir -p config/registry-auth
docker run --rm --entrypoint htpasswd httpd:2 -Bbn artifusion "your-password" > config/registry-auth/htpasswd
```

**Important:** Use the same password as `DOCKER_REGISTRY_PASSWORD` in `.env`

### 3. Start Services with Auth

```bash
docker-compose -f docker-compose.with-auth.yml up -d
```

### 4. Verify Health

```bash
docker-compose ps
curl http://localhost:8080/health
```

### 5. Test (Same as No-Auth Mode)

From the client perspective, testing is identical! You still use your GitHub PAT:

```bash
# Docker
docker login localhost:8080
Username: your-github-username
Password: ghp_your_github_token

# NPM
npm config set //localhost:8080/npm/:_authToken ghp_your_github_token

# Maven - same as above
```

Artifusion handles backend authentication transparently.

---

## Switching Deployment Modes

### Switch to No-Auth Mode

```bash
# Method 1: Use default symlink
docker-compose down
docker-compose up -d

# Method 2: Explicit file
docker-compose -f docker-compose.no-auth.yml up -d
```

### Switch to With-Auth Mode

```bash
# Ensure credentials are set up first!
docker-compose down
docker-compose -f docker-compose.with-auth.yml up -d
```

---

## Common Commands

```bash
# View logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f artifusion

# Restart services
docker-compose restart

# Stop services
docker-compose down

# Stop and remove all data (WARNING!)
docker-compose down -v

# Check health
curl http://localhost:8080/health
curl http://localhost:8080/ready

# View metrics
curl http://localhost:8080/metrics
```

---

## File Structure Overview

```
deployments/docker/
├── docker-compose.yml              # Symlink → docker-compose.no-auth.yml
├── docker-compose.no-auth.yml      # No backend auth (default)
├── docker-compose.with-auth.yml    # With backend auth
├── docker-compose.original.yml     # Original reference
│
├── .env.example                    # Environment template
├── .env                            # Your configuration (create from .env.example)
│
├── config/
│   ├── artifusion.yaml            # Main Artifusion config
│   ├── oci-registry-upstream.yaml # OCI upstream registries
│   │
│   ├── verdaccio.yaml             # Verdaccio with auth
│   ├── verdaccio-no-auth.yaml     # Verdaccio without auth
│   │
│   ├── reposilite.cdn/            # Reposilite with auth
│   │   └── configuration.cdn
│   └── reposilite-no-auth.cdn/    # Reposilite without auth
│       └── configuration.cdn
│
└── Documentation:
    ├── QUICKSTART.md              # This file
    ├── README.md                  # Full deployment guide
    ├── DEPLOYMENT_MODES.md        # Auth vs no-auth comparison
    ├── AUTHENTICATION_SETUP.md    # Detailed auth setup
    ├── TESTING.md                 # Testing procedures
    ├── MAVEN_SETUP.md             # Maven-specific guide
    └── NPM-SETUP.md               # NPM-specific guide
```

---

## Troubleshooting Quick Fixes

### Can't login to Docker registry

```bash
# Check Artifusion is running
docker-compose ps

# Check logs for auth errors
docker-compose logs artifusion | grep -i auth

# Verify your GitHub token is valid
curl -H "Authorization: Bearer ghp_xxx" https://api.github.com/user
```

### Image pull returns 404

```bash
# Check if image exists in upstreams
docker pull docker.io/nginx:latest

# Check Artifusion logs to see which upstreams were tried
docker-compose logs artifusion | grep -i upstream
```

### NPM install fails

```bash
# Verify npm registry is set
npm config get registry

# Verify auth token is set
npm config get //localhost:8080/npm/:_authToken

# Test Verdaccio directly
curl http://localhost:4873/-/ping
```

### Maven deploy fails

```bash
# Check Reposilite is running
curl http://localhost:8081/api/maven/details/releases

# Verify settings.xml has correct credentials
cat ~/.m2/settings.xml

# Check Artifusion logs
docker-compose logs artifusion | grep -i maven
```

---

## Next Steps

- [Read the full deployment guide](README.md)
- [Learn about authentication setup](AUTHENTICATION_SETUP.md)
- [Understand deployment modes](DEPLOYMENT_MODES.md)
- [Test your deployment thoroughly](TESTING.md)
- [Configure Maven clients](MAVEN_SETUP.md)
- [Configure NPM clients](NPM-SETUP.md)

---

## Getting Help

1. Check service health: `docker-compose ps`
2. View logs: `docker-compose logs -f`
3. Review configuration: `docker-compose config`
4. Read the [Troubleshooting](README.md#troubleshooting) section
5. Check the [GitHub repository](https://github.com/yourorg/artifusion) for issues

---

## Production Deployment

For production deployments:

1. ✅ Use **with-auth mode** for defense in depth
2. ✅ Deploy behind **HTTPS** (nginx/Traefik with TLS)
3. ✅ Use **secrets management** (Vault, AWS Secrets Manager)
4. ✅ Enable **audit logging**
5. ✅ Set up **monitoring** (Prometheus, Grafana)
6. ✅ Configure **backups** for volumes
7. ✅ Use **specific version tags** (not `latest`)
8. ✅ Implement **disaster recovery** procedures

See [Production Considerations](README.md#production-considerations) for details.
