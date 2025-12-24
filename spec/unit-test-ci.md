# Unit Test CI Workflow Specification

**Author**: Team
**Created**: 2025-12-22
**Status**: Approved
**Architecture**: Single-Job Sequential Workflow

## Overview

A GitHub Actions workflow that runs unit tests with race detection and uploads coverage to Codecov on every pull request targeting the main branch.

## Architecture Decision

**Chosen Approach**: Single-Job Sequential Workflow
**Rationale**: Simple, follows existing workflow patterns in the repository, minimal GitHub Actions minutes cost, no multi-version testing needed.

### Files to Create
- `.github/workflows/unit-tests.yml` - GitHub Actions workflow definition

### Files to Modify
- None

## Workflow Configuration

### Trigger

| Property | Value |
| -------- | ----- |
| Event | `pull_request` |
| Branches | `main` |
| Types | `opened`, `synchronize`, `reopened` |
| Path Filter | `**.go`, `go.mod`, `go.sum` |

### Environment

| Property | Value |
| -------- | ----- |
| Runner | `ubuntu-latest` |
| Go Version | `1.25` |
| Caching | Enabled via `actions/setup-go@v5` |

### Secrets Required

| Secret | Description |
| ------ | ----------- |
| `CODECOV_TOKEN` | Token for uploading coverage reports to Codecov |

## Workflow Steps

### Step 1: Checkout Code

**Action**: `actions/checkout@v4`
**Purpose**: Clone repository at PR HEAD

```yaml
- name: Checkout code
  uses: actions/checkout@v4
  with:
    fetch-depth: 1
```

### Step 2: Set up Go

**Action**: `actions/setup-go@v5`
**Purpose**: Install Go and enable module caching

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.25'
    cache: true
    cache-dependency-path: go.sum
```

### Step 3: Download Dependencies

**Command**: `go mod download`
**Purpose**: Download and verify all modules

```yaml
- name: Download dependencies
  run: go mod download
```

### Step 4: Build

**Command**: `go build -v ./...`
**Purpose**: Compile all packages, catch syntax/import errors

```yaml
- name: Build
  run: go build -v ./...
```

### Step 5: Run go vet

**Command**: `go vet ./...`
**Purpose**: Static analysis for common Go mistakes

```yaml
- name: Run go vet
  run: go vet ./...
```

### Step 6: Run Tests

**Command**: `go test -v -short -race -coverprofile=coverage.out -covermode=atomic ./...`
**Purpose**: Execute unit tests with race detection and coverage

```yaml
- name: Run tests with race detection
  run: go test -v -short -race -coverprofile=coverage.out -covermode=atomic ./...
```

**Flags**:
| Flag | Purpose |
| ---- | ------- |
| `-v` | Verbose output |
| `-short` | Skip long-running tests |
| `-race` | Enable race detector |
| `-coverprofile` | Generate coverage report |
| `-covermode=atomic` | Thread-safe coverage counting |

### Step 7: Upload Coverage

**Action**: `codecov/codecov-action@v4`
**Purpose**: Upload coverage report to Codecov

```yaml
- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v4
  with:
    files: ./coverage.out
    token: ${{ secrets.CODECOV_TOKEN }}
    fail_ci_if_error: false
    verbose: true
```

## Complete Workflow Definition

```yaml
name: Unit Tests

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
  test:
    name: Test and Coverage
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

      - name: Build
        run: go build -v ./...

      - name: Run go vet
        run: go vet ./...

      - name: Run tests with race detection
        run: go test -v -short -race -coverprofile=coverage.out -covermode=atomic ./...

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          token: ${{ secrets.CODECOV_TOKEN }}
          fail_ci_if_error: false
          verbose: true
```

## Behavior

### Success Path
1. PR opened/updated with Go file changes
2. Workflow triggers
3. All steps pass
4. Coverage uploaded to Codecov
5. PR check shows green checkmark

### Failure Scenarios

| Failure | Step | Resolution |
| ------- | ---- | ---------- |
| Compilation error | Build | Fix syntax/import errors |
| Vet issues | go vet | Fix reported issues |
| Test failure | Run tests | Fix failing tests |
| Race condition | Run tests | Fix data race with sync primitives |
| Codecov upload fails | Upload coverage | Check token, workflow continues |

### Path Filter Behavior
- Go file changes: Workflow runs
- Docs-only changes: Workflow skipped (shows as "skipped" not "passed")

## Implementation Steps

1. [x] Create `.github/workflows/unit-tests.yml` with workflow definition
2. [x] Verify `CODECOV_TOKEN` secret exists in repository settings
3. [x] Test by opening a PR with Go code changes
4. [x] Verify workflow completes successfully
5. [x] Verify coverage appears in Codecov

## Out of Scope

- Integration tests (separate workflow)
- Multi-Go-version matrix testing
- Linting (golangci-lint)
- Performance benchmarks
- Merge queue workflow

## Open Questions

- [x] Go version to use? → 1.25
- [x] Include race detection? → Yes
- [x] Coverage service? → Codecov
- [x] Path filtering? → Yes
