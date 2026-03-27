package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubmitApprovalRouteContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	registerAIHandlersForTest(v1)

	routes := r.Routes()
	seen := make(map[string]bool, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = true
	}

	if !seen[http.MethodPost+" /api/v1/ai/approvals/:id/submit"] {
		t.Fatalf("expected submit route to be registered")
	}
	if seen[http.MethodPost+" /api/v1/ai/approvals/:id/confirm"] {
		t.Fatalf("legacy confirm route must not be registered")
	}
	if seen[http.MethodPost+" /api/v1/ai/chains/:chainId/approvals/:nodeId/decision"] {
		t.Fatalf("legacy decision route must not be registered")
	}
	if seen[http.MethodPost+" /api/v1/ai/approvals/:id/resume"] {
		t.Fatalf("legacy resume route must not be registered")
	}
}

func TestSubmitApproval_ErrorTaxonomy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("not found maps to not found response", func(t *testing.T) {
		db := newApprovalSubmitTestDB(t)
		router := newApprovalSubmitTestRouter(db, 42)

		recorder := performApprovalSubmit(t, router, approvalSubmitTestRequest{
			approvalID: "missing-approval",
			body: map[string]any{
				"approved": true,
			},
		})

		resp := decodeApprovalSubmitResponse(t, recorder)
		if resp.Code != uint32(xcode.NotFound) {
			t.Fatalf("expected not found code %d, got %d body=%s", xcode.NotFound, resp.Code, recorder.Body.String())
		}
	})

	t.Run("non owner maps to forbidden response", func(t *testing.T) {
		db := newApprovalSubmitTestDB(t)
		seedApprovalSubmitTask(t, db, approvalSubmitTaskSeed{
			approvalID: "approval-owned",
			runID:      "run-owned",
			sessionID:  "session-owned",
			userID:     7,
			status:     "pending",
		})
		router := newApprovalSubmitTestRouter(db, 42)

		recorder := performApprovalSubmit(t, router, approvalSubmitTestRequest{
			approvalID: "approval-owned",
			body: map[string]any{
				"approved": true,
			},
		})

		resp := decodeApprovalSubmitResponse(t, recorder)
		if resp.Code != uint32(xcode.Forbidden) {
			t.Fatalf("expected forbidden code %d, got %d body=%s", xcode.Forbidden, resp.Code, recorder.Body.String())
		}
	})

	t.Run("already handled maps to conflict style business response", func(t *testing.T) {
		db := newApprovalSubmitTestDB(t)
		now := time.Now().UTC().Truncate(time.Second)
		seedApprovalSubmitTask(t, db, approvalSubmitTaskSeed{
			approvalID: "approval-decided",
			runID:      "run-decided",
			sessionID:  "session-decided",
			userID:     42,
			status:     "approved",
			approvedBy: 42,
			decidedAt:  &now,
		})
		router := newApprovalSubmitTestRouter(db, 42)

		recorder := performApprovalSubmit(t, router, approvalSubmitTestRequest{
			approvalID: "approval-decided",
			body: map[string]any{
				"approved": true,
			},
		})

		resp := decodeApprovalSubmitResponse(t, recorder)
		if resp.Code != uint32(xcode.ParamError) {
			t.Fatalf("expected conflict business code %d, got %d body=%s", xcode.ParamError, resp.Code, recorder.Body.String())
		}
	})

	t.Run("same idempotency key returns first result without re-running transition", func(t *testing.T) {
		db := newApprovalSubmitTestDB(t)
		seedApprovalSubmitTask(t, db, approvalSubmitTaskSeed{
			approvalID: "approval-idempotent",
			runID:      "run-idempotent",
			sessionID:  "session-idempotent",
			userID:     42,
			status:     "pending",
		})
		router := newApprovalSubmitTestRouter(db, 42)

		first := performApprovalSubmit(t, router, approvalSubmitTestRequest{
			approvalID: "approval-idempotent",
			headers: map[string]string{
				"Idempotency-Key": "approval-submit-key-1",
			},
			body: map[string]any{
				"approved": true,
				"comment":  "ship it",
			},
		})
		firstResp := decodeApprovalSubmitResponse(t, first)
		if firstResp.Code != uint32(xcode.Success) {
			t.Fatalf("expected first submit success, got %d body=%s", firstResp.Code, first.Body.String())
		}

		second := performApprovalSubmit(t, router, approvalSubmitTestRequest{
			approvalID: "approval-idempotent",
			headers: map[string]string{
				"Idempotency-Key": "approval-submit-key-1",
			},
			body: map[string]any{
				"approved": true,
				"comment":  "ship it",
			},
		})
		secondResp := decodeApprovalSubmitResponse(t, second)
		if secondResp.Code != uint32(xcode.Success) {
			t.Fatalf("expected retry success, got %d body=%s", secondResp.Code, second.Body.String())
		}
		if !jsonEqual(firstResp.Data, secondResp.Data) {
			t.Fatalf("expected same idempotent response payload, first=%s second=%s", string(firstResp.Data), string(secondResp.Data))
		}

		task, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), "approval-idempotent")
		if err != nil {
			t.Fatalf("reload approval task: %v", err)
		}
		if task == nil || task.Comment != "ship it" {
			t.Fatalf("expected original decision snapshot to remain, got %#v", task)
		}

		var outboxCount int64
		if err := db.Model(&model.AIApprovalOutboxEvent{}).
			Where("approval_id = ? AND event_type = ?", "approval-idempotent", "ai.approval.decided").
			Count(&outboxCount).Error; err != nil {
			t.Fatalf("count decision outbox rows: %v", err)
		}
		if outboxCount != 1 {
			t.Fatalf("expected one approval_decided outbox row, got %d", outboxCount)
		}
	})
}

