# ADR 001: Modular monolith

Status: Accepted (2026-08-25).

InfraGraph needs consistent transactions across provenance, effective state, graph, changes, findings, and audit. We deploy one Go control plane with internal modules and one separately trusted collector binary. This minimizes network failure modes and operational components while preserving interfaces for future extraction. Module coupling must occur through typed services, not handler-to-table shortcuts.

