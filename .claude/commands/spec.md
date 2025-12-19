# Spec-Driven Development

Help the user define requirements through structured questions, then generate a specification document.

## Workflow

### 1. Gather Requirements

When the user describes a feature or requirement:
- Listen to their initial description
- Identify areas that need clarification

### 2. Ask Clarifying Questions

Ask questions **one by one** (not all at once). Cover:
- **Scope**: What's included/excluded?
- **Data model**: What fields? What types? Required/optional?
- **Relationships**: How does this relate to existing entities?
- **API design**: Endpoints? Request/response format?
- **Business rules**: Validation? Constraints? Edge cases?
- **Authentication**: Public or protected?
- **Pagination/sorting**: If listing data
- **Error handling**: What can go wrong?

Wait for the user's answer before asking the next question.

### 3. Generate Summary

After gathering all information, present a summary:

```
## Summary

**Feature**: [name]
**Description**: [1-2 sentences]

### Data Model
- field1: type (required/optional) - description
- field2: type (required/optional) - description

### API Endpoints
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| ... | ... | ... | ... |

### Business Rules
- Rule 1
- Rule 2

### Out of Scope
- Item 1
- Item 2
```

Ask the user to review and approve before proceeding.

### 4. Generate Spec Document

After approval, create a spec document at `spec/[feature-name].md` with this format:

```markdown
# [Feature Name] Specification

**Author**: [ask user or use "Team"]
**Created**: [current date in YYYY-MM-DD format]
**Status**: Draft

## Overview

[Brief description of the feature]

## Data Model

### [Entity Name]

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| ... | ... | ... | ... |

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

| Status | Code | Description |
|--------|------|-------------|
| 400 | INVALID_INPUT | ... |
| 404 | NOT_FOUND | ... |

## Business Rules

1. Rule 1
2. Rule 2

## Out of Scope

- Item 1
- Item 2

## Open Questions

- [ ] Question 1
- [ ] Question 2
```

### 5. Confirm Completion

After generating the spec:
- Show the file path
- Ask if the user wants to make any changes
- Offer to start implementation when ready

## Guidelines

- Be thorough but concise
- Use the existing codebase conventions (camelCase for JSON fields)
- Reference existing patterns in the project
- Keep specs focused on WHAT, not HOW (implementation details come later)
