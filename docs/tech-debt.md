# Technical Debt Registry

**Last Updated:** 2025-12-23

This document tracks known technical debt in the codebase. Items are prioritized and categorized to help guide future improvements.

## Summary

| Priority | Count | Categories                                       |
| -------- | ----- | ------------------------------------------------ |
| High     | 0     | -                                                |
| Medium   | 7     | Specs (2), Code (1), Tests (2), CI (2)           |
| Low      | 4     | Specs (2), Code (2)                              |

---

## High Priority

_No high priority items._

---

## Medium Priority

### 1. Open Questions in Logout API Spec

| Attribute         | Value                                                                                                                                            |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Category**      | Specs                                                                                                                                            |
| **Location**      | `spec/logout-api.md:323-326`                                                                                                                     |
| **Description**   | Two open questions remain unresolved: (1) Should refresh tokens be rotated on each refresh? (2) Should "logout from all devices" be implemented? |
| **Suggested Fix** | Create backlog tickets for these features. Refresh token rotation improves security; "logout from all devices" is a common user expectation.     |

### 2. Open Questions in Team Spec

| Attribute         | Value                                                                                                                                                         |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | Specs                                                                                                                                                         |
| **Location**      | `spec/team-and-shared-voice-memos.md:765-768`                                                                                                                 |
| **Description**   | Three deferred items: (1) Email notifications for invitations, (2) Background job for cleaning expired invitations, (3) Rate limiting on invitation creation. |
| **Suggested Fix** | Create backlog tickets. Background job for expired invitations should be prioritized to prevent database bloat.                                               |

### 3. Inconsistent ID Validation Pattern

| Attribute         | Value                                                                                                                                                                                                          |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | Code                                                                                                                                                                                                           |
| **Location**      | `internal/service/user_service.go:33-37, 65-68, 83-87`                                                                                                                                                         |
| **Description**   | Voice memo handlers validate IDs in the handler layer (returning 400 for invalid format), but User handlers still validate IDs in the service layer (returning 404). This causes inconsistent error responses. |
| **Suggested Fix** | Update `UserService` methods to accept `primitive.ObjectID` instead of strings. Move `ObjectIDFromHex` parsing to `UserHandler`. Return 400 Bad Request for invalid ID format.                                 |

### 4. Missing Unit Tests for Core Packages

| Attribute         | Value                                                                                                                                                                                                                          |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Category**      | Tests                                                                                                                                                                                                                          |
| **Location**      | `internal/config/`, `internal/database/`, `internal/storage/`, `internal/cache/`, `internal/validator/`, `internal/authz/`                                                                                                     |
| **Description**   | Several core packages lack unit tests: config (parsing, validation), database (connection handling), storage/s3 (pre-signed URLs), cache/redis, validator (custom validators), and authz/local_authorizer (permission checks). |
| **Suggested Fix** | Add unit tests. Priority: (1) config - critical for startup, (2) authz - security-critical, (3) validator - affects all input validation.                                                                                      |

### 5. No Linting in CI Pipeline

| Attribute         | Value                                                                                                                                                                                                                   |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | CI                                                                                                                                                                                                                      |
| **Location**      | `.github/workflows/unit-tests.yml:41`                                                                                                                                                                                   |
| **Description**   | CI runs `go vet` but not `golangci-lint`. While `go vet` catches some issues, `golangci-lint` provides comprehensive static analysis including unused code, complexity metrics, security issues, and style enforcement. |
| **Suggested Fix** | Add `golangci/golangci-lint-action@v4` to CI. Create `.golangci.yml` for project-specific rules.                                                                                                                        |

### 6. No Security Scanning in CI

| Attribute         | Value                                                                                                                                                                             |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | CI                                                                                                                                                                                |
| **Location**      | `.github/workflows/`                                                                                                                                                              |
| **Description**   | No workflow for dependency vulnerability scanning. The project uses external dependencies (MongoDB driver, Redis client, MinIO SDK, JWT library) that could have vulnerabilities. |
| **Suggested Fix** | (1) Enable GitHub Dependabot, (2) Add `govulncheck` to CI, (3) Consider CodeQL analysis workflow.                                                                                 |

### 7. Voice Memo Delete Tests - Partial Coverage

