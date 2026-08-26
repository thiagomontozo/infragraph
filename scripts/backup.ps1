param([Parameter(Mandatory=$true)][string]$Output)
$ErrorActionPreference='Stop'
docker compose exec -T postgres pg_dump -U infragraph -d infragraph -Fc > $Output
