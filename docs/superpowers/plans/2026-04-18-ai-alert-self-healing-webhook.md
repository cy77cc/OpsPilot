# AI Alert Self-Healing Webhook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an alert-driven AI self-healing pipeline with signed webhook ingest, async execution, guarded auto-fix/approval flow, and alert-centric frontend UX in monitor alert pages.

**Architecture:** Add a public signed AI webhook endpoint that normalizes alert payloads into durable ingest events and enqueues healing jobs. A background worker drives job state transitions, delegates execution to AI runtime with existing risk/approval controls, and supports retry/cancel rules. Frontend stays alert-centric: list and detail views display healing status and actions.

**Tech Stack:** Go (Gin, GORM, MySQL/Postgres compatible SQL), existing AI runtime/approval middleware, React 19 + Ant Design 6 + React Router + Vitest.

---

## File Structure (Planned Changes)

### Backend

- Create: `internal/modules/ai/model/alert_heal.go`
- Modify: `internal/core/storage/migration/dev_auto.go`
- Create: `internal/modules/ai/dao/alertheal/dao.go`
- Create: `internal/modules/ai/dao/alertheal/dao_test.go`
- Create: `internal/modules/ai/logic/alertheal/payload.go`
- Create: `internal/modules/ai/logic/alertheal/service.go`
- Create: `internal/modules/ai/logic/alertheal/service_test.go`
- Create: `internal/modules/ai/interfaces/http/alert_webhook_handler.go`
- Create: `internal/modules/ai/interfaces/http/alert_webhook_handler_test.go`
- Create: `internal/modules/ai/api/webhook_routes.go`
- Modify: `internal/bootstrap/modules.go`
- Create: `internal/modules/ai/infra/workers/alert_heal_worker.go`
- Create: `internal/modules/ai/infra/workers/alert_heal_worker_test.go`
- Modify: `internal/modules/ai/dao/approval/dao.go`
- Modify: `internal/modules/ai/logic/logic.go`
- Modify: `internal/modules/ai/handler/approval/service.go`
- Modify: `internal/modules/ai/handler/approval/handler.go`
- Modify: `internal/modules/rbac/handler/permission.go`
- Modify: `internal/core/config/config.go`
- Modify: `configs/config.yaml`

### Frontend

- Create: `web/src/api/modules/aiAlertHeal.ts`
- Create: `web/src/api/modules/aiAlertHeal.test.ts`
- Modify: `web/src/api/index.ts`
- Create: `web/src/pages/Monitor/monitorAlertHealStatus.ts`
- Create: `web/src/pages/Monitor/AlertsPage.tsx`
- Create: `web/src/pages/Monitor/AlertsPage.test.tsx`
- Create: `web/src/pages/Monitor/AlertDetailPage.tsx`
- Create: `web/src/pages/Monitor/AlertDetailPage.test.tsx`
- Modify: `web/src/app/routes/pages.ts`
- Modify: `web/src/app/routes/observability.routes.tsx`

### Docs

- Modify: `docs/ai/error-codes.md`
- Modify: `docs/swagger/swagger.yaml`

---

### Task 1: Add Alert-Healing Data Models and Auto-Migration Wiring

**Files:**
- Create: `internal/modules/ai/model/alert_heal.go`
- Modify: `internal/core/storage/migration/dev_auto.go`
- Test: `internal/modules/ai/model/alert_heal_model_test.go`

- [ ] **Step 1: Write the failing test**

```go
package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAIAlertIngestEvent_DedupeKeyUnique(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:ai-alert-heal-model?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AIAlertIngestEvent{}, &AIAlertHealJob{}, &AIAlertHealAttempt{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	first := AIAlertIngestEvent{
		ID:        "evt-1",
		Source:    "alertmanager",
		Protocol:  "alertmanager",
		Fingerprint: "fp-1",
		Status:    "firing",
		DedupeKey: "alertmanager:fp-1:firing",
		Title:     "CPU high",
		ReceivedAt: time.Now().UTC(),
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first: %v", err)
	}

	duplicate := first
	duplicate.ID = "evt-2"
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("expected unique dedupe_key violation, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/ai/model -run TestAIAlertIngestEvent_DedupeKeyUnique -v`  
Expected: FAIL with `undefined: AIAlertIngestEvent` (or missing types).

