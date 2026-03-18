# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Development (frontend + backend separate)
make dev-backend    # Start backend at http://127.0.0.1:8080
make dev-frontend   # Start frontend at http://127.0.0.1:5173

# Production build
make build-all      # Build frontend + backend (embeds web/dist)
make run            # Run the compiled binary

# Testing
make test                      # Run all Go tests
make test-coverage             # Generate coverage report
make test-coverage-check       # Fail if coverage < 40%
make test-ai                   # Run AI service tests
make test-cluster              # Run cluster service tests
make test-deployment           # Run deployment service tests
go test ./internal/ai/... -v   # Run AI module tests
make web-test                  # Run frontend tests
make web-test-coverage         # Run frontend tests with coverage

# Database migrations
make migrate-up      # Apply migrations
make migrate-status  # Check migration status
make migrate-down    # Rollback one step
```

## Architecture Overview

This is a PaaS platform (OpsPilot) for K8s cluster management with an AI assistant. The stack is:
- **Backend**: Go 1.26 + Gin + GORM + Cobra CLI
- **Frontend**: React 19 + Vite + Ant Design 6 + @ant-design/x-sdk
- **AI**: CloudWeGo Eino framework for LLM orchestration

### Backend Layered Architecture

```
internal/
├── cmd/           # CLI entrypoint (cobra commands)
├── config/        # Viper config loading
├── server/        # HTTP server setup
├── middleware/    # Auth, CORS, rate limiting
├── model/         # GORM models (organized by domain)
├── dao/           # Data access layer (CRUD only)
├── service/       # HTTP handlers + business logic
│   └── <domain>/  # Each domain has routes.go + handler/ + logic/
├── httpx/         # HTTP response helpers
├── xcode/         # Business error codes
└── ai/            # AI orchestration (see below)
```

### AI Module Architecture (`internal/ai/`)

The AI module uses CloudWeGo Eino ADK with a Plan-Execute-Replan architecture:

```
internal/ai/
├── orchestrator.go       # Runner + CheckPointStore for stateful execution
├── agents/
│   ├── agent.go          # NewAgent: assembles Planner → Executor → Replanner
│   ├── planner/          # Task decomposition
│   ├── executor/         # Tool execution with human-in-the-loop
│   └── replan/           # Dynamic plan adjustment
├── chatmodel/            # LLM client initialization
└── tools/                # Domain-organized tools
    ├── common/           # PlatformDeps, ToolMeta, ToolResult
    ├── platform/         # Resource discovery (clusters, hosts, services, namespaces, metrics)
    ├── kubernetes/       # K8s operations (readonly + write with approval)
    ├── host/             # Host management
    ├── service/          # Service operations
    ├── monitor/          # Monitoring tools
    ├── cicd/             # CI/CD operations
    ├── deployment/       # Deployment tools
    ├── infrastructure/   # Infrastructure management
    ├── governance/       # Policy & approval tools
    └── middleware/       # Approval middleware for write operations
