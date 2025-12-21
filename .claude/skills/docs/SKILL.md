---
name: Documentation Updater
description: Use this skill when the user asks to "update documentation", "sync docs", "update CLAUDE.md", "check documentation", or after code implementation is complete. Also use after the implementation phase in the /spec workflow.
version: 1.0.0
---

# Documentation Updater

Analyze code changes and update project documentation accordingly.

## When This Skill Activates

- User says "update documentation" or "sync docs"
- After code implementation is complete
- When user asks to check if docs are in sync
- During Phase 10 of the /spec workflow

---

## Documentation Files

| File | Purpose | When to Update |
|------|---------|----------------|
| `CLAUDE.md` | Project overview, conventions | Structure changes, new patterns |
| `docs/architecture.md` | Layer conventions, DTOs | Layer changes, new conventions |
| `docs/design-patterns.md` | Design patterns used | New patterns, caching, tokens |
| `swagger/swagger.yaml` | API documentation | Run `task swagger` after endpoint changes |
| `.env.example` | Environment variables | New env vars added |
| `docker-compose.yml` | Local services | New services added |
| `Taskfile.yml` | Task commands | New tasks added |

---

## Analysis Process

### Step 1: Identify Changes

Determine what changed by either:
- **From /spec workflow**: Review the spec document and implementation
- **Standalone**: Ask user what was changed, or analyze git diff

```bash
git diff --name-only HEAD~1  # Recent changes
git diff --staged --name-only  # Staged changes
```

### Step 2: Map Changes to Documentation

Use this decision matrix:

| Change Type | Documentation Action |
|-------------|---------------------|
| New API endpoint | Verify `task swagger` was run |
| New environment variable | Add to `.env.example` with description |
| New project structure (folder/file pattern) | Update `CLAUDE.md` "Project Structure" |
| New layer convention | Update `docs/architecture.md` |
| New design pattern | Update `docs/design-patterns.md` |
| New local service | Update `docker-compose.yml` docs |
| New task command | Update `Taskfile.yml` or `CLAUDE.md` |

### Step 3: Check Each Documentation File

#### CLAUDE.md Checks

| Section | Check For |
|---------|-----------|
| Project Structure | New directories or file patterns |
| API Endpoints | New route groups |
| Quick Commands | New frequently-used tasks |
| Coding Conventions | New naming patterns |

#### docs/architecture.md Checks

| Section | Check For |
|---------|-----------|
| Layer responsibilities | New layer conventions |
| Error handling flow | Changes to error handling |
| Request/Response DTOs | New DTO patterns |
| Migration Status | Completed migrations |

#### docs/design-patterns.md Checks

| Section | Check For |
|---------|-----------|
| Dependency Injection | New DI patterns |
| Soft Delete | Changes to soft delete |
| Pagination | New pagination patterns |
| Caching | New cache keys or strategies |
| Token Pattern | New token flows |

### Step 4: Propose Updates

For each file that needs updates, present:

```markdown
## Proposed Documentation Updates

### CLAUDE.md
**Section**: Project Structure
**Change**: Add `internal/newmodule/` description
**Reason**: New module added for [feature]

### docs/architecture.md
**Section**: [section name]
**Change**: [what to add/modify]
**Reason**: [why this change is needed]
```

### Step 5: Apply Updates (with approval)

After user approves:
1. Read the target file
2. Make the specific edits
3. Confirm changes were applied

---

## Standalone Usage

When invoked directly:

1. **Ask**: "What changes were made?" or "Should I analyze recent git changes?"
2. **Analyze**: Determine which docs might need updates
3. **Propose**: Show proposed updates
4. **Apply**: Make changes after approval
5. **Summary**: List all documentation updates made

---

## Quick Checks

### After Adding New Endpoint
- [ ] `task swagger` was run
- [ ] Route group documented in CLAUDE.md (if new group)

### After Adding New Environment Variable
- [ ] Added to `.env.example` with comment
- [ ] Documented in CLAUDE.md if critical

### After Adding New Design Pattern
- [ ] Documented in `docs/design-patterns.md`
- [ ] Example code included

### After Changing Project Structure
- [ ] `CLAUDE.md` "Project Structure" updated
- [ ] Any new directories explained

---

## Output Format

After completing documentation updates:

```markdown
## Documentation Update Summary

### Files Updated
- `CLAUDE.md` - Added [section/content]
- `docs/architecture.md` - Updated [section]

### Files Verified (no changes needed)
- `docs/design-patterns.md`
- `.env.example`

### Reminders
- [ ] Run `task swagger` if endpoints changed
- [ ] Review changes before committing

### Source of Truth Status
| File | Status |
|------|--------|
| swagger/swagger.yaml | [Updated via task swagger / No changes] |
| .env.example | [Updated / No changes] |
| docker-compose.yml | [No changes] |
| Taskfile.yml | [No changes] |
```
