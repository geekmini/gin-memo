# Voice Memo API Specification

**Author**: Team
**Created**: 2025-12-20
**Status**: Implemented

## Overview

API for fetching a user's voice memos. This is a **read-only** API - voice memos are created through a separate upload process.

## Data Model

### VoiceMemo

| Field         | Type     | Description                              |
| ------------- | -------- | ---------------------------------------- |
| id            | string   | Unique identifier (MongoDB ObjectID)     |
| userId        | string   | Owner's user ID (foreign key)            |
| title         | string   | Memo title                               |
| transcription | string   | Full text transcription of the audio     |
| audioFileUrl  | string   | Pre-signed S3 URL (expires after 1 hour) |
| duration      | int      | Audio duration in seconds                |
| fileSize      | int      | File size in bytes                       |
| audioFormat   | string   | Audio format (mp3, wav, m4a, etc.)       |
| tags          | []string | Array of tags                            |
| isFavorite    | bool     | Whether memo is starred                  |
| createdAt     | datetime | When the recording was created           |

### Relationships

- One User → Many VoiceMemos
- Users can only access their own memos

## Endpoints

### List Voice Memos

Retrieve paginated list of the authenticated user's voice memos.

```
GET /api/v1/voice-memos
```

**Authentication:** Required (Bearer token)

**Query Parameters:**

| Param | Type | Default | Description              |
| ----- | ---- | ------- | ------------------------ |
| page  | int  | 1       | Page number (1-indexed)  |
| limit | int  | 10      | Items per page (max: 10) |

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "507f1f77bcf86cd799439011",
        "userId": "507f1f77bcf86cd799439012",
        "title": "Meeting notes",
        "transcription": "Today we discussed the Q4 roadmap...",
        "audioFileUrl": "https://bucket.s3.amazonaws.com/audio/123.mp3?X-Amz-Signature=...",
        "duration": 180,
        "fileSize": 2890000,
        "audioFormat": "mp3",
        "tags": ["work", "meeting"],
        "isFavorite": false,
        "createdAt": "2024-01-15T09:30:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "totalItems": 42,
      "totalPages": 5
    }
  }
}
```

**Sort Order:** Newest first (by `createdAt` descending)

**Errors:**

| Status | Description                             |
| ------ | --------------------------------------- |
| 401    | Unauthorized - invalid or missing token |
| 500    | Internal server error                   |

## Future Endpoints (Not in Scope)

- `GET /api/v1/voice-memos/:id` - Get single memo by ID

## Technical Notes

### Audio Storage

- Files stored in S3 (use MinIO for local development)
- API returns pre-signed URLs that expire after 1 hour
- URLs are generated on each request

### Database

- Collection: `voice_memos`
- Index on `userId` for efficient queries
- Index on `createdAt` for sorting

## Out of Scope

- Create/Update/Delete operations
- Multi-language support
- Cross-user memo sharing