```

**Key concepts:**
- Uses `adk.Runner` with `Interrupt/ResumeWithParams` for human-in-the-loop
- `feature_flags.ai_assistant_v2: true` enables this runtime (default)
- Max 20 iterations per conversation turn

**Tool Categories:**

| Category | Tools | Approval Required |
|----------|-------|-------------------|
| **Platform Discovery** | `platform_discover_resources` | No |
| **K8s Readonly** | `k8s_query`, `k8s_list_resources`, `k8s_events`, `k8s_logs` | No |
| **K8s Write** | `k8s_scale_deployment`, `k8s_restart_deployment`, `k8s_delete_pod`, `k8s_rollback_deployment`, `k8s_delete_deployment` | Yes |
| **Host Readonly** | `host_list_inventory`, `os_get_cpu_mem`, etc. | No |
| **Host Write** | `host_batch`, `host_batch_exec_apply`, `host_batch_status_update` | Yes |

**Approval Flow:**
1. Write tool invoked → `StatefulInterrupt` pauses execution
2. `ApprovalInfo` sent via SSE to frontend
3. User approves/rejects via UI
4. Result passed via `ResumeWithParams`
5. Execution continues or returns rejection message

### Frontend Structure

```
web/src/
├── api/modules/<domain>/   # API clients by domain
├── components/<Feature>/   # Feature-grouped components
├── pages/                  # Route pages
├── hooks/<concern>/        # Custom hooks by concern
└── utils/<purpose>/        # Utilities by purpose
```

## Key Conventions

### HTTP Response Format

All business APIs return HTTP 200 with unified body:
```json
{ "code": 1000, "msg": "请求成功", "data": {...} }
```

Use `internal/httpx` functions:
- `httpx.OK(c, data)` - Success (code 1000)
- `httpx.Fail(c, xcode.XXX, "message")` - Business error
- `httpx.BindErr(c, err)` - Parameter binding error
- `httpx.ServerErr(c, err)` - Server error (code 3000)
- `httpx.NotFound(c, msg)` - Resource not found (code 2005)

Never use `c.JSON()` with inline `gin.H{}`.

**Business error code ranges** (see `internal/xcode/code.go`):
| Range | Category | Examples |
|-------|----------|----------|
| 1000-1999 | Success | 1000 OK, 1001 Created |
| 2000-2999 | Client errors | 2000 ParamError, 2003 Unauthorized |
| 3000-3999 | Server errors | 3000 ServerError, 3001 DatabaseError |
| 4000-4999 | Business errors | 4005 TokenExpired, 4007 PermissionDenied |

### Code Organization

Directory file count thresholds:
| Category | Max Files |
|----------|-----------|
| Models | 10 |
| Handlers | 8 |
| Components | 12 |
| API Modules | 15 |

When exceeded, split into subdirectories by domain/feature.

### Database

- **No foreign keys** - Relationships handled at application layer
- Use `storage/migration/` for versioned migrations
- Auto-migrate available via `app.auto_migrate: true` in dev

### Comment Style

All Go code MUST have four-level Chinese comments:
1. Package-level (before `package` declaration)
2. Type/struct-level
3. Method/function-level (with parameters, returns, side effects)
4. Inline (explain "why", not "what")

Example:
```go
// Package orchestrator 实现 AI 编排核心逻辑。
//
// 架构概览:
//   Planner → Executor → Replanner
package ai

// Run 启动编排流水线。
//
// 参数:
//   - ctx: 上下文
//   - req: 请求参数
//
// 返回: 成功返回 nil，失败返回错误
func (o *Orchestrator) Run(ctx context.Context, req RunRequest) error
```

### Testing

- Table-driven tests preferred
- Test files alongside source (not separate `test/` directory)
- Coverage threshold: 40% minimum

## Configuration

Config file: `configs/config.yaml` (set via `--config` flag)

**Required environment variables** (`.env` file):
- `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD` - Database
- `REDIS_ADDR`, `REDIS_PASSWORD` - Cache
- `LLM_API_KEY` - LLM provider (qwen/ark/ollama)
- `MILVUS_HOST`, `MILVUS_PORT`, `MILVUS_DATABASE`, `MILVUS_COLLECTION` - Vector DB
- `PROMETHEUS_ADDRESS` - Metrics endpoint
- `SERVER_SALT`, `SERVER_SECRET` - Security
- `SECURITY_ENCRYPTION_KEY` - Data encryption

**Key feature flags:**
- `feature_flags.ai_assistant_v2: true` - Use Plan-Execute runtime (default)
- `feature_flags.ai_governed_host_execution: true` - Require AI approval for host operations
- `feature_flags.host_health_diagnostics: true` - Enable host health checks
- `feature_flags.host_maintenance_mode: true` - Enable maintenance mode

## API Endpoints

- All API routes prefixed with `/api/v1`
- Frontend served at `/` (embedded in production)
- SSE streaming for AI chat: `/api/v1/ai/chat`, `/api/v1/ai/resume/step/stream`
- Metrics endpoint: `/metrics` (when `metrics.enable: true`)

## Documentation

- `docs/ai/` - AI knowledge base for RAG (JSONL format)
- `docs/user/` - Help center and FAQ
- `openspec/specs/` - Historical feature specifications (completed tasks)