func TestRetryResumeApproval_ErrorTaxonomy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("retry resume route contract is registered", func(t *testing.T) {
		r := gin.New()
		v1 := r.Group("/api/v1")
		registerAIHandlersForTest(v1)

		routes := r.Routes()
		seen := make(map[string]bool, len(routes))
		for _, route := range routes {
			seen[route.Method+" "+route.Path] = true
		}
		if !seen[http.MethodPost+" /api/v1/ai/approvals/:id/retry-resume"] {
			t.Fatalf("expected retry-resume route to be registered")
		}
	})

	t.Run("retryable run requeues successfully", func(t *testing.T) {
		db := newApprovalSubmitTestDB(t)
		seedApprovalSubmitTask(t, db, approvalSubmitTaskSeed{
			approvalID: "approval-retryable",
			runID:      "run-retryable",
			sessionID:  "session-retryable",
			userID:     42,
			status:     "approved",
			approvedBy: 42,
			runStatus:  "resume_failed_retryable",
		})
		seedApprovalDecisionOutbox(t, db, "approval-retryable", "run-retryable", "session-retryable", "approved")
		router := newApprovalSubmitTestRouter(db, 42)

		recorder := performRetryResumeApproval(t, router, retryResumeApprovalTestRequest{
			approvalID: "approval-retryable",
			body: map[string]any{
				"trigger_id": "retry-trigger-1",
			},
		})

		resp := decodeApprovalSubmitResponse(t, recorder)
		if resp.Code != uint32(xcode.Success) {
			t.Fatalf("expected retry-resume success, got %d body=%s", resp.Code, recorder.Body.String())
		}

		var outbox model.AIApprovalOutboxEvent
		if err := db.Where("approval_id = ? AND event_type = ?", "approval-retryable", "ai.approval.decided").First(&outbox).Error; err != nil {
			t.Fatalf("load decision outbox after retry: %v", err)
		}
		if outbox.Status != "pending" {
			t.Fatalf("expected retry-resume to requeue approval_decided outbox, got %q", outbox.Status)
		}
	})

	t.Run("same trigger id returns original outcome without requeue", func(t *testing.T) {
		db := newApprovalSubmitTestDB(t)
		seedApprovalSubmitTask(t, db, approvalSubmitTaskSeed{
			approvalID: "approval-idempotent-retry",
			runID:      "run-idempotent-retry",
			sessionID:  "session-idempotent-retry",
			userID:     42,
			status:     "approved",
			approvedBy: 42,
			runStatus:  "resume_failed_retryable",
		})
		seedApprovalDecisionOutbox(t, db, "approval-idempotent-retry", "run-idempotent-retry", "session-idempotent-retry", "approved")
		router := newApprovalSubmitTestRouter(db, 42)

		first := performRetryResumeApproval(t, router, retryResumeApprovalTestRequest{
			approvalID: "approval-idempotent-retry",
			body: map[string]any{
				"trigger_id": "retry-trigger-same",
			},
		})
		firstResp := decodeApprovalSubmitResponse(t, first)
		if firstResp.Code != uint32(xcode.Success) {
			t.Fatalf("expected first retry-resume success, got %d body=%s", firstResp.Code, first.Body.String())
		}

		if err := db.Model(&model.AIApprovalOutboxEvent{}).
			Where("approval_id = ? AND event_type = ?", "approval-idempotent-retry", "ai.approval.decided").
			Update("status", "done").Error; err != nil {
			t.Fatalf("mark requeued outbox done: %v", err)
		}

		second := performRetryResumeApproval(t, router, retryResumeApprovalTestRequest{
			approvalID: "approval-idempotent-retry",
			body: map[string]any{
				"trigger_id": "retry-trigger-same",
			},
		})
		secondResp := decodeApprovalSubmitResponse(t, second)
		if secondResp.Code != uint32(xcode.Success) {
			t.Fatalf("expected repeated trigger success, got %d body=%s", secondResp.Code, second.Body.String())
		}
		if !jsonEqual(firstResp.Data, secondResp.Data) {
			t.Fatalf("expected repeated trigger to return original payload, first=%s second=%s", string(firstResp.Data), string(secondResp.Data))
		}
	})

	t.Run("new trigger against resuming run conflicts", func(t *testing.T) {
		db := newApprovalSubmitTestDB(t)
		seedApprovalSubmitTask(t, db, approvalSubmitTaskSeed{
			approvalID: "approval-resuming",
			runID:      "run-resuming",
			sessionID:  "session-resuming",
			userID:     42,
			status:     "approved",
			approvedBy: 42,
			runStatus:  "resuming",
		})
		seedApprovalDecisionOutbox(t, db, "approval-resuming", "run-resuming", "session-resuming", "approved")
		router := newApprovalSubmitTestRouter(db, 42)

		recorder := performRetryResumeApproval(t, router, retryResumeApprovalTestRequest{
			approvalID: "approval-resuming",
			body: map[string]any{
				"trigger_id": "retry-trigger-new",
			},
		})

		resp := decodeApprovalSubmitResponse(t, recorder)
		if resp.Code != uint32(xcode.ParamError) {
			t.Fatalf("expected resuming retry-resume conflict, got %d body=%s", resp.Code, recorder.Body.String())
		}
	})

	t.Run("unauthorized caller is forbidden", func(t *testing.T) {
		db := newApprovalSubmitTestDB(t)
		seedApprovalSubmitTask(t, db, approvalSubmitTaskSeed{
			approvalID: "approval-unauthorized",
			runID:      "run-unauthorized",
			sessionID:  "session-unauthorized",
			userID:     7,
			status:     "approved",
			approvedBy: 7,
			runStatus:  "resume_failed_retryable",
		})
		seedApprovalDecisionOutbox(t, db, "approval-unauthorized", "run-unauthorized", "session-unauthorized", "approved")
		router := newApprovalSubmitTestRouter(db, 42)

		recorder := performRetryResumeApproval(t, router, retryResumeApprovalTestRequest{
			approvalID: "approval-unauthorized",
			body: map[string]any{
				"trigger_id": "retry-trigger-unauthorized",
			},
		})

		resp := decodeApprovalSubmitResponse(t, recorder)
		if resp.Code != uint32(xcode.Forbidden) {
			t.Fatalf("expected forbidden retry-resume response, got %d body=%s", resp.Code, recorder.Body.String())
		}
	})
}

