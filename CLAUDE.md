# gin-sample

A REST API for user management and voice memos built with Go, Gin, MongoDB, Redis, and S3.

## Quick Commands

```bash
task setup        # Install git hooks (run once after clone)
task dev          # Start MongoDB + Redis + MinIO, run app with hot reload
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
internal/service        # Business logic, caching, S3 URLs
    ↓
internal/repository     # Database operations (MongoDB)
```

**Data flow:** Request → Router → Handler → Service → Repository → MongoDB

## Project Structure

```
cmd/server/main.go       # Entry point
internal/
  config/                # Environment config (godotenv)
  models/                # Data structs (User, VoiceMemo, requests, responses)
  repository/            # MongoDB CRUD (interface-based)
  service/               # Business logic + Redis caching + S3 URLs
  handler/               # HTTP handlers with Swagger annotations
  middleware/            # Auth (JWT), CORS
  router/                # Route setup
  errors/                # Centralized app errors
  cache/                 # Redis client
  storage/               # S3 client (pre-signed URLs)
pkg/
  auth/                  # JWT + bcrypt utilities
  response/              # Standard API response format
swagger/                 # Generated API docs (swag)
docs/                    # Project documentation
spec/                    # API specifications
```

## Local Services (Docker Compose)

| Service       | Port  | Purpose                        |
| ------------- | ----- | ------------------------------ |
| MongoDB       | 27017 | Database                       |
| Redis         | 6379  | Cache                          |
| MinIO         | 9000  | S3-compatible storage (API)    |
| MinIO Console | 9001  | Web UI (minioadmin/minioadmin) |

Start all services:
```bash
task docker:up
```

## Git Commits

This project uses [Conventional Commits](https://www.conventionalcommits.org/). Pre-commit hooks enforce this.

**Format:** `<type>[optional scope]: <description>`

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style (formatting, semicolons)
- `refactor`: Code change (no feature/fix)
- `perf`: Performance improvement
- `test`: Adding/updating tests
- `build`: Build system or dependencies
- `ci`: CI configuration
- `chore`: Other changes

**Examples:**
```
feat: add user authentication
fix(api): handle null response
docs: update README
feat!: breaking change to API
```

## Documentation

### Diagrams
- Always use **Mermaid** for diagrams in markdown files
- Supported diagram types: `flowchart`, `sequenceDiagram`, `graph`, `gantt`, `classDiagram`
- Example:
  ```mermaid
  flowchart TD
      A[Start] --> B{Decision}
      B -->|Yes| C[Action]
      B -->|No| D[End]
  ```

### Documentation Files
- Place in `docs/` directory
- Use descriptive filenames with kebab-case (`logout-strategies.md`)
- API specifications go in `spec/` directory

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

### S3 Storage
- Use `internal/storage/s3.go` for S3 operations
- Generate pre-signed URLs for private files (1 hour expiry)
- Store S3 key in database, generate URL on request
- MinIO for local development, real S3 in production

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
# Server
SERVER_PORT=8080
GIN_MODE=debug

# MongoDB
MONGO_URI=localhost:27017
MONGO_DATABASE=gin_sample

# Redis
REDIS_URI=localhost:6379

# Auth Tokens
ACCESS_TOKEN_SECRET=your-secret-key
ACCESS_TOKEN_EXPIRY=15m
REFRESH_TOKEN_EXPIRY=168h

# S3 (MinIO for local)
S3_ENDPOINT=localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=voice-memos
S3_USE_SSL=false
```

## Testing

```bash
task test         # Run all tests
go test ./...     # Same thing
```

## API Documentation

- Swagger UI: http://localhost:8080/docs/index.html
- Regenerate after handler changes: `task swagger`

## API Endpoints

### Auth (Public)
- `POST /api/v1/auth/register` - Register new user, returns access + refresh tokens
- `POST /api/v1/auth/login` - Login, returns access + refresh tokens
- `POST /api/v1/auth/refresh` - Exchange refresh token for new access token

### Auth (Protected)
- `POST /api/v1/auth/logout` - Invalidate refresh token

### Users (Protected)
- `GET /api/v1/users` - List all users
- `GET /api/v1/users/:id` - Get user by ID
- `PUT /api/v1/users/:id` - Update user
- `DELETE /api/v1/users/:id` - Delete user

### Voice Memos (Protected)
- `GET /api/v1/voice-memos` - List user's voice memos (paginated)
