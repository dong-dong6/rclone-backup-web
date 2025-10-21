# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rclone Backup Web V2.0 is a distributed backup management system built on a Hub-and-Spoke architecture. The Hub (central node) manages backup tasks and configuration, while Agents (distributed nodes) execute backup operations using Rclone. The system is written in Go (backend) and React + TypeScript (frontend), deployed via Docker Compose.

## Architecture

**Hub-and-Spoke Model:**
- **Hub**: Central management node running on `hub/` - PostgreSQL database, REST API (Gin framework), Web UI (React + Vite), and SSE for real-time updates
- **Agent**: Distributed execution nodes running on `agent/` - **Deployed as standalone binary**, connects to Hub via heartbeat, executes Rclone tasks, supports local fallback when Hub is unreachable
- **Note**: Agents are **no longer containerized** - they run as native Go binaries on target machines

**Key Services:**
- `hub-api`: Gin-based REST API server (hub/main.go:19)
- `web-ui`: React SPA with Ant Design (hub/web/)
- `postgres`: Primary data store
- `redis`: Session and cache storage

**Communication Flow:**
1. Agents send heartbeats to Hub (`/api/v1/agent/heartbeat`) every 30s
2. Hub responds with actions (SYNC_CONFIG, EXECUTE_TASK)
3. Agents execute tasks and stream logs back via SSE
4. If Hub unreachable >5min, Agents use cached config for local fallback

## Development Commands

### Building and Running

```bash
# Build all Docker images (local build, no external registry)
make build

# Start Hub services
make up

# Stop all services
make down

# Restart services
make restart
```

### Testing

```bash
# Run end-to-end tests
make test

# Run scheduling tests
make test-scheduling

# API health check
make test-api
```

### Frontend Development

```bash
cd hub/web
npm install
npm run dev      # Development server with hot reload
npm run build    # Production build
```

### Backend Development

```bash
# Format Go code
make fmt

# Run linters (requires golangci-lint)
make lint

# Development mode with live reload
make dev

# Enter Hub container shell
make shell

# Enter database shell
make db-shell
```

### Deployment Script

```bash
# Interactive deployment with auto-config
./deploy.sh hub              # Deploy Hub services

# Other operations
./deploy.sh status           # View service status
./deploy.sh logs [service]   # View logs
./deploy.sh backup           # Backup data directory
./deploy.sh clean            # Interactive data cleanup
```

## Agent Deployment

**Agents are deployed as standalone Go binaries, not Docker containers.**

### Building Agent Binary

```bash
cd agent
go build -o rclone-backup-agent main.go
```

### Running Agent

```bash
# Set environment variables
export HUB_URL=http://hub-server:8080
export AGENT_API_KEY=<api-key-from-hub-ui>
export AGENT_ID=<agent-id>
export HEARTBEAT_INTERVAL=30s

# Run agent
./rclone-backup-agent
```

Agents communicate with the Hub via REST API and execute backup tasks using Rclone installed on the host machine.

## Code Organization

### Hub Structure (`hub/`)

- `main.go`: Entry point, initializes DB, services, router, and graceful shutdown
- `api/`: HTTP handlers and middleware
  - `handler.go`: Main handler struct with dependencies
  - `admin_handlers.go`: Admin-facing endpoints (tasks, remotes, executions)
  - `agent_handlers.go`: Agent-facing endpoints (heartbeat, task sync, log streaming)
  - `middleware.go`: Auth middleware (AdminAuthMiddleware, AgentAuthMiddleware)
- `models/`: Database models and queries
  - `db.go`: Database connection pooling (pgx/v5)
  - `task.go`, `agent.go`, `execution.go`, `remote.go`, `user.go`: Domain models
- `services/`: Business logic
  - `scheduler.go`: Task scheduling coordinator
  - `cron_scheduler.go`: Cron expression parsing (robfig/cron/v3)
  - `execution_monitor.go`: Monitors task execution timeouts
  - `sse.go`: Server-Sent Events for real-time updates
  - `auth.go`: JWT token generation/validation
  - `crypto.go`: AES-256 encryption for sensitive data (rclone configs, passwords)
- `web/`: React frontend
  - `src/pages/`: Page components (Dashboard, Tasks, Agents, Remotes, Executions)
  - `src/components/`: Reusable UI components
  - `src/services/`: API client and auth service
  - `src/i18n/`: Internationalization (i18next)

### Agent Structure (`agent/`)

- `main.go`: Main agent loop with heartbeat, task execution, local fallback
- `main_sidecar.go`: Sidecar mode using Rclone RC API
- `main_standalone.go`: Standalone mode with embedded Rclone
- `services/`:
  - `hub_client.go`: HTTP client for Hub communication
  - `scheduler.go`: Local cron scheduler for fallback execution
- `rclone_manager/`: Rclone configuration and lifecycle management
- `executor/`: Task execution isolation (sandbox_executor.go)

**Local Fallback Mechanism:**
- Agent caches task config to disk (`/var/lib/rclone-agent/tasks.json`)
- If Hub unreachable >5min (`shouldExecuteLocalFallback()` in agent/main.go:482), Agent executes scheduled tasks from cache
- Prevents backup failures during Hub outages

## Database

**Schema Location:** `database/schema.sql` and `database/migrations/`

**Key Tables:**
- `users`: Admin users with password hashing
- `agents`: Registered agents with API keys
- `remotes`: Rclone remote configurations (encrypted)
- `tasks`: Backup task definitions with cron schedules
- `executions`: Task execution history and logs
- `registration_tokens`: One-time tokens for agent registration
- `audit_logs`: Security audit trail

