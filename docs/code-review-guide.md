# Code Review Guide

Based on Google Engineering Practices.

## Core Principle

> **Approve a PR if it improves overall code health, even if it isn't perfect.**

Seek continuous improvement, not perfection. Don't block PRs for minor issues. There is no "perfect" code—only better code.

## What to Review

### Design
- Is the architecture sound and appropriate for the system?
- Does this change belong in this codebase, or a library?
- Does it integrate well with the rest of the system?
- Is now the right time to add this functionality?

### Functionality
- Does the code do what the author intended?
- Is the behavior good for users (both end-users and developers)?
- Are edge cases handled correctly?
- Are there potential concurrency issues (race conditions, deadlocks)?

### Complexity
- Can the code be understood quickly by other developers?
- Is it over-engineered for the current requirements?
- Are there unnecessary abstractions or premature optimizations?
- Will developers introduce bugs when modifying this code?

### Tests
- Are there appropriate unit, integration, or end-to-end tests?
- Are tests correct and actually testing the intended behavior?
- Will tests fail when the code breaks?
- Are edge cases covered?

### Naming
- Are names clear and descriptive?
- Do they communicate intent without being overly long?
- Do they follow project conventions (see CLAUDE.md)?

### Comments
- Do comments explain *why*, not *what*?
- Is the code clear enough to be self-documenting where possible?
- Are TODOs tracked with issue references?

### Style
- Does the code follow CLAUDE.md conventions?
- Is it consistent with the surrounding codebase?
- Are style-only changes separate from functional changes?

### Documentation
- Are docs updated for user-facing changes?
- Are API changes reflected in relevant documentation?
- Is the PR description clear and complete?

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

Help the author understand *why* you're suggesting a change:

```
# Bad
"Use a map instead of a slice."

# Good
"A map would give O(1) lookup here instead of O(n), which matters
since this runs in a hot path during request handling."
```

### Label Comments by Severity

| Label | Meaning | Author Action |
|-------|---------|---------------|
| `Blocking:` | Must be fixed before approval | Required |
| `Suggestion:` | Recommended improvement | Strongly encouraged |
| `Nit:` | Minor polish, stylistic preference | Optional |
| `Question:` | Clarification needed | Please explain |
| `FYI:` | Educational, for future reference | No action needed |

### Acknowledge Good Practices

Don't only point out problems. Call out what's done well:

- Clean abstractions
- Good test coverage
- Clear naming
- Thoughtful error handling
- Well-structured commits

## Decision Criteria

### Approve When
- Code improves overall system health
- Issues found are minor (`Nit` or `Suggestion` level)
- You're confident the author will address remaining comments
- Changes are low-risk and well-tested

### Request Changes When
- There are blocking issues (bugs, security vulnerabilities, design flaws)
- The code would degrade system health
- Critical tests are missing
- The design needs fundamental rethinking

### LGTM with Comments

You can approve while leaving unresolved comments when:
- You trust the author to address them appropriately
- The suggestions are minor improvements
- Waiting would significantly delay the author (e.g., timezone differences)

## Handling Disagreements

1. **Seek consensus** - Discuss the technical tradeoffs
2. **Focus on principles** - Reference this guide or CLAUDE.md
3. **Consider the data** - Let technical facts guide decisions
4. **Escalate if needed** - Don't let PRs sit blocked indefinitely

If the author disagrees with your feedback:
- They may be closer to the code and have better insight
- Ask for their reasoning before insisting
- Be willing to be wrong

## Review Checklist

Before approving, verify:

- [ ] Code compiles and tests pass
- [ ] Design is sound and fits the system
- [ ] Functionality works correctly with edge cases handled
- [ ] No obvious bugs or security issues
- [ ] Appropriate test coverage exists
- [ ] Code is readable and maintainable
- [ ] Style follows project conventions
- [ ] Documentation is updated if needed

## Output Format for Automated Reviews

Structure automated reviews as:

```markdown
## Summary
[One-line assessment of the PR]

## Strengths
- [Specific things done well]

## Feedback
- **Blocking:** [Critical issues that must be fixed]
- **Suggestion:** [Recommended improvements]
- **Nit:** [Minor polish items]

## Verdict
[Approve / Request Changes] - [Brief reasoning]
```

