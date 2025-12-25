# CI Code Review Workflow

## Arguments

```
$ARGUMENTS
```

## PR Number Resolution

Parse arguments for a PR number. If not provided, auto-detect from current branch:

1. Get current branch:
   ```bash
   git branch --show-current
   ```

2. Find PR for current branch:
   ```bash
   gh pr view --json number --jq '.number'
   ```

3. If no PR found, show error:
   ```
   Error: No PR found for current branch.

   Usage:
     /review-pr-ci        # Auto-detect PR from branch
     /review-pr-ci 123    # Review PR #123
   ```

## Repository Info

Get owner and repo from the current repository:
```bash
gh repo view --json owner,name --jq '"\(.owner.login)/\(.name)"'
```

---

## Error Handling

All GitHub API calls should implement basic error handling:

1. **Rate Limiting:** If rate limited (HTTP 403 with `X-RateLimit-Remaining: 0`):
   - Log the error with reset time
   - Exit gracefully: "GitHub API rate limit reached. Review will retry on next push."

2. **Permission Errors:** If bot lacks permissions (HTTP 403/401):
   - Log the specific permission needed
   - Post a summary comment (if possible) indicating permission issue
   - Exit with error code

3. **Network Failures:** Retry once with 5-second delay, then fail

4. **PR State Changes:** If PR is closed/merged during execution:
   - Exit gracefully: "PR was closed or merged during review"

5. **GraphQL Errors:** Check response for `errors` field:
   - Log the GraphQL error message
   - Fall back to REST API if available, otherwise skip that operation

**Critical vs Non-Critical Operations:**
- **Critical** (block execution): Fetching PR diff, posting summary comment
- **Non-Critical** (log and continue): Individual inline comments, orphan cleanup

---

## Phase 1: Code Review Analysis

### Step 1.1: Read Review Guidelines

Read and apply the review criteria:
1. Read `docs/code-review-guideline.md` for detailed review criteria
2. Reference `CLAUDE.md` for project standards (if exists)

### Step 1.2: Execute Review

1. Use `gh pr view` to understand PR context and description
2. Use `gh pr diff` to see the actual changes
3. Read relevant source files for deeper context using `Read`, `Grep`, `Glob`
4. Evaluate against focus areas defined in your guideline

### Step 1.3: Classify Issues

For each issue found, classify as:

**CRITICAL** (will be posted as inline comments):
- Bugs that will cause failures
- Security vulnerabilities
- Code that definitively worsens code health
- Violations of architectural principles
- Anti-patterns specific to your tech stack (defined in guideline)

**SUGGESTIONS** (will go to summary table only, NOT inline comments):
- Minor style inconsistencies
- Alternative approaches worth considering
- Educational comments about best practices
- Optional refactoring ideas

Store findings in two lists:
- `critical_issues`: Array of `{path, line, body}`
- `suggestions`: Array of `{path, line, description}`

---

## Phase 2: State Discovery

### Step 2.1: Fetch Existing Bot Comments

Fetch all existing inline comments from the bot (include body for similarity check):

```bash
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments \
  --jq '[.[] | select(.user.login == "github-actions[bot]" or .user.login == "claude[bot]") | {id, path, line, original_line, body}]'
```

### Step 2.2: Fetch Review Thread Resolution Status

Query GraphQL to get which threads are resolved:

```bash
gh api graphql -f query='
  query($owner: String!, $repo: String!, $pr: Int!) {
    repository(owner: $owner, name: $repo) {
      pullRequest(number: $pr) {
        reviewThreads(first: 100) {
          pageInfo { hasNextPage }
          nodes {
            id
            isResolved
            comments(first: 1) {
              nodes {
                databaseId
                path
                line
              }
            }
          }
        }
      }
    }
  }
' -f owner="{owner}" -f repo="{repo}" -F pr={pr_number} \
  --jq '.data.repository.pullRequest.reviewThreads'
```

**Pagination check:** If `pageInfo.hasNextPage` is true, add a warning in the summary.

Build a mapping: `comment_databaseId → {thread_id, isResolved}`

### Step 2.3: Filter to Unresolved Comments

From the existing bot comments (Step 2.1), filter to only those where:
- The corresponding thread has `isResolved: false`
- OR no matching thread found (treat as unresolved)

Store as `unresolved_comments`: Array of `{id, path, line, body, thread_id}`

### Step 2.4: Get Current PR Diff Files

Get the list of files currently in the PR diff:

```bash
gh pr diff {pr_number} --name-only
```

Store as `diff_files`: Array of file paths currently in the PR.

---

## Phase 3: Semantic Deduplication (MANDATORY)

**CRITICAL:** This phase MUST be executed rigorously. Skipping or rushing this phase will result in duplicate comments.

For each CRITICAL issue you want to post, check if a similar unresolved comment already exists.

### Step 3.1: Find Existing Comments at Same Location

For each critical issue at `{path, line}`:

1. Find ALL unresolved comments where:
   - Same `path` AND
   - `line` or `original_line` is within **±5 lines** of the issue location
   - **Note:** GitHub sets `line: null` when the commented code is no longer in the diff. Use `original_line` for proximity checks.

2. If no matching comments at this location → Mark as `POST_NEW`

3. If one or more existing comments found → Proceed to Step 3.2

