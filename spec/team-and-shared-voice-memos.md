# Team & Shared Voice Memos Specification

**Author**: Team
**Created**: 2025-12-21
**Status**: Draft
**Architecture**: Service-Layer Business Logic + Isolated Authorization Module

## Overview

Add team support so users can collaborate on voice memos. Users maintain their private space and can create/join multiple teams, each with shared voice memo access. The authorization module is designed for future migration to API Gateway/SpiceDB.

## Architecture Decision

**Chosen Approach**: Isolated Authorization Module + Domain Handlers

**Rationale**:
- Authorization logic isolated in `internal/authz/` for easy migration to SpiceDB/Gateway
- Services contain pure business logic (no permission checks)
- Middleware provides thin authorization wrapper (easy to remove for gateway)
- Handlers focus on HTTP concerns
- Follows existing codebase patterns

### Future Migration Path

| Phase | Authorization Location | Implementation |
|-------|----------------------|----------------|
| **Current** | Middleware | `LocalAuthorizer` with DB lookup |
| **SpiceDB** | Middleware | `SpiceDBAuthorizer` calls SpiceDB |
| **Gateway** | Gateway | Remove middleware, services trust headers |

---

## Files to Create

### Authorization Module
- `internal/authz/authorizer.go` - Interface and action constants
- `internal/authz/local_authorizer.go` - DB-based implementation

### Models
- `internal/models/team.go` - Team model, requests, responses
- `internal/models/team_member.go` - TeamMember, TeamRole enum
- `internal/models/team_invitation.go` - TeamInvitation model

### Repositories
- `internal/repository/team_repository.go` - Team CRUD
- `internal/repository/team_member_repository.go` - Membership operations
- `internal/repository/team_invitation_repository.go` - Invitation lifecycle

### Services
- `internal/service/team_service.go` - Team business logic
- `internal/service/team_member_service.go` - Member management
- `internal/service/team_invitation_service.go` - Invitation handling

### Handlers
- `internal/handler/team_handler.go` - Team endpoints
- `internal/handler/team_member_handler.go` - Member endpoints
- `internal/handler/team_invitation_handler.go` - Invitation endpoints

### Middleware
- `internal/middleware/team_authz.go` - Authorization middleware

---

## Files to Modify

- `internal/models/voice_memo.go` - Add `TeamID` field
- `internal/repository/voice_memo_repository.go` - Add `FindByTeamID` method
- `internal/service/voice_memo_service.go` - Add team memo methods
- `internal/handler/voice_memo_handler.go` - Add team memo handlers
- `internal/errors/errors.go` - Add team-related errors
- `internal/router/router.go` - Add team routes with authz middleware
- `cmd/server/main.go` - Wire new dependencies

---

## Data Models

### Team

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | ObjectID | Yes | Primary key |
| name | string | Yes | Team display name (2-100 chars) |
| slug | string | Yes | URL-friendly identifier (unique, 2-50 chars) |
| description | string | No | Team description (max 500 chars) |
| logoURL | string | No | Team logo image URL |
| ownerId | ObjectID | Yes | User who owns the team |
| seats | int | Yes | Max members + invitations (default: 10) |
| retentionDays | int | Yes | Soft-delete retention period (default: 30) |
| createdAt | time | Yes | Creation timestamp |
| updatedAt | time | Yes | Last update timestamp |
| deletedAt | *time | No | Soft delete timestamp |

### TeamMember

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | ObjectID | Yes | Primary key |
| teamId | ObjectID | Yes | Reference to Team |
| userId | ObjectID | Yes | Reference to User |
| role | TeamRole | Yes | owner / admin / member |
| joinedAt | time | Yes | When user joined |

### TeamInvitation

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | ObjectID | Yes | Primary key |
| teamId | ObjectID | Yes | Reference to Team |
| email | string | Yes | Invitee email |
| invitedBy | ObjectID | Yes | User who sent invitation |
| role | TeamRole | Yes | Role to assign on accept (admin/member) |
| expiresAt | time | Yes | 7 days from creation |
| createdAt | time | Yes | Creation timestamp |

### VoiceMemo (Modified)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| ... | ... | ... | (existing fields) |
| teamId | *ObjectID | No | null = private memo, set = team memo |

---

## Authorization Actions

```go
const (
    ActionTeamView         = "team:view"
    ActionTeamUpdate       = "team:update"
    ActionTeamDelete       = "team:delete"
    ActionTeamTransfer     = "team:transfer"
    ActionMemberInvite     = "member:invite"
    ActionMemberRemove     = "member:remove"
    ActionMemberUpdateRole = "member:update_role"
    ActionMemoView         = "memo:view"
    ActionMemoCreate       = "memo:create"
    ActionMemoUpdate       = "memo:update"
    ActionMemoDelete       = "memo:delete"
)
```

