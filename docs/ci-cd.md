# CI/CD

`ci.yml` builds/tests Go with PostgreSQL and MinIO, runs race and frontend checks, validates Docker discovery and Playwright with timeouts, and uploads coverage. `security.yml` runs govulncheck, npm high audit, gitleaks, and Trivy critical gates. CodeQL scans Go and JavaScript/TypeScript; Scorecard uses its official action. Dependabot proposes Go/npm/Actions/Docker updates without automerge.

`container.yml` builds API/web/collector images on main and pull requests without exposing release credentials to untrusted code. `release.yml` runs only for `v*` tags or explicit dispatch, validates first, builds amd64/arm64 GHCR images, attaches SBOM/checksums/provenance, uses GitHub OIDC for keyless Cosign, and creates release notes. Only release workflows receive package/content/id-token write permissions.

