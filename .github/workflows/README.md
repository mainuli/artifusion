# Release Workflow

Automated release workflow for building and publishing Docker images and Helm charts to ghcr.io.

## Trigger

Push a git tag with `v` prefix:
```bash
git tag v1.0.0
git push origin v1.0.0
```

## What It Does

1. **Test** - Runs test suite with race detection
2. **Build** - Multi-arch Docker image (amd64, arm64)
3. **Scan** - Trivy vulnerability scanning (fails on HIGH/CRITICAL)
4. **Sign** - Cosign keyless signing
5. **Publish** - Push to ghcr.io with SLSA provenance
6. **Helm** - Package and push chart to ghcr.io/charts
7. **Release** - Create GitHub release with artifacts

## Security Features

- ✅ Trivy vulnerability scanning
- ✅ Cosign image signing
- ✅ SLSA Build Level 3 provenance
- ✅ SBOM generation (SPDX)
- ✅ Manual approval gate

## Setup Required

### 1. Production Environment (Manual Approval)
```
Settings → Environments → New Environment
Name: production
Add required reviewers
```

### 2. Slack Notifications (Optional)
```bash
# Add webhook URL to secrets
gh secret set SLACK_WEBHOOK_URL --body "https://hooks.slack.com/..."

# Uncomment lines 374-393 in release.yml
```

## Artifacts Published

**Docker Image**: `ghcr.io/mainuli/artifusion:1.0.0`, `latest`
**Helm Chart**: `oci://ghcr.io/mainuli/charts/artifusion:1.0.0`

## Verify Image Signature

```bash
cosign verify ghcr.io/mainuli/artifusion@sha256:... \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  --certificate-identity-regexp="^https://github.com/mainuli/artifusion/"
```

## Rollback

1. Delete GitHub release and git tag
2. Delete image tags from ghcr.io (Packages UI)
3. Delete Helm chart version from ghcr.io (Packages UI)
4. Run `helm rollback <release-name>` on clusters
