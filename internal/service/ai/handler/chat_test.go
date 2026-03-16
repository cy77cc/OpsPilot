package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/gin-gonic/gin"
)

type stubIntentRouter struct {
	decision Decision
}

func (s stubIntentRouter) Route(_ context.Context, _ string) (Decision, error) {
	return s.decision, nil
}

type stubQAAgent struct {
	result QAResult
}

func (s stubQAAgent) Answer(_ context.Context, _ QARequest) (QAResult, error) {
	return s.result, nil
}

type stubDiagnosisAgent struct {
	result DiagnosisResult
}

func (s stubDiagnosisAgent) Diagnose(_ context.Context, _ DiagnosisRequest) (DiagnosisResult, error) {
	return s.result, nil
}

func TestChatHandler_QAFlowCreatesSessionRunAndAssistantMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := New(Dependencies{
		ChatDAO:            aidao.NewAIChatDAO(db),
		RunDAO:             aidao.NewAIRunDAO(db),
		DiagnosisReportDAO: aidao.NewAIDiagnosisReportDAO(db),
		IntentRouter:       stubIntentRouter{decision: Decision{IntentType: IntentTypeQA, AssistantType: "qa", RiskLevel: "low"}},
		QAAgent:            stubQAAgent{result: QAResult{Text: "Namespaces isolate resources."}},
		DiagnosisAgent:     stubDiagnosisAgent{},
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(101))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"What does a namespace do?"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", contentType)
	}

	events := decodeSSEEvents(t, recorder.Body.String())
	expectedOrder := []string{"init", "intent", "status", "delta", "done"}
	if len(events) != len(expectedOrder) {
		t.Fatalf("expected %d events, got %d: %#v", len(expectedOrder), len(events), events)
	}
	for i, event := range expectedOrder {
		if events[i].Event != event {
			t.Fatalf("expected event %d to be %q, got %#v", i, event, events[i])
		}
	}

	sessions, err := h.deps.ChatDAO.ListSessions(context.Background(), 101)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected one session, got %d", len(sessions))
	}
	messages, err := h.deps.ChatDAO.ListMessagesBySession(context.Background(), sessions[0].ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected two messages, got %d", len(messages))
	}
	runData := events[0].Data.(map[string]any)
	runID, _ := runData["run_id"].(string)
	run, err := h.deps.RunDAO.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run == nil || run.Status != "completed" {
		t.Fatalf("expected completed run, got %#v", run)
	}
}

func TestChatHandler_DiagnosisFlowCreatesReportAndStreamsProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := New(Dependencies{
		ChatDAO:            aidao.NewAIChatDAO(db),
		RunDAO:             aidao.NewAIRunDAO(db),
		DiagnosisReportDAO: aidao.NewAIDiagnosisReportDAO(db),
		IntentRouter:       stubIntentRouter{decision: Decision{IntentType: IntentTypeDiagnosis, AssistantType: "diagnosis", RiskLevel: "medium"}},
		QAAgent:            stubQAAgent{},
		DiagnosisAgent: stubDiagnosisAgent{result: DiagnosisResult{
			Progress: []string{"Checking rollout", "Inspecting events"},
			Report: DiagnosisReport{
				Summary:         "Rollout blocked by quota exhaustion",
				Evidence:        []string{"events show quota exceeded"},
				RootCauses:      []string{"namespace quota exhausted"},
				Recommendations: []string{"increase quota"},
			},
		}},
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(202))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"Diagnose why the rollout is failing"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	events := decodeSSEEvents(t, recorder.Body.String())
	expectedOrder := []string{"init", "intent", "status", "progress", "progress", "report_ready", "done"}
	if len(events) != len(expectedOrder) {
		t.Fatalf("expected %d events, got %d: %#v", len(expectedOrder), len(events), events)
	}
	for i, event := range expectedOrder {
		if events[i].Event != event {
			t.Fatalf("expected event %d to be %q, got %#v", i, event, events[i])
		}
	}

	initData := events[0].Data.(map[string]any)
	runID, _ := initData["run_id"].(string)
	report, err := h.deps.DiagnosisReportDAO.GetReportByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("get report by run id: %v", err)
	}
	if report == nil || report.Summary == "" {
		t.Fatalf("expected diagnosis report, got %#v", report)
	}
}

func decodeSSEEvents(t *testing.T, body string) []chatEvent {
	t.Helper()

	lines := strings.Split(body, "\n")
	events := make([]chatEvent, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event chatEvent
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			t.Fatalf("decode SSE event %q: %v", raw, err)
		}
		events = append(events, event)
	}
	return events
}

type chatEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}
