# ADR 016: Exclude Docker and Kubernetes secrets

Status: Accepted (2026-08-25).

Docker environment/log/secret/mounted content and Kubernetes Secret/ServiceAccount/ConfigMap data are outside inventory purpose and may contain credentials. Connectors collect metadata only and tests assert sensitive endpoints are not called. Operators cannot override this through generic configuration in V1.

