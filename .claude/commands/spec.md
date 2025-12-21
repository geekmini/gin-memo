# Spec-Driven Development

A comprehensive workflow that combines structured requirements gathering, codebase exploration, architecture design, spec documentation, Postman integration, and optional implementation with quality review.

## Workflow Overview

```
1. Discovery          → Understand the feature request
2. Codebase Exploration → Explore existing patterns (agents)
3. Clarifying Questions → Resolve ambiguities
4. Summary & Approval  → Confirm understanding
5. Architecture Design → Propose 2-3 approaches (agents)
6. Generate Spec      → Create spec document (skill)
7. Add to Postman     → Add endpoints to collection (skill, optional)
8. Implementation     → Build the feature + regenerate Swagger (skill, optional)
9. Quality Review     → Review code quality (agents)
10. Documentation     → Update CLAUDE.md if needed (skill), final summary
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

## Phase 8: Implementation (Optional)

**Ask the user**: "Would you like me to implement this feature now?"

If yes, **use the `/feature-dev` skill** for guided implementation:

1. The skill will use the spec document as the implementation guide
2. It follows the chosen architecture approach from Phase 5
3. Uses existing patterns discovered in Phase 2
4. Creates/modifies files as specified in the spec
5. Runs tests and linters
6. **Runs `task swagger`** to regenerate API documentation

**Note**: The `/feature-dev` skill provides guided feature development with codebase understanding and architecture focus.

If no, skip to Phase 10.

---

## Phase 9: Quality Review

**Only run if implementation was done in Phase 8.**

**Use the Task tool with `subagent_type: "feature-dev:code-reviewer"`** to review the implementation.

The code reviewer checks for:
- Bugs and logic errors
- Security vulnerabilities
- Code quality (DRY, simplicity)
- Adherence to project conventions (CLAUDE.md)
- Proper error handling

Present findings:
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

Fix any critical issues before proceeding.

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

### Next Steps
- [ ] Review and update spec status to "Approved"
- [ ] [Additional follow-up items]
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
