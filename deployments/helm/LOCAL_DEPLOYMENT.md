# Local Kubernetes Deployment on lvh.me

This guide documents the successful deployment of Artifusion to your local Kubernetes cluster using the lvh.me domain.

## Deployment Summary

**Status**: ✅ Successfully Deployed

**Date**: October 20, 2025

**Cluster**: OrbStack (local Kubernetes)

**Domain**: artifacts.lvh.me

**Routing Mode**: Path-based routing on single domain

## Deployed Components

All components are running in the `artifusion` namespace:

```bash
kubectl get pods -n artifusion
```

### Pods:
- `artifusion-artifusion` - Main reverse proxy (1 replica)
- `artifusion-oci-registry-0` - OCI pull-through cache (GHCR + Docker Hub)
- `artifusion-registry-0` - Local hosted OCI registry (for push operations)
- `artifusion-reposilite-0` - Maven repository manager
- `artifusion-verdaccio-0` - NPM registry cache

### Services:
- `artifusion-artifusion` - Main service (ClusterIP:8080)
- `artifusion-oci-registry` - OCI cache service
- `artifusion-registry` - Hosted registry service
- `artifusion-reposilite` - Maven service
- `artifusion-verdaccio` - NPM service

### Ingress:
- **Host**: artifacts.lvh.me
- **Class**: nginx
- **TLS**: Disabled (local development)
- **IP**: 192.168.139.2

## Protocol Endpoints

All protocols are accessible via path-based routing on artifacts.lvh.me:

### 1. OCI/Docker Registry
- **Endpoint**: `http://artifacts.lvh.me/v2/`
- **Path**: `/v2` (fixed per OCI Distribution Spec)
- **Authentication**: Required (GitHub PAT)

**Usage**:
```bash
# Login with GitHub credentials
docker login artifacts.lvh.me -u <github-username> -p <github-pat>

# Pull image (cascades through GHCR, Docker Hub, local cache)
docker pull artifacts.lvh.me/library/nginx:latest

# Tag and push to local hosted registry
docker tag myimage:latest artifacts.lvh.me/myorg/myimage:latest
docker push artifacts.lvh.me/myorg/myimage:latest
```

### 2. Maven Repository
- **Endpoint**: `http://artifacts.lvh.me/maven/`
- **Path Prefix**: `/maven`
- **Authentication**: Required (GitHub PAT)

**Usage** (settings.xml):
```xml
<settings>
  <servers>
    <server>
      <id>artifusion</id>
      <username>your-github-username</username>
      <password>ghp_your_github_pat</password>
    </server>
  </servers>

  <mirrors>
    <mirror>
      <id>artifusion</id>
      <name>Artifusion Maven</name>
      <url>http://artifacts.lvh.me/maven/releases</url>
      <mirrorOf>central</mirrorOf>
    </mirror>
  </mirrors>
</settings>
```

### 3. NPM Registry
- **Endpoint**: `http://artifacts.lvh.me/npm/`
- **Path Prefix**: `/npm`
- **Authentication**: Required (GitHub PAT)

**Usage**:
```bash
# Configure npm to use Artifusion
npm config set registry http://artifacts.lvh.me/npm/

# Login with GitHub credentials
npm login --registry=http://artifacts.lvh.me/npm/
# Username: your-github-username
# Password: ghp_your_github_pat
# Email: your-email@example.com

# Install packages (cached from npmjs.org)
npm install express
```

## Infrastructure Endpoints

### Health Check
```bash
curl http://artifacts.lvh.me/health
```

**Response**:
```json
{
  "status": "healthy",
  "version": "aa343d0",
  "uptime": "1m7.52621403s",
  "time": "2025-10-20T18:06:47.244668446Z"
}
```

### Readiness Check
```bash
curl http://artifacts.lvh.me/ready
```

**Response**:
```json
{
  "status": "ready",
  "checks": {
    "github_api": "healthy"
  },
  "time": "2025-10-20T18:06:50.894241296Z"
}
```

### Prometheus Metrics
```bash
curl http://artifacts.lvh.me/metrics
```

## Configuration

The deployment uses the custom values file: `values-local-lvh.yaml`

**Key Configuration**:
- **GitHub Org**: Not set (allows any GitHub user)
- **Log Level**: debug
- **Log Format**: console
- **Rate Limiting**: 1000 req/sec global, 100 req/sec per-user
- **Storage**: 5Gi per backend (OCI cache, Registry, Maven, NPM)
- **Resources**: Optimized for local development (100-500m CPU, 128Mi-1Gi RAM)

## Managing the Deployment

### View Logs
```bash
# Artifusion main proxy logs
kubectl logs -n artifusion deployment/artifusion-artifusion -f

# Backend logs
kubectl logs -n artifusion artifusion-oci-registry-0 -f
kubectl logs -n artifusion artifusion-registry-0 -f
kubectl logs -n artifusion artifusion-reposilite-0 -f
kubectl logs -n artifusion artifusion-verdaccio-0 -f
```

### Check Status
```bash
# All pods
kubectl get pods -n artifusion

# Ingress
kubectl get ingress -n artifusion

# Services
kubectl get svc -n artifusion

# Persistent Volume Claims
kubectl get pvc -n artifusion
```

### Access Metrics Dashboard
If you have Prometheus/Grafana installed:
```bash
# Port forward to Artifusion metrics
kubectl port-forward -n artifusion svc/artifusion-artifusion 9090:8080

# Access metrics at: http://localhost:9090/metrics
```

## Persistent Storage

