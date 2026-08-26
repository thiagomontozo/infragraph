# Performance

The default local smoke generates 2,000 synthetic assets and 5,000 relationships, then performs a depth-3 traversal capped at 500 nodes with a three-second deadline. CI may set 10,000/25,000. Integration profiling should additionally measure paginated asset list, trigram search, neighbors, reconciliation batch, and database depth-3 queries.

Record hardware, Docker/Go/PostgreSQL versions, dataset shape, cold/warm state, timings, and query plan for every report. Passing this bounded smoke is not a claim that InfraGraph supports one million assets or guarantees latency on different hardware. Production capacity requires representative load and retention data.

Observed locally on 2026-08-25 using Go 1.26.6 on Windows/amd64: the 2,000 asset / 5,000 relationship in-memory depth-3 traversal completed in approximately 0.7 ms. This single warm synthetic observation excludes PostgreSQL and network latency and is recorded only as a regression baseline.
