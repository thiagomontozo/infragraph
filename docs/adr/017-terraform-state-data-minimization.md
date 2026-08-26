# ADR 017: Minimize Terraform state

Status: Accepted (2026-08-25).

Raw Terraform state is not persisted by default. A positive allowlist extracts addresses, provider/type, identifiers, location, and tags; outputs and sensitive/arbitrary attributes are dropped. This reduces analytic flexibility but materially limits credential leakage. New allowed fields require fixtures and threat review.

