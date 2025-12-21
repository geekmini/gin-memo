# Project Setup

Help a new engineer set up the gin-sample project for local development.

## Steps

1. **Check Prerequisites**
   Run the following commands to check if required tools are installed:
   - `go version` (need 1.25+)
   - `docker --version` or check OrbStack is running
   - `task --version`
   - `lefthook version`
   - `golangci-lint --version`
   - `air -v`
   - `swag --version`

2. **Install Missing Tools**
   For any missing tools, provide the installation command:
   - Go: `brew install go`
   - OrbStack: `brew install --cask orbstack`
   - Task: `go install github.com/go-task/task/v3/cmd/task@latest`
   - lefthook: `go install github.com/evilmartians/lefthook@latest`
   - golangci-lint: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
   - air: `go install github.com/air-verse/air@latest`
   - swag: `go install github.com/swaggo/swag/cmd/swag@latest`

   Remind user to add `$HOME/go/bin` to PATH if tools installed via `go install` are not found.

3. **Install Claude Code Plugins**
   The `/dev` workflow requires these plugins:
   - `feature-dev` - Code exploration, architecture, and review agents
   - `commit-commands` - Commit, push, and PR creation

   Check if plugins are installed:
   ```bash
   claude plugins list
   ```

   Install missing plugins:
   ```bash
   claude plugins add feature-dev
   claude plugins add commit-commands
   ```

4. **Environment Setup**
   - Check if `.env` exists, if not copy from `.env.example`
   - Run `task setup` to install git hooks

5. **Start Services**
   - Run `task docker:up` to start MongoDB, Redis, MinIO
   - Wait for services to be healthy

6. **Create Database Indexes**
   - Run `task index` to create MongoDB indexes
   - This is required for optimal query performance

7. **Seed Test Data** (optional)
   - Ask if user wants to seed test data
   - If yes, run `task seed`

8. **Verify Setup**
   - Verify Docker services are running: `docker compose ps`
   - Confirm MongoDB, Redis, and MinIO are healthy

9. **Summary**
   Print a summary of:
   - All tools installed
   - Claude Code plugins installed
   - Services running
   - Test credentials (if seeded): alice@example.com / password123

Be interactive and helpful. If any step fails, diagnose the issue and help fix it.
