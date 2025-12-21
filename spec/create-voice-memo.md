# Create Voice Memo API Specification

**Author**: Team
**Created**: 2025-12-21
**Status**: Implemented
**Architecture**: Status-Driven Workflow with In-Memory Queue

## Overview

Create voice memos with metadata, upload audio via pre-signed S3 URL, and automatically transcribe audio to text. Supports both private and team memos with a status-driven workflow that tracks upload and transcription progress.

## Architecture Decision

**Chosen Approach**: Status-Driven Workflow with In-Memory Queue

**Rationale**:
- Follows existing codebase patterns (handler → service → repository)
- Clean separation of concerns (create → upload → transcribe)
- Atomic operations prevent race conditions
- Interface-based transcription service allows easy swap to real implementation
- Pre-signed URLs keep server lightweight (client uploads directly to S3)

### Files to Create

| File | Purpose |
|------|---------|
| `internal/transcription/service.go` | Transcription interface + mock implementation |
| `internal/queue/memory_queue.go` | In-memory job queue with configurable workers |
| `internal/queue/processor.go` | Transcription job processor with retry logic |

### Files to Modify

| File | Changes |
|------|---------|
| `internal/models/voice_memo.go` | Add `Status` field, `CreateVoiceMemoRequest`, `CreateVoiceMemoResponse` |
| `internal/repository/voice_memo_repository.go` | Add `Create()`, `UpdateStatus()`, `UpdateStatusWithOwnership()`, `UpdateStatusWithTeam()`, `UpdateTranscriptionAndStatus()` |
| `internal/storage/s3.go` | Add `GetPresignedPutURL()` method |
| `internal/service/voice_memo_service.go` | Add create, confirm-upload, retry methods; update constructor for queue dependency |
| `internal/handler/voice_memo_handler.go` | Add 6 new handlers with Swagger annotations |
| `internal/errors/errors.go` | Add `ErrVoiceMemoInvalidStatus` |
| `internal/router/router.go` | Register 6 new routes with appropriate middleware |
| `cmd/server/main.go` | Wire queue, processor, transcription service dependencies |

## Data Model

### CreateVoiceMemoRequest

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| title | string | Yes | 1-200 chars | Name of the voice memo |
| duration | int | No | >= 0, default 0 | Duration in seconds |
| fileSize | int64 | Yes | > 0, max 100MB | File size in bytes |
| audioFormat | string | Yes | mp3, wav, m4a, webm, aac | Audio file format |
| tags | []string | No | max 10 tags, each max 50 chars | Categorization tags |
| isFavorite | bool | No | default false | Mark as favorite |

### CreateVoiceMemoResponse

| Field | Type | Description |
|-------|------|-------------|
| memo | VoiceMemo | Created voice memo object |
| uploadUrl | string | Pre-signed S3 PUT URL (15 min expiry) |

### VoiceMemo.Status (New Field)

| Status | Description |
|--------|-------------|
| `pending_upload` | Memo created, waiting for audio upload |
| `transcribing` | Audio uploaded, transcription in progress |
| `ready` | Transcription complete, memo fully available |
| `failed` | Transcription failed (can retry) |

### Status State Machine

```
pending_upload ──► transcribing ──► ready
                        │
                        ▼
                     failed ──► (retry) ──► transcribing
```

## API Endpoints

### 1. Create Private Voice Memo

**Endpoint**: `POST /api/v1/voice-memos`
**Authentication**: JWT Required
**Description**: Create a new private voice memo and receive a pre-signed S3 URL for audio upload

#### Request

```json
{
  "title": "Meeting Notes",
  "duration": 120,
  "fileSize": 1048576,
  "audioFormat": "mp3",
  "tags": ["work", "meeting"],
  "isFavorite": false
}
```

#### Response

**Success (201 Created):**
```json
{
  "success": true,
  "data": {
    "memo": {
      "id": "507f1f77bcf86cd799439011",
      "userId": "507f1f77bcf86cd799439012",
      "title": "Meeting Notes",
      "transcription": "",
      "duration": 120,
      "fileSize": 1048576,
      "audioFormat": "mp3",
      "tags": ["work", "meeting"],
      "isFavorite": false,
      "status": "pending_upload",
      "version": 0,
      "createdAt": "2025-12-21T10:00:00Z",
      "updatedAt": "2025-12-21T10:00:00Z"
    },
    "uploadUrl": "https://s3.amazonaws.com/bucket/voice-memos/507f1f77bcf86cd799439012/507f1f77bcf86cd799439011.mp3?X-Amz-Algorithm=..."
  }
}
```

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Validation error | Invalid request body (missing required fields, invalid format) |
| 401 | User not authenticated | Missing or invalid JWT token |
| 500 | Internal server error | S3 URL generation failed |