### Step 3.2: LLM Semantic Similarity Check

For each existing unresolved comment at the same location, perform semantic analysis:

**Compare the existing comment body with your new issue. Ask yourself:**
- Are they addressing the SAME underlying code problem?
- Would fixing one issue also fix the other?
- Is this the same anti-pattern or bug, even if worded differently?

**Decision:**
- If ANY matching comment is **SIMILAR** → `SKIP` - Do not post
- If ANY matching comment is **EVOLVED** → `UPDATE` - Update existing comment
- If ALL matching comments are **DIFFERENT** → `POST_NEW`

### Step 3.3: Output Deduplication Decision Table (MANDATORY)

**BEFORE proceeding to Phase 4, you MUST output a decision table:**

```
Deduplication Analysis:
| New Issue | Line | Existing Comment | Line | Semantic Match | Decision |
|-----------|------|------------------|------|----------------|----------|
| Issue A   | 26   | "Similar..."     | 24   | SIMILAR        | SKIP     |
| Issue B   | 43   | (none nearby)    | -    | -              | POST_NEW |
```

This table is MANDATORY.

Store decisions for each critical issue: `{path, line, action: POST_NEW | SKIP | UPDATE, existing_comment_id?}`

---

## Phase 4: Comment Reconciliation

**PREREQUISITE:** Phase 3's Deduplication Decision Table MUST be completed before this phase.

### Step 4.1: Post New Critical Issues

For each critical issue marked `POST_NEW` in the decision table:

Create a new inline comment:
```
mcp__github_inline_comment__create_inline_comment
```

### Step 4.2: Update Evolved Issues

For each critical issue marked `UPDATE`:

```bash
gh api repos/{owner}/{repo}/pulls/comments/{existing_comment_id} -X PATCH -f body="<updated comment body>"
```

### Step 4.3: Skip Duplicates

For each critical issue marked `SKIP`:
- Log: "Skipped duplicate: {path}:{line} - similar issue already flagged"
- Do NOT post any comment

---

## Phase 5: State Synchronization

### Step 5.1: Notify Fixed Issues (User Resolves)

When an issue appears to be fixed, post an informative reply but **do NOT auto-resolve** the thread.

**IMPORTANT:** Group comments by unique issue first to avoid duplicate replies.

For each **unique issue group**:

1. **Check if issue still exists in current review findings**

2. **If issue appears FIXED**:
   - Post ONE reply to the FIRST comment:
     ```bash
     gh api repos/{owner}/{repo}/pulls/{pr_number}/comments \
       -X POST \
       -f body="✅ This issue appears to be addressed. Please verify and resolve this thread." \
       -F in_reply_to={first_comment_id}
     ```
   - **Do NOT auto-resolve** - let the user verify and resolve manually

3. **If issue PERSISTS:** Leave comments as-is

### Step 5.2: Delete Orphaned Comments

For each unresolved bot comment:

1. **Check if file still in PR diff**

2. **If file NOT in diff**:
   ```bash
   gh api repos/{owner}/{repo}/pulls/comments/{comment_id} -X DELETE
   ```
   Log: "Deleted orphaned comment: {path}:{line} - file no longer in PR"

3. **If file still in diff:** Keep the comment

---

## Phase 6: Summary Comment

### Step 6.1: Find Existing Summary Comment

```bash
gh api repos/{owner}/{repo}/issues/{pr_number}/comments \
  --jq '[.[] | select((.user.login == "github-actions[bot]" or .user.login == "claude[bot]") and (.body | contains("## Code Review Summary"))) | .id] | first'
```

### Step 6.2: Update or Create Summary

**If comment ID found:** Update existing comment
**If no comment found:** Create new comment

### Summary Comment Format

```markdown
## Code Review Summary

### Overview
<Brief description of what the PR does>

### Critical Issues
<If critical issues were posted as inline comments:>
**{N} critical issue(s) posted as inline comments.** Please review and address each one.

<If no critical issues:>
**None found.** The changes look good.

### Suggestions

| File | Line | Issue |
|------|------|-------|
| `path/to/file` | 42 | Description |

### What's Good
<Acknowledge positive aspects>

### Review Actions
- Critical issues posted: {count}
- Duplicates skipped: {count}
- Fixed issues notified: {count}
- Orphaned comments deleted: {count}

### Files Reviewed

| File | Status |
|------|--------|
| `path/to/file` | Summary |

---
**Review Cost**
- Total cost: $X.XX

Generated with [Claude Code](https://claude.com/claude-code)
```

---

## Constraints

- **CI-only**: This skill MUST only run in CI environments
- **Critical issues only as inline**: Suggestions go to summary table
- **Semantic deduplication is MANDATORY**
- **Fixed issue notification**: Post reply, do NOT auto-resolve
- **Smart orphan cleanup**: Only delete when file removed from PR
- Only ONE summary comment per PR
- **Pagination limits**: GraphQL queries fetch up to 100 review threads

---

## Edge Cases

### Clean PR (No Issues)
- Still post a summary comment
- Critical Issues: "**None found.** The changes look good."

### Zero Diff Files
- Delete ALL existing bot inline comments
- Post summary noting: "PR has no file changes to review."

### Bot Identity
Check for both `github-actions[bot]` and `claude[bot]`.
