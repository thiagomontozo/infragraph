# ADR 015: Bounded graph traversal

Status: Accepted (2026-08-25).

Every traversal specifies depth, node cap, organization, direction, and deadline and maintains a visited set. Requests exceeding configured ceilings are rejected. This prevents accidental whole-tenant rendering and graph-amplification denial of service, at the cost of explicitly truncated exploration.

