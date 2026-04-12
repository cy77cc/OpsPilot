# Backend Restructuring Phase 1 & 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the Go backend into a Modular Monolith by consolidating infrastructure into `internal/core/` and business logic into `internal/modules/`.

**Architecture:** 
- `internal/core/`: Cross-cutting concerns (logger, config, middleware, httpx, cache, storage).
- `internal/modules/`: Domain-specific vertical slices (model, repository, service, handler).
- `cmd/opspilot/`: Standard Go entrypoints.

**Tech Stack:** Go 1.26.1, Gorm, Gin, Zap.

---

### Task 1: Setup Core Infrastructure

**Files:**
- Create: `internal/core/logger/`, `internal/core/config/`, `internal/core/middleware/`, `internal/core/httpx/`, `internal/core/cache/`, `internal/core/utils/`
- Move: `internal/logger/*` -> `internal/core/logger/`
- Move: `internal/config/*` -> `internal/core/config/`
- Move: `internal/middleware/*` -> `internal/core/middleware/`
- Move: `internal/httpx/*` -> `internal/core/httpx/`
- Move: `internal/cache/*` -> `internal/core/cache/`
- Move: `internal/utils/*` -> `internal/core/utils/`

- [ ] **Step 1: Create new core directories**
Run: `mkdir -p internal/core/logger internal/core/config internal/core/middleware internal/core/httpx internal/core/cache internal/core/utils`

- [ ] **Step 2: Move logger files and update imports**
Move all files from `internal/logger/` to `internal/core/logger/`.
Update package name to `logger`.
Update all imports in the project from `github.com/cy77cc/OpsPilot/internal/core/logger` to `github.com/cy77cc/OpsPilot/internal/core/logger`.

- [ ] **Step 3: Move config files and update imports**
Move all files from `internal/config/` to `internal/core/config/`.
Update package name to `config`.
Update all imports in the project from `github.com/cy77cc/OpsPilot/internal/core/config` to `github.com/cy77cc/OpsPilot/internal/core/config`.

- [ ] **Step 4: Move middleware files and update imports**
Move all files from `internal/middleware/` to `internal/core/middleware/`.
Update package name to `middleware`.
Update all imports in the project from `github.com/cy77cc/OpsPilot/internal/core/middleware` to `github.com/cy77cc/OpsPilot/internal/core/middleware`.

- [ ] **Step 5: Move httpx files and update imports**
Move all files from `internal/httpx/` to `internal/core/httpx/`.
Update package name to `httpx`.
Update all imports in the project from `github.com/cy77cc/OpsPilot/internal/core/httpx` to `github.com/cy77cc/OpsPilot/internal/core/httpx`.

- [ ] **Step 6: Move cache files and update imports**
Move all files from `internal/cache/` to `internal/core/cache/`.
Update package name to `cache`.
Update all imports in the project from `github.com/cy77cc/OpsPilot/internal/core/cache` to `github.com/cy77cc/OpsPilot/internal/core/cache`.

- [ ] **Step 7: Move utils files and update imports**
Move all files from `internal/utils/` to `internal/core/utils/`.
Update package name to `utils`.
Update all imports in the project from `github.com/cy77cc/OpsPilot/internal/core/utils` to `github.com/cy77cc/OpsPilot/internal/core/utils`.

- [ ] **Step 8: Verify build**
Run: `go build ./...`

- [ ] **Step 9: Commit**
```bash
git add .
git commit -m "refactor: move infrastructure to internal/core"
```

### Task 2: Consolidate Storage

**Files:**
- Create: `internal/core/storage/`
- Move: `storage/*` -> `internal/core/storage/`

- [ ] **Step 1: Create storage directory**
Run: `mkdir -p internal/core/storage`

- [ ] **Step 2: Move storage files and update imports**
Move all files from `storage/` to `internal/core/storage/`.
Update package name to `storage`.
Update all imports in the project from `github.com/cy77cc/OpsPilot/storage` to `github.com/cy77cc/OpsPilot/internal/core/storage`.

- [ ] **Step 3: Verify build**
Run: `go build ./...`

- [ ] **Step 4: Commit**
```bash
git add .
git commit -m "refactor: move storage to internal/core/storage"
```

### Task 3: Move CMD entrypoints

**Files:**
- Create: `cmd/opspilot/`
- Move: `internal/cmd/*` -> `cmd/opspilot/`

- [ ] **Step 1: Create cmd directory**
Run: `mkdir -p cmd/opspilot`

- [ ] **Step 2: Move cmd files**
Move all files from `internal/cmd/` to `cmd/opspilot/`.
Note: `main.go` at root should be updated to point to the new `cmd/opspilot`.

- [ ] **Step 3: Update main.go**
Update root `main.go` to import the new cmd package if necessary, or move root `main.go` to `cmd/opspilot/main.go`.
Standard layout: `cmd/opspilot/main.go` is the entrypoint. Root `main.go` often just calls it or is deleted.

- [ ] **Step 4: Verify build**
Run: `go build -o bin/opspilot ./cmd/opspilot`

- [ ] **Step 5: Commit**
```bash
git add .
git commit -m "refactor: move cmd to cmd/opspilot"
```

### Task 4: Proof of Concept - AI Module

**Files:**
- Create: `internal/modules/ai/`
- Move: `internal/model/ai.go` -> `internal/modules/ai/model.go`
- Move: `internal/dao/ai/*` -> `internal/modules/ai/repository.go` (consolidated or kept as separate files)
- Move: `internal/service/ai/*` -> `internal/modules/ai/service.go`
- Move: `internal/server/ai/*` and `internal/svc/ai/*` -> `internal/modules/ai/handler.go`
- Move: `internal/ai/*` -> `internal/modules/ai/` (special logic)

- [ ] **Step 1: Create AI module directory**
Run: `mkdir -p internal/modules/ai`

- [ ] **Step 2: Move AI model and update package**
Move `internal/model/ai.go` to `internal/modules/ai/model.go`.
Change package to `ai`.
Update all imports of `github.com/cy77cc/OpsPilot/internal/model` that specifically used `Ai` related structs. This might be tricky if `internal/model` is used elsewhere.
Wait: In DDD, we want `internal/modules/ai/model.go` to contain AI-specific models.
If other packages depend on these models, they now import `github.com/cy77cc/OpsPilot/internal/modules/ai`.

- [ ] **Step 3: Move AI DAO to repository**
Move files from `internal/dao/ai/` to `internal/modules/ai/`.
Rename them if appropriate (e.g., `chat_dao.go` -> `repository_chat.go`).
Change package to `ai`.

- [ ] **Step 4: Move AI Service**
Move files from `internal/service/ai/` to `internal/modules/ai/`.
Change package to `ai`.

- [ ] **Step 5: Move AI Server/Svc to handler**
Move files from `internal/server/ai/` and `internal/svc/ai/` to `internal/modules/ai/`.
Change package to `ai`.

- [ ] **Step 6: Move existing internal/ai logic**
Move everything from `internal/ai/` to `internal/modules/ai/`.

- [ ] **Step 7: Update all project-wide imports for AI**
This is the most complex step. All code that interacted with AI must now point to the single `ai` package.

- [ ] **Step 8: Verify build and tests**
Run: `go build ./...`
Run AI-related tests.

- [ ] **Step 9: Commit**
```bash
git add .
git commit -m "refactor: migrate AI module to Modular Monolith structure"
```
