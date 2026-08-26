# Release process

Keep `VERSION` at `1.0.0-rc.1` until all 53 gates pass. Run unit, integration, synthetic Docker E2E, frontend/Playwright, image builds/scans, restore, performance, static/fuzz/security review, documentation/OpenAPI checks, production config validation, clean Docker verification, and clean Git review. Push main, observe every remote workflow, fix concrete project failures for at most five cycles, and configure main protection/rulesets when API/plan permits.

Only then update to 1.0.0, changelog, image/docs references, commit, push, and create a signed `v1.0.0` tag through the controlled release process. Never create a tag to test release. If any critical gate is failed or unrun, publish only the release candidate status and precise blockers.