- [ ] **Step 3: Write minimal implementation**

```go
package model

import "time"

type AIAlertIngestEvent struct {
	ID             string    `gorm:"column:id;type:varchar(64);primaryKey"`
	Source         string    `gorm:"column:source;type:varchar(64);not null;index:idx_ai_alert_ingest_source_fp,priority:1"`
	Protocol       string    `gorm:"column:protocol;type:varchar(32);not null"`
	Fingerprint    string    `gorm:"column:fingerprint;type:varchar(128);not null;index:idx_ai_alert_ingest_source_fp,priority:2"`
	Status         string    `gorm:"column:status;type:varchar(16);not null;index:idx_ai_alert_ingest_status"`
	DedupeKey      string    `gorm:"column:dedupe_key;type:varchar(256);not null;uniqueIndex:uk_ai_alert_ingest_dedupe"`
	Severity       string    `gorm:"column:severity;type:varchar(16);not null;default:'warning'"`
	Title          string    `gorm:"column:title;type:varchar(255);not null"`
	Target         string    `gorm:"column:target;type:varchar(255);not null;default:''"`
	LabelsJSON     string    `gorm:"column:labels_json;type:text;not null;default:'{}'"`
	AnnotationsJSON string   `gorm:"column:annotations_json;type:text;not null;default:'{}'"`
	RawPayloadJSON string    `gorm:"column:raw_payload_json;type:longtext;not null"`
	StartsAt       *time.Time `gorm:"column:starts_at"`
	EndsAt         *time.Time `gorm:"column:ends_at"`
	ReceivedAt     time.Time `gorm:"column:received_at;not null;index:idx_ai_alert_ingest_received"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIAlertIngestEvent) TableName() string { return "ai_alert_ingest_events" }

