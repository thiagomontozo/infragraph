# ADR 007: No weak-identifier automatic merge

Status: Accepted (2026-08-25).

Hostname or IP equality can cross time, environment, NAT, or reuse boundaries. Only contextual immutable identifiers may auto-resolve. Weak matches create reviewable merge candidates with reasons; environment contradictions prevent automatic merge. This tolerates temporary duplicates to avoid destructive identity corruption.

