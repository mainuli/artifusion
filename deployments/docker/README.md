# Artifusion Docker Deployment

Complete Docker Compose stack for running Artifusion with all backend services (Docker Registry, Maven/Reposilite, NPM/Verdaccio).

## Quick Start

**Get started in 5 minutes:**

```bash
# 1. Setup environment
cp .env.example .env
# Edit .env and set GITHUB_PACKAGES_TOKEN

# 2. Start services
docker-compose up -d

# 3. Test Docker login
docker login localhost:8080
# Username: your-github-username
# Password: ghp_your_github_token
```

**[→ Full Quick Start Guide](docs/QUICKSTART.md)**

---

## Deployment Modes

Choose between two deployment configurations:

### 1. No Backend Authentication (Default)
- **File:** `docker-compose.no-auth.yml` (default)
- Backend services run without auth, protected by Artifusion
- Simpler setup, fewer credentials
- **Best for:** Internal networks, development

### 2. With Backend Authentication
- **File:** `docker-compose.with-auth.yml`
- Backend services have their own authentication (defense in depth)
- Requires additional setup (htpasswd, tokens)
- **Best for:** Public deployments, compliance requirements

**[→ Learn About Deployment Modes](docs/DEPLOYMENT_MODES.md)**

---

## File Structure

```
deployments/docker/
├── docker-compose.yml              # Default (symlink → no-auth)
├── docker-compose.no-auth.yml      # No backend authentication
├── docker-compose.with-auth.yml    # With backend authentication
│
├── .env.example                    # Environment template
├── .env                            # Your configuration (gitignored)
│
├── config/                         # Service configurations
│   ├── artifusion.yaml
│   ├── verdaccio.yaml
│   ├── verdaccio-no-auth.yaml
│   ├── reposilite.cdn/
│   └── reposilite-no-auth.cdn/
│
├── auth/                           # Authentication files
├── scripts/                        # Utility scripts
│
└── docs/                           # Documentation
    ├── QUICKSTART.md               # 5-minute quick start
    ├── DEPLOYMENT_MODES.md         # Auth modes comparison
    ├── README.md                   # Full deployment guide
    ├── AUTHENTICATION_SETUP.md     # Detailed auth setup
    ├── TESTING.md                  # Testing procedures
    ├── MAVEN_SETUP.md              # Maven configuration
    └── NPM-SETUP.md                # NPM configuration
```

---

## Architecture

```
Client (docker/npm/maven)
         ↓
    Artifusion (Port 8080)
    GitHub PAT Auth
         ↓
    ┌────┴────┬────────┐
    ↓         ↓        ↓
OCI Ops   Maven    NPM
    ↓         ↓        ↓
registry  reposilite verdaccio
```

---

## Common Commands

```bash
# Start services
docker-compose up -d

# Start with authentication mode
docker-compose -f docker-compose.with-auth.yml up -d

# View logs
docker-compose logs -f

# Check health
curl http://localhost:8080/health

# Stop services
docker-compose down

# View status
docker-compose ps
```

---

## Documentation

| Guide | Description |
|-------|-------------|
| [Quick Start](docs/QUICKSTART.md) | Get running in 5 minutes |
| [Environment Variables](docs/ENVIRONMENT_VARIABLES.md) | Complete env var reference |
| [Deployment Modes](docs/DEPLOYMENT_MODES.md) | Auth vs no-auth comparison |
| [Full Guide](docs/README.md) | Complete deployment documentation |
| [Authentication](docs/AUTHENTICATION_SETUP.md) | Detailed auth configuration |
| [Testing](docs/TESTING.md) | Test procedures and verification |
| [Maven Setup](docs/MAVEN_SETUP.md) | Maven client configuration |
| [NPM Setup](docs/NPM-SETUP.md) | NPM client configuration |

---

## Support

- Check service health: `docker-compose ps`
- View logs: `docker-compose logs -f`
- Read the [troubleshooting guide](docs/README.md#troubleshooting)
- Review the [full documentation](docs/README.md)

---

## What You Get

This deployment includes:

- **Artifusion** - GitHub PAT authentication proxy
- **OCI Registry** - Multi-upstream Docker image cache
- **Docker Registry** - Local image storage
- **Reposilite** - Maven repository with GitHub Packages proxy
- **Verdaccio** - NPM registry with npmjs.org proxy

All integrated and ready to use with a single GitHub PAT for authentication.
