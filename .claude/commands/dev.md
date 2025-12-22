# Feature Development Workflow

A comprehensive workflow that combines structured requirements gathering, codebase exploration, architecture design, spec documentation, Postman integration, implementation, and quality review.

---

## Session Management

This workflow uses a checkpoint file (`.claude/dev-checkpoint.md`) to persist progress across conversations. This ensures you can resume from where you left off even after context compaction.

### On `/dev` Invocation

**Step 1: Check for existing session**

```bash
# Check if checkpoint file exists
ls .claude/dev-checkpoint.md
```

**Step 2: If checkpoint exists → Offer to resume**

Read the checkpoint file and present:
```
Found existing dev session:
- Feature: [feature name]
- Current Phase: [phase number] - [phase name]
- Last Updated: [timestamp]

Options:
1. Resume this session
2. Start fresh (archives existing checkpoint)
```

If user chooses to resume:
- Read checkpoint file for full context
- Continue from the current phase listed
- Update checkpoint as you progress

If user chooses to start fresh:
- Move existing checkpoint to `.claude/dev-checkpoints/[feature]-[timestamp].md`
- Create new checkpoint file

**Step 3: If no checkpoint exists → Start new session**

After Phase 1 (Discovery), create the checkpoint file.

### Checkpoint File Format

Location: `.claude/dev-checkpoint.md`

```markdown
# Dev Session: [feature-name]

**Current Phase:** [number] - [phase name]
**Spec File:** spec/[feature-name].md (if created)
**Branch:** [branch name] (if created)
**Last Updated:** [ISO timestamp]

---

## Completed Phases

### Phase 1: Discovery
[Summary of what user wants - 2-3 sentences]

### Phase 2: Codebase Exploration
[Key patterns found, relevant files]

### Phase 3: Clarifying Questions
[Key decisions from Q&A - bullet points]

### Phase 4: Summary & Approval
[Confirmed: Yes/No, any changes requested]

### Phase 5: Architecture
**Decision:** [Option chosen]
**Rationale:** [Why this option]

### Phase 6: Spec Generated
**File:** spec/[feature-name].md

### Phase 7: Spec Approved
[Approved: Yes, timestamp]

### Phase 8: Implementation
**Status:** [In progress / Complete]
**Completed:**
- [x] Models
- [x] Repository
- [ ] Service (3/5 methods)
- [ ] Handler
- [ ] Router

### Phase 9: Finalization
**Local Review:** [Passed / Issues Found]
**Issues Fixed:** [N/A / List]
**Docs Updated:** [Yes/No]
**Postman:** [Added N endpoints / Skipped]
**PR:** [#number or N/A]
**CI Review:** [Pending / Passed / Issues Fixed]

---

## Current Context

[Any important context for resuming - what was being worked on, blockers, next immediate step]

---

## Next Steps

1. [Immediate next action]
2. [Following action]
```

### Checkpoint Update Rules

Update the checkpoint file:
- **After each phase completes** - Add phase summary to "Completed Phases"
- **On phase transitions** - Update "Current Phase"
- **When blocked or pausing** - Update "Current Context" with state
- **At key decision points** - Record decisions and rationale

### Session Completion

When Phase 9 completes successfully:
1. Archive checkpoint to `.claude/dev-checkpoints/[feature]-[timestamp].md`
2. Delete `.claude/dev-checkpoint.md`
3. Confirm: "Dev session complete. Checkpoint archived."

---

## Workflow Overview

```
1. Discovery          → Understand the feature request
2. Codebase Exploration → Explore existing patterns (agents)
3. Clarifying Questions → Resolve ambiguities
4. Summary & Approval  → Confirm understanding
5. Architecture Design → Propose 2-3 approaches (agents)
6. Generate Spec      → Create spec document (skill) [Status: Draft]
7. Approve Spec       → User approves spec [Status: Draft → Approved]
8. Implementation     → Build the feature + regenerate Swagger
9. Finalization       → Review, fix, docs, Postman, PR (LAST PHASE)
```

