# Docker connector

The Docker connector lists only containers carrying the configured scope label and derives host, container, image, network, and named-volume assets plus `RUNS_ON`, `USES_IMAGE`, `CONNECTED_TO`, and `USES_VOLUME` relationships. Synthetic dependency labels are supported for isolated acceptance fixtures. It does not read environment variables, secret contents, logs, command output, or mounted file contents and implements only Docker API `GET` inventory calls.

Docker socket access is security-sensitive: a read-only filesystem mount does not make the Docker API safe. The collector, never the API/web, receives access. Production should use a dedicated collector host and/or restricted Docker API proxy, a dedicated service identity, network egress controls, and host monitoring. There is no endpoint for users to supply raw Docker paths, API verbs, or container actions.

