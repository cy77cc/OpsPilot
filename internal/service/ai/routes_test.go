package ai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/testutil"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRegisterAIHandlersRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterAIHandlers(v1, &svc.ServiceContext{})

	type route struct {
		name   string
		method string
		path   string
		body   string
	}

	routes := []route{
		{name: "Chat", method: http.MethodPost, path: "/api/v1/ai/chat", body: `{"message":"hello"}`},
		{name: "ListSessions", method: http.MethodGet, path: "/api/v1/ai/sessions"},
		{name: "CreateSession", method: http.MethodPost, path: "/api/v1/ai/sessions", body: `{"scene":"cluster"}`},
		{name: "GetSession", method: http.MethodGet, path: "/api/v1/ai/sessions/session-123"},
		{name: "DeleteSession", method: http.MethodDelete, path: "/api/v1/ai/sessions/session-123"},
		{name: "GetRun", method: http.MethodGet, path: "/api/v1/ai/runs/run-987"},
		{name: "GetDiagnosis", method: http.MethodGet, path: "/api/v1/ai/diagnosis/report-123"},
	}

	for _, routeCase := range routes {
		routeCase := routeCase
		t.Run(routeCase.name, func(t *testing.T) {
			var bodyReader io.Reader
			if routeCase.body != "" {
				bodyReader = strings.NewReader(routeCase.body)
			}

			req := httptest.NewRequest(routeCase.method, routeCase.path, bodyReader)
			if routeCase.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusNotFound {
				t.Fatalf("%s returned 404", routeCase.name)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s returned %d instead of 401", routeCase.name, rec.Code)
			}
		})
	}
}

type apiEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func withTestJWTConfig(t *testing.T) {
	t.Helper()
	original := config.CFG.JWT
	config.CFG.JWT = config.JWT{
		Secret:        "test-secret",
		Expire:        time.Hour,
		RefreshExpire: 2 * time.Hour,
		Issuer:        "test-issuer",
	}
	utils.MySecret = []byte(config.CFG.JWT.Secret)
	t.Cleanup(func() {
		config.CFG.JWT = original
		utils.MySecret = []byte(config.CFG.JWT.Secret)
	})
}

