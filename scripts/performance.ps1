$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try { $env:INFRAGRAPH_PERF='1'; if (-not $env:INFRAGRAPH_PERF_ASSETS) {$env:INFRAGRAPH_PERF_ASSETS='2000'}; if (-not $env:INFRAGRAPH_PERF_RELATIONSHIPS) {$env:INFRAGRAPH_PERF_RELATIONSHIPS='5000'}; go test -count=1 -v -run TestPerformanceSmoke ./internal/graph }
finally { Remove-Item Env:INFRAGRAPH_PERF -ErrorAction SilentlyContinue; Pop-Location }