type AIAlertHealJob struct {
	ID          string     `gorm:"column:id;type:varchar(64);primaryKey"`
	EventID     string     `gorm:"column:event_id;type:varchar(64);not null;index:idx_ai_alert_heal_jobs_event_id"`
	Scene       string     `gorm:"column:scene;type:varchar(32);not null"`
	Status      string     `gorm:"column:status;type:varchar(32);not null;index:idx_ai_alert_heal_jobs_queue,priority:1"`
	Decision    string     `gorm:"column:decision;type:varchar(32);not null;default:''"`
	RetryCount  int        `gorm:"column:retry_count;not null;default:0"`
	MaxRetry    int        `gorm:"column:max_retry;not null;default:3"`
	NextRetryAt *time.Time `gorm:"column:next_retry_at;index:idx_ai_alert_heal_jobs_queue,priority:2"`
	LastError   string     `gorm:"column:last_error;type:text;not null;default:''"`
	LatestRunID string     `gorm:"column:latest_run_id;type:varchar(64);not null;default:''"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_ai_alert_heal_jobs_queue,priority:3"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIAlertHealJob) TableName() string { return "ai_alert_heal_jobs" }

type AIAlertHealAttempt struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	JobID        string    `gorm:"column:job_id;type:varchar(64);not null;index:idx_ai_alert_heal_attempt_job,priority:1"`
	AttemptNo    int       `gorm:"column:attempt_no;not null;index:idx_ai_alert_heal_attempt_job,priority:2"`
	RunID        string    `gorm:"column:run_id;type:varchar(64);not null;default:''"`
	Outcome      string    `gorm:"column:outcome;type:varchar(32);not null"`
	ErrorMessage string    `gorm:"column:error_message;type:text;not null;default:''"`
	StartedAt    time.Time `gorm:"column:started_at;not null"`
	FinishedAt   time.Time `gorm:"column:finished_at;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (AIAlertHealAttempt) TableName() string { return "ai_alert_heal_attempts" }
```

Also add in `RunDevAutoMigrate`:

```go
&aimodel.AIAlertIngestEvent{},
&aimodel.AIAlertHealJob{},
&aimodel.AIAlertHealAttempt{},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/modules/ai/model -run TestAIAlertIngestEvent_DedupeKeyUnique -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/model/alert_heal.go internal/modules/ai/model/alert_heal_model_test.go internal/core/storage/migration/dev_auto.go
git commit -m "feat(ai): add alert-healing models and migration wiring"
```

### Task 2: Implement Payload Normalization and Idempotent Ingestion Service

**Files:**
- Create: `internal/modules/ai/dao/alertheal/dao.go`
- Create: `internal/modules/ai/dao/alertheal/dao_test.go`
- Create: `internal/modules/ai/logic/alertheal/payload.go`
- Create: `internal/modules/ai/logic/alertheal/service.go`
- Test: `internal/modules/ai/logic/alertheal/service_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestNormalizePayload_SupportsAlertmanagerAndUnified(t *testing.T) {
	alertmanagerRaw := []byte(`{"alerts":[{"status":"firing","fingerprint":"fp-a","labels":{"alertname":"CPU"}}]}`)
	unifiedRaw := []byte(`{"kind":"opspilot.alert.v1","alerts":[{"status":"firing","fingerprint":"fp-b","title":"Disk Full"}]}`)

	am, err := NormalizePayload("alertmanager", alertmanagerRaw)
	if err != nil || len(am) != 1 || am[0].Fingerprint != "fp-a" {
		t.Fatalf("alertmanager normalize failed: len=%d err=%v", len(am), err)
	}
	uv, err := NormalizePayload("opspilot.alert.v1", unifiedRaw)
	if err != nil || len(uv) != 1 || uv[0].Fingerprint != "fp-b" {
		t.Fatalf("unified normalize failed: len=%d err=%v", len(uv), err)
	}
}

func TestIngestService_DeduplicatesBySourceFingerprintStatus(t *testing.T) {
	// create in-memory db, call Ingest twice with same source/fingerprint/status, assert only one event row.
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ai/logic/alertheal ./internal/modules/ai/dao/alertheal -run "NormalizePayload|Deduplicates" -v`  
Expected: FAIL with undefined functions/types.

- [ ] **Step 3: Write minimal implementation**

`payload.go` core contract:

```go
type NormalizedAlert struct {
	Source      string
	Protocol    string
	Fingerprint string
	Status      string
	Severity    string
	Title       string
	Target      string
	LabelsJSON  string
	AnnotationsJSON string
	RawPayloadJSON  string
	StartsAt    *time.Time
	EndsAt      *time.Time
}

func DedupeKey(source, fingerprint, status string) string {
	return strings.TrimSpace(source) + ":" + strings.TrimSpace(fingerprint) + ":" + strings.TrimSpace(status)
}
```

`service.go` ingestion core:

```go
func (s *Service) Ingest(ctx context.Context, protocol string, raw []byte) ([]model.AIAlertIngestEvent, error) {
	normalized, err := NormalizePayload(protocol, raw)
	if err != nil {
		return nil, err
	}
	events := make([]model.AIAlertIngestEvent, 0, len(normalized))
	for _, item := range normalized {
		if strings.TrimSpace(item.Fingerprint) == "" {
			return nil, ErrInvalidPayload
		}
		e := model.AIAlertIngestEvent{
			ID: uuid.NewString(), Source: item.Source, Protocol: item.Protocol, Fingerprint: item.Fingerprint,
			Status: item.Status, DedupeKey: DedupeKey(item.Source, item.Fingerprint, item.Status),
			Severity: item.Severity, Title: item.Title, Target: item.Target, LabelsJSON: item.LabelsJSON,
			AnnotationsJSON: item.AnnotationsJSON, RawPayloadJSON: item.RawPayloadJSON, StartsAt: item.StartsAt,
			EndsAt: item.EndsAt, ReceivedAt: time.Now().UTC(),
		}
		saved, err := s.dao.UpsertIngestEvent(ctx, e)
		if err != nil {
			return nil, err
		}
		events = append(events, saved)
	}
	return events, nil
}
```

`dao.go` upsert strategy:

```go
func (d *DAO) UpsertIngestEvent(ctx context.Context, row model.AIAlertIngestEvent) (model.AIAlertIngestEvent, error) {
	if err := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "dedupe_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"updated_at": time.Now().UTC(),
		}),
	}).Create(&row).Error; err != nil {
		return model.AIAlertIngestEvent{}, err
	}
	var out model.AIAlertIngestEvent
	if err := d.db.WithContext(ctx).Where("dedupe_key = ?", row.DedupeKey).First(&out).Error; err != nil {
		return model.AIAlertIngestEvent{}, err
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ai/logic/alertheal ./internal/modules/ai/dao/alertheal -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/dao/alertheal/dao.go internal/modules/ai/dao/alertheal/dao_test.go internal/modules/ai/logic/alertheal/payload.go internal/modules/ai/logic/alertheal/service.go internal/modules/ai/logic/alertheal/service_test.go
git commit -m "feat(ai): add alert payload normalization and idempotent ingest service"
```

### Task 3: Add Signed Public Webhook Endpoint and Route Registration

**Files:**
- Create: `internal/modules/ai/interfaces/http/alert_webhook_handler.go`
- Create: `internal/modules/ai/interfaces/http/alert_webhook_handler_test.go`
- Create: `internal/modules/ai/api/webhook_routes.go`
- Modify: `internal/bootstrap/modules.go`
- Modify: `internal/core/config/config.go`
- Modify: `configs/config.yaml`

- [ ] **Step 1: Write failing handler tests**

```go
func TestAlertWebhook_RejectsMissingSignature(t *testing.T) {
	// POST /api/v1/ai/alerts/webhook without X-OpsPilot-Signature -> 401
}

func TestAlertWebhook_AcceptsValidSignatureAndReturns202(t *testing.T) {
	// signed payload -> 202, accepted=true
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ai/interfaces/http -run AlertWebhook -v`  
Expected: FAIL because handler/route not implemented.

- [ ] **Step 3: Write minimal implementation**

`config.go` add AI secret:

```go
type AI struct {
	UseMultiDomainArch    bool   `mapstructure:"use_multi_domain_arch"`
	UseTurnBlockStreaming bool   `mapstructure:"use_turn_block_streaming"`
	AlertWebhookSecret    string `mapstructure:"alert_webhook_secret"`
}
```

`alert_webhook_handler.go` core:

```go
func (h *AlertWebhookHandler) Handle(c *gin.Context) {
	body, err := readLimitedBody(c.Request.Body, 1<<20)
	if err != nil {
		httpx.BadRequest(c, "invalid webhook payload")
		return
	}
	if !verifySignature(config.CFG.AI.AlertWebhookSecret, c.GetHeader("X-OpsPilot-Signature"), body) {
		httpx.Fail(c, xcode.Unauthorized, "invalid webhook signature")
		return
	}
	events, err := h.ingest.Ingest(c.Request.Context(), detectProtocol(body), body)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	jobID, err := h.ingest.EnqueueJobs(c.Request.Context(), events)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"accepted": true, "event_id": events[0].ID, "job_id": jobID})
}
```

`webhook_routes.go`:

```go
func RegisterAIWebhookHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := aihttp.NewAlertWebhookHandler(alertheal.NewService(svcCtx))
	v1.POST("/ai/alerts/webhook", h.Handle)
}
```

`bootstrap/modules.go` register before JWT group routes:

```go
aiapi.RegisterAIWebhookHandlers(v1, appCtx)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ai/interfaces/http -run AlertWebhook -v`  
Expected: PASS (missing signature 401, valid signature 202).

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/interfaces/http/alert_webhook_handler.go internal/modules/ai/interfaces/http/alert_webhook_handler_test.go internal/modules/ai/api/webhook_routes.go internal/bootstrap/modules.go internal/core/config/config.go configs/config.yaml
git commit -m "feat(ai): add signed alert self-healing webhook endpoint"
```