---

## Phase 1: Discovery

When the user describes a feature or requirement:
- Listen to their initial description
- Identify the core functionality requested
- Note any explicit constraints or preferences mentioned

**Checkpoint:** Create `.claude/dev-checkpoint.md` with feature name, set Current Phase to "2 - Codebase Exploration", and add Phase 1 summary.

---

## Phase 2: Codebase Exploration

**Use the Task tool with `subagent_type: "code-explorer"`** to understand existing patterns.

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

**Checkpoint:** Update Current Phase to "3 - Clarifying Questions", add Phase 2 summary with key patterns and files found.

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

**Checkpoint:** Update Current Phase to "4 - Summary & Approval", add Phase 3 summary with key decisions from Q&A.

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

**Checkpoint:** Update Current Phase to "5 - Architecture Design", add Phase 4 summary (Confirmed: Yes/No).

---

## Phase 5: Architecture Design

**Use the Task tool with `subagent_type: "code-architect"`** to design the implementation.

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

**Checkpoint:** Update Current Phase to "6 - Generate Spec", add Phase 5 summary with decision and rationale.

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

**Checkpoint:** Update Current Phase to "7 - Approve Spec", add Phase 6 summary with spec file path, update Spec File field.

---

## Phase 7: Approve Spec

Before starting implementation, get explicit user approval on the spec document.

**Ask the user**: "The spec document is ready at `spec/[feature-name].md`. Would you like to review it and approve for implementation?"

### On Approval

Update the spec document status:

1. Change `**Status**: Draft` to `**Status**: Approved`
2. Confirm: "Spec status updated to 'Approved'. Proceeding to implementation."

### Status Lifecycle

| Status          | Meaning                                 |
| --------------- | --------------------------------------- |
| **Draft**       | Spec created, pending review            |
| **Approved**    | User approved, ready for implementation |
| **Implemented** | Code complete and verified              |

**Note**: Do not proceed to Phase 8 until user explicitly approves.

**Checkpoint:** Update Current Phase to "8 - Implementation", add Phase 7 summary (Approved: Yes, timestamp).

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

**Checkpoint:** Update Current Phase to "9 - Finalization", add Phase 8 summary with implementation status and completed layers checklist.

---

## Phase 9: Finalization

This is the **final phase** that covers quality review, documentation, Postman updates, PR creation, and CI review handling.

### Step 1: Local Code Review Loop (Iterative)

**IMPORTANT:** This step repeats until the code review passes with no critical/high issues.

```
┌─────────────────────────────────────┐
│  Run code-reviewer agent            │
└────────────────┬────────────────────┘
                 ▼
         ┌───────────────┐
         │ Issues found? │
         └───────┬───────┘
                 │
     ┌───────────┴───────────┐
     │ Yes                   │ No
     ▼                       ▼
┌────────────────┐    ┌─────────────────┐
│ Run /pr-fix    │    │ Proceed to      │
│ (Local mode)   │    │ Step 2          │
└───────┬────────┘    └─────────────────┘
        │
        ▼
┌────────────────┐
│ Loop back to   │
│ code-reviewer  │
└────────────────┘
```

**Use the Task tool with `subagent_type: "code-reviewer"`** to review the implementation.

The code reviewer checks for:
- Bugs and logic errors
- Security vulnerabilities
- Code quality (DRY, simplicity)
- Adherence to project conventions (CLAUDE.md)
- Proper error handling

If issues are found:
1. Run `/pr-fix` in **Local mode**
2. Present each issue with context and severity
3. User decides: Fix / Skip / Need more context
4. Apply fixes **only after user approval**
5. **Re-run code-reviewer** to verify fixes
6. Repeat until all critical/high issues resolved

See `.claude/skills/pr-fix/SKILL.md` for the full process.

### Step 2: Update Spec Status

After local review passes, update the spec document:

