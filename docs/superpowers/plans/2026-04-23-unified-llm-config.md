# Unified LLM Config & Hot Update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify LLM configuration by migrating from `config.yaml` to the database, ensuring all API keys are encrypted, and enabling hot updates for model selection.

**Architecture:** Implement a bootstrap migration check that triggers if the database is empty, move API key encryption into the logic layer, and refactor the runtime model factory to prioritize database lookups.

**Tech Stack:** Go (Golang), GORM, AES-GCM (internal utility), Eino/Cloudwego (AI framework).

---

### Task 1: API Key Encryption in Logic Layer

**Files:**
- Modify: `internal/modules/llmprovider/logic/service.go`
- Modify: `internal/modules/llmprovider/dao/dao.go`

- [ ] **Step 1: Implement automatic encryption in Create/Update**
Modify `internal/modules/llmprovider/logic/service.go` to encrypt the `APIKey` before calling the DAO.

- [ ] **Step 2: Commit**
```bash
git add internal/modules/llmprovider/logic/service.go
git commit -m "feat(llm): add automatic encryption to provider logic"
```

---

### Task 2: Bootstrap Migration (YAML to DB)

**Files:**
- Modify: `internal/svc/ai_runtime.go`
- Create: `internal/svc/llm_migration.go`

- [ ] **Step 1: Create migration logic**
Implement the one-time migration from `config.yaml` to the database.

- [ ] **Step 2: Hook migration into initialization**
Call the migration function in `initAIRuntime` in `internal/svc/ai_runtime.go`.

- [ ] **Step 3: Commit**
```bash
git add internal/svc/ai_runtime.go internal/svc/llm_migration.go
git commit -m "feat(llm): implement bootstrap migration from yaml to db"
```

---

### Task 3: Refactor Runtime Client for Hot Updates

**Files:**
- Modify: `internal/modules/llmprovider/client/model.go`

- [ ] **Step 1: Ensure DB-first lookup and remove YAML fallback**
Refactor `GetDefaultChatModel` to strictly use the database and remove the `newConfiguredChatModel` fallback if a default exists.

- [ ] **Step 2: Commit**
```bash
git add internal/modules/llmprovider/client/model.go
git commit -m "refactor(llm): prioritize database for hot updates and remove yaml fallback"
```

---

### Task 4: Add New Providers (DeepSeek, Claude, Gemini, Azure)

**Files:**
- Create: `internal/modules/llmprovider/client/deepseek.go`
- Create: `internal/modules/llmprovider/client/claude.go`
- Create: `internal/modules/llmprovider/client/gemini.go`
- Create: `internal/modules/llmprovider/client/azure.go`

- [ ] **Step 1: Implement DeepSeek factory**
- [ ] **Step 2: Implement Claude factory**
- [ ] **Step 3: Implement Gemini factory**
- [ ] **Step 4: Implement Azure OpenAI factory**
- [ ] **Step 5: Commit**
```bash
git add internal/modules/llmprovider/client/*.go
git commit -m "feat(llm): add DeepSeek, Claude, Gemini, and Azure providers"
```

---

### Task 5: Final Verification

- [ ] **Step 1: Run integration test for migration**
- [ ] **Step 2: Verify API Key is encrypted in DB**
- [ ] **Step 3: Verify Hot Update (Change default in DB, check runtime effect)**
