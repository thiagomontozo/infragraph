# ADR 018: Organization isolation

Status: Accepted (2026-08-25).

Organization is the mandatory tenant boundary. Authenticated session or collector credential supplies it; browser/snapshot organization claims are cross-checked, never trusted. Queries and uniqueness include organization where cardinality requires it. Integration tests attempt cross-tenant access for every resource family.

