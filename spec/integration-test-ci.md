# Integration Test CI Pipeline

**Status**: Implemented

## Overview

Add a GitHub Actions workflow to run repository-level integration tests on pull requests. This workflow mirrors the existing unit test workflow for consistency.

## Requirements

- Run repository integration tests (`internal/repository/...`) on PRs to main
- Use testcontainers for MongoDB (no external services needed)
- Block PR merge on failure (required check)
- Upload coverage to Codecov

## Architecture Decision

**Option A: Mirror Unit Test Workflow** - Selected for consistency and simplicity.

## Files to Create

| File | Description |
|------|-------------|
| `.github/workflows/integration-tests.yml` | Integration test workflow |

## Files to Modify

| File | Change |
|------|--------|
| `CLAUDE.md` | Add integration test CI documentation |

## Workflow Specification

### Triggers

```yaml
on:
  pull_request:
    branches:
      - main
    types:
      - opened
      - synchronize
      - reopened
    paths:
      - '**.go'
      - 'go.mod'
      - 'go.sum'
```

### Job: Integration Tests

| Step | Command/Action |
|------|----------------|
| Checkout | `actions/checkout@v4` |
| Setup Go | `actions/setup-go@v5` with Go 1.25 |
| Download dependencies | `go mod download` |
| Run integration tests | `go test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/repository/...` |
| Upload coverage | `codecov/codecov-action@v4` |

### Full Workflow YAML

```yaml
name: Integration Tests

on:
  pull_request:
    branches:
      - main
    types:
      - opened
      - synchronize
      - reopened
    paths:
      - '**.go'
      - 'go.mod'
      - 'go.sum'

jobs:
  integration-test:
    name: Integration Tests
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 1

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache: true
          cache-dependency-path: go.sum

      - name: Download dependencies
        run: go mod download

      - name: Run integration tests
        run: go test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/repository/...

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          flags: integration
          token: ${{ secrets.CODECOV_TOKEN }}
          fail_ci_if_error: false
          verbose: true
```

## Documentation Updates

### CLAUDE.md Changes

Add to "Continuous Integration" section:

```markdown
### Integration Tests

Integration tests run automatically on every PR via GitHub Actions:
- **Workflow**: `.github/workflows/integration-tests.yml`
- **Triggers**: PR to main (only on Go file changes)
- **Scope**: Repository tests (`internal/repository/...`)
- **Services**: MongoDB via testcontainers (auto-managed)
- **Steps**: Checkout → Setup Go → Download deps → Run tests → Codecov upload
```

## Implementation Checklist

- [x] Create `.github/workflows/integration-tests.yml`
- [x] Update `CLAUDE.md` documentation
- [ ] Verify workflow triggers correctly on test PR

## Out of Scope

- API integration tests (`test/api/...`)
- Manual/scheduled triggers
- Retry logic for flaky tests