func makeAuthedRequest(t *testing.T, method, path, body string, userID uint) *http.Request {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	token, err := utils.GenToken(userID, false)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) apiEnvelope {
	t.Helper()
	var resp apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestAIHandlers_SessionLifecycleAndReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withTestJWTConfig(t)

	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterAIHandlers(v1, suite.SvcCtx)

	userID := uint(42)
	now := time.Now().UTC()
	session := &model.AIChatSession{
		ID:        uuid.NewString(),
		UserID:    uint64(userID),
		Scene:     "cluster",
		Title:     "Existing Session",
		CreatedAt: now.Add(-time.Hour),
		UpdatedAt: now,
	}
	testutil.RequireNoError(t, suite.DB.Create(session).Error)
	testutil.RequireNoError(t, suite.DB.Create(&model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: session.ID,
		Role:      "user",
		Content:   "why is this pod unhealthy?",
	}).Error)
	testutil.RequireNoError(t, suite.DB.Create(&model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: session.ID,
		Role:      "assistant",
		Content:   "investigating",
	}).Error)
	report := &model.AIDiagnosisReport{
		ID:                  uuid.NewString(),
		RunID:               uuid.NewString(),
		SessionID:           session.ID,
		Summary:             "summary",
		ImpactScope:         "impact",
		SuspectedRootCauses: "root cause",
		Evidence:            "evidence",
		Recommendations:     "recommend",
		RawToolRefs:         "raw",
		Status:              "ready",
	}
	testutil.RequireNoError(t, suite.DB.Create(report).Error)

	t.Run("create session", func(t *testing.T) {
		req := makeAuthedRequest(t, http.MethodPost, "/api/v1/ai/sessions", `{"scene":"service","title":"Fresh Session"}`, userID)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		testutil.AssertEqual(t, http.StatusOK, rec.Code)
		resp := decodeEnvelope(t, rec)
		testutil.AssertEqual(t, 1000, resp.Code)

		var payload CreateSessionResponse
		testutil.RequireNoError(t, json.Unmarshal(resp.Data, &payload))
		testutil.AssertEqual(t, "service", payload.Scene)
		testutil.AssertEqual(t, "Fresh Session", payload.Title)
		testutil.AssertTrue(t, payload.ID != "", "session id should be generated")
		testutil.AssertDBRecordExists(t, suite.DB, &model.AIChatSession{}, "id = ?", payload.ID)
	})

	t.Run("list sessions", func(t *testing.T) {
		req := makeAuthedRequest(t, http.MethodGet, "/api/v1/ai/sessions?scene=cluster", "", userID)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		testutil.AssertEqual(t, http.StatusOK, rec.Code)
		resp := decodeEnvelope(t, rec)
		testutil.AssertEqual(t, 1000, resp.Code)

		var payload []SessionResponse
		testutil.RequireNoError(t, json.Unmarshal(resp.Data, &payload))
		testutil.AssertLen[SessionResponse](t, 1, payload)
		testutil.AssertEqual(t, session.ID, payload[0].ID)
		testutil.AssertLen[ChatMessageResponse](t, 2, payload[0].Messages)
		testutil.AssertEqual(t, "investigating", payload[0].Messages[1].Content)
	})

	t.Run("get session with messages", func(t *testing.T) {
		req := makeAuthedRequest(t, http.MethodGet, "/api/v1/ai/sessions/"+session.ID, "", userID)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		testutil.AssertEqual(t, http.StatusOK, rec.Code)
		resp := decodeEnvelope(t, rec)
		testutil.AssertEqual(t, 1000, resp.Code)

		var payload SessionResponse
		testutil.RequireNoError(t, json.Unmarshal(resp.Data, &payload))
		testutil.AssertEqual(t, session.ID, payload.ID)
		testutil.AssertLen[ChatMessageResponse](t, 2, payload.Messages)
	})

	t.Run("delete session", func(t *testing.T) {
		deleteSession := &model.AIChatSession{
			ID:     uuid.NewString(),
			UserID: uint64(userID),
			Scene:  "cluster",
			Title:  "Delete Me",
		}
		testutil.RequireNoError(t, suite.DB.Create(deleteSession).Error)

		req := makeAuthedRequest(t, http.MethodDelete, "/api/v1/ai/sessions/"+deleteSession.ID, "", userID)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		testutil.AssertEqual(t, http.StatusOK, rec.Code)
		resp := decodeEnvelope(t, rec)
		testutil.AssertEqual(t, 1000, resp.Code)
		testutil.AssertTrue(t, len(resp.Data) == 0 || strings.TrimSpace(string(resp.Data)) == "null", "delete response should not carry a payload")
		testutil.AssertDBRecordNotExists(t, suite.DB, &model.AIChatSession{}, "id = ?", deleteSession.ID)
	})

	t.Run("get diagnosis report", func(t *testing.T) {
		req := makeAuthedRequest(t, http.MethodGet, "/api/v1/ai/diagnosis/"+report.ID, "", userID)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		testutil.AssertEqual(t, http.StatusOK, rec.Code)
		resp := decodeEnvelope(t, rec)
		testutil.AssertEqual(t, 1000, resp.Code)

		var payload DiagnosisReportResponse
		testutil.RequireNoError(t, json.Unmarshal(resp.Data, &payload))
		testutil.AssertEqual(t, report.ID, payload.ID)
		testutil.AssertEqual(t, report.SessionID, payload.SessionID)
		testutil.AssertEqual(t, report.Status, payload.Status)
	})

	t.Run("get session not found for another user", func(t *testing.T) {
		req := makeAuthedRequest(t, http.MethodGet, "/api/v1/ai/sessions/"+session.ID, "", 7)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		testutil.AssertEqual(t, http.StatusOK, rec.Code)
		resp := decodeEnvelope(t, rec)
		testutil.AssertEqual(t, 2005, resp.Code)
	})

	t.Run("delete session not found for another user", func(t *testing.T) {
		protectedSession := &model.AIChatSession{
			ID:     uuid.NewString(),
			UserID: uint64(userID),
			Scene:  "cluster",
			Title:  "Protected",
		}
		testutil.RequireNoError(t, suite.DB.Create(protectedSession).Error)

		req := makeAuthedRequest(t, http.MethodDelete, "/api/v1/ai/sessions/"+protectedSession.ID, "", 7)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		resp := decodeEnvelope(t, rec)
		testutil.AssertEqual(t, 2005, resp.Code)
		testutil.AssertDBRecordExists(t, suite.DB, &model.AIChatSession{}, "id = ?", protectedSession.ID)
	})

	t.Run("get diagnosis report not found for another user", func(t *testing.T) {
		req := makeAuthedRequest(t, http.MethodGet, "/api/v1/ai/diagnosis/"+report.ID, "", 7)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		resp := decodeEnvelope(t, rec)
		testutil.AssertEqual(t, 2005, resp.Code)
	})

	t.Run("missing session and report return not found", func(t *testing.T) {
		req := makeAuthedRequest(t, http.MethodGet, "/api/v1/ai/sessions/"+uuid.NewString(), "", userID)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		testutil.AssertEqual(t, 2005, decodeEnvelope(t, rec).Code)

		req = makeAuthedRequest(t, http.MethodDelete, "/api/v1/ai/sessions/"+uuid.NewString(), "", userID)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		testutil.AssertEqual(t, 2005, decodeEnvelope(t, rec).Code)

		req = makeAuthedRequest(t, http.MethodGet, "/api/v1/ai/diagnosis/"+uuid.NewString(), "", userID)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		testutil.AssertEqual(t, 2005, decodeEnvelope(t, rec).Code)
	})
}
