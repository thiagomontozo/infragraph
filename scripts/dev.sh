#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
trap 'docker compose down --remove-orphans' EXIT
docker compose up --build
