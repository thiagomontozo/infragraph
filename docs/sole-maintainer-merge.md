# Sole-maintainer protected merge

InfraGraph requires review and approval after the last push on `main`. GitHub does not allow the pull-request author or last pusher to satisfy that approval, even when that person owns the repository. When no second maintainer is available, every owner-authored merge must use the repository helper instead of manually editing branch protection.

The helper accepts only an open, non-draft pull request targeting `main`. It requires every reported pull-request check to be complete and successful, records the current review settings, temporarily sets `required_approving_review_count` to `0` and `require_last_push_approval` to `false`, merges the exact checked head SHA, and restores both original values in a PowerShell `finally` block or POSIX shell exit trap.

Disabling only `require_last_push_approval` is insufficient because GitHub still prevents an author from approving their own pull request. The helper changes both review fields for the shortest possible interval. Required status checks, strict update enforcement, administrator enforcement, conversation resolution, force-push prevention, and deletion prevention remain unchanged.

Run from an authenticated clone with a GitHub CLI identity allowed to administer branch protection:

```powershell
./scripts/merge-protected-pr.ps1 -PullRequest 123
```

```sh
./scripts/merge-protected-pr.sh 123
```

Squash is the default. Pass `-Method merge` or `-Method rebase` in PowerShell, or a second `merge`/`rebase` argument in the shell, only when the pull request requires another history policy.

After every run, verify the pull request is `MERGED` and `main` again reports the expected review count and last-push approval. GitHub records both branch-protection changes and the merge. Retain those events with the pull-request checks as release evidence.

An abrupt machine termination can prevent any client-side cleanup from running. If the helper reports a critical restoration failure or the host loses power, restore the policy before any other repository operation:

```sh
gh api --method PATCH repos/thiagomontozo/infragraph/branches/main/protection/required_pull_request_reviews \
  -F required_approving_review_count=1 \
  -F require_last_push_approval=true
```

Then confirm the complete branch protection response. A second independent reviewer remains preferable whenever one is available; this procedure is an explicit single-maintainer exception, not an approval substitute for a multi-maintainer repository.
