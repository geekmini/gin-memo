# Feature Development Workflow

A comprehensive workflow that combines structured requirements gathering, codebase exploration, architecture design, spec documentation, Postman integration, implementation, and quality review.

## Sub-agent Architecture

This workflow uses sub-agents to minimize main context size. Heavy exploration and analysis runs in sub-agents, returning only summaries to the main conversation.

```
Main Context (orchestrator)
│
├─ Phase 1: Discovery ────────────────── [main] User input
├─ Phase 2: Exploration ──────────────── [sub-agent: code-explorer]
├─ Phase 3: Questions ────────────────── [main] Interactive Q&A
├─ Phase 4: Approval ─────────────────── [main] User decision
├─ Phase 5: Architecture ─────────────── [sub-agent: code-architect]
├─ Phase 6: Spec Gen ─────────────────── [agent: @spec-gen-agent]
├─ Phase 7: Approval ─────────────────── [main] User decision
├─ Phase 8: Implementation ───────────── [main] File editing
├─ Phase 9: Documentation ────────────── [agent: @docs-agent + @postman-agent]
└─ Phase 10: Review & PR ─────────────── [sub-agent: code-reviewer] + [agent: @pr-fix-agent]
```

**Agent Types:**
- `code-explorer`, `code-architect`, `code-reviewer` - Built-in sub-agent types (Task tool)
- `@spec-gen-agent`, `@docs-agent`, `@postman-agent`, `@pr-fix-agent` - Custom agents (`.claude/agents/`)

**Why sub-agents?**
- Exploration/analysis generates lots of file reads → offload to sub-agent
- Main context only keeps summaries and decisions
- Checkpoint file persists full state for resumability

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

### Phase 9: Documentation
**Docs Updated:** [Yes/No]
**Files:** [list of updated files]
**Postman:** [Added N endpoints / Skipped]

### Phase 10: Review & PR
**Local Review:** [Passed / Issues Found]
**Issues Fixed:** [N/A / List]
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

When Phase 10 completes successfully:
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
9. Documentation      → Update docs, Postman (skills)
10. Review & PR       → Iterative review/fix, PR, CI review (LAST PHASE)
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

**[SUB-AGENT: code-explorer]** - Runs in separate context, returns summary.

### Launch Sub-agent

```
Task tool:
  subagent_type: "code-explorer"
  prompt: |
    Explore the codebase to understand patterns for implementing: [feature from Phase 1]

    Investigate:
    - Similar features already implemented
    - Existing patterns (handlers, services, repositories)
    - Related data models and relationships
    - Authentication/authorization patterns
    - Error handling conventions

    Return a structured summary with:
    1. Relevant patterns found (with file paths)
    2. Related components
    3. Recommended approach based on existing patterns
```

### Expected Output (to main context)

The sub-agent returns a summary like:
```markdown
## Codebase Analysis

### Relevant Existing Patterns
- [Pattern 1]: Found in [files] - [brief description]
- [Pattern 2]: Found in [files] - [brief description]

### Related Components
- [Component]: [how it relates to new feature]

### Recommended Approach
Based on existing patterns, this feature should follow [approach]
```

**Store this summary in checkpoint**, then present to user.

**Checkpoint:** Update Current Phase to "3 - Clarifying Questions", add Phase 2 summary.

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

**[SUB-AGENT: code-architect]** - Runs in separate context, returns options.

### Launch Sub-agent

```
Task tool:
  subagent_type: "code-architect"
  prompt: |
    Design implementation for: [feature from Phase 1]

    Context from Phase 2:
    [paste codebase analysis summary from checkpoint]

    Requirements from Phase 3-4:
    [paste confirmed requirements from checkpoint]

    Provide 2-3 implementation approaches with:
    - Description
    - Files to create/modify
    - Pros and cons
    - Recommendation with reasoning
```

### Expected Output (to main context)

The sub-agent returns:
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

**Store in checkpoint**, present to user, **wait for user to choose**.

**Checkpoint:** Update Current Phase to "6 - Generate Spec", add Phase 5 summary with decision and rationale.

---

## Phase 6: Generate Spec Document

**[AGENT: @spec-gen-agent]** - Generates spec file, returns path.

### Launch Agent

```
@spec-gen-agent

Feature: [feature name from Phase 1]

Inputs:
- Codebase patterns: [from Phase 2 checkpoint]
- Requirements: [from Phase 3-4 checkpoint]
- Architecture: [from Phase 5 checkpoint - chosen option]
```

The agent will:
1. Read the spec-gen skill template
2. Generate spec at: `spec/[feature-name-kebab-case].md`
3. Set status to "Draft"
4. Return the file path

### Expected Output (to main context)

```
Spec generated: spec/[feature-name].md
Status: Draft
```

**Checkpoint:** Update Current Phase to "7 - Approve Spec", add Phase 6 summary with spec file path.

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

