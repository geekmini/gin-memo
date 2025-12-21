# Feature Development Workflow

A comprehensive workflow that combines structured requirements gathering, codebase exploration, architecture design, spec documentation, Postman integration, implementation, and quality review.

## Workflow Overview

```
1. Discovery          → Understand the feature request
2. Codebase Exploration → Explore existing patterns (agents)
3. Clarifying Questions → Resolve ambiguities
4. Summary & Approval  → Confirm understanding
5. Architecture Design → Propose 2-3 approaches (agents)
6. Generate Spec      → Create spec document (skill)
7. Add to Postman     → Add endpoints to collection (skill, optional)
8. Implementation     → Build the feature + regenerate Swagger
9. Quality Review & Fix → Review + fix issues (agents + skill)
10. Documentation & PR → Update docs (skill), commit/push/PR (skill)
```

---

## Phase 1: Discovery

When the user describes a feature or requirement:
- Listen to their initial description
- Identify the core functionality requested
- Note any explicit constraints or preferences mentioned

---

## Phase 2: Codebase Exploration

**Use the Task tool with `subagent_type: "feature-dev:code-explorer"`** to understand existing patterns.

Launch parallel code-explorer agents to investigate:
- Similar features already implemented
- Existing patterns for the feature type (handlers, services, repositories)
- Related data models and their relationships
- Authentication/authorization patterns used
- Error handling conventions

Present findings to the user:
```
## Codebase Analysis

### Relevant Existing Patterns
- [Pattern 1]: Found in [files] - [brief description]
- [Pattern 2]: Found in [files] - [brief description]

### Related Components
- [Component]: [how it relates to new feature]

### Recommended Approach
Based on existing patterns, this feature should follow [approach]
```

---

## Phase 3: Clarifying Questions

Ask questions **one by one** (not all at once), informed by codebase exploration. Cover:

- **Scope**: What's included/excluded?
- **Data model**: What fields? What types? Required/optional?
- **Relationships**: How does this relate to existing entities?
- **API design**: Endpoints? Request/response format?
- **Business rules**: Validation? Constraints? Edge cases?
- **Authentication**: Public or protected?
- **Pagination/sorting**: If listing data
- **Error handling**: What can go wrong?

Wait for the user's answer before asking the next question.

---

## Phase 4: Summary & Approval

After gathering all information, present a summary:

```markdown
## Summary

**Feature**: [name]
**Description**: [1-2 sentences]

### Data Model
- field1: type (required/optional) - description
- field2: type (required/optional) - description

### API Endpoints
| Method | Endpoint | Description | Auth |
| ------ | -------- | ----------- | ---- |
| ...    | ...      | ...         | ...  |

### Business Rules
- Rule 1
- Rule 2

### Out of Scope
- Item 1
- Item 2
```

**Ask the user to review and approve before proceeding.**

---

## Phase 5: Architecture Design

**Use the Task tool with `subagent_type: "feature-dev:code-architect"`** to design the implementation.

Present **2-3 implementation approaches** with trade-offs:

```markdown
## Architecture Options

### Option A: Minimal Changes
**Description**: [approach description]
**Files to modify**: [list]
**Pros**: Fast to implement, low risk
**Cons**: [trade-offs]

### Option B: Clean Architecture
**Description**: [approach description]
**Files to create**: [list]
**Files to modify**: [list]
**Pros**: Better separation of concerns, easier to test
**Cons**: More code, higher complexity

### Option C: Pragmatic Balance (Recommended)
**Description**: [approach description]
**Files to create**: [list]
**Files to modify**: [list]
**Pros**: Balances simplicity with good design
**Cons**: [trade-offs]

## Recommendation
I recommend **Option [X]** because [reasoning based on codebase patterns].
```

**Wait for user to choose an approach before proceeding.**

---

## Phase 6: Generate Spec Document

**The Spec Generator skill** will automatically activate to generate the specification document.

The skill will:
1. Validate all required inputs from previous phases
2. Determine file path: `spec/[feature-name-kebab-case].md`
3. Generate structured spec using the standard template
4. Write the file and confirm creation

**Inputs gathered from previous phases:**
- Feature name and description (from Phase 1)
- Data model and endpoints (from Phase 3)
- Architecture decision and file changes (from Phase 5)
- Business rules and out of scope items (from Phase 3-4)

See `.claude/skills/spec-gen/SKILL.md` for the full template structure.

---

## Phase 7: Add Endpoints to Postman (Optional)

### Source of Truth

| Source      | Purpose                                                                    |
| ----------- | -------------------------------------------------------------------------- |
| **Swagger** | API contract (endpoints, params, schemas) - regenerated via `task swagger` |
| **Postman** | Team workflow (tests, examples, env configs) - managed via Postman skill   |

**Note:** Postman's OpenAPI sync has limitations (doesn't delete removed endpoints, can overwrite team customizations). The Postman Collection Manager skill handles incremental updates rather than full collection sync.

### When to Skip

- The feature doesn't involve new API endpoints
- The endpoints already exist in Postman
- The user prefers to add them manually later

**Ask the user**: "This feature includes [N] new endpoint(s). Would you like me to add them to Postman?"

### If Yes, Add Endpoints

**The Postman Collection Manager skill** will activate with "Add Endpoints from Spec" operation:

1. The skill will read the spec file created in Phase 6
2. Extract API endpoints from the spec
3. Add each endpoint to the Postman collection with proper folder organization
4. Report success/failure for each endpoint