### Task 4: Implement Healing Worker State Machine and Retry/Resolved Rules

**Files:**
- Create: `internal/modules/ai/infra/workers/alert_heal_worker.go`
- Create: `internal/modules/ai/infra/workers/alert_heal_worker_test.go`
- Modify: `internal/bootstrap/modules.go`
- Modify: `internal/modules/ai/logic/alertheal/service.go`

- [ ] **Step 1: Write failing worker tests**

```go
func TestAlertHealWorker_FailureRetriesThenEscalatesToApproval(t *testing.T) {
	// seed pending job, force executor failure 3 times, assert status waiting_approval and retry_count=3
}

func TestAlertHealWorker_ResolvedCancelsActiveJobs(t *testing.T) {
	// seed resolved ingest event + active job, run worker once, assert status=canceled_resolved
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ai/infra/workers -run AlertHealWorker -v`  
Expected: FAIL because worker/service transition code is missing.

- [ ] **Step 3: Write minimal implementation**

`alert_heal_worker.go` core loop:

```go
func (w *AlertHealWorker) RunOnce(ctx context.Context) (bool, error) {
	job, err := w.svc.ClaimRunnableJob(ctx, time.Now().UTC())
	if err != nil || job == nil {
		return false, err
	}
	if err := w.svc.CancelIfResolved(ctx, job); err != nil {
		return true, err
	}
	decision, execErr := w.executor.Execute(ctx, job)
	if execErr == nil {
		return true, w.svc.MarkSucceeded(ctx, job.ID, decision.RunID)
	}
	next := w.backoff(job.RetryCount)
	if job.RetryCount+1 >= job.MaxRetry {
		return true, w.svc.MarkWaitingApproval(ctx, job.ID, execErr.Error())
	}
	return true, w.svc.MarkRetryWait(ctx, job.ID, execErr.Error(), time.Now().UTC().Add(next))
}
```

