# Layered Architecture Conventions

This document describes the layered architecture conventions used in this project.

## Overview

```
Handler → Service → Repository → MongoDB
```

- **Handler**: HTTP concerns, input validation, error mapping
- **Service**: Business logic, caching, orchestration
- **Repository**: Data access, CRUD operations

## Layer Responsibilities

| Layer          | Responsibility                                 | Receives                            | Returns               |
| -------------- | ---------------------------------------------- | ----------------------------------- | --------------------- |
| **Handler**    | HTTP concerns, input validation, error mapping | `*gin.Context`                      | HTTP response         |
| **Service**    | Business logic, caching, orchestration         | Typed values (`primitive.ObjectID`) | Domain models, errors |
| **Repository** | Data access, CRUD operations                   | Typed values (`primitive.ObjectID`) | Domain models, errors |

## Handler Layer (`internal/handler/`)

**Responsibilities:**
- Parse HTTP request (path params, query params, body)
- Validate input format (ID format, JSON binding)
- Convert string IDs to `primitive.ObjectID`
- Call service with typed values
- Map service errors to HTTP responses
- Return standardized JSON responses

**Should:**
```go
// Validate ID format and convert to ObjectID
memoID, err := primitive.ObjectIDFromHex(c.Param("id"))
if err != nil {
    response.BadRequest(c, "invalid id format")  // 400, not 404
    return
}

// Bind and validate request body
var req models.CreateRequest
if err := c.ShouldBindJSON(&req); err != nil {
    response.BadRequest(c, err.Error())
    return
}

// Call service with typed values
result, err := h.service.DoSomething(ctx, memoID, userID)
```

**Should NOT:**
- Contain business logic
- Access repository directly
- Know about database implementation

## Service Layer (`internal/service/`)

**Responsibilities:**
- Business logic and validation rules
- Ownership/authorization checks
- Caching (cache-aside pattern)
- Orchestrating multiple repository calls
- Generating pre-signed URLs (S3)

**Should:**
```go
// Receive typed values - no string parsing
func (s *Service) DeleteMemo(ctx context.Context, memoID, userID primitive.ObjectID) error {
    // Business logic: ownership check
    memo, err := s.repo.FindByID(ctx, memoID)
    if err != nil {
        return err
    }
    if memo.UserID != userID {
        return apperrors.ErrUnauthorized
    }
    return s.repo.SoftDelete(ctx, memoID)
}
```

**Should NOT:**
- Parse string IDs (that's handler's job)
- Know about HTTP status codes
- Return HTTP-specific errors

## Repository Layer (`internal/repository/`)

**Responsibilities:**
- Database CRUD operations
- Query building and execution
- Mapping database errors to app errors
- Soft-delete filtering

**Should:**
```go
// Interface-based for testability
type VoiceMemoRepository interface {
    FindByID(ctx context.Context, id primitive.ObjectID) (*models.VoiceMemo, error)
    SoftDelete(ctx context.Context, id primitive.ObjectID) error
}

// Return app errors, not mongo errors
if errors.Is(err, mongo.ErrNoDocuments) {
    return nil, apperrors.ErrVoiceMemoNotFound
}
```

**Should NOT:**
- Contain business logic
- Handle caching (that's service's job)
- Know about HTTP layer

## Request/Response DTOs (`internal/models/`)

**Request DTOs:**
```go
// Use binding tags for validation
type CreateUserRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
    Name     string `json:"name" binding:"required,min=2"`
}

// Use pointers for optional fields
type UpdateUserRequest struct {
    Email *string `json:"email" binding:"omitempty,email"`
    Name  *string `json:"name" binding:"omitempty,min=2"`
}
```

**Response DTOs:**
```go
// Exclude sensitive fields with json:"-"
type User struct {
    ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Email    string             `json:"email" bson:"email"`
    Password string             `json:"-" bson:"password"`  // Never in JSON
}
```

## Error Handling Flow

```
Handler                          Service                         Repository
   |                                |                                |
   |-- validate ID format --------->|                                |
   |   (400 Bad Request if invalid) |                                |
   |                                |                                |
   |-- call service(typed ID) ----->|                                |
   |                                |-- call repo(typed ID) -------->|
   |                                |                                |
   |                                |<-- ErrNotFound ----------------|
   |<-- ErrNotFound ----------------|                                |
   |                                |                                |
   |-- map to 404 Not Found ------->|                                |
```

## Migration Status

Some older code validates IDs in the service layer. New code should follow the handler-layer validation pattern. See `spec/delete-voice-memo.md` for the refactor plan.

| API                       | Current State      | Target State       |
| ------------------------- | ------------------ | ------------------ |
| `DELETE /voice-memos/:id` | Handler validation | Done               |
| `GET /users/:id`          | Service validation | Handler validation |
| `PUT /users/:id`          | Service validation | Handler validation |
| `DELETE /users/:id`       | Service validation | Handler validation |
| `GET /voice-memos`        | Service validation | Handler validation |