type approvalSubmitTestRequest struct {
	approvalID string
	headers    map[string]string
	body       map[string]any
}

type retryResumeApprovalTestRequest struct {
	approvalID string
	headers    map[string]string
	body       map[string]any
}

type approvalSubmitEnvelope struct {
	Code uint32          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type approvalSubmitTaskSeed struct {
	approvalID string
	runID      string
	sessionID  string
	userID     uint64
	status     string
	approvedBy uint64
	decidedAt  *time.Time
	runStatus  string
}

func newApprovalSubmitTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AIChatSession{},
		&model.AIChatMessage{},
		&model.AIRun{},
		&model.AIRunEvent{},
		&model.AIRunProjection{},
		&model.AIRunContent{},
		&model.AIApprovalTask{},
		&model.AIApprovalOutboxEvent{},
	); err != nil {
		t.Fatalf("auto migrate approval submit tables: %v", err)
	}
	return db
}

func newApprovalSubmitTestRouter(db *gorm.DB, userID uint64) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("uid", userID)
		c.Next()
	})
	h := newAIHandlerTestHarness(db)
	router.POST("/ai/approvals/:id/submit", h.SubmitApproval)
	router.POST("/ai/approvals/:id/retry-resume", h.RetryResumeApproval)
	return router
}

func seedApprovalSubmitTask(t *testing.T, db *gorm.DB, seed approvalSubmitTaskSeed) {
	t.Helper()

	if err := db.Create(&model.AIChatSession{
		ID:     seed.sessionID,
		UserID: seed.userID,
		Scene:  "ai",
		Title:  "approval submit test",
	}).Error; err != nil {
		t.Fatalf("seed chat session: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           seed.runID + "-user",
		SessionID:    seed.sessionID,
		SessionIDNum: 1,
		Role:         "user",
		Content:      "please continue",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           seed.runID + "-assistant",
		SessionID:    seed.sessionID,
		SessionIDNum: 2,
		Role:         "assistant",
		Content:      "",
		Status:       "in_progress",
	}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := db.Create(&model.AIRun{
		ID:                 seed.runID,
		SessionID:          seed.sessionID,
		ClientRequestID:    seed.runID,
		UserMessageID:      seed.runID + "-user",
		AssistantMessageID: seed.runID + "-assistant",
		Status:             defaultString(seed.runStatus, "waiting_approval"),
		TraceJSON:          `{}`,
	}).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}

	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if err := db.Create(&model.AIApprovalTask{
		ApprovalID:     seed.approvalID,
		CheckpointID:   seed.approvalID + "-checkpoint",
		SessionID:      seed.sessionID,
		RunID:          seed.runID,
		UserID:         seed.userID,
		ToolName:       "exec_command",
		ToolCallID:     seed.approvalID + "-tool-call",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         seed.status,
		ApprovedBy:     seed.approvedBy,
		TimeoutSeconds: 300,
		ExpiresAt:      &expiresAt,
		DecidedAt:      seed.decidedAt,
	}).Error; err != nil {
		t.Fatalf("seed approval task: %v", err)
	}
}