**Checkpoint:** Update Current Phase to "9 - Documentation", add Phase 8 summary with implementation status and completed layers checklist.

---

## Phase 9: Documentation

**[AGENTS: @docs-agent + @postman-agent]** - Updates docs and Postman, returns summary.

### Step 1: Update Project Documentation

```
@docs-agent

Feature: [feature name]
Spec file: [from Phase 6 checkpoint]
Files changed: [from Phase 8 - list files created/modified]
```

The agent will:
1. Check and propose updates to project docs (CLAUDE.md, docs/, .env.example)
2. Ask for approval before making changes
3. Return summary of updates

### Step 2: Update Postman Collection

```
@postman-agent

Feature: [feature name]
Spec file: [from Phase 6 checkpoint]
Operation: Add endpoints from spec
```

The agent will:
1. Extract API endpoints from spec
2. Add to Postman collection
3. Return summary of endpoints added

### Expected Output (to main context)

```markdown
## Documentation Summary

### Files Updated
- `CLAUDE.md`: Added [section] for [feature]
- `.env.example`: No changes needed

### Postman
- Added 3 endpoints to collection
```

**Checkpoint:** Update Current Phase to "10 - Review & PR", add Phase 9 summary.

---

## Phase 10: Review & PR

This is the **final phase** that covers quality review, PR creation, and CI review handling.

### Step 1: Local Code Review Loop (Iterative)

**[SUB-AGENT: code-reviewer]** - Reviews code, returns issues list.

**IMPORTANT:** This step repeats until the code review passes with no critical/high issues.

```
┌─────────────────────────────────────┐
│  Run code-reviewer sub-agent        │
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

### Launch Sub-agent

```
Task tool:
  subagent_type: "code-reviewer"
  prompt: |
    Review implementation for: [feature name]

    Files to review: [list from Phase 8 checkpoint]
    Spec file: [from Phase 6 checkpoint]

    Check for:
    - Bugs and logic errors
    - Security vulnerabilities
    - Code quality (DRY, simplicity)
    - Adherence to project conventions (CLAUDE.md)
    - Proper error handling

    Return issues list with severity and file:line references.
    Only report issues with ≥80% confidence.
```

### Expected Output (to main context)

The sub-agent returns:
```markdown
## Code Review Results

### Issues Found
- [Critical] Missing error check - `internal/handler/user.go:45`
- [High] SQL injection risk - `internal/repository/user.go:123`

### Overall: Needs Changes
```

### Fix Issues (in main context)

If issues found, launch `@pr-fix-agent` in **Local mode**:

```
@pr-fix-agent

Mode: Local
Review output: [paste code-reviewer output above]
```

The agent will:
1. Present each issue with context
2. Wait for user decision: Fix / Skip / Need context
3. Apply fixes after approval
4. **Re-run code-reviewer sub-agent** to verify
5. Repeat until clean

### Step 2: Update Spec Status

After local review passes, update the spec document:

1. Change `**Status**: Approved` to `**Status**: Implemented`
2. Mark all implementation checklist items as complete (`[x]`)
3. Confirm: "Spec status updated to 'Implemented'."

### Step 3: Commit, Push & Create PR

**Ask the user**: "Would you like me to commit, push, and create a PR?"

If yes, run `/commit-push-pr` which will:
1. Create a new branch if on main
2. Commit with conventional commit message
3. Push to remote
4. Create PR via `gh pr create`

### Step 4: Address CI Review Comments (Iterative)

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

When CI review comments arrive, launch `@pr-fix-agent` in **Remote mode**:

```
@pr-fix-agent

Mode: Remote
PR: #[number]
```

The agent will:
1. Fetch comments from the PR
2. Present each comment with context
3. Wait for user decision: Fix / Skip / Need more context
4. Apply fixes, commit, push
5. Wait for CI to re-run
6. Repeat until CI review passes

### Step 5: Archive Checkpoint & Summary

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

### Documentation (Phase 9)
- Files Updated: [list or "No updates needed"]
- Postman: [Added N endpoints / Skipped]

### Local Review
- Iterations: [N]
- Issues Fixed: [list or N/A]

### PR
- PR: #[number] - [title]
- URL: [link]

### CI Review
- Iterations: [N]
- Issues Fixed: [list or N/A]
- Status: Passed
```

**Checkpoint:** Mark Phase 10 complete. Archive checkpoint file.

---

## Guidelines

- Be thorough but concise
- Use existing codebase conventions (camelCase for JSON fields)
- Reference existing patterns discovered during exploration
- Keep specs focused on WHAT, not HOW (until architecture phase)
- Always wait for user approval at key decision points (Phase 4, Phase 5, Phase 7)
- Use specialized agents for exploration, architecture, and review
- Confidence threshold for code review issues: only report issues with ≥80% confidence
- Phase 10 loops are iterative - keep running until clean (local review, CI review)
