# Flaky Test Detection Strategy

A practical, cost-effective approach to detecting flaky tests in CI pipelines.

## Overview

Flaky tests are tests that produce inconsistent results (pass/fail) without code changes. They erode confidence in the test suite and slow down development.

This document describes a **combined approach** that balances detection effectiveness with CI cost.

## Strategy Overview

```
┌─────────────────────────────────┐
│  PR Opened                      │
└────────────────┬────────────────┘
                 ▼
┌─────────────────────────────────┐
│  Stage 1: Static Analysis       │  ← Pattern detection (no test runs)
│  AI/linter checks for anti-     │
│  patterns that cause flakiness  │
└────────────────┬────────────────┘
                 ▼
┌─────────────────────────────────┐
│  Stage 2: Run Tests Once        │  ← Normal CI cost
└────────────────┬────────────────┘
                 │
         ┌───────┴───────┐
         │   Failed?     │
         └───────┬───────┘
                 ▼ Yes
┌─────────────────────────────────┐
│  Stage 3: Retry Failed Only     │  ← Minimal extra cost
│  Log tests that pass on retry   │
└────────────────┬────────────────┘
                 ▼
┌─────────────────────────────────┐
│  Stage 4: Aggregate & Report    │  ← Scheduled (weekly)
│  Track flaky candidates, create │
│  issues for persistent ones     │
└─────────────────────────────────┘
```

## Stage 1: Static Analysis

Detect flaky test patterns **before** running tests. This is the cheapest and most proactive approach.

### Anti-Patterns to Detect

| Pattern | Risk | Description |
|---------|------|-------------|
| Fixed delays | High | Using sleep/wait without polling for condition |
| Missing cleanup | High | Resources not cleaned up between tests |
| Shared mutable state | High | Global variables modified by tests |
| Hardcoded ports/paths | Medium | Port conflicts when tests run in parallel |
| Unsynchronized concurrency | High | Async operations without proper synchronization |
| Time-sensitive assertions | Medium | Assertions on timestamps or durations |
| External dependencies | Medium | Network calls, file I/O without mocking |
| Order-dependent tests | High | Tests that assume execution order |

### Example Static Analysis Prompt

For AI-based code review (Claude, GPT, etc.):

```
Analyze test files for flaky test patterns:

1. TIMING ISSUES
   - Sleep/delay without polling or condition checking
   - Hardcoded timeouts that may be too short
   - Assertions on elapsed time with tight tolerances

2. STATE MANAGEMENT
   - Missing cleanup/teardown for created resources
   - Shared variables without synchronization
   - Global state modifications without reset

3. CONCURRENCY
   - Async operations without wait mechanisms
   - Race conditions between test threads
   - Uncontrolled parallel execution

4. EXTERNAL DEPENDENCIES
   - Hardcoded ports, URLs, or file paths
   - Network calls without retry/timeout
   - Assumptions about external service state

5. ORDER DEPENDENCY
   - Tests relying on other tests' side effects
   - Assumptions about execution order
   - Shared database/cache state

Output format:
- File:Line - Risk (High/Medium/Low) - Pattern - Suggested fix
```

### Linter Rules

Many languages have linters that can detect flaky patterns:

| Language | Tool | Relevant Rules |
|----------|------|----------------|
| Any | Custom regex | `sleep\(`, `Thread.sleep`, `time.sleep` |
| JavaScript | ESLint | `no-await-in-loop`, custom rules |
| Python | pylint | Custom plugins for test patterns |
| Java | ErrorProne | `ThreadSleep`, `FlakyTest` |
| Go | staticcheck | Race detector integration |

## Stage 2: Run Tests Once

Standard test execution - no additional cost.

### Best Practices

- Use test randomization (shuffle order) to expose order dependencies
- Enable race/thread sanitizers when available
- Set reasonable timeouts to catch hanging tests
- Output results in machine-readable format (JSON, JUnit XML)

## Stage 3: Retry Failed Tests

Only retry tests that failed - minimal additional cost.

### Implementation

```yaml
# Pseudocode for CI workflow

steps:
  - name: Run tests
    id: initial_test
    run: |
      run_tests --output=json > results.json
    continue_on_error: true

  - name: Retry failed tests
    if: steps.initial_test.outcome == 'failure'
    run: |
      # Extract failed test names from results
      failed_tests=$(parse_failures results.json)

      for test in $failed_tests; do
        echo "Retrying: $test"

        if run_single_test "$test"; then
          # Test passed on retry = flaky candidate
          echo "FLAKY CANDIDATE: $test"
          echo "$test" >> flaky-candidates.txt
        else
          # Test failed again = likely real failure
          echo "CONFIRMED FAILURE: $test"
        fi
      done

  - name: Upload flaky candidates
    run: upload_artifact flaky-candidates.txt
```

