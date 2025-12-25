---
name: review-and-fix
description: Run a local code review and interactively fix issues one by one. Works without a PR - compares current branch against base branch. Use when a developer wants to review their changes locally before creating a PR, or when working on code that isn't yet pushed.
---

# Local Code Review

Review local changes and fix issues interactively without needing a PR.

## Quick Workflow

1. **Detect branches** - Current branch vs base (main/master)
2. **Read guidelines** from `docs/code-review-guideline.md` and `CLAUDE.md` (if exists)
3. **Analyze diff** using `git diff base...HEAD`
4. **Classify & sort** - Critical → Major → Minor
5. **Interactive loop** - Fix, Skip, or Document each issue
6. **Commit changes** if fixes were applied

## Branch Detection

Auto-detect base branch or accept as argument:
```bash
git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@'
```
Falls back to `main` or `master`.

## Issue Classification

**Critical** - Must fix:
- Bugs, security vulnerabilities
- Anti-patterns per guideline

**Major** - Strong suggestions:
- Performance issues, poor maintainability

**Minor** - Nits:
- Style, naming, alternatives

## User Options

- **Fix** - Apply the change
- **Skip** - Move to next
- **Document** - Record as tech debt in `docs/technical_debt.md`

## Detailed Workflow

See [references/workflow.md](references/workflow.md) for:
- Complete review phases
- Interactive fix loop details
- Tech debt documentation format
- Final options

## Constraints

- Do NOT post GitHub comments (local-only mode)
- Do NOT interact with GitHub API
- Show clear progress indicators