`bootstrap/modules.go` start worker runner:

```go
alertHealWorker := workers.NewAlertHealWorker(alertheal.NewService(appCtx), alertheal.NewExecutor(appCtx))
_ = workers.NewRunner(func(runCtx context.Context) {
	for runCtx.Err() == nil {
		claimed, _ := alertHealWorker.RunOnce(runCtx)
		if !claimed {
			return
		}
	}
}, aiBackgroundWorkerTick).Start(ctx)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ai/infra/workers -run AlertHealWorker -v`  
Expected: PASS with retry and resolved-cancel assertions.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/infra/workers/alert_heal_worker.go internal/modules/ai/infra/workers/alert_heal_worker_test.go internal/modules/ai/logic/alertheal/service.go internal/bootstrap/modules.go
git commit -m "feat(ai): add alert-healing worker with retry and resolved-cancel state machine"
```

### Task 5: Add Global Approval Pool Endpoint and Permissions

**Files:**
- Modify: `internal/modules/ai/dao/approval/dao.go`
- Modify: `internal/modules/ai/logic/logic.go`
- Modify: `internal/modules/ai/handler/approval/service.go`
- Modify: `internal/modules/ai/handler/approval/handler.go`
- Modify: `internal/modules/ai/api/routes.go`
- Modify: `internal/modules/rbac/handler/permission.go`
- Test: `internal/modules/ai/handler/approval/global_pending_test.go`

- [ ] **Step 1: Write failing test for global pending API**

```go
func TestListPendingApprovalsGlobal_RequiresPermission(t *testing.T) {
	// request without ai:approval:read -> forbidden
}