---

### 2. Create Team Voice Memo

**Endpoint**: `POST /api/v1/teams/:teamId/voice-memos`
**Authentication**: JWT + TeamAuthz (memo:create)
**Description**: Create a new team voice memo and receive a pre-signed S3 URL for audio upload

#### Request

Same as Create Private Voice Memo.

#### Response

**Success (201 Created):**
```json
{
  "success": true,
  "data": {
    "memo": {
      "id": "507f1f77bcf86cd799439011",
      "userId": "507f1f77bcf86cd799439012",
      "teamId": "507f1f77bcf86cd799439013",
      "title": "Team Standup",
      "transcription": "",
      "duration": 300,
      "fileSize": 2097152,
      "audioFormat": "m4a",
      "tags": ["standup"],
      "isFavorite": false,
      "status": "pending_upload",
      "version": 0,
      "createdAt": "2025-12-21T10:00:00Z",
      "updatedAt": "2025-12-21T10:00:00Z"
    },
    "uploadUrl": "https://s3.amazonaws.com/bucket/voice-memos/507f1f77bcf86cd799439013/507f1f77bcf86cd799439012/507f1f77bcf86cd799439011.m4a?X-Amz-Algorithm=..."
  }
}
```

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Validation error | Invalid request body or team ID format |
| 401 | User not authenticated | Missing or invalid JWT token |
| 403 | Forbidden | User lacks memo:create permission for team |
| 500 | Internal server error | S3 URL generation failed |

---

### 3. Confirm Upload (Private)

**Endpoint**: `POST /api/v1/voice-memos/:id/confirm-upload`
**Authentication**: JWT Required
**Description**: Confirm audio file has been uploaded to S3 and trigger transcription

#### Request

No request body required.

#### Response

**Success (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "upload confirmed, transcription started"
  }
}
```

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid voice memo id format | ID is not a valid ObjectID |
| 400 | Invalid voice memo status transition | Memo is not in pending_upload status |
| 401 | User not authenticated | Missing or invalid JWT token |
| 403 | Forbidden | User does not own this memo |
| 404 | Voice memo not found | Memo does not exist |
| 500 | Internal server error | Queue is full or other error |

---

### 4. Confirm Upload (Team)

**Endpoint**: `POST /api/v1/teams/:teamId/voice-memos/:id/confirm-upload`
**Authentication**: JWT + TeamAuthz (memo:create)
**Description**: Confirm team memo audio upload and trigger transcription

#### Request

No request body required.

#### Response

Same as Confirm Upload (Private).

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid voice memo id format | ID is not a valid ObjectID |
| 400 | Invalid voice memo status transition | Memo is not in pending_upload status |
| 401 | User not authenticated | Missing or invalid JWT token |
| 403 | Forbidden | User lacks memo:create permission |
| 404 | Voice memo not found | Memo does not exist or doesn't belong to team |
| 500 | Internal server error | Queue is full or other error |

---

### 5. Retry Transcription (Private)

**Endpoint**: `POST /api/v1/voice-memos/:id/retry-transcription`
**Authentication**: JWT Required
**Description**: Manually retry transcription for a failed voice memo

#### Request

No request body required.

#### Response

**Success (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "transcription retry initiated"
  }
}
```

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid voice memo id format | ID is not a valid ObjectID |
| 400 | Can only retry failed transcriptions | Memo status is not "failed" |
| 401 | User not authenticated | Missing or invalid JWT token |
| 403 | Forbidden | User does not own this memo |
| 404 | Voice memo not found | Memo does not exist |
| 500 | Internal server error | Queue is full or other error |

---

### 6. Retry Transcription (Team)

**Endpoint**: `POST /api/v1/teams/:teamId/voice-memos/:id/retry-transcription`
**Authentication**: JWT + TeamAuthz (memo:create)
**Description**: Manually retry transcription for a failed team voice memo

#### Request

No request body required.

#### Response

Same as Retry Transcription (Private).

#### Error Responses

| Status | Message | Description |
|--------|---------|-------------|
| 400 | Invalid voice memo id format | ID is not a valid ObjectID |
| 400 | Can only retry failed transcriptions | Memo status is not "failed" |
| 401 | User not authenticated | Missing or invalid JWT token |
| 403 | Forbidden | User lacks memo:create permission |
| 404 | Voice memo not found | Memo does not exist or doesn't belong to team |
| 500 | Internal server error | Queue is full or other error |

