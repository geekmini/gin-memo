# Voice Memo Concurrency Control Specification

**Author**: Team
**Created**: 2025-12-21
**Status**: Implemented
**Architecture**: Atomic Delete with FindOneAndUpdate

## Overview

Implement idempotent soft delete operations for voice memos and add version/timestamp fields to prepare for future optimistic locking on updates. Uses MongoDB's atomic `FindOneAndUpdate` to eliminate race conditions.

## Architecture Decision

**Chosen Approach**: Atomic Delete with FindOneAndUpdate

**Rationale**:
- Single database query for successful deletes (vs 2-3 in alternative approaches)
- Atomic operation eliminates race conditions between ownership check and delete
- Simplifies service layer to thin pass-through
- MongoDB best practice for read-modify-write operations
- Version increment prepares for future optimistic locking on updates

### Files to Modify

- `internal/models/voice_memo.go` - Add `Version` and `UpdatedAt` fields
- `internal/repository/voice_memo_repository.go` - Add atomic delete methods with ownership/team checks
- `internal/service/voice_memo_service.go` - Simplify delete methods to delegate to repository
- `internal/handler/voice_memo_handler.go` - Change response from 200 to 204 No Content
- `pkg/response/response.go` - Add `NoContent()` helper function

## Data Model

### VoiceMemo (Updated Fields)

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| version | int | Yes | Optimistic locking version, increments on each modification |
| updatedAt | time.Time | Yes | Timestamp of last modification |

**Note**: These fields are added to the existing VoiceMemo model. All other fields remain unchanged.

