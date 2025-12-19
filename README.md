# gin-sample

A REST API built with Go, Gin, MongoDB, Redis, and JWT authentication.

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- [Task](https://taskfile.dev/) - `brew install go-task`

## Quick Start

```bash
# Start MongoDB and Redis
task docker:up

# Run the server
task run

# Test health endpoint
curl http://localhost:8080/health
```

## Available Commands

| Command            | Description           |
| ------------------ | --------------------- |
| `task run`         | Run the server        |
| `task build`       | Compile to binary     |
| `task test`        | Run tests             |
| `task sync`        | Sync dependencies     |
| `task fmt`         | Format code           |
| `task lint`        | Run linter            |
| `task docker:up`   | Start MongoDB + Redis |
| `task docker:down` | Stop Docker services  |
| `task docker:logs` | View Docker logs      |

## Project Structure

```
gin-sample/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── config/                  # Configuration
│   ├── models/                  # Data models
│   ├── repository/              # Database operations
│   ├── service/                 # Business logic
│   ├── handler/                 # HTTP handlers
│   ├── middleware/              # Auth, CORS
│   ├── cache/                   # Redis caching
│   └── router/                  # Route definitions
├── pkg/
│   ├── jwt/                     # JWT utilities
│   └── response/                # API response helpers
├── docker-compose.yml
├── Dockerfile
├── Taskfile.yml
├── .env
└── go.mod
```

## API Endpoints

| Method | Endpoint              | Description   | Auth |
| ------ | --------------------- | ------------- | ---- |
| GET    | /health               | Health check  | No   |
| POST   | /api/v1/auth/register | Register user | No   |
| POST   | /api/v1/auth/login    | Login         | No   |
| GET    | /api/v1/users         | List users    | Yes  |
| GET    | /api/v1/users/:id     | Get user      | Yes  |
| PUT    | /api/v1/users/:id     | Update user   | Yes  |
| DELETE | /api/v1/users/:id     | Delete user   | Yes  |