| Attribute         | Value                                                                                                                                                                                                                                                                                                       |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | Tests                                                                                                                                                                                                                                                                                                       |
| **Location**      | `spec/voice-memo-concurrency-control.md:203-215`, `test/api/voice_memos_test.go`, `test/api/team_voice_memos_test.go`                                                                                                                                                                                       |
| **Description**   | The spec checklist shows 10 unchecked items, but **7 tests actually exist**. Missing tests: (1) Delete team memo from wrong team returns 404, (2) Version field increments on delete, (3) UpdatedAt field is set on delete. Existing tests cover: delete success, idempotent delete, not found, forbidden. |
| **Suggested Fix** | Add 3 missing test cases, then update spec checklist to mark all 10 items as complete.                                                                                                                                                                                                                      |

---

## Low Priority

### 8. Create Voice Memo Spec - Checklist Not Updated

| Attribute         | Value                                                                                                                                                                              |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | Specs                                                                                                                                                                              |
| **Location**      | `spec/create-voice-memo.md:330-377`                                                                                                                                                |
| **Description**   | The implementation checklist shows 37 items with `[ ]` rather than `[x]`, but the feature is fully implemented in code. This creates confusion about actual implementation status. |
| **Suggested Fix** | Audit the implementation against the spec checklist and update all completed items to `[x]`.                                                                                       |

### 9. CI Specs - Verification Pending

| Attribute         | Value                                                                                                                                  |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | Specs                                                                                                                                  |
| **Location**      | `spec/unit-test-ci.md:223-226`, `spec/integration-test-ci.md:133-135`, `spec/api-test-ci.md:133-135`                                   |
| **Description**   | Implementation checklists have unchecked items like "Test by opening a PR", "Verify workflow completes", "Verify coverage in Codecov". |
| **Suggested Fix** | Run verification by opening a test PR, then update specs.                                                                              |

### 10. Missing DeleteObject in Storage Interface

| Attribute         | Value                                                                                                                                                         |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | Code                                                                                                                                                          |
| **Location**      | `internal/storage/interface.go:12-19`                                                                                                                         |
| **Description**   | Storage interface lacks `DeleteObject` method. Per delete-voice-memo spec, S3 deletion is intentionally deferred, but will be needed for hard delete feature. |
| **Suggested Fix** | Add `DeleteObject(ctx context.Context, key string) error` when implementing hard delete.                                                                      |

### 11. Hardcoded Configuration Values

| Attribute         | Value                                                                                                                                                         |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Category**      | Code                                                                                                                                                          |
| **Location**      | `internal/service/voice_memo_service.go:19-22`, `internal/service/user_service.go:16`, `cmd/server/main.go:84,96`                                             |
| **Description**   | Several values are hardcoded: presigned URL expiry (1 hour), upload URL expiry (15 minutes), user cache TTL (15 minutes), queue size (100), worker count (2). |
| **Suggested Fix** | Add to `config.go`: `PRESIGNED_URL_EXPIRY`, `UPLOAD_URL_EXPIRY`, `USER_CACHE_TTL`, `TRANSCRIPTION_QUEUE_SIZE`, `TRANSCRIPTION_WORKERS`.                       |

---

## Recommended Action Order

### Short-term (Medium Priority)
1. Complete UserHandler ID validation refactor (#3)
2. Add 3 missing delete tests and update spec (#7)
3. Add `golangci-lint` to CI (#5)
4. Add unit tests for core packages (#4)

### Medium-term
5. Add security scanning to CI (#6)
6. Create backlog tickets for deferred features (#1, #2)

### Long-term (Low Priority)
7. Update spec checklists (#8, #9)
8. Make hardcoded values configurable (#11)
9. Add `DeleteObject` when implementing hard delete (#10)

---

## Changelog

| Date       | Changes                                                                                          |
| ---------- | ------------------------------------------------------------------------------------------------ |
| 2025-12-23 | Fixed #1 (env config) - updated `.env.example` with correct variable names; 11 items remain      |
| 2025-12-23 | Audited all items; merged env issues (#7+#11→#1); verified tests exist for delete; 12 items now  |
| 2025-12-23 | Initial tech debt audit - 14 items identified                                                    |
