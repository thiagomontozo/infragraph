# Release process

Keep `VERSION` at `1.0.0-rc.1` until every item in the explicit [53-gate production-readiness matrix](production-readiness.md#release-gate-matrix-53-gates) has recorded evidence. Run the gates in increasing cost: static/unit, integration, synthetic Docker, frontend/Playwright, recovery/performance, security/image scans, then target-environment acceptance.

Push the candidate commit, observe every required remote workflow for that exact SHA, and fix project failures before reassessing. Configure the main ruleset so CI, CodeQL, and Security cannot be bypassed by an ordinary push. Operational gates require links to dated evidence; prose claims are not evidence.

For an owner-authored pull request in this single-maintainer repository, always use the [protected merge helper](sole-maintainer-merge.md). It verifies the checks and exact head SHA, temporarily disables the approval count and `require_last_push_approval`, performs the merge, and restores the original review protection even when the merge fails. Do not edit the protection in the GitHub UI or leave the exception active between merges.

Only then update to 1.0.0, changelog, image/docs references, commit, push, and create a signed `v1.0.0` tag through the controlled release process. The manual release workflow accepts only an existing tag whose value matches `VERSION`; it is not a tag-creation mechanism. Never create a tag to test release. If any critical gate is failed or unrun, publish only the release-candidate status and precise blockers.

