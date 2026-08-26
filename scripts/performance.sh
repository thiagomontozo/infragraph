#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
INFRAGRAPH_PERF=1 INFRAGRAPH_PERF_ASSETS="${INFRAGRAPH_PERF_ASSETS:-2000}" INFRAGRAPH_PERF_RELATIONSHIPS="${INFRAGRAPH_PERF_RELATIONSHIPS:-5000}" go test -count=1 -v -run TestPerformanceSmoke ./internal/graph
