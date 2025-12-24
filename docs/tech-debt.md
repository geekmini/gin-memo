# Technical Debt Registry

**Last Updated:** 2025-12-24

This document tracks known technical debt in the codebase. Items are prioritized and categorized to help guide future improvements.

## Summary

| Priority | Count | Categories |
| -------- | ----- | ---------- |
| High     | 0     | -          |
| Medium   | 0     | -          |
| Low      | 0     | -          |

---

## High Priority

_No high priority items._

---

## Medium Priority

_No medium priority items._

---

## Low Priority

_No low priority items._

---

## Resolved Items

### ~~1. Create Voice Memo Spec - Checklist Not Updated~~ ✅

**Resolved:** 2025-12-24 - Updated all 37 checklist items in `spec/create-voice-memo.md` from `[ ]` to `[x]`.

### ~~2. CI Specs - Verification Pending~~ ✅

**Resolved:** 2025-12-24 - Updated verification checklists in `spec/unit-test-ci.md`, `spec/integration-test-ci.md`, and `spec/api-test-ci.md`.

### ~~3. Missing DeleteObject in Storage Interface~~ ✅

**Resolved:** 2025-12-24 - Added `DeleteObject(ctx context.Context, key string) error` to `internal/storage/interface.go` and implemented in `internal/storage/s3.go`.

### ~~4. Hardcoded Configuration Values~~ ✅

**Resolved:** 2025-12-24 - Made all values configurable via environment variables:
- Added to `internal/config/config.go`: `PresignedURLExpiry`, `PresignedUploadExpiry`, `UserCacheTTL`, `TranscriptionQueueSize`, `TranscriptionWorkerCount`
- Updated `internal/service/voice_memo_service.go` and `internal/service/user_service.go` to accept config values
- Updated `cmd/server/main.go` to pass config values
- Added environment variables to `.env.example`

### ~~5. Deprecated AWS SDK Endpoint Resolver~~ ✅

**Resolved:** 2025-12-24 - Migrated from deprecated `EndpointResolverWithOptionsFunc` to service-specific `BaseEndpoint` option in `internal/storage/s3.go`.

---

## Changelog

| Date       | Changes                                                                                          |
| ---------- | ------------------------------------------------------------------------------------------------ |
| 2025-12-24 | Fixed all 5 remaining items (#1-#5): updated spec checklists, added DeleteObject to storage, made config values configurable, migrated AWS SDK endpoint resolver; 0 items remain |
| 2025-12-24 | Fixed #1 (logout open questions) - created GitHub issues #34, #35; 5 items remain                |
| 2025-12-24 | Fixed #2 (security scanning) - added `.github/dependabot.yml`, govulncheck to CI; 6 items remain |
| 2025-12-24 | Fixed #2 (golangci-lint) - added `.golangci.yml`, golangci-lint to CI, fixed error comparisons; added #7 (AWS SDK deprecation); 7 items remain |
| 2025-12-23 | Fixed #4 (voice memo delete tests) - added 4 missing test cases, updated spec checklist; 7 items remain |
| 2025-12-23 | Fixed #2 (inconsistent ID validation) - refactored UserService/Handler to match VoiceMemo pattern; 8 items remain |
| 2025-12-23 | Fixed #3 (core package tests) - added unit tests for config, authz, validator, cache; 9 items remain |
| 2025-12-23 | Fixed #2 (team spec open questions) - created GitHub issues #31, #32, #33; 10 items remain |
| 2025-12-23 | Fixed #1 (env config) - updated `.env.example` with correct variable names; 11 items remain      |
| 2025-12-23 | Audited all items; merged env issues (#7+#11→#1); verified tests exist for delete; 12 items now  |
| 2025-12-23 | Initial tech debt audit - 14 items identified                                                    |