**Version field behavior**: Existing documents without a version field will default to 0 (MongoDB's default for missing int fields). After the first modification (e.g., soft delete), version becomes 1. New documents should initialize version to 1 when create endpoints are implemented.

### Complete Model Structure

```go
type VoiceMemo struct {
    ID            primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
    UserID        primitive.ObjectID  `json:"userId" bson:"userId"`
    TeamID        *primitive.ObjectID `json:"teamId,omitempty" bson:"teamId,omitempty"`
    Title         string              `json:"title" bson:"title"`
    Transcription string              `json:"transcription" bson:"transcription"`
    AudioFileKey  string              `json:"-" bson:"audioFileKey"`
    AudioFileURL  string              `json:"audioFileUrl" bson:"-"`
    Duration      int                 `json:"duration" bson:"duration"`
    FileSize      int64               `json:"fileSize" bson:"fileSize"`
    AudioFormat   string              `json:"audioFormat" bson:"audioFormat"`
    Tags          []string            `json:"tags" bson:"tags"`
    IsFavorite    bool                `json:"isFavorite" bson:"isFavorite"`
    Version       int                 `json:"version" bson:"version"`           // NEW
    CreatedAt     time.Time           `json:"createdAt" bson:"createdAt"`
    UpdatedAt     time.Time           `json:"updatedAt" bson:"updatedAt"`       // NEW
    DeletedAt     *time.Time          `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
}
```

## API Endpoints

### Delete Private Voice Memo

**Endpoint**: `DELETE /api/v1/voice-memos/:id`
**Authentication**: Required (JWT)
**Description**: Soft delete a private voice memo. Idempotent - returns 204 even if already deleted.

#### Request

No request body. Voice memo ID in path parameter.

#### Response

**Success (204 No Content):**
No response body.

#### Error Responses

| Status | Message | Description |
| ------ | ------- | ----------- |
| 400 | Invalid voice memo id format | ID is not a valid ObjectID |
| 401 | User not authenticated | Missing or invalid JWT token |
| 403 | Unauthorized to access this voice memo | User does not own this memo |
| 404 | Voice memo not found | Memo never existed |

### Delete Team Voice Memo

**Endpoint**: `DELETE /api/v1/teams/:teamId/voice-memos/:id`
**Authentication**: Required (JWT + Team Membership)
**Authorization**: Owner, Admin, or Member role with `memo:delete` permission
**Description**: Soft delete a team voice memo. Idempotent - returns 204 even if already deleted.

#### Request

No request body. Team ID and Voice memo ID in path parameters.

#### Response

**Success (204 No Content):**
No response body.

#### Error Responses

| Status | Message | Description |
| ------ | ------- | ----------- |
| 400 | Invalid voice memo id format | ID is not a valid ObjectID |
| 401 | User not authenticated | Missing or invalid JWT token |
| 403 | Forbidden | User lacks permission to delete team memos |
| 404 | Voice memo not found | Memo never existed or doesn't belong to team |

## Business Rules

1. **Idempotent Delete**: Delete operations return 204 No Content even if the memo is already deleted
2. **Ownership Check**: Private memo deletes verify the requesting user owns the memo
3. **Team Check**: Team memo deletes verify the memo belongs to the specified team
4. **Version Increment**: Version field is incremented on soft delete (for future audit/consistency)
5. **UpdatedAt Tracking**: `updatedAt` timestamp is set when memo is deleted
6. **Atomic Operation**: Ownership/team check and delete happen in a single atomic MongoDB operation
7. **Error Distinction**:
   - 404 = memo never existed OR memo doesn't belong to user/team
   - 403 = memo exists but user lacks ownership (private) or permission (team)

## Implementation Steps

1. [x] Add `Version` and `UpdatedAt` fields to VoiceMemo model
2. [x] Add `NoContent()` helper to `pkg/response/response.go`
3. [x] Add `SoftDeleteWithOwnership()` method to repository interface and implementation
4. [x] Add `SoftDeleteWithTeam()` method to repository interface and implementation
5. [x] Update `DeleteVoiceMemo` service method to use new repository method
6. [x] Update `DeleteTeamVoiceMemo` service method to use new repository method
7. [x] Update `DeleteVoiceMemo` handler to return 204 No Content
8. [x] Update `DeleteTeamVoiceMemo` handler to return 204 No Content
9. [x] Update Swagger annotations for both handlers
10. [x] Run `task swagger` to regenerate API documentation

## Repository Method Details

### SoftDeleteWithOwnership

```go
// SoftDeleteWithOwnership atomically soft-deletes if user owns the memo.
// Returns nil if memo is already deleted (idempotent).
// Returns ErrVoiceMemoNotFound if memo doesn't exist.
// Returns ErrVoiceMemoUnauthorized if memo exists but user doesn't own it.
func (r *voiceMemoRepository) SoftDeleteWithOwnership(ctx context.Context, id, userID primitive.ObjectID) error
```

**MongoDB Filter**:
```json
{
  "_id": "<id>",
  "userId": "<userID>",
  "deletedAt": { "$exists": false }
}
```

**MongoDB Update**:
```json
{
  "$set": {
    "deletedAt": "<now>",
    "updatedAt": "<now>"
  },
  "$inc": { "version": 1 }
}
```

### SoftDeleteWithTeam

```go
// SoftDeleteWithTeam atomically soft-deletes if memo belongs to team.
// Returns nil if memo is already deleted (idempotent).
// Returns ErrVoiceMemoNotFound if memo doesn't exist or doesn't belong to team.
func (r *voiceMemoRepository) SoftDeleteWithTeam(ctx context.Context, id, teamID primitive.ObjectID) error
```

**MongoDB Filter**:
```json
{
  "_id": "<id>",
  "teamId": "<teamID>",
  "deletedAt": { "$exists": false }
}
```

## Out of Scope

- Update endpoints for voice memos (don't exist yet)
- Create endpoints for voice memos (separate upload process)
- Hard delete operations
- Optimistic locking enforcement (Version field added but not validated until update endpoints exist)
- Database migration for existing records (existing records will have Version=0 and no UpdatedAt)

## Testing Checklist

> Source of truth for testing requirements.

- [x] Delete existing private memo returns 204
- [x] Delete already-deleted private memo returns 204 (idempotent)
- [x] Delete non-existent private memo returns 404
- [x] Delete another user's private memo returns 403
- [x] Delete existing team memo returns 204
- [x] Delete already-deleted team memo returns 204 (idempotent)
- [x] Delete non-existent team memo returns 404
- [x] Delete team memo from wrong team returns 404
- [x] Version field increments on delete
- [x] UpdatedAt field is set on delete

## Open Questions

None - all questions resolved during discovery phase.