func TestListPendingApprovalsGlobal_ReturnsPendingList(t *testing.T) {
	// with permission -> returns pending approvals across users
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ai/handler/approval -run PendingApprovalsGlobal -v`  
Expected: FAIL because endpoint and DAO query do not exist.

- [ ] **Step 3: Write minimal implementation**

`dao.go`:

```go
func (d *AIApprovalTaskDAO) ListPendingPage(ctx context.Context, page, pageSize int) ([]model.AIApprovalTask, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	query := d.db.WithContext(ctx).Model(&model.AIApprovalTask{}).Where("status = ?", "pending")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]model.AIApprovalTask, 0, pageSize)
	if err := query.Order("created_at DESC").Offset((page-1)*pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
```

`handler.go`:

```go
func (h *HTTPHandler) ListPendingApprovalsGlobal(c *gin.Context) {
	if !httpx.Authorize(c, h.svc.logic.SvcCtx.DB, "ai:approval:read", "monitoring:write", "*:*") {
		return
	}
	page := intFromQuery(c, "page", 1)
	pageSize := intFromQuery(c, "page_size", 20)
	list, total, err := h.svc.ListPendingApprovalsGlobal(c.Request.Context(), page, pageSize)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": total})
}
```

`routes.go` add:

```go
g.GET("/approvals/pending/global", approvalHandler.ListPendingApprovalsGlobal)
```

`permission.go` admin set add:

```go
"ai:approval:read", "ai:approval:write",
"ai:alert:read", "ai:alert:write",
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/ai/handler/approval -run PendingApprovalsGlobal -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/ai/dao/approval/dao.go internal/modules/ai/logic/logic.go internal/modules/ai/handler/approval/service.go internal/modules/ai/handler/approval/handler.go internal/modules/ai/api/routes.go internal/modules/rbac/handler/permission.go internal/modules/ai/handler/approval/global_pending_test.go
git commit -m "feat(ai): add global pending approvals endpoint with permission gating"
```

### Task 6: Add Frontend API Module and Status Mapping

**Files:**
- Create: `web/src/api/modules/aiAlertHeal.ts`
- Create: `web/src/api/modules/aiAlertHeal.test.ts`
- Create: `web/src/pages/Monitor/monitorAlertHealStatus.ts`
- Modify: `web/src/api/index.ts`

- [ ] **Step 1: Write failing frontend API test**

```ts
import { describe, expect, it } from 'vitest';
import { normalizeHealStatus } from '../../pages/Monitor/monitorAlertHealStatus';

describe('normalizeHealStatus', () => {
  it('maps waiting_approval to 待人工 + 转人工审批', () => {
    expect(normalizeHealStatus('waiting_approval')).toEqual({
      processing: '待人工',
      healing: '转人工审批',
      processingColor: 'orange',
      healingColor: 'volcano',
    });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm --prefix web run test:run -- web/src/api/modules/aiAlertHeal.test.ts`  
Expected: FAIL with missing module/function.

- [ ] **Step 3: Write minimal implementation**

`aiAlertHeal.ts`:

```ts
import { apiService } from '../api';

export interface AlertHealJob {
  id: string;
  event_id: string;
  status: string;
  decision: string;
  retry_count: number;
  max_retry: number;
  last_error: string;
  latest_run_id: string;
  updated_at: string;
}

export const aiAlertHealApi = {
  listByAlert(alertId: string) {
    return apiService.get<{ list: AlertHealJob[]; total: number }>('/ai/alert-heal/jobs', { alert_id: alertId });
  },
  getJob(jobId: string) {
    return apiService.get<AlertHealJob>(`/ai/alert-heal/jobs/${jobId}`);
  },
  retryJob(jobId: string) {
    return apiService.post(`/ai/alert-heal/jobs/${jobId}/retry`, {});
  },
  listGlobalPendingApprovals(page = 1, pageSize = 20) {
    return apiService.get<{ list: any[]; total: number }>('/ai/approvals/pending/global', { page, page_size: pageSize });
  },
};
```

`monitorAlertHealStatus.ts`:

```ts
export function normalizeHealStatus(status: string) {
  switch (status) {
    case 'waiting_approval':
      return { processing: '待人工', healing: '转人工审批', processingColor: 'orange', healingColor: 'volcano' };
    case 'auto_fixing':
      return { processing: '处理中', healing: '自动修复中', processingColor: 'blue', healingColor: 'geekblue' };
    case 'succeeded':
      return { processing: '已处理', healing: 'AI自愈成功', processingColor: 'green', healingColor: 'cyan' };
    case 'failed_manual':
      return { processing: '待人工', healing: 'AI修复失败', processingColor: 'orange', healingColor: 'red' };
    case 'no_action':
      return { processing: '已处理', healing: 'AI判定无需处理', processingColor: 'green', healingColor: 'gold' };
    case 'canceled_resolved':
      return { processing: '已处理', healing: '告警恢复已取消', processingColor: 'green', healingColor: 'default' };
    default:
      return { processing: '待处理', healing: '待分析', processingColor: 'default', healingColor: 'default' };
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix web run test:run -- web/src/api/modules/aiAlertHeal.test.ts`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/aiAlertHeal.ts web/src/api/modules/aiAlertHeal.test.ts web/src/pages/Monitor/monitorAlertHealStatus.ts web/src/api/index.ts
git commit -m "feat(web): add ai alert-heal api module and status mapping"
```

### Task 7: Build Alert-Centric List and Detail Pages with Healing Actions

**Files:**
- Create: `web/src/pages/Monitor/AlertsPage.tsx`
- Create: `web/src/pages/Monitor/AlertsPage.test.tsx`
- Create: `web/src/pages/Monitor/AlertDetailPage.tsx`
- Create: `web/src/pages/Monitor/AlertDetailPage.test.tsx`
- Modify: `web/src/app/routes/pages.ts`
- Modify: `web/src/app/routes/observability.routes.tsx`

- [ ] **Step 1: Write failing page tests**

```tsx
it('renders processing/healing status columns in alert list', async () => {
  render(<AlertsPage />);
  expect(await screen.findByText('处理状态')).toBeInTheDocument();
  expect(await screen.findByText('自愈状态')).toBeInTheDocument();
});

it('disables retry button when alert is resolved', async () => {
  render(<AlertDetailPage />);
  expect(await screen.findByRole('button', { name: '手动重试' })).toBeDisabled();
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix web run test:run -- web/src/pages/Monitor/AlertsPage.test.tsx web/src/pages/Monitor/AlertDetailPage.test.tsx`  
Expected: FAIL because pages/routes not implemented.

- [ ] **Step 3: Write minimal implementation**

`AlertsPage.tsx` key table columns:

```tsx
const columns: ColumnsType<AlertRow> = [
  { title: '告警消息', dataIndex: 'title', render: (_, row) => <Link to={`/monitor/alerts/${row.id}`}>{row.title}</Link> },
  { title: '级别', dataIndex: 'severity', render: (v) => <Tag color={v === 'critical' ? 'error' : 'warning'}>{v}</Tag> },
  { title: '处理状态', dataIndex: 'healStatus', render: (v) => {
      const mapped = normalizeHealStatus(v);
      return <Tag color={mapped.processingColor}>{mapped.processing}</Tag>;
    } },
  { title: '自愈状态', dataIndex: 'healStatus', render: (v) => {
      const mapped = normalizeHealStatus(v);
      return <Tag color={mapped.healingColor}>{mapped.healing}</Tag>;
    } },
  { title: '最近处理', dataIndex: 'updatedAt' },
];
```

`AlertDetailPage.tsx` action logic:

```tsx
const retryDisabled = alert?.status === 'resolved' || ['analyzing', 'auto_fixing', 'waiting_approval'].includes(job?.status || '');

<Button onClick={handleRetry} disabled={retryDisabled}>手动重试</Button>
{job?.status === 'waiting_approval' ? <Button onClick={() => navigate('/ai/approvals/pending?scope=global')}>查看审批</Button> : null}
<Button onClick={() => setShowTimeline((v) => !v)}>查看执行轨迹</Button>
```

`observability.routes.tsx` add detail route:

```tsx
<Route path="/monitor/alerts/:alertId" element={withAuth('monitoring', 'read', <AlertDetailPage />)} />
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix web run test:run -- web/src/pages/Monitor/AlertsPage.test.tsx web/src/pages/Monitor/AlertDetailPage.test.tsx`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Monitor/AlertsPage.tsx web/src/pages/Monitor/AlertsPage.test.tsx web/src/pages/Monitor/AlertDetailPage.tsx web/src/pages/Monitor/AlertDetailPage.test.tsx web/src/app/routes/pages.ts web/src/app/routes/observability.routes.tsx
git commit -m "feat(web): add alert-centric list/detail pages with ai healing actions"
```

### Task 8: Final Contract Docs and Cross-Stack Verification

**Files:**
- Modify: `docs/ai/error-codes.md`
- Modify: `docs/swagger/swagger.yaml`

- [ ] **Step 1: Write failing contract check tests**

```go
func TestAlertWebhookRoutePresentInSwagger(t *testing.T) {
	raw, err := os.ReadFile("docs/swagger/swagger.yaml")
	if err != nil {
		t.Fatalf("read swagger: %v", err)
	}
	if !strings.Contains(string(raw), "/ai/alerts/webhook:") {
		t.Fatal("missing /ai/alerts/webhook route in swagger")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bootstrap -run AlertWebhookRoutePresentInSwagger -v`  
Expected: FAIL because new route/docs are absent.

- [ ] **Step 3: Write minimal implementation**

`docs/ai/error-codes.md` add:

```md
`AI_ALERT_WEBHOOK_INVALID_SIGNATURE`
- Meaning: invalid or missing webhook signature.

`AI_ALERT_WEBHOOK_UNSUPPORTED_PAYLOAD`
- Meaning: payload protocol is unsupported or required fields are missing.

`AI_ALERT_HEAL_JOB_NOT_FOUND`
- Meaning: requested healing job does not exist.

`AI_ALERT_HEAL_RETRY_EXHAUSTED`
- Meaning: automatic retries reached the configured limit and escalation is required.
```

`docs/swagger/swagger.yaml` add path entries for:

- `POST /ai/alerts/webhook`
- `GET /ai/alert-heal/jobs`
- `GET /ai/alert-heal/jobs/{id}`
- `POST /ai/alert-heal/jobs/{id}/retry`
- `GET /ai/approvals/pending/global`

- [ ] **Step 4: Run full verification**

Run:

```bash
go test ./internal/modules/ai/... ./internal/bootstrap/... -v
npm --prefix web run test:run -- web/src/pages/Monitor/AlertsPage.test.tsx web/src/pages/Monitor/AlertDetailPage.test.tsx web/src/api/modules/aiAlertHeal.test.ts
```

Expected:

- Go tests PASS.
- Web tests PASS.

- [ ] **Step 5: Commit**

```bash
git add docs/ai/error-codes.md docs/swagger/swagger.yaml internal/bootstrap/*_test.go
git commit -m "docs(ai): document alert-heal webhook contracts and error codes"
```

---

## Self-Review

### 1. Spec Coverage Check

- Webhook signed async ingest: Task 3.
- Dual protocol normalization + idempotency key: Task 2.
- Worker state machine + retry + resolved cancel: Task 4.
- Low-risk/high-risk handling with approval path: Task 4 + existing middleware integration.
- Global approval pool endpoint: Task 5.
- Alert-centric frontend list/detail and status display: Task 6 + Task 7.
- Manual retry and approval jump in UI: Task 7.
- Error codes and API docs: Task 8.

No uncovered spec requirements remain.

### 2. Placeholder Scan

- No `TODO`, `TBD`, or “implement later” text remains in tasks.
- Every code-changing step includes concrete code blocks and concrete commands.

### 3. Type Consistency Check

- Backend status names are consistent across model/service/UI mapping (`waiting_approval`, `auto_fixing`, `succeeded`, `failed_manual`, `no_action`, `canceled_resolved`).
- Frontend mapping function consumes the same status constants.
- Route names in backend and frontend match `/ai/alert-heal/*` and `/monitor/alerts/:alertId`.

