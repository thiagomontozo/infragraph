# Kubernetes connector

The shipped collector supports Kubernetes metadata discovery by setting `INFRAGRAPH_CONNECTOR_TYPE=KUBERNETES`. One enrolled identity represents one connector type; changing an existing Docker identity to Kubernetes is rejected. Use a separate data directory, enrollment token, and connector identity for every cluster.

## Discovered inventory

The connector lists namespaces, nodes, pods, services, Deployments, StatefulSets, DaemonSets, ReplicaSets, ingresses, persistent volumes, and persistent volume claims. It emits a cluster asset and deterministic relationships for containment, workload ownership, pod placement, service selectors, ingress backends, StatefulSet services, and PVC/PV binding.

Cluster identity defaults to the immutable `kube-system` namespace UID. Set `INFRAGRAPH_KUBERNETES_CLUSTER_ID` only when that namespace cannot be listed; the value must remain stable for the cluster lifetime. `INFRAGRAPH_KUBERNETES_CLUSTER_NAME` is display metadata and does not control identity.

Responses are paginated and bounded to 500 items per request and 100,000 resources per run by default. Each response is capped at 10 MiB. Discovery fails explicitly rather than sending a partial successful snapshot when authorization, TLS, pagination, or limits fail.

## Credential and data minimization

The collector reads the projected ServiceAccount token and CA from files. It requires HTTPS outside loopback tests, TLS 1.2 or newer, and a bearer token. Kubernetes API requests are direct and do not inherit process proxy variables, preventing a ServiceAccount credential from being sent through an unexpected proxy.

Only allowlisted metadata and selected placement/selector fields are decoded. The connector never requests Secrets, ConfigMaps, ServiceAccounts, token requests, logs, exec, attach, port-forward, or arbitrary API paths. Unknown response fields—including `data` and `stringData`—are discarded before observations are constructed.

Apply [the example RBAC](../deploy/kubernetes-collector-rbac.yaml) to a dedicated ServiceAccount. It grants only cluster-wide `list` for the eleven enumerated resource types; there are no mutation, `get`, `watch`, secret, or subresource permissions.

## Deployment

1. Bootstrap a single-use collector enrollment token.
2. Create `infragraph-collector-enrollment` in the `infragraph-collector` namespace with a `token` key.
3. Apply `deploy/kubernetes-collector-rbac.yaml`.
4. Replace the control-plane URL and zero digest in `deploy/kubernetes-collector.example.yaml` with the approved HTTPS URL and immutable collector image digest.
5. Apply the deployment and confirm `HEALTHY` heartbeats and successful snapshots.
6. Delete the enrollment Secret after identity is persisted on the PVC. Its reference is optional for subsequent restarts.

The example deliberately uses one replica with `Recreate` and a `ReadWriteOnce` PVC because identity, monotonic sequence, and spool state must not be shared concurrently. Losing the PVC requires revocation and re-enrollment. Restrict egress to DNS, the Kubernetes API, and the InfraGraph control plane, and rotate/revoke the ServiceAccount credential after suspected collector compromise.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `INFRAGRAPH_CONNECTOR_TYPE` | `DOCKER` | Set to `KUBERNETES` before first enrollment. |
| `INFRAGRAPH_KUBERNETES_API_URL` | `https://kubernetes.default.svc` | Kubernetes API endpoint. |
| `INFRAGRAPH_KUBERNETES_TOKEN_FILE` | projected ServiceAccount token | Bearer-token file. |
| `INFRAGRAPH_KUBERNETES_CA_FILE` | projected `ca.crt` | PEM CA bundle. |
| `INFRAGRAPH_KUBERNETES_CLUSTER_ID` | derived from `kube-system` UID | Stable identity override. |
| `INFRAGRAPH_KUBERNETES_CLUSTER_NAME` | cluster ID | Display name. |
| `INFRAGRAPH_KUBERNETES_TIMEOUT` | `30s` | Per-request timeout. |
| `INFRAGRAPH_KUBERNETES_PAGE_SIZE` | `500` | API list page size, capped at 1,000. |
| `INFRAGRAPH_KUBERNETES_MAX_RESOURCES` | `100000` | Run-wide resource cap. |

## Verification

`scripts/kubernetes-e2e.sh` and `scripts/kubernetes-e2e.ps1` create an ephemeral pinned `kind` cluster, apply the exact RBAC and synthetic Deployment/Pod/Service/Secret fixtures, authenticate over TLS with the ServiceAccount token, and assert assets, relationships, and Secret exclusion. The CI job is named `kubernetes-discovery`.