## Business Rules

1. **Audio Format Validation**: Only mp3, wav, m4a, webm, and aac formats are allowed
2. **File Size Limit**: Maximum file size is 100 MB (104,857,600 bytes)
3. **S3 Key Format**:
   - Private: `voice-memos/{userId}/{memoId}.{format}`
   - Team: `voice-memos/{teamId}/{userId}/{memoId}.{format}`
4. **Pre-signed URL Expiry**: Upload URLs expire after 15 minutes
5. **Transcription Auto-Retry**: Failed transcriptions automatically retry up to 3 times
6. **Manual Retry**: Users can manually retry only when status is "failed"
7. **Status Transitions**:
   - `pending_upload` → `transcribing` (on confirm-upload)
   - `transcribing` → `ready` (on success) or `failed` (on error)
   - `failed` → `transcribing` (on retry)
8. **Team Authorization**: Team memo creation requires `memo:create` permission
9. **Ownership**: Users can only confirm/retry their own private memos

## Implementation Steps

### Phase 1: Foundation (Infrastructure)
1. [ ] Add `Status` field to `VoiceMemo` model in `internal/models/voice_memo.go`
2. [ ] Add `CreateVoiceMemoRequest` struct with validation tags
3. [ ] Add `CreateVoiceMemoResponse` struct
4. [ ] Add `ErrVoiceMemoInvalidStatus` to `internal/errors/errors.go`
5. [ ] Create `internal/transcription/service.go` with interface + mock
6. [ ] Create `internal/queue/memory_queue.go` with worker pool
7. [ ] Create `internal/queue/processor.go` with retry logic
8. [ ] Add `GetPresignedPutURL()` to `internal/storage/s3.go`

### Phase 2: Data Layer
9. [ ] Add `Create()` method to `VoiceMemoRepository` interface and implementation
10. [ ] Add `UpdateStatus()` method to repository
11. [ ] Add `UpdateStatusWithOwnership()` method to repository
12. [ ] Add `UpdateStatusWithTeam()` method to repository
13. [ ] Add `UpdateTranscriptionAndStatus()` method to repository

### Phase 3: Service Layer
14. [ ] Update `VoiceMemoService` constructor to accept queue dependency
15. [ ] Implement `CreateVoiceMemo()` method
16. [ ] Implement `CreateTeamVoiceMemo()` method
17. [ ] Implement `ConfirmUpload()` method
18. [ ] Implement `ConfirmTeamUpload()` method
19. [ ] Implement `RetryTranscription()` method
20. [ ] Implement `RetryTeamTranscription()` method

### Phase 4: Handler Layer
21. [ ] Implement `CreateVoiceMemo` handler with Swagger annotations
22. [ ] Implement `CreateTeamVoiceMemo` handler with Swagger annotations
23. [ ] Implement `ConfirmUpload` handler with Swagger annotations
24. [ ] Implement `ConfirmTeamUpload` handler with Swagger annotations
25. [ ] Implement `RetryTranscription` handler with Swagger annotations
26. [ ] Implement `RetryTeamTranscription` handler with Swagger annotations

### Phase 5: Integration
27. [ ] Wire queue, processor, transcription service in `cmd/server/main.go`
28. [ ] Start queue workers with context cancellation
29. [ ] Update `VoiceMemoService` instantiation with queue dependency
30. [ ] Add private memo routes to `internal/router/router.go`
31. [ ] Add team memo routes with TeamAuthz middleware

### Phase 6: Testing & Verification
32. [ ] Run `task swagger` to regenerate API docs
33. [ ] Test private memo creation flow
34. [ ] Test team memo creation flow
35. [ ] Test confirm-upload flow
36. [ ] Test retry-transcription flow
37. [ ] Test error cases (invalid status, unauthorized, not found)

## Out of Scope

- **Transcription service implementation**: Only placeholder/mock for MVP; real implementation (OpenAI Whisper, AWS Transcribe) deferred
- **Persistent job queue**: In-memory queue acceptable for MVP; Redis/RabbitMQ upgrade later
- **Real-time status updates**: Frontend will poll; WebSocket/SSE deferred
- **Upload progress tracking**: Client-side responsibility
- **Audio file validation**: S3 handles content-type; server-side validation deferred

## Open Questions

- [x] Audio upload strategy → Hybrid: metadata + pre-signed URL
- [x] Transcription service → Placeholder interface for now
- [x] Job queue → In-memory with 3 workers
- [x] Duration field → Optional, client provides if available
