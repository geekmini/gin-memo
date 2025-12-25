# Local Code Review Workflow

## Arguments

```
$ARGUMENTS
```

## Branch Detection

**If base branch provided in arguments:**
- Use that as the base branch

**If no arguments:**
1. Get current branch
2. Detect default base branch:
   ```bash
   git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@'
   ```
   Fall back to `main` or `master` if that fails.

## Phase 1: Execute Review

1. Read `docs/code-review-guideline.md` for review criteria
2. Reference `CLAUDE.md` for project standards (if exists)

Execute:
```bash
git diff --name-only <base_branch>...HEAD
git diff <base_branch>...HEAD
```

## Phase 2: Classify and Sort Issues

**Critical** - Must fix:
- Bugs, security vulnerabilities
- Anti-patterns per your guideline

**Major** - Strong suggestions:
- Performance issues, poor maintainability

**Minor** - Nits:
- Style, naming, alternatives

**Sort by severity: Critical → Major → Minor**

## Phase 3: Interactive Fix Loop

For each issue:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Processing issue X of Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**File:** path/to/file:42
**Severity:** Critical | Major | Minor
**Issue:** Description
**Proposed Fix:** How to fix

[Show proposed code change]
```

**Options:**
- **Fix** - Apply the change
- **Skip** - Move to next
- **Document** - Record as tech debt

**Document** appends to `docs/technical_debt.md`:
```markdown
## [YYYY-MM-DD] Branch: <branch_name>

### <Brief title>
- **File:** <path>:<line>
- **Severity:** <level>
- **Issue:** <description>
- **Why deferred:** Acknowledged during review
```

### Progress Tracking

```
Progress: X of Y issues (F fixed, S skipped, D documented)
```

## Phase 4: Commit Changes

**If code changes made:**
```bash
git add -A
git commit -m "fix: address code review feedback

Resolved:
- [List fixes]
"
```

## Phase 5: Final Options

**Options:**
- **Re-review** - Run full review again
- **Done** - Exit

## Constraints

- Do NOT post GitHub comments (local-only mode)
- Do NOT interact with GitHub API
- Show clear progress indicators
