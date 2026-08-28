#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."

cluster_name=infragraph-e2e
kind_version=v0.33.0
kind_sha256=aee6151561422756b764a4ae28e7f44cda5af5a9eead3cc9985112b1de8d8e0d
node_image=kindest/node:v1.35.8@sha256:07b2536e30b803ed61d1677a79df6115f798ce64c80f9e22f6ed45afd09323c0
temporary_directory="$(mktemp -d)"
kind_binary="$temporary_directory/kind"
ca_file="$temporary_directory/ca.crt"

cleanup() {
  "$kind_binary" delete cluster --name "$cluster_name" >/dev/null 2>&1 || true
}
trap cleanup EXIT

curl --fail --silent --show-error --location "https://github.com/kubernetes-sigs/kind/releases/download/$kind_version/kind-linux-amd64" --output "$kind_binary"
echo "$kind_sha256  $kind_binary" | sha256sum --check --status
chmod +x "$kind_binary"
"$kind_binary" create cluster --name "$cluster_name" --image "$node_image" --wait 120s

kubectl apply -f deploy/kubernetes-collector-rbac.yaml
kubectl apply -f internal/connectors/kubernetes/testdata/e2e.yaml
kubectl -n infragraph-e2e rollout status deployment/infragraph-e2e --timeout=120s

server="$(kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.server}')"
kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 --decode > "$ca_file"
token="$(kubectl -n infragraph-collector create token infragraph-collector --duration=10m)"

INFRAGRAPH_KUBERNETES_E2E=1 \
INFRAGRAPH_KUBERNETES_E2E_URL="$server" \
INFRAGRAPH_KUBERNETES_E2E_CA_FILE="$ca_file" \
INFRAGRAPH_KUBERNETES_E2E_TOKEN="$token" \
  go test -tags=integration -count=1 -run TestKindDiscoveryWithServiceAccount ./internal/connectors/kubernetes
