# gin-sample

A REST API for user management built with Go, Gin, MongoDB, and Redis.

## Quick Commands

```bash
task dev          # Start MongoDB + Redis, run app with hot reload
task run          # Run without hot reload
task swagger      # Regenerate API docs after handler changes
task docker:prod  # Run full stack in Docker
task sync         # go mod tidy
```

## Architecture

Layered architecture with dependency injection:

```
cmd/server/main.go      # Entry point, wires dependencies
    ↓
internal/router         # HTTP routes, middleware chain
    ↓
internal/handler        # HTTP handlers (request/response)
    ↓
internal/service        # Business logic, caching
    ↓
internal/repository     # Database operations (MongoDB)
```

**Data flow:** Request → Router → Handler → Service → Repository → MongoDB

## Project Structure

```
cmd/server/main.go       # Entry point
internal/
  config/                # Environment config (godotenv)
  models/                # Data structs (User, requests, responses)
  repository/            # MongoDB CRUD (interface-based)
  service/               # Business logic + Redis caching
  handler/               # HTTP handlers with Swagger annotations
  middleware/            # Auth (JWT), CORS
  router/                # Route setup
  errors/                # Centralized app errors
  cache/                 # Redis client
pkg/
  auth/                  # JWT + bcrypt utilities
  response/              # Standard API response format
swagger/                 # Generated API docs (swag)
docs/                    # Project documentation
```

## Coding Conventions

### Naming
- Packages: lowercase, single word (`handler`, `service`, `repository`)
- Files: snake_case (`user_handler.go`, `user_service.go`)
- Exported: PascalCase (`UserService`, `NewUserHandler`)
- Unexported: camelCase (`userCacheTTL`, `jwtManager`)
- Interfaces: describe behavior (`UserRepository`, not `IUserRepository`)

### Error Handling
- Define errors in `internal/errors/errors.go`
- Use `errors.Is()` for comparison
- Return errors up the stack, handle at handler level
- Example:
  ```go
  if errors.Is(err, apperrors.ErrUserNotFound) {
      response.NotFound(c, err.Error())
      return
  }
  ```

### API Responses
Always use `pkg/response` helpers:
```go
response.Success(c, data)      // 200
response.Created(c, data)      // 201
response.BadRequest(c, msg)    // 400
response.Unauthorized(c, msg)  // 401
response.NotFound(c, msg)      // 404
response.Conflict(c, msg)      // 409
response.InternalError(c)      // 500
```

### Caching (Redis)
- Cache-aside pattern in service layer
- Cache key format: `user:{id}`
- TTL: 15 minutes for user data
- Invalidate on update/delete
- Cache errors are non-fatal (best effort)

## Adding New Features

### New Endpoint
1. Add model in `internal/models/`
2. Add repository method in `internal/repository/`
3. Add service method in `internal/service/`
4. Add handler with Swagger annotations in `internal/handler/`
5. Add route in `internal/router/router.go`
6. Run `task swagger` to regenerate docs

### Swagger Annotations
Add before each handler:
```go
// HandlerName godoc
// @Summary      Short description
// @Description  Longer description
// @Tags         tag-name
// @Accept       json
// @Produce      json
// @Param        name  path/body  type  required  "description"
// @Success      200   {object}   response.Response{data=models.Type}
// @Failure      400   {object}   response.Response
// @Security     BearerAuth  // for protected routes
// @Router       /path [method]
```

### New Error Type
Add to `internal/errors/errors.go`:
```go
var ErrSomething = errors.New("something went wrong")
```

## Authentication

- JWT tokens in Authorization header: `Bearer {token}`
- Token contains only user ID (no email)
- Protected routes use `middleware.Auth(jwtManager)`
- JWT secret from `JWT_SECRET` env var

## Environment Variables

Required in `.env`:
```
SERVER_PORT=8080
GIN_MODE=debug
MONGO_URI=localhost:27017
MONGO_DATABASE=gin_sample
REDIS_URI=localhost:6379
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h
```

## Testing

```bash
task test         # Run all tests
go test ./...     # Same thing
```

## API Documentation

- Swagger UI: http://localhost:8080/docs/index.html
- Regenerate after handler changes: `task swagger`
