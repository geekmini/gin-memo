# Spec-Driven Development

A comprehensive workflow that combines structured requirements gathering, codebase exploration, architecture design, spec documentation, Postman integration, and optional implementation with quality review.

## Workflow Overview

```
1. Discovery          → Understand the feature request
2. Codebase Exploration → Explore existing patterns (agents)
3. Clarifying Questions → Resolve ambiguities
4. Summary & Approval  → Confirm understanding
5. Architecture Design → Propose 2-3 approaches (agents)
6. Generate Spec      → Create spec document
7. Add to Postman     → Add endpoints to collection (optional)
8. Implementation     → Build the feature (optional)
9. Quality Review     → Review code quality (agents)
10. Completion        → Summary and next steps
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

After architecture approval, create a spec document at `spec/[feature-name].md`:

```markdown
# [Feature Name] Specification

**Author**: [ask user or use "Team"]
**Created**: [current date in YYYY-MM-DD format]
**Status**: Draft
**Architecture**: [chosen option from Phase 5]

## Overview

[Brief description of the feature]

## Architecture Decision

**Chosen Approach**: [Option name]
**Rationale**: [Why this approach was selected]

### Files to Create
- `path/to/file.go` - [purpose]

### Files to Modify
- `path/to/file.go` - [changes needed]

## Data Model

### [Entity Name]

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| ...   | ...  | ...      | ...         |

## API Endpoints

### [Endpoint Name]

**Endpoint**: `METHOD /path`
**Authentication**: Required/None
**Description**: [what it does]

#### Request

```json
{
  "field": "value"
}
```

#### Response

```json
{
  "field": "value"
}
```

#### Error Responses

| Status | Code          | Description |
| ------ | ------------- | ----------- |
| 400    | INVALID_INPUT | ...         |
| 404    | NOT_FOUND     | ...         |

## Business Rules

1. Rule 1
2. Rule 2

## Implementation Steps

1. [ ] Step 1 - [description]
2. [ ] Step 2 - [description]
3. [ ] Step 3 - [description]

## Out of Scope

- Item 1
- Item 2

## Open Questions

- [ ] Question 1
- [ ] Question 2
```

---

## Phase 7: Add Endpoints to Postman (Optional)

**Skip this phase if:**
- The feature doesn't involve new API endpoints
- The endpoints already exist in Postman
- The user prefers to add them manually later

**Ask the user**: "This feature includes [N] new endpoint(s). Would you like me to add them to Postman?"

If yes, add the new API endpoints using MCP tools:

1. Use the Postman MCP tools to add each new endpoint to the collection
2. Use the workspace and collection IDs from CLAUDE.md:
   - Workspace: `1ee078be-5479-45b9-9d5a-883cd4c6ef50` (golang)
   - Collection: `25403495-bb644262-dce4-42ac-8cc4-810d8a328fc9` (go-sample)
3. For each endpoint, set:
   - Name: Descriptive name (e.g., "Create Voice Memo")
   - Method: GET/POST/PUT/DELETE
   - URL: `{{base_url}}/api/v1/...`
   - Headers: Content-Type if needed
   - Body: Sample request body for POST/PUT requests
4. Verify the endpoints were added correctly

If no or no endpoints to add, skip to Phase 8.

---

## Phase 8: Implementation (Optional)

**Ask the user**: "Would you like me to implement this feature now?"

If yes, proceed with implementation:
1. Follow the implementation steps from the spec
2. Follow the chosen architecture approach
3. Use existing patterns discovered in Phase 2
4. Create/modify files as specified
5. Run tests if available
6. Run linters/formatters

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

## Phase 10: Completion

Present final summary:

```markdown
## Completion Summary

### Spec Document
- Created: `spec/[feature-name].md`

### Postman Endpoints (if added)
- [Endpoint 1]: [METHOD] [URL]
- [Endpoint 2]: [METHOD] [URL]
- Or: "Skipped - no new endpoints" / "Skipped - user preference"

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
