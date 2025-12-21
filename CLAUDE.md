# gin-sample

A REST API for user management and voice memos built with Go, Gin, MongoDB, Redis, and S3.

## Quick Commands

See `Taskfile.yml` for all available tasks. Common ones:

```bash
task --list       # Show all tasks
task dev          # Start services + hot reload
task swagger      # Regenerate API docs
task index        # Create MongoDB indexes
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

**Detailed documentation:**
- [Layered Architecture Conventions](docs/architecture.md) - Layer responsibilities, error handling flow
- [Design Patterns](docs/design-patterns.md) - DI, soft delete, pagination, caching, tokens

## Project Structure

```
cmd/
  server/main.go         # Entry point
  index/main.go          # MongoDB index creation script
internal/
  config/                # Environment config (godotenv)
  models/                # Data structs (User, VoiceMemo, Team, requests, responses)
  repository/            # MongoDB CRUD (interface-based)
  service/               # Business logic + Redis caching + S3 URLs
  handler/               # HTTP handlers with Swagger annotations
  middleware/            # Auth (JWT), CORS
  router/                # Route setup
  errors/                # Centralized app errors
  cache/                 # Redis client
  storage/               # S3 client (pre-signed URLs)
  authz/                 # Authorization (team permissions)
pkg/
  auth/                  # JWT + bcrypt utilities
  response/              # Standard API response format
swagger/                 # Generated API docs (swag)
docs/                    # Project documentation
spec/                    # API specifications
```

## Local Services (Docker Compose)

See `docker-compose.yml` for service definitions. Start all services:

```bash
task docker:up
```

## Git Commits

Uses [Conventional Commits](https://www.conventionalcommits.org/). Pre-commit hooks enforce this.

**Format:** `<type>[optional scope]: <description>`

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`

## Documentation

- **Diagrams:** Use Mermaid in markdown files
- **Docs:** Place in `docs/` with kebab-case filenames
- **Specs:** Place in `spec/` directory

## Coding Conventions

### Naming
- Packages: lowercase, single word (`handler`, `service`)
- Files: snake_case (`user_handler.go`)
- Exported: PascalCase (`UserService`)
- Unexported: camelCase (`userCacheTTL`)
- Interfaces: describe behavior (`UserRepository`, not `IUserRepository`)

### Error Handling
- Define errors in `internal/errors/errors.go`
- Use `errors.Is()` for comparison
- Return errors up the stack, handle at handler level

### API Responses
Use `pkg/response` helpers:
```go
response.Success(c, data)      // 200
response.Created(c, data)      // 201
response.BadRequest(c, msg)    // 400
response.Unauthorized(c, msg)  // 401
response.Forbidden(c, msg)     // 403
response.NotFound(c, msg)      // 404
response.Conflict(c, msg)      // 409
response.InternalError(c)      // 500
```

## Adding New Features

### New Endpoint
1. Add model in `internal/models/`
2. Add repository method in `internal/repository/`
3. Add service method in `internal/service/`
4. Add handler with Swagger annotations in `internal/handler/`
5. Add route in `internal/router/router.go`
6. Run `task swagger` to regenerate docs

### Swagger Annotations
```go
// HandlerName godoc
// @Summary      Short description
// @Tags         tag-name
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
- Token contains only user ID
- Protected routes use `middleware.Auth(jwtManager)`

## Environment Variables

See `.env.example` for all required variables. Copy to `.env` and update values:

```bash
cp .env.example .env
```

## Testing

```bash
task test         # Run all tests
go test ./...     # Same thing
```

## API Endpoints

See Swagger for complete API documentation:
- **Swagger UI:** http://localhost:8080/docs/index.html
- **Swagger YAML:** `swagger/swagger.yaml`
- **Regenerate:** `task swagger`

**Route groups:** `/api/v1/auth/*`, `/api/v1/users/*`, `/api/v1/voice-memos/*`, `/api/v1/teams/*`, `/api/v1/invitations/*`

## Postman

**Default workspace and collection for MCP operations:**

| Resource   | Name      | ID                                           |
| ---------- | --------- | -------------------------------------------- |
| Workspace  | golang    | `1ee078be-5479-45b9-9d5a-883cd4c6ef50`       |
| Collection | go-sample | `25403495-bb644262-dce4-42ac-8cc4-810d8a328fc9` |
