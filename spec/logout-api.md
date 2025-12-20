# Logout API Specification

**Author**: Team
**Created**: 2025-12-20
**Status**: Draft

## Overview

API endpoint to logout users by invalidating all their active sessions. Uses token versioning strategy where each logout increments a version number, causing all previously issued JWTs to become invalid.

## Data Model Changes

### User (Modified)

| Field        | Type | Default | Description                                      |
| ------------ | ---- | ------- | ------------------------------------------------ |
| tokenVersion | int  | 1       | Incremented on logout to invalidate all tokens   |

### JWT Claims (Modified)

| Claim        | Type   | Description                          |
| ------------ | ------ | ------------------------------------ |
| userId       | string | User's ID (existing)                 |
| tokenVersion | int    | Version at time of token generation  |
| exp          | int    | Expiration timestamp (existing)      |

## Storage Strategy

### Dual Storage (Redis + MongoDB)

- **MongoDB**: Persistent storage of `tokenVersion` in User document
- **Redis**: Cache for fast lookups during token validation
  - Key format: `user:{userId}` (existing user cache, add tokenVersion field)
  - TTL: 15 minutes (matches existing user cache TTL)

### Lookup Flow

1. Check Redis for cached `tokenVersion`
2. If cache miss, fetch from MongoDB
3. Cache result in Redis with 15 min TTL

## Endpoints

### Logout

Invalidate all user sessions by incrementing token version.

```
POST /api/v1/auth/logout
```

**Authentication:** Required (Bearer token)

**Request Body:** None

**Response:** `204 No Content`

(Empty response body)

**Behavior:**
- Increment `tokenVersion` in MongoDB
- Update `tokenVersion` in Redis cache
- All existing tokens become invalid (version mismatch)

**Errors:**

| Status | Description                             |
| ------ | --------------------------------------- |
| 401    | Unauthorized - invalid or missing token |
| 500    | Internal server error                   |

**Idempotency:** Always returns 204, even if called multiple times or with an already-invalid token (as long as token was once valid for this user).

## Business Rules

1. **Token Versioning**
   - New users start with `tokenVersion = 1`
   - Each logout increments `tokenVersion` by 1
   - JWT must include `tokenVersion` claim at time of login

2. **Token Validation (Auth Middleware)**
   - Extract `tokenVersion` from JWT claims
   - Fetch current `tokenVersion` from Redis (or MongoDB on cache miss)
   - Reject token if versions don't match (return 401)

3. **Logout Scope**
   - Logs out ALL sessions (all devices, all browsers)
   - No per-device/per-session logout support

4. **Cache Consistency**
   - On logout: update both MongoDB and Redis
   - Redis TTL: 15 minutes (consistent with user cache)
   - Cache miss falls back to MongoDB

## Implementation Touches

| File | Changes |
| ---- | ------- |
| `internal/models/user.go` | Add `TokenVersion` field to User struct |
| `pkg/auth/jwt.go` | Include `tokenVersion` in JWT claims |
| `internal/middleware/auth.go` | Validate `tokenVersion` matches stored version |
| `internal/handler/auth_handler.go` | Add `Logout` handler |
| `internal/service/user_service.go` | Add `Logout` method (increment version) |
| `internal/repository/user_repository.go` | Add `IncrementTokenVersion` method |
| `internal/router/router.go` | Add logout route |

## Technical Notes

### Token Validation Flow

```
Request with JWT
       ↓
Extract userId + tokenVersion from JWT
       ↓
Fetch stored tokenVersion (Redis → MongoDB fallback)
       ↓
Compare versions
       ↓
Match? → Continue to handler
Mismatch? → 401 Unauthorized
```

### Login Flow Changes

```
User logs in
       ↓
Fetch user from DB (includes tokenVersion)
       ↓
Generate JWT with tokenVersion in claims
       ↓
Return token to client
```

## Out of Scope

- Per-device/per-session logout
- Refresh tokens
- "Logout from specific device" functionality
- Session listing (see active sessions)

## Open Questions

- [ ] None - all requirements clarified