### Action to Role Mapping

| Action | Owner | Admin | Member |
|--------|-------|-------|--------|
| team:view | ✓ | ✓ | ✓ |
| team:update | ✓ | ✓ | ✗ |
| team:delete | ✓ | ✗ | ✗ |
| team:transfer | ✓ | ✗ | ✗ |
| member:invite | ✓ | ✓ | ✗ |
| member:remove | ✓ | ✓ | ✗ |
| member:update_role | ✓ | ✓ | ✗ |
| memo:view | ✓ | ✓ | ✓ |
| memo:create | ✓ | ✓ | ✓ |
| memo:update | ✓ | ✓ | ✓ |
| memo:delete | ✓ | ✓ | ✓ |

---

## API Endpoints

### Team Management

#### Create Team
- **Endpoint**: `POST /api/v1/teams`
- **Authentication**: Required
- **Authorization**: Any authenticated user (limit: 1 team for free users)

**Request**:
```json
{
  "name": "Engineering Team",
  "slug": "engineering",
  "description": "Our engineering team workspace"
}
```

**Response** (201 Created):
```json
{
  "success": true,
  "data": {
    "id": "...",
    "name": "Engineering Team",
    "slug": "engineering",
    "description": "Our engineering team workspace",
    "logoUrl": "",
    "ownerId": "...",
    "seats": 10,
    "retentionDays": 30,
    "createdAt": "2025-12-21T10:00:00Z",
    "updatedAt": "2025-12-21T10:00:00Z"
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 400 | validation error | Invalid input |
| 403 | team limit reached | Free user already owns a team |
| 409 | slug taken | Slug already exists |

---

#### List My Teams
- **Endpoint**: `GET /api/v1/teams`
- **Authentication**: Required
- **Query Parameters**: `page` (default: 1), `limit` (default: 10)

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "items": [...],
    "pagination": {
      "page": 1,
      "limit": 10,
      "totalItems": 3,
      "totalPages": 1
    }
  }
}
```

---

#### Get Team
- **Endpoint**: `GET /api/v1/teams/:teamId`
- **Authentication**: Required
- **Authorization**: `team:view` (any team member)

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "id": "...",
    "name": "Engineering Team",
    "slug": "engineering",
    ...
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 403 | not a team member | User is not a member |
| 404 | team not found | Team doesn't exist |

---

#### Update Team
- **Endpoint**: `PUT /api/v1/teams/:teamId`
- **Authentication**: Required
- **Authorization**: `team:update` (owner, admin)

