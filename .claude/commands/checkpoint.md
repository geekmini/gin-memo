---
description: Check dev session status and manage checkpoint
---

# Checkpoint Command

Check the current dev session status and manage the checkpoint file.

## Your Task

**Step 1: Check if checkpoint file exists**

```bash
ls .claude/dev-checkpoint.md
```

**Step 2: If checkpoint does NOT exist**

Report: "No active dev session found. Use `/dev` to start a new feature development session."

Stop here.

**Step 3: If checkpoint exists, read and analyze it**

Read `.claude/dev-checkpoint.md` and check the **Current Phase** field.

### Case A: Session is COMPLETED

If `Current Phase` contains "COMPLETED":

1. Show summary:
   ```
   Dev session completed:
   - Feature: [feature name from title]
   - Status: COMPLETED
   - Last Updated: [timestamp]
   ```

2. Ask user: "Would you like to archive and cleanup this completed session?"

3. If user confirms:
   - Create `.claude/dev-checkpoints/` directory if it doesn't exist
   - Move `.claude/dev-checkpoint.md` to `.claude/dev-checkpoints/[feature-name]-[timestamp].md`
   - Confirm: "Session archived to `.claude/dev-checkpoints/[feature-name]-[timestamp].md`"

4. If user declines:
   - Confirm: "Checkpoint preserved. Run `/checkpoint` again when ready to cleanup."

### Case B: Session is IN PROGRESS

If `Current Phase` is any phase from 1-10 (not completed):

1. Show current status:
   ```
   Active dev session found:
   - Feature: [feature name from title]
   - Current Phase: [phase number] - [phase name]
   - Last Updated: [timestamp]

   Completed Phases:
   - [list completed phases from checkpoint]

   Remaining Phases:
   - [list remaining phases until Phase 10]
   ```

2. Ask user with options:
   ```
   What would you like to do?
   1. Continue this session (resume from current phase)
   2. Abandon session (archive without completing)
   3. Keep checkpoint (do nothing)
   ```

3. Handle user choice:
   - **Continue**: Say "Run `/dev` to resume from Phase [N]. The session context will be loaded automatically."
   - **Abandon**: Archive to `.claude/dev-checkpoints/[feature-name]-abandoned-[timestamp].md` and delete checkpoint
   - **Keep**: Confirm "Checkpoint preserved."

## Phase Reference

| Phase | Name |
|-------|------|
| 1 | Discovery |
| 2 | Codebase Exploration |
| 3 | Clarifying Questions |
| 4 | Summary & Approval |
| 5 | Architecture Design |
| 6 | Generate Spec |
| 7 | Approve Spec |
| 8 | Implementation |
| 9 | Documentation |
| 10 | Review & PR |
| COMPLETED | All phases finished |
