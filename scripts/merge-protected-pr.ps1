param(
    [Parameter(Mandatory = $true)]
    [ValidateRange(1, [int]::MaxValue)]
    [int]$PullRequest,

    [ValidateSet('squash', 'merge', 'rebase')]
    [string]$Method = 'squash'
)

$ErrorActionPreference = 'Stop'

function Invoke-GhText {
    param([Parameter(Mandatory = $true)][string[]]$GhArguments)

    $output = & gh @GhArguments
    if ($LASTEXITCODE -ne 0) {
        throw "gh $($GhArguments -join ' ') failed with exit code $LASTEXITCODE"
    }
    return (($output | Out-String).Trim())
}

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    throw 'GitHub CLI (gh) is required.'
}

$repository = Invoke-GhText -GhArguments @('repo', 'view', '--json', 'nameWithOwner', '--jq', '.nameWithOwner')
$pullRequestData = Invoke-GhText -GhArguments @('pr', 'view', "$PullRequest", '--json', 'state,isDraft,baseRefName,headRefOid,url') | ConvertFrom-Json
if ($pullRequestData.state -ne 'OPEN') {
    throw "Pull request #$PullRequest is not open."
}
if ($pullRequestData.isDraft) {
    throw "Pull request #$PullRequest is still a draft."
}
if ($pullRequestData.baseRefName -ne 'main') {
    throw "Pull request #$PullRequest targets '$($pullRequestData.baseRefName)', not 'main'."
}

& gh pr checks $PullRequest
if ($LASTEXITCODE -ne 0) {
    throw "Pull request #$PullRequest has failed or pending checks."
}

$reviewsEndpoint = "repos/$repository/branches/main/protection/required_pull_request_reviews"
$reviewProtection = Invoke-GhText -GhArguments @('api', $reviewsEndpoint) | ConvertFrom-Json
$restoreBody = @{
    required_approving_review_count = [int]$reviewProtection.required_approving_review_count
    require_last_push_approval       = [bool]$reviewProtection.require_last_push_approval
} | ConvertTo-Json
$temporaryBody = @{
    required_approving_review_count = 0
    require_last_push_approval       = $false
} | ConvertTo-Json
$protectionRelaxed = $false

try {
    # Mark first so a partial/ambiguous API failure also triggers restoration.
    $protectionRelaxed = $true
    $temporaryBody | & gh api --method PATCH $reviewsEndpoint --input - --silent
    if ($LASTEXITCODE -ne 0) {
        throw 'Failed to apply the temporary sole-maintainer review exception.'
    }

    # Auto-merge may complete as soon as the incompatible review requirements are relaxed.
    $state = Invoke-GhText -GhArguments @('pr', 'view', "$PullRequest", '--json', 'state', '--jq', '.state')
    if ($state -eq 'OPEN') {
        $mergeFlag = "--$Method"
        & gh pr merge $PullRequest $mergeFlag --match-head-commit $pullRequestData.headRefOid
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to merge pull request #$PullRequest."
        }
    }

    $state = Invoke-GhText -GhArguments @('pr', 'view', "$PullRequest", '--json', 'state', '--jq', '.state')
    if ($state -ne 'MERGED') {
        throw "Pull request #$PullRequest did not reach MERGED state."
    }
}
finally {
    if ($protectionRelaxed) {
        $restoreBody | & gh api --method PATCH $reviewsEndpoint --input - --silent
        if ($LASTEXITCODE -ne 0) {
            throw 'CRITICAL: merge finished but the original review protection could not be restored.'
        }
        $protectionRelaxed = $false
    }
}

Write-Output "Merged $($pullRequestData.url) and restored the original main review protection."
