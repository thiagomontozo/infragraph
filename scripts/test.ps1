$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
$frontendImage = "infragraph-frontend-test:$PID"
try {
	$unformatted = @(gofmt -l ./cmd ./internal)
	if ($unformatted.Count -gt 0) { throw "Go files require gofmt: $($unformatted -join ', ')" }
  go test ./cmd/... ./internal/...
	$buildArgs = @('build', '--target', 'test', '-f', 'web/Dockerfile', '-t', $frontendImage)
  if ($env:INFRAGRAPH_NPM_CA_FILE) {
		$buildArgs += @('--secret', "id=npm_ca,src=$($env:INFRAGRAPH_NPM_CA_FILE)")
  }
	$buildArgs += 'web'
	docker @buildArgs
	if ($LASTEXITCODE -ne 0) { throw "frontend test image build failed" }
} finally {
	docker image rm -f $frontendImage 2>$null | Out-Null
  Pop-Location
}
