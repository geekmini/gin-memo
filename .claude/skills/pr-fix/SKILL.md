---
name: Review Comments Fixer
description: Use this skill when the user asks to "fix review comments", "address review feedback", "fix PR comments", "handle review issues", or after a code review (local or remote) identifies issues that need to be addressed.
version: 1.1.0
---

# Review Comments Fixer

Address review comments from local code review (agents) or remote PR review (GitHub).

## When This Skill Activates

- After local code review (Phase 9 of /dev) identifies issues
- User says "fix review comments" or "address review feedback"
- User says "fix PR comments" or "handle PR comments"
- After receiving PR review notification from GitHub

---

## Modes

| Mode       | Source                                   | Trigger                                 |
| ---------- | ---------------------------------------- | --------------------------------------- |
| **Local**  | `feature-dev:code-reviewer` agent output | After Phase 9 local review              |
| **Remote** | GitHub PR comments                       | User manually activates after PR review |

---

## Mode 1: Local Review (from code-reviewer agent)

### Input Format

The code-reviewer agent produces output like:

```markdown
## Code Review Results

### Issues Found
- [Issue 1]: [severity] - [description] - [file:line]
- [Issue 2]: [severity] - [description] - [file:line]

### Suggestions
- [Suggestion 1]
- [Suggestion 2]

### Overall Assessment
[Pass/Needs Changes] - [summary]
```

### Process

1. **Parse Issues**: Extract issues from the review output
2. **Filter by Severity**: Focus on critical/high severity first
3. **Present One by One**: Show each issue with context
4. **User Decision**: Fix / Skip / Need context
5. **Apply Fix**: Make the change after approval
6. **Summary**: Report all fixes made

### Presenting Local Issues

```markdown
## Issue [N of Total]

**Severity**: [Critical / High / Medium / Low]
**File**: `path/to/file.go:123`
**Issue**: [description]

### Current Code
```go
// Show the problematic code
```

### Suggested Fix
[Explanation of what needs to change]

---

**Options**:
1. Fix this issue
2. Skip (intentional / won't fix)
3. Need more context
```

---

## Mode 2: Remote PR Review (from GitHub)

### Step 1: Identify the PR

```bash
# Get current branch
git branch --show-current

# Find PR for current branch
gh pr view --json number,title,url,state
```

If no PR found, ask user for PR number or URL.

### Step 2: Fetch All Comments

```bash
# Get PR number
PR_NUMBER=$(gh pr view --json number -q '.number')

# Fetch review comments (code comments)
gh api repos/{owner}/{repo}/pulls/{PR_NUMBER}/comments

# Fetch issue comments (general PR comments)
gh api repos/{owner}/{repo}/issues/{PR_NUMBER}/comments
```

### Step 3: Filter Comments

Include:
- Comments from others (not the PR author)
- Comments from `claude[bot]` (automated review)
- Unresolved comments

Exclude:
- Comments from the current user
- Already resolved comments

### Step 4: Present Comments One by One

```markdown
## Comment [N of Total]

**From**: @username (or claude[bot])
**File**: `path/to/file.go:123`
**Type**: [Code Review / General Comment]

### Comment
> [The actual comment text]

### Context
[Show the relevant code snippet]

---

**Options**:
1. Fix this issue
2. Skip (won't fix)
3. Need more context
4. Mark as resolved (no code change needed)
```

### Step 5: Handle User Decision

#### Option 1: Fix this issue
1. Read the file mentioned
2. Understand the requested change
3. Propose a fix
4. Apply after approval
5. Reply to comment: "Fixed in latest commit"

#### Option 2: Skip
1. Note the skipped comment
2. Move to next
3. Include in summary

#### Option 3: Need more context
1. Show more surrounding code
2. Explain what's being asked
3. Re-present options

#### Option 4: Mark as resolved (Remote only)
1. Reply indicating resolution
2. Move to next comment

---

## Commands for Remote Mode

### Fetch PR Comments

```bash
# Review comments (on specific lines)
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments \
  --jq '.[] | {id, user: .user.login, path: .path, line: .line, body: .body}'

# Issue comments (general discussion)
gh api repos/{owner}/{repo}/issues/{pr_number}/comments \
  --jq '.[] | {id, user: .user.login, body: .body}'
```

### Reply to Comment

```bash
# Reply to a review comment
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments/{comment_id}/replies \
  -f body="Fixed in latest commit"

# React to a comment
gh api repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions \
  -f content="+1"
```

---

## Output Format

### Summary (Both Modes)

```markdown
## Review Comments Summary

**Source**: [Local Code Review / PR #123]
**Total Issues**: 7

### Fixed (4)
- `internal/handler/user.go:45` - Added error handling
- `internal/service/auth.go:123` - Fixed validation logic
- `CLAUDE.md:50` - Updated documentation
- General - Addressed code style

### Skipped (2)
- `internal/models/user.go:10` - "Consider using UUID" (intentional design)
- General - Out of scope

### Marked Resolved (1) [Remote only]
- `go.mod` - Dependency question answered

### Next Steps
- [ ] Run tests to verify fixes
- [ ] [Local] Proceed to Phase 10 (Documentation)
- [ ] [Remote] Push changes and request re-review
```

---

## Integration with /dev Workflow

### Phase 9 Integration (Local Mode)

After `feature-dev:code-reviewer` agent completes:

1. If issues found with severity ≥ High → Activate this skill
2. Process issues one by one
3. Fix or skip each issue
4. Re-run code review if critical fixes were made
5. Proceed to Phase 10 when all critical issues resolved

### Manual Invocation (Remote Mode)

After GitHub Action PR review completes:

1. User says "fix PR comments" or similar trigger
2. Skill fetches comments from PR
3. Process comments one by one
4. Push fixes when done

---

## Error Handling

| Error                   | Action                                |
| ----------------------- | ------------------------------------- |
| No issues found         | Report "No issues to fix" and proceed |
| No PR found (Remote)    | Ask user for PR number                |
| File not found          | Report and skip to next issue         |
| API rate limit (Remote) | Wait and retry, or continue later     |
