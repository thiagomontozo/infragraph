# ADR 003: Collector separated from control plane

Status: Accepted (2026-08-25).

Infrastructure credentials and Docker/Kubernetes access have a materially different trust profile from user APIs. `infragraph-collector` is a separate outbound process; the control plane never mounts Docker socket or opens an administrative collector channel. This adds enrollment/deployment work but sharply limits blast radius and preserves network segmentation.