1. Change `**Status**: Approved` to `**Status**: Implemented`
2. Mark all implementation checklist items as complete (`[x]`)
3. Confirm: "Spec status updated to 'Implemented'."

### Step 3: Update Documentation

**The Documentation Updater skill** will analyze changes and update documentation.

**Files checked:**
- `CLAUDE.md` - Project structure, conventions
- `docs/architecture.md` - Layer conventions, DTOs
- `docs/design-patterns.md` - Design patterns, caching, tokens
- `.env.example` - Environment variables

See `.claude/skills/docs/SKILL.md` for the decision matrix.

### Step 4: Add Endpoints to Postman (Optional)

If the feature includes new API endpoints:

**Ask the user**: "This feature includes [N] new endpoint(s). Would you like me to add them to Postman?"

If yes, **the Postman Collection Manager skill** will:
1. Read the spec file from Phase 6
2. Extract API endpoints
3. Add each endpoint to the Postman collection
4. Report success/failure

See `.claude/skills/postman/SKILL.md` for details.

### Step 5: Commit, Push & Create PR

**Ask the user**: "Would you like me to commit, push, and create a PR?"

If yes, run `/commit-push-pr` which will:
1. Create a new branch if on main
2. Commit with conventional commit message
3. Push to remote
4. Create PR via `gh pr create`

### Step 6: Address CI Review Comments (Iterative)

After the PR is created, GitHub Actions runs automated code review (claude-reviewer).

**IMPORTANT:** This step repeats until CI review passes with no issues.

```
┌─────────────────────────────────────┐
│  Wait for CI review comments        │
└────────────────┬────────────────────┘
                 ▼
         ┌───────────────┐
         │ Comments?     │
         └───────┬───────┘
                 │
     ┌───────────┴───────────┐
     │ Yes                   │ No
     ▼                       ▼
┌────────────────┐    ┌─────────────────┐
│ Run /pr-fix    │    │ Proceed to      │
│ (Remote mode)  │    │ Step 7          │
└───────┬────────┘    └─────────────────┘
        │
        ▼
┌────────────────┐
│ Push fixes,    │
│ loop back      │
└────────────────┘
```

When CI review comments arrive:
1. Run `/pr-fix` in **Remote mode**
2. Present each comment with context
3. User decides: Fix / Skip / Need more context
4. Apply fixes, commit, push
5. Wait for CI to re-run
6. Repeat until CI review passes

### Step 7: Archive Checkpoint & Summary

After CI review passes:
1. Create `.claude/dev-checkpoints/` directory if needed
2. Move `.claude/dev-checkpoint.md` to `.claude/dev-checkpoints/[feature-name]-[timestamp].md`
3. Confirm: "Dev session archived."

Present final summary:

```markdown
## Completion Summary

### Spec Document
- Created: `spec/[feature-name].md`
- Status: Draft → Approved → Implemented

### Local Review
- Iterations: [N]
- Issues Fixed: [list or N/A]

### Documentation Updates
- `CLAUDE.md`: [Updated: section] / [No updates needed]
- `docs/architecture.md`: [Updated: section] / [No updates needed]

### Postman Endpoints
- [Added N endpoints] / [Skipped]

### PR
- PR: #[number] - [title]
- URL: [link]

### CI Review
- Iterations: [N]
- Issues Fixed: [list or N/A]
- Status: Passed
```

**Checkpoint:** Mark Phase 9 complete. Archive checkpoint file.

---

## Guidelines

- Be thorough but concise
- Use existing codebase conventions (camelCase for JSON fields)
- Reference existing patterns discovered during exploration
- Keep specs focused on WHAT, not HOW (until architecture phase)
- Always wait for user approval at key decision points (Phase 4, Phase 5, Phase 7)
- Use specialized agents for exploration, architecture, and review
- Confidence threshold for code review issues: only report issues with ≥80% confidence
- Phase 9 loops are iterative - keep running until clean (local review, CI review)
