# Kubernetes connector

The Kubernetes connector lists metadata for clusters/namespaces/nodes, workloads, pods, services, ingresses, PVs, and PVCs. It excludes `Secret.data`, `Secret.stringData`, ServiceAccount tokens, and ConfigMap data. Names, namespaces, UIDs, labels, placement, selectors, and selected non-sensitive resource metadata are enough for identity and graph construction.

Apply the example RBAC in `deploy/kubernetes-collector-rbac.yaml` to a dedicated namespace/service account. It grants only `get`, `list`, and `watch` for enumerated resource types, never `cluster-admin`, mutation verbs, secrets, configmaps, token requests, exec, attach, or logs. Rotate the service-account credential and restrict API/network reachability.

