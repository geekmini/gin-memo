# Testcontainers vs Docker Compose for Testing

## Overview

This document compares two approaches for running integration tests with external dependencies (databases, caches, etc.).

## Comparison

| Aspect | Testcontainers | Docker Compose |
|--------|---------------|----------------|
| **Lifecycle** | Per-test or per-suite | Shared across all tests |
| **Isolation** | Each test gets fresh container | All tests share same containers |
| **State** | Clean slate every time | State accumulates between tests |
| **Port conflicts** | Random ports, no conflicts | Fixed ports, potential conflicts |
| **Parallelization** | Safe parallel execution | Risky - shared state |

## Test Isolation

### Docker Compose Problem

```
Test A: Creates user "john@test.com"
Test B: Creates user "john@test.com"  ← FAILS (duplicate key)
Test C: Expects empty DB              ← FAILS (has data from A, B)
```

### Testcontainers Solution

```
Test A: Fresh MongoDB → Creates user → Container destroyed
Test B: Fresh MongoDB → Creates user → Works fine
Test C: Fresh MongoDB → Empty as expected
```

## Flaky Tests

| Cause | Docker Compose | Testcontainers |
|-------|----------------|----------------|
| **Leftover data** | Common - previous test data | None - fresh container |
| **Port conflicts** | Possible (fixed ports) | Impossible (random ports) |
| **Test order dependency** | High risk | No risk |
| **Parallel test interference** | High risk | No risk |
| **"Works locally, fails in CI"** | Common | Rare |

## Code Examples

### Docker Compose Approach

Requires manual cleanup to avoid state leakage:

```go
func TestCreateUser(t *testing.T) {
    // Must manually clean up before/after
    db.Collection("users").DeleteMany(ctx, bson.M{})  // Cleanup needed!

    // ... test code

    // Cleanup again to not affect other tests
    db.Collection("users").DeleteMany(ctx, bson.M{})
}
```

### Testcontainers Approach

Automatic isolation, no cleanup needed:

```go
func TestCreateUser(t *testing.T) {
    container := mongodb.Run(ctx, "mongo:7")  // Fresh container
    defer container.Terminate(ctx)             // Auto cleanup

    // ... test code - no cleanup needed, container is destroyed
}
```

## Trade-offs

| Aspect | Testcontainers | Docker Compose |
|--------|----------------|----------------|
| **Speed** | Slower (container startup per test/suite) | Faster (reuse running containers) |
| **Reliability** | Higher (isolation) | Lower (shared state) |
| **CI simplicity** | Just run `go test` | Need `docker-compose up` first |
| **Local dev** | No pre-setup needed | Must run `docker-compose up` |
| **Resource usage** | Higher (multiple containers) | Lower (shared containers) |

## When to Use Each

### Use Testcontainers When

- Running tests in CI/CD pipelines
- Test isolation is critical
- Tests need to run in parallel
- You want reproducible test results
- Different tests need different database states

### Use Docker Compose When

- Local development with long-running services
- Manual/exploratory testing
- Performance testing (avoid container startup overhead)
- Services need to communicate with each other

## Project Usage

This project uses testcontainers for API tests (`test/api/...`):

```
test/api/testdb/
├── mongo.go   # MongoDB testcontainer
├── redis.go   # Redis testcontainer
└── minio.go   # MinIO testcontainer (S3-compatible)
```

Each test suite gets fresh containers, ensuring complete isolation.

## Conclusion

Testcontainers trades some execution speed for deterministic, non-flaky tests. The isolation guarantee is worth the extra startup time, especially in CI where flaky tests waste developer time debugging phantom failures.
