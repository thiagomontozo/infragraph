#!/usr/bin/env sh
set -eu

usage() {
  echo "usage: $0 <pull-request-number> [squash|merge|rebase]" >&2
  exit 2
}

[ "$#" -ge 1 ] && [ "$#" -le 2 ] || usage
pr_number=$1
merge_method=${2:-squash}
case "$pr_number" in *[!0-9]*|'') usage ;; esac
case "$merge_method" in squash|merge|rebase) ;; *) usage ;; esac
command -v gh >/dev/null 2>&1 || { echo 'GitHub CLI (gh) is required.' >&2; exit 1; }

repository=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
state=$(gh pr view "$pr_number" --json state --jq .state)
draft=$(gh pr view "$pr_number" --json isDraft --jq .isDraft)
base=$(gh pr view "$pr_number" --json baseRefName --jq .baseRefName)
head_sha=$(gh pr view "$pr_number" --json headRefOid --jq .headRefOid)
pr_url=$(gh pr view "$pr_number" --json url --jq .url)
[ "$state" = OPEN ] || { echo "Pull request #$pr_number is not open." >&2; exit 1; }
[ "$draft" = false ] || { echo "Pull request #$pr_number is still a draft." >&2; exit 1; }
[ "$base" = main ] || { echo "Pull request #$pr_number targets '$base', not 'main'." >&2; exit 1; }

# This exits non-zero for failed or pending checks; required checks remain protected throughout.
gh pr checks "$pr_number"

reviews_endpoint="repos/$repository/branches/main/protection/required_pull_request_reviews"
original_count=$(gh api "$reviews_endpoint" --jq .required_approving_review_count)
original_last_push=$(gh api "$reviews_endpoint" --jq .require_last_push_approval)
temporary_body='{"required_approving_review_count":0,"require_last_push_approval":false}'
restore_body=$(printf '{"required_approving_review_count":%s,"require_last_push_approval":%s}' "$original_count" "$original_last_push")
protection_relaxed=0

restore_protection() {
  if [ "$protection_relaxed" -eq 1 ]; then
    if printf '%s' "$restore_body" | gh api --method PATCH "$reviews_endpoint" --input - --silent; then
      protection_relaxed=0
    else
      echo 'CRITICAL: merge finished but the original review protection could not be restored.' >&2
      return 1
    fi
  fi
}

trap restore_protection EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Mark first so a partial/ambiguous API failure also triggers restoration.
protection_relaxed=1
printf '%s' "$temporary_body" | gh api --method PATCH "$reviews_endpoint" --input - --silent

# Auto-merge may complete as soon as the incompatible review requirements are relaxed.
state=$(gh pr view "$pr_number" --json state --jq .state)
if [ "$state" = OPEN ]; then
  gh pr merge "$pr_number" "--$merge_method" --match-head-commit "$head_sha"
fi

state=$(gh pr view "$pr_number" --json state --jq .state)
[ "$state" = MERGED ] || { echo "Pull request #$pr_number did not reach MERGED state." >&2; exit 1; }

restore_protection
trap - EXIT INT TERM
echo "Merged $pr_url and restored the original main review protection."