### Retry Decision Matrix

| Initial | Retry 1 | Classification |
|---------|---------|----------------|
| Pass | - | Stable |
| Fail | Pass | Flaky candidate |
| Fail | Fail | Real failure |

## Stage 4: Aggregate and Report

Track flaky tests over time to identify persistent issues.

### Flakiness Score

```
Flakiness Rate = (Inconsistent Runs / Total Runs) × 100%
```

| Rate | Action |
|------|--------|
| < 1% | Monitor |
| 1-5% | Investigate |
| > 5% | Quarantine or fix immediately |

### Tracking Options

**Option A: Simple file-based tracking**

Store flaky candidates in a file in the repository:

```
# .github/flaky-tests.txt
# Format: test_name | first_seen | occurrence_count | last_seen
TestUserCreation | 2024-01-15 | 3 | 2024-01-20
TestPaymentFlow | 2024-01-18 | 7 | 2024-01-22
```

**Option B: GitHub Issues**

Create issues automatically for persistent flaky tests:

```yaml
- name: Create issue for flaky tests
  if: steps.retry.outputs.flaky_count > 0
  run: |
    gh issue create \
      --title "Flaky test detected: $TEST_NAME" \
      --label "flaky-test" \
      --body "Test $TEST_NAME passed on retry. Detected $COUNT times."
```

**Option C: External services**

- BuildPulse
- Datadog CI Visibility
- Codecov Test Analytics
- Allure TestOps

### Weekly Aggregation Job

```yaml
# Scheduled job to summarize flaky tests
name: Flaky Test Report
on:
  schedule:
    - cron: '0 9 * * 1'  # Every Monday at 9 AM

jobs:
  report:
    runs-on: ubuntu-latest
    steps:
      - name: Aggregate flaky test data
        run: |
          # Collect from past week's CI runs
          # Generate summary report

      - name: Post to Slack/Email
        run: |
          # Send notification with top flaky tests
```

## Quarantine Strategy

For tests with high flakiness rate, quarantine them to prevent blocking PRs:

### Quarantine Process

1. **Identify**: Flakiness rate > 5% over 2 weeks
2. **Tag**: Add quarantine tag/marker to test
3. **Separate**: Run quarantined tests in non-blocking job
4. **Track**: Create issue with deadline to fix
5. **Fix**: Address root cause
6. **Restore**: Remove quarantine tag, monitor

### Implementation

```python
# Example: Mark test as quarantined
@pytest.mark.quarantine
def test_flaky_payment():
    ...

# Example: Skip quarantined in main CI
@pytest.mark.skipif(os.getenv('CI') == 'true', reason="Quarantined")
def test_flaky_payment():
    ...
```

## Cost Comparison

| Approach | CI Minutes | Effectiveness |
|----------|------------|---------------|
| Run every test 10x | 10x baseline | High but expensive |
| Run every test 3x | 3x baseline | Medium |
| **Combined approach** | ~1.1x baseline | High |
| Static analysis only | No extra | Medium (preventive) |

The combined approach achieves high effectiveness at minimal cost by:
- Catching patterns before they become flaky tests (Stage 1)
- Only retrying failed tests, not all tests (Stage 3)
- Aggregating data over time rather than running multiple times (Stage 4)

## Checklist for Implementation

- [ ] Set up static analysis for flaky patterns in PR review
- [ ] Configure test output in machine-readable format
- [ ] Implement retry-on-failure with logging
- [ ] Create storage for flaky test tracking (file, issues, or service)
- [ ] Set up weekly aggregation and reporting
- [ ] Define quarantine process and thresholds
- [ ] Document how developers should handle flaky test alerts

## References

- [Google Testing Blog: Flaky Tests](https://testing.googleblog.com/2016/05/flaky-tests-at-google-and-how-we.html)
- [Spotify: Test Flakiness](https://engineering.atspotify.com/2019/11/test-flakiness-methods-for-identifying-and-dealing-with-flaky-tests/)
- [GitHub Actions: Retry Action](https://github.com/nick-fields/retry)
