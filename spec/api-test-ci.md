# API Test CI Pipeline

**Status**: Implemented

## Overview

Add a GitHub Actions workflow to run full API integration tests on pull requests. These tests exercise the complete application stack with real MongoDB, Redis, and MinIO services via testcontainers.

## Requirements

- Run API tests (`test/api/...`) with `-tags=api` on PRs to main
- Use testcontainers for MongoDB, Redis, MinIO (auto-managed)
- Block PR merge on failure (required check)
- Upload coverage to Codecov with `api` flag

## Architecture Decision

Mirror existing `integration-tests.yml` pattern for consistency.

## Files to Create

| File | Description |
|------|-------------|
| `.github/workflows/api-tests.yml` | API test workflow |

## Files to Modify

| File | Change |
|------|--------|
| `CLAUDE.md` | Add API test CI documentation |

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

### Job: API Tests

| Step | Command/Action |
|------|----------------|
| Checkout | `actions/checkout@v4` |
| Setup Go | `actions/setup-go@v5` with Go 1.25 |
| Download dependencies | `go mod download` |
| Run API tests | `go test -tags=api -v -race -coverprofile=coverage.out -covermode=atomic ./test/api/...` |
| Upload coverage | `codecov/codecov-action@v4` with `flags: api` |

### Full Workflow YAML

```yaml
name: API Tests

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
  api-test:
    name: API Tests
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

      - name: Run API tests
        run: go test -tags=api -v -race -coverprofile=coverage.out -covermode=atomic ./test/api/...

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          flags: api
          token: ${{ secrets.CODECOV_TOKEN }}
          fail_ci_if_error: false
          verbose: true
```

## Documentation Updates

### CLAUDE.md Changes

Add to "Continuous Integration" section under "Integration Tests":

```markdown
### API Tests

API tests run automatically on every PR via GitHub Actions:
- **Workflow**: `.github/workflows/api-tests.yml`
- **Triggers**: PR to main (only on Go file changes)
- **Scope**: Full API tests (`test/api/...`) with `-tags=api`
- **Services**: MongoDB, Redis, MinIO via testcontainers (auto-managed)
- **Steps**: Checkout → Setup Go → Download deps → Run tests → Codecov upload
```

## Implementation Checklist

- [x] Create `.github/workflows/api-tests.yml`
- [x] Update `CLAUDE.md` documentation
- [x] Verify workflow triggers correctly on test PR

## Out of Scope

- Manual/scheduled triggers
- Retry logic for flaky tests
- Parallel test execution
