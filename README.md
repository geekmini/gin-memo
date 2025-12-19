# gin-sample

A REST API for user management and voice memos built with Go, Gin, MongoDB, Redis, and S3.

## Features

- User CRUD with JWT authentication
- Voice memos with paginated listing
- Redis caching (cache-aside pattern)
- S3 pre-signed URLs for audio files
- Swagger API documentation
- Docker Compose for local development

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- [Task](https://taskfile.dev/) - `brew install go-task`

## Quick Start

```bash
# Clone and setup
cp .env.example .env

# Start all services (MongoDB, Redis, MinIO)
task docker:up

# Run with hot reload
task dev

# Or run without hot reload
task run
```

## Available Commands

| Command | Description |
|---------|-------------|
| `task dev` | Start services + hot reload |
| `task run` | Run the server |
| `task build` | Compile to binary |
| `task test` | Run tests |
| `task sync` | Sync dependencies |
| `task swagger` | Regenerate API docs |
| `task fmt` | Format code |
| `task lint` | Run linter |
| `task docker:up` | Start MongoDB + Redis + MinIO |
| `task docker:prod` | Start full stack in Docker |
| `task docker:down` | Stop Docker services |
| `task docker:logs` | View Docker logs |

## Local Services

| Service | Port | URL |
|---------|------|-----|
| API | 8080 | http://localhost:8080 |
| Swagger Docs | 8080 | http://localhost:8080/docs/index.html |
| MongoDB | 27017 | mongodb://localhost:27017 |
| Redis | 6379 | localhost:6379 |
| MinIO API | 9000 | http://localhost:9000 |
| MinIO Console | 9001 | http://localhost:9001 |

## Project Structure

```
gin-sample/
├── cmd/server/              # Entry point
├── internal/
│   ├── config/              # Environment config
│   ├── models/              # Data models
│   ├── repository/          # Database operations
│   ├── service/             # Business logic
│   ├── handler/             # HTTP handlers
│   ├── middleware/          # Auth, CORS
│   ├── cache/               # Redis client
│   ├── storage/             # S3 client
│   ├── errors/              # App errors
│   └── router/              # Route definitions
├── pkg/
│   ├── auth/                # JWT + bcrypt
│   └── response/            # API response helpers
├── swagger/                 # Generated API docs
├── spec/                    # API specifications
├── docs/                    # Project documentation
├── docker-compose.yml
├── Dockerfile
├── Taskfile.yml
└── CLAUDE.md                # AI assistant context
```

## API Endpoints

### Auth (Public)
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/register | Register new user |
| POST | /api/v1/auth/login | Login, get JWT |

### Users (Protected)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/users | List all users |
| GET | /api/v1/users/:id | Get user by ID |
| PUT | /api/v1/users/:id | Update user |
| DELETE | /api/v1/users/:id | Delete user |

### Voice Memos (Protected)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/voice-memos | List user's memos (paginated) |

### Other
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /health | Health check |
| GET | /docs/* | Swagger UI |

## Environment Variables

See `.env.example` for all required variables.

## License

MIT
