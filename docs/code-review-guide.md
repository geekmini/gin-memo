# Code Review Guide

Based on Google Engineering Practices.

## Core Principle

> **Focus on whether the PR improves overall code health.**

Seek continuous improvement, not perfection. There is no "perfect" code—only better code.

## What to Review

### Design
- Is the architecture sound and appropriate for the system?
- Does this change belong in this codebase, or a library?
- Does it integrate well with the rest of the system?

### Functionality
- Does the code do what the author intended?
- Are edge cases handled correctly?
- Are there potential concurrency issues (race conditions, deadlocks)?

### Complexity
- Can the code be understood quickly by other developers?
- Is it over-engineered for the current requirements?
- Will developers introduce bugs when modifying this code?

### Tests
- Are there appropriate unit, integration, or end-to-end tests?
- Will tests fail when the code breaks?
- Are edge cases covered?

### Naming
- Are names clear and descriptive?
- Do they follow project conventions (see CLAUDE.md)?

### Comments
- Do comments explain *why*, not *what*?
- Is the code clear enough to be self-documenting where possible?

### Style
- Does the code follow CLAUDE.md conventions?
- Is it consistent with the surrounding codebase?

### Documentation
- Are docs updated for user-facing changes?
- Are API changes reflected in relevant documentation?

## Comment Guidelines

### Focus on Code, Not the Author

```
# Bad
"Why did you use a global variable here?"

# Good
"Using a global variable here could cause issues in concurrent scenarios.
Consider passing the state explicitly."
```

### Explain Your Reasoning

```
# Bad
"Use a map instead of a slice."

# Good
"A map would give O(1) lookup here instead of O(n), which matters
since this runs in a hot path during request handling."
```

### Label Comments by Severity

| Label | Meaning |
|-------|---------|
| `Blocking:` | Must be fixed - bugs, security issues, design flaws |
| `Suggestion:` | Recommended improvement |
| `Nit:` | Minor polish, optional |

### Acknowledge Good Practices

Call out what's done well: clean abstractions, good test coverage, clear naming.

## Output Format

```markdown
## Summary
[One-line assessment of the PR]

## Strengths
- [Specific things done well]

## Feedback
- **Blocking:** [Critical issues]
- **Suggestion:** [Recommended improvements]
- **Nit:** [Minor polish items]
```
