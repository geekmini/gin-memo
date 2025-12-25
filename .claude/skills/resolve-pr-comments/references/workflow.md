# Resolve PR Comments Workflow

## Arguments

```
$ARGUMENTS
```

## PR Number Resolution

Parse arguments for a PR number. If not provided, auto-detect from current branch:

**If PR number provided in arguments:**
```bash
gh pr view <PR_NUMBER> --json number,title --jq '"\(.number)|\(.title)"'
```

**If no PR number provided:**
1. Get current branch:
   ```bash
   git branch --show-current
   ```

2. Find PR for current branch:
   ```bash
   gh pr view --json number,title --jq '"\(.number)|\(.title)"'
   ```

**If no PR found, show error:**
```
Error: No PR found for current branch.

Usage:
  /resolve-pr-comments        # Auto-detect PR from branch
  /resolve-pr-comments 123    # Resolve comments on PR #123
```

## Repository Info

```bash
gh repo view --json owner,name --jq '"\(.owner.login)/\(.name)"'
```

## Phase 1: Fetch All PR Comments

### Step 1: Fetch Inline Review Comments

```bash
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments \
  --jq '[.[] | {id, path, line, original_line, body, user: .user.login, in_reply_to_id, created_at, comment_type: "inline"}]'
```

### Step 2: Fetch Review Threads

```bash
gh api graphql -f query='
  query($owner: String!, $repo: String!, $pr: Int!) {
    repository(owner: $owner, name: $repo) {
      pullRequest(number: $pr) {
        reviewThreads(first: 100) {
          nodes {
            id
            isResolved
            comments(first: 10) {
              nodes {
                id
                databaseId
                body
                author { login }
                path
                line
              }
            }
          }
        }
      }
    }
  }
' -f owner="{owner}" -f repo="{repo}" -F pr={pr_number}
```

### Step 2.5: Build Thread Mapping and Filter to Unresolved

1. Build mapping from `comment_databaseId → thread_id`
2. Skip comments whose thread has `isResolved: true`
3. Store `thread_id` for use when resolving

### Step 3: Fetch General PR Comments

```bash
gh api repos/{owner}/{repo}/issues/{pr_number}/comments \
  --jq '[.[] | {id, body, user: .user.login, created_at, comment_type: "general"}]'
```

### Processing Order

1. **First**: Process unresolved inline review comments
2. **Last**: Process general PR comments

**If no unresolved comments:**
```
No unresolved review comments found on PR #{pr_number}!
```

## Phase 2: Interactive Resolution Loop

For each comment:

### Step 1: Analysis

1. **Relevance Check:** Is this comment actionable?
2. **Severity Assessment:** Blocker / Major / Minor / Questionable

### Step 2: Presentation

**For inline comments:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Processing comment X of Y [Inline Review Comment]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**File:** path/to/file:42
**Reviewer:** @username
**Comment:** "The comment text"
**Assessment:** [Severity] - [Reasoning]
**Proposed Fix:** [Description]

[Show proposed code change]
```

### Step 3: User Decision

**For inline comments - Options:**
- **Fix** - Apply code change and reply
- **Skip** - Move to next without changes
- **Resolve** - Mark resolved without code changes

**For general comments - Options:**
- **Fix** - Apply code changes and reply
- **Skip** - Move to next without changes
- **Acknowledge** - Reply with explanation

### Handling Choices

**Fix (inline):**
1. Apply code change
2. Post reply: "Fixed in latest commit. [Automated reply]"
3. Resolve thread via GraphQL:
   ```bash
   gh api graphql -f query='
     mutation($threadId: ID!) {
       resolveReviewThread(input: {threadId: $threadId}) {
         thread { id isResolved }
       }
     }
   ' -f threadId="$thread_id"
   ```

**Resolve (inline):**
1. Post reply with reason
2. Resolve thread via GraphQL (same mutation as above)

**Fix/Acknowledge (general):**
1. Apply changes if needed
2. Post reply as new comment

### Progress Tracking

```
Progress: X of Y comments processed (F fixed, S skipped, R resolved, A acknowledged)
```

## Phase 3: Commit Changes

**If code changes made:**
```bash
git add -A
git commit -m "fix: address PR review feedback"
```

## Phase 4: Final Options

**Options:**
- **Push** - Push commit to remote
- **Done** - Exit without pushing

## Constraints

- Process comments one by one
- Always post replies before resolving threads
- Do NOT modify other users' comments
- Show clear progress indicators
