# CI/CD Practices and Merge Queue

This document covers CI/CD best practices, GitHub merge queue setup, and strategies for optimizing pipeline performance.

## GitHub Merge Queue

### What is Merge Queue?

Merge queue automates PR merging by queuing PRs and running CI checks on the merged result before actually merging. This prevents "semantic merge conflicts" where two PRs pass CI individually but break when combined.

### How It Works

1. Instead of clicking "Merge", you click "Add to merge queue"
2. GitHub creates a temporary merge commit (your PR + main + any PRs ahead in queue)
3. CI runs on this combined state
4. If CI passes, the PR merges automatically
5. If CI fails, the PR is removed from the queue

### Benefits

- **Prevents broken main**: CI runs on the actual merged state
- **Handles concurrent PRs**: Multiple PRs are tested together
- **Automation**: No manual waiting/merging after CI passes

### How to Enable

1. Go to **Settings** → **Rules** → **Rulesets**
2. Create/edit a rule for `main` branch
3. Enable **"Require merge queue"**
4. Configure merge method and batching options

## The CI Performance Problem

A common issue with merge queue:

```
PR Created → CI (30 min) → Review → Merge Queue → CI again (30 min) → Merged
                                    └── Total: 1+ hour
```

Running the same full test suite twice is wasteful and slows development velocity.

## Solutions and Best Practices

### 1. Tiered Testing (Recommended)

Split tests by speed and run different tiers at different stages:

| Stage | Tests | Target Time |
|-------|-------|-------------|
| PR Created | Lint, unit tests, type check | ~3-5 min |
| Merge Queue | Integration + E2E | ~15-20 min |

```yaml
# .github/workflows/pr-checks.yml
name: PR Checks
on: [pull_request]

jobs:
  fast-checks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go build ./...
      - run: go vet ./...
      - run: go test -short ./...  # Skip long-running tests
```

```yaml
# .github/workflows/merge-queue.yml
name: Merge Queue
on: [merge_group]

jobs:
  full-suite:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go test ./...  # Full test suite
      - run: ./scripts/integration-tests.sh
      - run: ./scripts/e2e-tests.sh
```

### 2. Parallel Test Sharding

Split slow tests across multiple runners:

```yaml
e2e-tests:
  runs-on: ubuntu-latest
  strategy:
    matrix:
      shard: [1, 2, 3, 4]
  steps:
    - run: go test ./... -parallel 4 -run "TestShard${{ matrix.shard }}"
```

**Result**: 30 min → ~8 min with 4 shards

### 3. Merge Queue Batching

Configure merge queue to batch multiple PRs together:

```
PR #1 ─┐
PR #2 ─┼─→ Single CI run → All merge together
PR #3 ─┘
```

Settings:
- **Batch size**: 3-5 PRs
- **Wait time**: 5 minutes (to accumulate PRs)

If batch fails, GitHub bisects to find the culprit.

### 4. Aggressive Caching

#### Go Modules

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.23'
    cache: true  # Caches go modules automatically
```

#### Docker Layer Caching

```yaml
- uses: docker/build-push-action@v5
  with:
    cache-from: type=gha
    cache-to: type=gha,mode=max
```

### 5. Affected Tests Only

Only run tests related to changed files:

```bash
# Get changed packages
CHANGED=$(git diff --name-only origin/main | grep '\.go$' | xargs -I {} dirname {} | sort -u)

# Test only affected packages
go test $CHANGED
```

Tools for monorepos:
- **Nx** (JavaScript/TypeScript)
- **Bazel** (polyglot)
- **Turborepo** (JavaScript/TypeScript)
- **pants** (Python, Go, Java)

### 6. Skip Redundant CI Runs

If PR head commit already passed CI and base hasn't changed:

```yaml
jobs:
  check-skip:
    runs-on: ubuntu-latest
    outputs:
      should-skip: ${{ steps.skip-check.outputs.should_skip }}
    steps:
      - id: skip-check
        uses: fkirc/skip-duplicate-actions@v5
        with:
          concurrent_skipping: 'same_content_newer'
```

## Recommended Setup for This Repo

```
┌─────────────────────────────────────────────────────┐
│ PR Created (fast feedback ~3 min)                   │
│  • go build                                         │
│  • go vet / staticcheck                             │
│  • go test -short ./...                             │
│  • golangci-lint                                    │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│ Review & Approval                                   │
│  • Claude Code Review (automated)                   │
│  • Human review                                     │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│ Merge Queue (thorough validation ~10-15 min)        │
│  • Full test suite (go test ./...)                  │
│  • Integration tests (with test containers)         │
│  • Batching enabled (2-3 PRs together)              │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│ Post-Merge (optional, async)                        │
│  • E2E tests against staging                        │
│  • Performance benchmarks                           │
│  • Security scans                                   │
└─────────────────────────────────────────────────────┘
```

## Quick Wins Checklist

- [ ] Add `on: merge_group` trigger for heavy tests
- [ ] Use `-short` flag for unit tests on PR, full tests in queue
- [ ] Enable merge queue batching (batch_size: 3-5)
- [ ] Cache go modules in CI
- [ ] Add parallel test sharding for slow test suites
- [ ] Configure skip-duplicate-actions for redundant runs

## References

- [GitHub Merge Queue Documentation](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue)
- [GitHub Actions Caching](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- [Go Test Flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags)
