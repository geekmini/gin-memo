---
name: Spec Generator
description: Use this skill when the user asks to "generate a spec", "create specification document", "write API spec", "create spec file", or when architecture decisions have been made and requirements are gathered. Also use after the architecture phase in the /dev workflow.
version: 1.0.0
---

# Spec Generator

Generate a structured API specification document from requirements.

## When This Skill Activates

- User says "generate a spec" or "create specification"
- After architecture decisions are made
- When requirements and data models are defined
- During Phase 6 of the /dev workflow

---

## Inputs Required

Before generating a spec, gather the following information:

| Input | Required | Source |
|-------|----------|--------|
| Feature name | Yes | User or discovery |
| Feature description | Yes | User or discovery |
| Data model (fields, types) | Yes | Clarifying questions |
| API endpoints | Yes | Clarifying questions |
| Business rules | Yes | Clarifying questions |
| Architecture decision | Yes | Architecture phase or user |
| Files to create/modify | Yes | Architecture phase |
| Out of scope items | No | Clarifying questions |

---

## Generation Process

### Step 1: Validate Inputs

Ensure all required inputs are available. If missing, ask the user:
- "What is the feature name?"
- "What data fields are needed?"
- "What endpoints should be created?"

### Step 2: Determine File Path

```
spec/[feature-name-kebab-case].md
```

Example: "User Profile Pictures" → `spec/user-profile-pictures.md`

### Step 3: Generate Spec Document

Use the following template structure:

```markdown
# [Feature Name] Specification

**Author**: [ask user or use "Team"]
**Created**: [YYYY-MM-DD]
**Status**: Draft
**Architecture**: [chosen approach]

## Overview

[Brief description - 1-2 sentences]

## Architecture Decision

**Chosen Approach**: [approach name]
**Rationale**: [why this approach]

### Files to Create
- `path/to/file.go` - [purpose]

### Files to Modify
- `path/to/file.go` - [changes needed]

## Data Model

### [Entity Name]

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| id | string | Yes | Unique identifier |
| ... | ... | ... | ... |

## API Endpoints

### [Endpoint Name]

**Endpoint**: `METHOD /api/v1/path`
**Authentication**: Required/None
**Description**: [what it does]

#### Request

```json
{
  "field": "value"
}
```

#### Response

**Success (200/201):**
```json
{
  "success": true,
  "data": {
    "field": "value"
  }
}
```

#### Error Responses

| Status | Message | Description |
| ------ | ------- | ----------- |
| 400 | Invalid input | [when this occurs] |
| 404 | Not found | [when this occurs] |

## Business Rules

1. [Rule 1]
2. [Rule 2]

## Implementation Steps

1. [ ] [Step 1 - description]
2. [ ] [Step 2 - description]
3. [ ] [Step 3 - description]

## Out of Scope

- [Item 1]
- [Item 2]

## Open Questions

- [ ] [Question 1]
```

### Step 4: Write File

Use the Write tool to create the spec file at `spec/[feature-name-kebab-case].md`.

### Step 5: Confirm

Present summary to user:
```
## Spec Generated

**File**: spec/[feature-name-kebab-case].md
**Endpoints**: [N] endpoints defined
**Data Models**: [N] models defined

Would you like to review or modify the spec?
```

---

## Standalone Usage

When invoked directly (not from /dev workflow):

1. Ask for feature name and description
2. Ask clarifying questions about data model and endpoints
3. Ask about architecture approach (or suggest based on codebase patterns)
4. Generate the spec document
5. Offer to proceed to implementation or Postman setup

---

## Integration with /dev Workflow

When called from Phase 6 of /dev:
- All inputs should already be gathered from previous phases
- Skip to Step 2 (Determine File Path)
- Generate and write the spec
- Return control to /dev for Phase 7

---

## Quality Checklist

Before finalizing, verify:
- [ ] All endpoints have request/response examples
- [ ] All fields have types and descriptions
- [ ] Business rules are clear and testable
- [ ] Implementation steps are actionable
- [ ] File paths follow project conventions