**Request**:
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "logoUrl": "https://example.com/logo.png"
}
```

**Response** (200 OK): Updated team object

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 403 | insufficient permissions | Not owner/admin |
| 404 | team not found | Team doesn't exist |
| 409 | slug taken | New slug already exists |

---

#### Delete Team
- **Endpoint**: `DELETE /api/v1/teams/:teamId`
- **Authentication**: Required
- **Authorization**: `team:delete` (owner only)

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "team deleted successfully"
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 403 | insufficient permissions | Not owner |
| 404 | team not found | Team doesn't exist |

---

#### Transfer Ownership
- **Endpoint**: `POST /api/v1/teams/:teamId/transfer`
- **Authentication**: Required
- **Authorization**: `team:transfer` (owner only)

**Request**:
```json
{
  "newOwnerId": "user-object-id"
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "ownership transferred successfully"
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 400 | invalid user id | New owner ID invalid |
| 403 | insufficient permissions | Not current owner |
| 404 | user not found | New owner not a team member |

---

### Team Membership

#### List Members
- **Endpoint**: `GET /api/v1/teams/:teamId/members`
- **Authentication**: Required
- **Authorization**: `team:view` (any member)

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "...",
        "teamId": "...",
        "userId": "...",
        "user": {
          "id": "...",
          "email": "user@example.com",
          "name": "John Doe"
        },
        "role": "owner",
        "joinedAt": "2025-12-21T10:00:00Z"
      }
    ]
  }
}
```

---

#### Remove Member
- **Endpoint**: `DELETE /api/v1/teams/:teamId/members/:userId`
- **Authentication**: Required
- **Authorization**: `member:remove` (owner, admin)

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "member removed successfully"
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 400 | cannot remove owner | Use transfer instead |
| 400 | cannot remove self | Use leave endpoint |
| 403 | insufficient permissions | Not owner/admin |
| 404 | member not found | User not in team |

---

#### Update Member Role
- **Endpoint**: `PUT /api/v1/teams/:teamId/members/:userId/role`
- **Authentication**: Required
- **Authorization**: `member:update_role` (owner, admin)

**Request**:
```json
{
  "role": "admin"
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "role updated successfully"
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 400 | cannot change owner role | Use transfer instead |
| 400 | invalid role | Must be "admin" or "member" |
| 403 | insufficient permissions | Not owner/admin |

---

#### Leave Team
- **Endpoint**: `POST /api/v1/teams/:teamId/leave`
- **Authentication**: Required
- **Authorization**: Any member (except owner)

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "left team successfully"
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 400 | owner cannot leave | Transfer ownership first |
| 404 | not a member | User not in team |

---

### Invitations

#### Create Invitation
- **Endpoint**: `POST /api/v1/teams/:teamId/invitations`
- **Authentication**: Required
- **Authorization**: `member:invite` (owner, admin)

**Request**:
```json
{
  "email": "newuser@example.com",
  "role": "member"
}
```

**Response** (201 Created):
```json
{
  "success": true,
  "data": {
    "id": "...",
    "teamId": "...",
    "email": "newuser@example.com",
    "role": "member",
    "expiresAt": "2025-12-28T10:00:00Z",
    "createdAt": "2025-12-21T10:00:00Z"
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 400 | already a member | User already in team |
| 400 | pending invitation | Invitation already sent |
| 403 | seats exceeded | members + invitations >= seats |
| 403 | insufficient permissions | Not owner/admin |

---

#### List Team Invitations
- **Endpoint**: `GET /api/v1/teams/:teamId/invitations`
- **Authentication**: Required
- **Authorization**: `member:invite` (owner, admin)

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "items": [...]
  }
}
```

---

#### Cancel Invitation
- **Endpoint**: `DELETE /api/v1/teams/:teamId/invitations/:id`
- **Authentication**: Required
- **Authorization**: `member:invite` (owner, admin)

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "invitation cancelled"
  }
}
```

---

#### List My Invitations
- **Endpoint**: `GET /api/v1/invitations`
- **Authentication**: Required
- **Description**: List pending invitations for the authenticated user's email

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "...",
        "team": {
          "id": "...",
          "name": "Engineering Team",
          "slug": "engineering"
        },
        "invitedBy": {
          "id": "...",
          "name": "John Doe"
        },
        "role": "member",
        "expiresAt": "2025-12-28T10:00:00Z",
        "createdAt": "2025-12-21T10:00:00Z"
      }
    ]
  }
}
```

---

#### Accept Invitation
- **Endpoint**: `POST /api/v1/invitations/:id/accept`
- **Authentication**: Required

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "invitation accepted",
    "teamId": "..."
  }
}
```

**Errors**:
| Status | Error | Description |
|--------|-------|-------------|
| 400 | invitation expired | Past expiry date |
| 403 | email mismatch | Invitation not for this user |
| 403 | seats exceeded | Team is full |
| 404 | invitation not found | Invalid or already used |

---

#### Decline Invitation
- **Endpoint**: `POST /api/v1/invitations/:id/decline`
- **Authentication**: Required

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "message": "invitation declined"
  }
}
```

---

### Team Voice Memos

#### List Team Voice Memos
- **Endpoint**: `GET /api/v1/teams/:teamId/voice-memos`
- **Authentication**: Required
- **Authorization**: `memo:view` (any member)
- **Query Parameters**: `page`, `limit`

**Response** (200 OK): Same structure as private voice memos list

---

#### Create Team Voice Memo
- **Endpoint**: `POST /api/v1/teams/:teamId/voice-memos`
- **Authentication**: Required
- **Authorization**: `memo:create` (any member)

**Request/Response**: Same as private voice memo creation

---

#### Get Team Voice Memo
- **Endpoint**: `GET /api/v1/teams/:teamId/voice-memos/:id`
- **Authentication**: Required
- **Authorization**: `memo:view` (any member)

---

#### Update Team Voice Memo
- **Endpoint**: `PUT /api/v1/teams/:teamId/voice-memos/:id`
- **Authentication**: Required
- **Authorization**: `memo:update` (any member)

---

#### Delete Team Voice Memo
- **Endpoint**: `DELETE /api/v1/teams/:teamId/voice-memos/:id`
- **Authentication**: Required
- **Authorization**: `memo:delete` (any member)

---

## Business Rules

1. **Team Ownership Limit**: Free users can create at most 1 team
2. **Team Membership**: Users can be invited to unlimited teams
3. **Seats Constraint**: `members + pending invitations <= seats` (default: 10)
4. **Invitation Expiry**: Invitations expire after 7 days
5. **Pending Invitations**: Non-registered emails can receive invitations (visible after registration)
6. **Re-invitation**: Users can be re-invited after declining
7. **Owner Departure**: Owner must transfer ownership before leaving
8. **Soft Delete**: Deleted teams retained for `retentionDays` (default: 30)
9. **Team Deletion Cascade**:
   - Soft delete team
   - Soft delete all team voice memos
   - Hard delete all team members
   - Hard delete all pending invitations