func seedApprovalDecisionOutbox(t *testing.T, db *gorm.DB, approvalID, runID, sessionID, status string) {
	t.Helper()

	payload := map[string]any{
		"approval_id": approvalID,
		"status":      status,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal decision outbox payload: %v", err)
	}
	if err := db.Create(&model.AIApprovalOutboxEvent{
		ApprovalID:  approvalID,
		RunID:       runID,
		SessionID:   sessionID,
		ToolCallID:  approvalID + "-tool-call",
		EventType:   "ai.approval.decided",
		PayloadJSON: string(payloadJSON),
		Status:      "done",
	}).Error; err != nil {
		t.Fatalf("seed approval decision outbox: %v", err)
	}
}

func performApprovalSubmit(t *testing.T, router *gin.Engine, req approvalSubmitTestRequest) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(req.body)
	if err != nil {
		t.Fatalf("marshal submit request: %v", err)
	}

	httpReq := httptest.NewRequest(http.MethodPost, "/ai/approvals/"+req.approvalID+"/submit", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.headers {
		httpReq.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpReq)
	return recorder
}

func performRetryResumeApproval(t *testing.T, router *gin.Engine, req retryResumeApprovalTestRequest) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(req.body)
	if err != nil {
		t.Fatalf("marshal retry-resume request: %v", err)
	}

	httpReq := httptest.NewRequest(http.MethodPost, "/ai/approvals/"+req.approvalID+"/retry-resume", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.headers {
		httpReq.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpReq)
	return recorder
}

func decodeApprovalSubmitResponse(t *testing.T, recorder *httptest.ResponseRecorder) approvalSubmitEnvelope {
	t.Helper()

	var resp approvalSubmitEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode submit response: %v body=%s", err, recorder.Body.String())
	}
	return resp
}

func jsonEqual(left, right []byte) bool {
	var leftValue any
	var rightValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