All backend services use persistent storage to cache artifacts:

```bash
kubectl get pvc -n artifusion
```

**PVCs**:
- `data-artifusion-oci-registry-0` - OCI cache (5Gi)
- `data-artifusion-registry-0` - Hosted registry (5Gi)
- `data-artifusion-reposilite-0` - Maven cache (5Gi)
- `data-artifusion-verdaccio-0` - NPM cache (5Gi)

**Note**: PVCs are configured to be RETAINED on helm uninstall, preserving cached data.

## Upgrading the Deployment

To update configuration:

```bash
# Edit values-local-lvh.yaml
# Then upgrade the release
helm upgrade artifusion ./deployments/helm/artifusion \
  -f ./deployments/helm/artifusion/values-local-lvh.yaml \
  -n artifusion
```

To rebuild and redeploy with new code:

```bash
# Rebuild Docker image
make docker-build

# Restart deployment to pull new image
kubectl rollout restart deployment/artifusion-artifusion -n artifusion
```

## Uninstalling

```bash
# Uninstall Helm release (keeps PVCs)
helm uninstall artifusion -n artifusion

# Delete namespace (WARNING: deletes PVCs and all cached data)
kubectl delete namespace artifusion
```

## Troubleshooting

### 404 Errors on Maven/NPM
The protocol detectors require specific request patterns. Simply accessing `/maven/` or `/npm/` with curl won't match any protocol. Use actual Maven/NPM clients or test with protocol-specific endpoints:

```bash
# Maven - test with metadata file
curl http://artifacts.lvh.me/maven/maven-metadata.xml

# NPM - test with ping endpoint
curl http://artifacts.lvh.me/npm/-/ping
```

### Authentication Issues
All protocols require GitHub PAT authentication. Get a token from:
https://github.com/settings/tokens

Required scopes: `read:packages`, `read:user`

### DNS Resolution
If `artifacts.lvh.me` doesn't resolve:
```bash
# Check if it resolves to localhost
ping artifacts.lvh.me

# If not, add to /etc/hosts:
echo "127.0.0.1 artifacts.lvh.me" | sudo tee -a /etc/hosts
```

### Ingress Not Working
```bash
# Check ingress controller
kubectl get pods -n ingress-nginx

# Check ingress resource
kubectl describe ingress -n artifusion artifusion
```

### Pod CrashLoopBackOff
```bash
# Check pod logs
kubectl logs -n artifusion <pod-name>

# Check pod events
kubectl describe pod -n artifusion <pod-name>
```

## Testing the Deployment

Run Helm tests:
```bash
helm test artifusion -n artifusion
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     artifacts.lvh.me                        │
│                  (Nginx Ingress Controller)                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Artifusion Reverse Proxy                    │
│  ┌──────────┐  ┌───────────┐  ┌──────────┐  ┌──────────┐  │
│  │   /v2/   │  │  /maven/  │  │  /npm/   │  │ /health  │  │
│  │   OCI    │  │   Maven   │  │   NPM    │  │ /metrics │  │
│  └────┬─────┘  └─────┬─────┘  └────┬─────┘  └──────────┘  │
└───────┼──────────────┼──────────────┼──────────────────────┘
        │              │              │
        │              │              │
   ┌────▼────┐    ┌────▼────┐   ┌────▼────┐
   │  OCI    │    │ Maven   │   │   NPM   │
   │Registry │    │Reposilite│  │Verdaccio│
   │ (Cache) │    │         │   │         │
   └────┬────┘    └────┬────┘   └────┬────┘
        │              │              │
   ┌────▼────┐         │              │
   │Registry │         │              │
   │(Hosted) │         │              │
   └─────────┘         │              │
        │              │              │
   ┌────▼──────────────▼──────────────▼────┐
   │      Persistent Volume Claims          │
   │  (Cached Artifacts & Repository Data)  │
   └────────────────────────────────────────┘
```

## Next Steps

1. **Configure GitHub Organization** (optional):
   ```bash
   helm upgrade artifusion ./deployments/helm/artifusion \
     -f ./deployments/helm/artifusion/values-local-lvh.yaml \
     --set artifusion.config.github.required_org="your-org" \
     -n artifusion
   ```

2. **Add GitHub/Docker Hub Credentials** (to enable GitHub Packages and avoid rate limits):
   ```bash
   helm upgrade artifusion ./deployments/helm/artifusion \
     -f ./deployments/helm/artifusion/values-local-lvh.yaml \
     --set secrets.github.username="your-username" \
     --set secrets.github.token="ghp_xxx" \
     --set secrets.dockerhub.username="your-username" \
     --set secrets.dockerhub.token="dckr_pat_xxx" \
     -n artifusion
   ```

3. **Enable Autoscaling** (for production):
   ```bash
   # Edit values-local-lvh.yaml:
   # artifusion.autoscaling.enabled: true
   # artifusion.autoscaling.minReplicas: 2
   # artifusion.autoscaling.maxReplicas: 10
   ```

4. **Add Monitoring**:
   - Install Prometheus Operator
   - Enable ServiceMonitor: `artifusion.serviceMonitor.enabled: true`
   - Create Grafana dashboards for Artifusion metrics

## References

- **Helm Chart**: `/deployments/helm/artifusion`
- **Values File**: `/deployments/helm/artifusion/values-local-lvh.yaml`
- **Docker Compose**: `/deployments/docker/` (for local testing without K8s)
- **Documentation**: `/docs/`
- **Configuration Example**: `/config/config.example.yaml`
