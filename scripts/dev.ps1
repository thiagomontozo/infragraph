$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try { docker compose up --build } finally { docker compose down --remove-orphans; Pop-Location }
