---
name: resolve-pr-comments
description: Resolve GitHub PR review comments interactively. Fetches unresolved comments, helps fix them, posts replies, and resolves threads. Use when a developer wants to address feedback from code reviewers on their pull request.
---

# Resolve PR Comments

Interactively resolve GitHub PR review comments.

## Quick Workflow

1. **Fetch comments** - Get all unresolved inline and general PR comments
2. **Present each** - Show comment with file context and severity assessment
3. **User decides** - Fix, Skip, Resolve, or Acknowledge
4. **Apply changes** - Make code changes, post replies, resolve threads
5. **Commit & push** - Stage changes and offer to push

## PR Detection

Auto-detect from current branch or accept PR number as argument:
```bash
gh pr view --json number,title --jq '"\(.number)|\(.title)"'
```

## Comment Processing Order

1. **First**: Unresolved inline review comments
2. **Last**: General PR comments

## User Options

**For inline comments:**
- **Fix** - Apply code change, post reply, resolve thread
- **Skip** - Move to next without changes
- **Resolve** - Mark resolved without code changes (with reason)

**For general comments:**
- **Fix** - Apply code changes and reply
- **Skip** - Move to next
- **Acknowledge** - Reply with explanation

## Detailed Workflow

See [references/workflow.md](references/workflow.md) for:
- Complete fetch and filter logic
- GraphQL thread resolution
- Interactive loop details
- Progress tracking
