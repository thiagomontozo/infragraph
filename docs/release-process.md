# Release process

Keep `VERSION` at `1.0.0-rc.1` until every item in the explicit [53-gate production-readiness matrix](production-readiness.md#release-gate-matrix-53-gates) has recorded evidence. Run the gates in increasing cost: static/unit, integration, synthetic Docker, frontend/Playwright, recovery/performance, security/image scans, then target-environment acceptance.

Push the candidate commit, observe every required remote workflow for that exact SHA, and fix project failures before reassessing. Configure the main ruleset so CI, CodeQL, and Security cannot be bypassed by an ordinary push. Operational gates require links to dated evidence; prose claims are not evidence.

Only then update to 1.0.0, changelog, image/docs references, commit, push, and create a signed `v1.0.0` tag through the controlled release process. The manual release workflow accepts only an existing tag whose value matches `VERSION`; it is not a tag-creation mechanism. Never create a tag to test release. If any critical gate is failed or unrun, publish only the release-candidate status and precise blockers.

