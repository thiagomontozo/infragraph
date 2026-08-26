#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
cleanup(){ docker rm -f infragraph-test-web infragraph-test-api infragraph-test-db >/dev/null 2>&1 || true; docker network rm infragraph-e2e-network >/dev/null 2>&1 || true; }
trap cleanup EXIT
docker network create --label com.infragraph.managed=true --label com.infragraph.purpose=test infragraph-e2e-network >/dev/null
docker run -d --name infragraph-test-db --network infragraph-e2e-network --label com.infragraph.test=true --label com.infragraph.managed=true --label com.infragraph.purpose=test --memory 64m --cpus .25 alpine:3.22 sleep 600 >/dev/null
docker run -d --name infragraph-test-api --network infragraph-e2e-network --label com.infragraph.test=true --label com.infragraph.managed=true --label com.infragraph.purpose=test --label com.infragraph.depends-on=infragraph-test-db --memory 64m --cpus .25 alpine:3.22 sleep 600 >/dev/null
docker run -d --name infragraph-test-web --network infragraph-e2e-network --label com.infragraph.test=true --label com.infragraph.managed=true --label com.infragraph.purpose=test --label com.infragraph.depends-on=infragraph-test-api --memory 64m --cpus .25 alpine:3.22 sleep 600 >/dev/null
docker run --rm --label com.infragraph.managed=true --label com.infragraph.purpose=test -e INFRAGRAPH_DOCKER_E2E=1 -v /var/run/docker.sock:/var/run/docker.sock -v "$PWD:/src" -w /src golang:1.26.6-bookworm go test -count=1 -run TestSyntheticDockerTopology ./internal/connectors/docker
