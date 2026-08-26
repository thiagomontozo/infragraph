#!/usr/bin/env sh
set -eu
containers=$(docker ps -aq --filter label=com.infragraph.managed=true)
[ -z "$containers" ] || docker rm -f $containers >/dev/null
for n in $(docker network ls -q --filter label=com.infragraph.managed=true); do docker network rm "$n" >/dev/null; done
for v in $(docker volume ls -q --filter label=com.infragraph.managed=true --filter label=com.infragraph.purpose=test); do docker volume rm "$v" >/dev/null; done
