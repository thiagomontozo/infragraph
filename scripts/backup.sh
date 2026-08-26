#!/usr/bin/env sh
set -eu
[ "$#" -eq 1 ] || { echo 'usage: backup.sh OUTPUT' >&2; exit 2; }
docker compose exec -T postgres pg_dump -U infragraph -d infragraph -Fc > "$1"