**Migrations:** Applied automatically on Hub startup via `models.InitDB()`

## Environment Configuration

All configuration via `.env` file (see `.env.example`):

**Critical Security Keys:**
- `JWT_SECRET`: 64-char hex for JWT signing (generate: `openssl rand -hex 32`)
- `ENCRYPTION_KEY`: 32-char hex for AES-256 (generate: `openssl rand -hex 16`)
- `DB_PASSWORD`: PostgreSQL password

**Ports:**
- `WEB_PORT=3000`: Web UI (nginx proxies all services through this port)
- Hub API runs internally on 8080 (not exposed, accessed via /api)
- Metrics on internal port 9090 (accessed via /metrics through nginx)

**Agent Configuration:**
- Agent credentials obtained after registration via Hub UI
- `AGENT_ID` and `AGENT_API_KEY`: Agent authentication credentials
- `HEARTBEAT_INTERVAL=30s`: How often Agent sends heartbeat
- `ENABLE_LOCAL_FALLBACK=true`: Enable local execution when Hub unreachable
- `LOCAL_FALLBACK_THRESHOLD=5m`: Time before local fallback activates

## Data Persistence

All data stored in `./data/` directory:
- `./data/postgres/`: PostgreSQL data (requires 700 permissions)
- `./data/redis/`: Redis persistence
- `./data/hub/`: Hub config and logs
- `./data/backups/`: Automated database backups

## Key Design Patterns

**Scheduler Anti-Duplication:**
- Hub's `cron_scheduler.go` prevents duplicate task execution
- Uses `lastExecutedAt` timestamp and execution record checks
- Cron runs with seconds precision (cron.WithSeconds())

**SSE Real-time Updates:**
- Hub broadcasts events via `services.SSEService` (services/sse.go)
- Agents stream logs during execution to `/api/v1/agent/executions/:id/logs`
- Frontend subscribes to `/events` for live dashboard updates

**Agent Authentication:**
- Registration: Agent POSTs to `/api/v1/agent/register` with registration token
- Receives permanent API key stored in `AGENT_API_KEY` env var
- All subsequent requests use `Authorization: Bearer <api_key>` header
- Middleware validates via `AgentAuthMiddleware` (api/middleware.go)

**Graceful Shutdown:**
- Hub: 30s timeout for in-flight requests (hub/main.go:177)
- Agent: Stops cron scheduler, completes running task (agent/main.go:574)

## Testing Strategy

**E2E Tests:** `test/e2e_test.sh` - Full deployment cycle
**Scheduling Tests:** `test/test_scheduling.sh` - Cron scheduler validation

When writing tests:
- Use `make test-api` to validate Hub API health
- Database state via `make db-shell` for manual verification

## Common Tasks

**Add a new API endpoint:**
1. Define handler in `hub/api/admin_handlers.go` or `hub/api/agent_handlers.go`
2. Add route in `hub/main.go` (line ~60 for admin, ~63 for agent)
3. Add middleware if auth required (AdminAuthMiddleware or AgentAuthMiddleware)
4. Update frontend API client in `hub/web/src/services/`

**Add database table:**
1. Create migration in `database/migrations/`
2. Add model struct in `hub/models/`
3. Implement CRUD methods on model
4. Run `make down && make up` to apply migration

**Modify Agent behavior:**
- Heartbeat logic: `agent/main.go` sendHeartbeat() and heartbeatLoop()
- Task execution: `agent/main.go` executeTask() and executeTaskDirect()
- Local fallback: `agent/main.go` shouldExecuteLocalFallback() and executeLocalFallback()
- Rclone integration: `agent/rclone_manager/manager.go`

**Update frontend:**
- Page components in `hub/web/src/pages/`
- API calls via `hub/web/src/services/api.ts`
- UI components use Ant Design v5
- Internationalization keys in `hub/web/src/i18n/locales/`

## Deployment Notes

**Production deployment uses nginx reverse proxy:**
- All services accessed through single port (WEB_PORT=3000)
- `/api/*` → hub-api:8080
- `/events` → hub-api:8080 (SSE)
- `/metrics` → hub-api:9090
- `/*` → web-ui:80

**Docker Compose profiles:**
- Default: Hub services only
- `db-backup`: Automated database backup service

**Version Detection:**
- Script auto-detects `docker compose` (V2) vs `docker-compose` (V1)
- All Makefiles and scripts handle both versions

## Important Go Packages

- `github.com/gin-gonic/gin`: HTTP router and middleware
- `github.com/jackc/pgx/v5`: PostgreSQL driver with connection pooling
- `github.com/golang-jwt/jwt/v5`: JWT token handling
- `github.com/robfig/cron/v3`: Cron expression parsing and scheduling
- `golang.org/x/crypto`: Password hashing (bcrypt)

## Important Frontend Libraries

- `react-router-dom`: Client-side routing
- `antd`: UI component library
- `axios`: HTTP client
- `recharts`: Dashboard charts
- `dayjs`: Date/time formatting
- `i18next`: Internationalization

## Security Considerations

- All sensitive data (rclone configs, remote passwords) encrypted via `CryptoService` using AES-256
- JWT tokens expire based on `SESSION_TIMEOUT` (default 24h)
- Agent API keys stored hashed in database
- CORS enabled for development (`Access-Control-Allow-Origin: *` in hub/main.go:44)
- No plaintext credentials in logs or error messages
