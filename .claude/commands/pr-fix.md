---
allowed-tools: Bash(git:*), Bash(gh:*), Read, Edit, Grep, Glob
description: Fix review comments from local code review or GitHub PR
---

## Context

- Current branch: !`git branch --show-current`
- PR status (if exists): !`gh pr view --json number,title,state,url 2>/dev/null || echo "No PR found for current branch"`
- Recent PR comments: !`gh pr view --json comments --jq '.comments[-3:] | .[] | "- \(.author.login): \(.body | split("\n")[0])"' 2>/dev/null || echo "No comments"`

## Your task

Address review comments using the `pr-fix` skill.

**Modes:**
1. **Remote mode** (default): Fetch and fix comments from the current branch's GitHub PR
2. **Local mode**: If user provides code review output, process those issues instead

**Process:**
1. Identify the source (PR comments or local review output)
2. Fetch all unresolved comments/issues
3. Present each issue ONE AT A TIME
4. Wait for user decision: Fix / Skip / Need context
5. Apply fix only after user approval
6. After all issues processed, show summary

**Critical:** Never batch fix. Always present issues individually and wait for user input.

Use the Skill tool to invoke `pr-fix` now.