10. **Memo Immobility**: Memos cannot be moved between private and team spaces

---

## MongoDB Indexes

```javascript
// teams collection
db.teams.createIndex({ "slug": 1 }, { unique: true })
db.teams.createIndex({ "ownerId": 1 })
db.teams.createIndex({ "deletedAt": 1 })

// team_members collection
db.team_members.createIndex({ "teamId": 1, "userId": 1 }, { unique: true })
db.team_members.createIndex({ "userId": 1 })

// team_invitations collection
db.team_invitations.createIndex({ "teamId": 1, "email": 1 })
db.team_invitations.createIndex({ "email": 1 })
db.team_invitations.createIndex({ "expiresAt": 1 })  // TTL or cleanup job

// voice_memos collection (add)
db.voice_memos.createIndex({ "teamId": 1, "createdAt": -1 })
```

---

## Error Definitions

Add to `internal/errors/errors.go`:

```go
// Team errors
var (
    ErrTeamNotFound            = errors.New("team not found")
    ErrTeamSlugTaken           = errors.New("team slug is already taken")
    ErrTeamLimitReached        = errors.New("free users can only create 1 team")
    ErrNotTeamMember           = errors.New("you are not a member of this team")
    ErrInsufficientPermissions = errors.New("insufficient permissions")
    ErrOwnerCannotLeave        = errors.New("owner must transfer ownership before leaving")
    ErrCannotRemoveOwner       = errors.New("cannot remove team owner")
    ErrCannotRemoveSelf        = errors.New("cannot remove yourself, use leave endpoint")
    ErrCannotChangeOwnerRole   = errors.New("cannot change owner role, use transfer")
    ErrSeatsExceeded           = errors.New("team seats limit exceeded")
    ErrInvalidRole             = errors.New("invalid role, must be admin or member")
)

// Invitation errors
var (
    ErrInvitationNotFound      = errors.New("invitation not found")
    ErrInvitationExpired       = errors.New("invitation has expired")
    ErrInvitationEmailMismatch = errors.New("invitation email does not match your account")
    ErrAlreadyMember           = errors.New("user is already a team member")
    ErrPendingInvitation       = errors.New("invitation already pending for this email")
)
```

---

## Implementation Steps

### Phase 1: Foundation
- [ ] Add team errors to `internal/errors/errors.go`
- [ ] Create `internal/authz/authorizer.go` - interface and actions
- [ ] Create `internal/authz/local_authorizer.go` - implementation
- [ ] Create `internal/models/team.go`
- [ ] Create `internal/models/team_member.go`
- [ ] Create `internal/models/team_invitation.go`
- [ ] Modify `internal/models/voice_memo.go` - add TeamID field

### Phase 2: Repositories
- [ ] Create `internal/repository/team_repository.go`
- [ ] Create `internal/repository/team_member_repository.go`
- [ ] Create `internal/repository/team_invitation_repository.go`
- [ ] Modify `internal/repository/voice_memo_repository.go` - add FindByTeamID

### Phase 3: Services
- [ ] Create `internal/service/team_service.go`
- [ ] Create `internal/service/team_member_service.go`
- [ ] Create `internal/service/team_invitation_service.go`
- [ ] Modify `internal/service/voice_memo_service.go` - add team memo methods

### Phase 4: Handlers & Middleware
- [ ] Create `internal/middleware/team_authz.go`
- [ ] Create `internal/handler/team_handler.go`
- [ ] Create `internal/handler/team_member_handler.go`
- [ ] Create `internal/handler/team_invitation_handler.go`
- [ ] Modify `internal/handler/voice_memo_handler.go` - add team memo handlers

### Phase 5: Wiring & Routes
- [ ] Modify `internal/router/router.go` - add team routes
- [ ] Modify `cmd/server/main.go` - wire dependencies

### Phase 6: Documentation & Testing
- [ ] Run `task swagger` to regenerate API docs
- [ ] Create MongoDB indexes
- [ ] Test all endpoints

---

## Out of Scope

- SSO per team
- Billing/subscription plans
- Moving memos between spaces
- Organization layer
- SpiceDB integration (architecture prepared)
- Email notifications for invitations
- Background job for invitation cleanup

---

## Open Questions

- [ ] Should we send email notifications for invitations? (deferred)
- [ ] Background job for cleaning expired invitations? (deferred)
- [ ] Rate limiting on invitation creation? (deferred)