The skill also supports "Add Single Endpoint" for manual additions.

See `.claude/skills/postman/SKILL.md` for all operations.

If no or no endpoints to add, skip to Phase 8.

---

## Phase 8: Implementation

Implement the feature using the spec document and architecture decisions from previous phases.

### Step 1: Follow the Spec

Use the spec document (`spec/[feature-name].md`) as the implementation guide:
- Create files listed in "Files to Create"
- Modify files listed in "Files to Modify"
- Implement each endpoint from the API section
- Apply business rules as specified

### Step 2: Use Existing Patterns

Reference patterns discovered in Phase 2:
- Follow existing handler/service/repository structure
- Use established error handling conventions
- Match existing code style and naming

### Step 3: Implementation Order

Follow the layered architecture (bottom-up):
1. **Models** - Add/update data structures in `internal/models/`
2. **Repository** - Add database operations in `internal/repository/`
3. **Service** - Add business logic in `internal/service/`
4. **Handler** - Add HTTP handlers with Swagger annotations in `internal/handler/`
5. **Router** - Register routes in `internal/router/router.go`

### Step 4: Verify

After implementation:
1. Run tests: `task test`
2. Run linter: `task lint` (if available)
3. **Run `task swagger`** to regenerate API documentation
4. Verify the feature works as expected

---

## Phase 9: Quality Review & Fix

### Step 1: Code Review

**Use the Task tool with `subagent_type: "feature-dev:code-reviewer"`** to review the implementation.

The code reviewer checks for:
- Bugs and logic errors
- Security vulnerabilities
- Code quality (DRY, simplicity)
- Adherence to project conventions (CLAUDE.md)
- Proper error handling

Review output format:
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

### Step 2: Fix Review Issues

If issues are found (especially Critical/High severity):

**The Review Comments Fixer skill** will activate in Local mode to address issues one by one:

1. Present each issue with context and severity
2. User decides: Fix / Skip / Need more context
3. Apply fixes after approval
4. Track all changes made

See `.claude/skills/pr-fix/SKILL.md` for the full process.

### Step 3: Re-review if Needed

If critical fixes were made:
- Re-run code reviewer to verify fixes
- Ensure no new issues introduced

Proceed to Phase 10 when:
- All critical/high severity issues are resolved
- Or user explicitly approves to proceed with remaining issues

---

## Phase 10: Documentation & Completion

### Documentation Update Check

**The Documentation Updater skill** will automatically activate to analyze changes and update documentation.

The skill will:
1. Identify what changed during implementation
2. Map changes to documentation files using decision matrix
3. Check each documentation file for required updates
4. Propose specific updates for user approval
5. Apply updates after approval

**Files checked:**
- `CLAUDE.md` - Project structure, conventions
- `docs/architecture.md` - Layer conventions, DTOs
- `docs/design-patterns.md` - Design patterns, caching, tokens
- `.env.example` - Environment variables
- `swagger/swagger.yaml` - API docs (via `task swagger`)

See `.claude/skills/docs/SKILL.md` for the full decision matrix and update process.

### Commit, Push & Create PR

**Ask the user**: "Would you like me to commit, push, and create a PR?"

If yes, **use the `/commit-commands:commit-push-pr` skill**:

1. Stage all changes
2. Create commit with conventional commit message
3. Push to remote branch
4. Create PR with summary from this workflow

The PR description will include:
- Feature summary from Phase 4
- Implementation details
- Files changed
- Test plan

**Note**: After the PR is created, GitHub Actions will run automated code review. When review comments arrive, use the Review Comments Fixer skill manually to address them.

### Final Summary

Present final summary:

```markdown
## Completion Summary

### Spec Document
- Created: `spec/[feature-name].md`

### Swagger Documentation
- [Updated via `task swagger`] / [Skipped - no implementation]

### Postman Endpoints (if added)
- [Endpoint 1]: [METHOD] [URL]
- [Endpoint 2]: [METHOD] [URL]
- Or: "Skipped - no new endpoints" / "Skipped - user preference"

### Documentation Updates
- `CLAUDE.md`: [Updated: section] / [No updates needed]
- `docs/architecture.md`: [Updated: section] / [No updates needed]
- `docs/design-patterns.md`: [No updates needed]

### Source of Truth Files
- `swagger/swagger.yaml`: [Regenerated via `task swagger`] / [No changes]
- `.env.example`: [Updated] / [No updates needed]
- `docker-compose.yml`: [No updates needed]
- `Taskfile.yml`: [No updates needed]

### Implementation Status
- [Completed / Skipped / Pending]

### Files Changed (if implemented)
- `path/to/file.go` - [created/modified]

### PR Status (if created)
- PR: #[number] - [title]
- URL: [link]
- Status: Awaiting review

### Next Steps
- [ ] Review and update spec status to "Approved"
- [ ] Wait for GitHub Action code review
- [ ] Use "fix PR comments" to address review feedback when ready
```

Ask if the user wants to make any changes.

---

## Guidelines

- Be thorough but concise
- Use existing codebase conventions (camelCase for JSON fields)
- Reference existing patterns discovered during exploration
- Keep specs focused on WHAT, not HOW (until architecture phase)
- Always wait for user approval at key decision points (Phase 4, Phase 5, Phase 8)
- Use specialized agents for exploration, architecture, and review
- Confidence threshold for code review issues: only report issues with ≥80% confidence
