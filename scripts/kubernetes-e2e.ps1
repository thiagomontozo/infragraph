$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
$clusterName = 'infragraph-e2e'
$kindVersion = 'v0.33.0'
$kindSha256 = '4b22adaa135368c5a465d56bbd8e520cbea87272a06ca00b6078e7b81515c9fc'
$nodeImage = 'kindest/node:v1.35.8@sha256:07b2536e30b803ed61d1677a79df6115f798ce64c80f9e22f6ed45afd09323c0'
$temporaryDirectory = Join-Path ([IO.Path]::GetTempPath()) ("infragraph-kind-" + [guid]::NewGuid().ToString('N'))
$kindBinary = Join-Path $temporaryDirectory 'kind.exe'
$caFile = Join-Path $temporaryDirectory 'ca.crt'
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null
try {
  Invoke-WebRequest -UseBasicParsing -Uri "https://github.com/kubernetes-sigs/kind/releases/download/$kindVersion/kind-windows-amd64" -OutFile $kindBinary
  if ((Get-FileHash -Algorithm SHA256 -LiteralPath $kindBinary).Hash.ToLowerInvariant() -ne $kindSha256) {
    throw 'kind checksum mismatch'
  }
  & $kindBinary create cluster --name $clusterName --image $nodeImage --wait 120s
  kubectl apply -f deploy/kubernetes-collector-rbac.yaml
  kubectl apply -f internal/connectors/kubernetes/testdata/e2e.yaml
  kubectl -n infragraph-e2e rollout status deployment/infragraph-e2e --timeout=120s
  $server = kubectl config view --minify --raw -o "jsonpath={.clusters[0].cluster.server}"
  $caData = kubectl config view --minify --raw -o "jsonpath={.clusters[0].cluster.certificate-authority-data}"
  [IO.File]::WriteAllBytes($caFile, [Convert]::FromBase64String($caData))
  $token = kubectl -n infragraph-collector create token infragraph-collector --duration=10m
  $env:INFRAGRAPH_KUBERNETES_E2E = '1'
  $env:INFRAGRAPH_KUBERNETES_E2E_URL = $server
  $env:INFRAGRAPH_KUBERNETES_E2E_CA_FILE = $caFile
  $env:INFRAGRAPH_KUBERNETES_E2E_TOKEN = $token
  go test -tags=integration -count=1 -run TestKindDiscoveryWithServiceAccount ./internal/connectors/kubernetes
  if ($LASTEXITCODE) { throw "Kubernetes E2E failed with exit code $LASTEXITCODE" }
} finally {
  if (Test-Path -LiteralPath $kindBinary) {
    & $kindBinary delete cluster --name $clusterName 2>$null
  }
  Pop-Location
  $resolvedTemp = [IO.Path]::GetFullPath($temporaryDirectory)
  if ($resolvedTemp.StartsWith([IO.Path]::GetTempPath(), [StringComparison]::OrdinalIgnoreCase)) {
    Remove-Item -LiteralPath $resolvedTemp -Recurse -Force -ErrorAction SilentlyContinue
  }
}
