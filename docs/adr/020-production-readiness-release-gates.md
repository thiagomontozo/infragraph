# ADR 020: Production-readiness release gates

Status: Accepted (2026-08-25).

Version stays `1.0.0-rc.1` and status Production Candidate unless all 53 documented gates—including remote CI, recovery, scanners, cleanup, clean Git, and successful push—pass. Unrun equals not passed. This prevents documentation or marketing from outrunning evidence; blockers are listed precisely rather than waived silently.

