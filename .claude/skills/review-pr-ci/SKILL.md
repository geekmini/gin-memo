---
name: review-pr-ci
description: Review a pull request in CI mode. Posts/updates GitHub inline comments and summary comment. Use when running automated code review in GitHub Actions or other CI environments. Includes semantic deduplication to prevent duplicate comments, smart orphan cleanup when files are removed from PR, and fixed issue notification.
---

# CI Code Review

Review PRs automatically in CI with intelligent comment management.

## CI Environment Check

**This skill is for CI automation only.**

Check if running in CI:
```bash
echo "CI=$CI, GITHUB_ACTIONS=$GITHUB_ACTIONS"
```

If NOT in CI (both empty/unset), show error and exit:
```
Error: This skill is for CI automation only.

For local use, choose:
  review-and-fix         # Local code review (no PR required)
  resolve-pr-comments    # Respond to human reviewer comments

To trigger automated CI review, push your branch and open/sync a PR.
```

## Quick Workflow

1. **Read guidelines** from `docs/code-review-guideline.md` and `CLAUDE.md` (if exists)
2. **Analyze PR** using `gh pr view` and `gh pr diff`
3. **Classify issues** as CRITICAL (inline comments) or SUGGESTIONS (summary only)
4. **Deduplicate** against existing unresolved comments (semantic matching)
5. **Post/update comments** and summary

## Detailed Workflow

See [references/workflow.md](references/workflow.md) for:
- Complete 6-phase workflow
- Semantic deduplication algorithm
- Comment reconciliation logic
- State synchronization
- Error handling

## Issue Classification

**CRITICAL** (inline comments):
- Bugs that will cause failures
- Security vulnerabilities
- Code that worsens code health
- Architectural violations
- Tech stack anti-patterns (per guideline)

**SUGGESTIONS** (summary table only):
- Minor style inconsistencies
- Alternative approaches
- Educational comments
- Optional refactoring
