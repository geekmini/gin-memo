---
allowed-tools: Bash(git:*), Bash(gh:*), Read, Edit, Grep, Glob
description: Fix review comments from local code review or GitHub PR
---

## Context

- Current branch: !`git branch --show-current`
- PR status (if exists): !`gh pr view --json number,title,state,url 2>/dev/null || echo "No PR found for current branch"`
- Recent PR comments: !`gh pr view --json comments --jq '.comments[-3:] | .[] | "- \(.author.login): \(.body | split("\n")[0])"' 2>/dev/null || echo "No comments"`

## Your task

Address review comments using `@pr-fix-agent`.

**Modes:**
1. **Remote mode** (default): Fetch and fix comments from the current branch's GitHub PR
2. **Local mode**: If user provides code review output, process those issues instead

**Critical:** The agent will present issues one at a time and wait for your decision on each.

Launch `@pr-fix-agent` now to process review comments.
