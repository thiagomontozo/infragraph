$ErrorActionPreference = 'Stop'
$containers = @(docker ps -aq --filter 'label=com.infragraph.managed=true')
if ($containers.Count -gt 0) { docker rm -f $containers | Out-Null }
$networks = @(docker network ls -q --filter 'label=com.infragraph.managed=true')
foreach ($network in $networks) { docker network rm $network | Out-Null }
$volumes = @(docker volume ls -q --filter 'label=com.infragraph.managed=true' --filter 'label=com.infragraph.purpose=test')
foreach ($volume in $volumes) { docker volume rm $volume | Out-Null }
Write-Output "InfraGraph managed containers remaining: $(@(docker ps -aq --filter 'label=com.infragraph.managed=true').Count)"
