package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestApprovalFlowEndToEnd(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	if err := suite.DB.Create(&model.Node{
		ID:      1,
		Name:    "node-1",
		IP:      "10.0.0.1",
		SSHUser: "root",
		Status:  "active",
	}).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	var resumeCalls []coreai.ResumeRequest
	handler := NewHTTPHandler(suite.SvcCtx)
	handler.orchestrator = &fakeRuntime{
		runFn: func(context.Context, coreai.RunRequest, coreai.StreamEmitter) error { return nil },
		resumeFn: func(_ context.Context, req coreai.ResumeRequest) (*coreai.ResumeResult, error) {
			resumeCalls = append(resumeCalls, req)
			return &coreai.ResumeResult{
				Resumed:   true,
				SessionID: req.SessionID,
				PlanID:    req.PlanID,
				StepID:    req.StepID,
				Status:    "completed",
				Message:   "审批已通过，待审批步骤会继续执行。",
			}, nil
		},
		resumeStreamFn: func(context.Context, coreai.ResumeRequest, coreai.StreamEmitter) (*coreai.ResumeResult, error) {
			return nil, nil
		},
	}

	createBody := `{"session_id":"sess-1","plan_id":"plan-1","step_id":"step-1","checkpoint_id":"cp-1","scene":"global","tool":"host_batch_exec_apply","summary":"重启 nginx","params":{"host_ids":[1],"command":"systemctl restart nginx"}}`
	createResp := performJSONRequest(t, handler.CreateApproval, "POST", "/ai/approvals", createBody)

	var created struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create approval response: %v", err)
	}
	approvalID, _ := created.Data["id"].(string)
	if approvalID == "" {
		t.Fatalf("approval response missing id: %#v", created.Data)
	}
	if created.Data["checkpoint_id"] != "cp-1" {
		t.Fatalf("checkpoint_id = %#v, want cp-1", created.Data["checkpoint_id"])
	}

	getResp := performJSONRequest(t, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: approvalID}}
		handler.GetApproval(c)
	}, "GET", "/ai/approvals/"+approvalID, "")
	var got struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(getResp.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal get approval response: %v", err)
	}
	if got.Data["summary"] != "重启 nginx" {
		t.Fatalf("summary = %#v, want 重启 nginx", got.Data["summary"])
	}

	listResp := performJSONRequest(t, handler.ListApprovals, "GET", "/ai/approvals?status=pending", "")
	var listed struct {
		Data struct {
			List []map[string]any `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("unmarshal list approvals response: %v", err)
	}
	if len(listed.Data.List) != 1 || listed.Data.List[0]["id"] != approvalID {
		t.Fatalf("listed approvals = %#v", listed.Data.List)
	}

	approveResp := performJSONRequest(t, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: approvalID}}
		handler.ApproveApproval(c)
	}, "POST", "/ai/approvals/"+approvalID+"/approve", `{"reason":"看起来没问题"}`)
	var approved struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(approveResp.Body.Bytes(), &approved); err != nil {
		t.Fatalf("unmarshal approve response: %v", err)
	}
	approvalPayload, _ := approved.Data["approval"].(map[string]any)
	executionPayload, _ := approved.Data["execution"].(map[string]any)
	if approvalPayload["status"] != "approved" {
		t.Fatalf("approval payload = %#v", approvalPayload)
	}
	if executionPayload["status"] != "success" {
		t.Fatalf("execution payload = %#v", executionPayload)
	}
	if len(resumeCalls) != 1 {
		t.Fatalf("resume calls = %#v, want 1", resumeCalls)
	}
	if resumeCalls[0].CheckpointID != "cp-1" || !resumeCalls[0].Approved {
		t.Fatalf("resume request = %#v", resumeCalls[0])
	}

	var approvalRow model.AIApproval
	if err := suite.DB.First(&approvalRow, "id = ?", approvalID).Error; err != nil {
		t.Fatalf("load approval row: %v", err)
	}
	if approvalRow.Status != "approved" || approvalRow.ExecutionID == "" {
		t.Fatalf("approval row = %#v", approvalRow)
	}

	var executionRow model.AIExecution
	if err := suite.DB.First(&executionRow, "id = ?", approvalRow.ExecutionID).Error; err != nil {
		t.Fatalf("load execution row: %v", err)
	}
	if executionRow.ApprovalID != approvalID || executionRow.CheckpointID != "cp-1" || executionRow.Status != "success" {
		t.Fatalf("execution row = %#v", executionRow)
	}
}

func TestRejectApprovalResumesRuntimeWithCheckpointIdentity(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	approval := model.AIApproval{
		ID:            "approval-reject",
		SessionID:     "sess-2",
		PlanID:        "plan-2",
		StepID:        "step-2",
		CheckpointID:  "cp-2",
		ApprovalKey:   "sess-2:plan-2:step-2",
		RequestUserID: 1,
		ToolName:      "host_batch_exec_apply",
		ToolMode:      "mutating",
		RiskLevel:     "high",
		Status:        "pending",
		Scene:         "global",
		Summary:       "拒绝测试",
	}
	if err := suite.DB.Create(&approval).Error; err != nil {
		t.Fatalf("seed approval: %v", err)
	}

	var resumeCalls []coreai.ResumeRequest
	handler := NewHTTPHandler(suite.SvcCtx)
	handler.orchestrator = &fakeRuntime{
		runFn: func(context.Context, coreai.RunRequest, coreai.StreamEmitter) error { return nil },
		resumeFn: func(_ context.Context, req coreai.ResumeRequest) (*coreai.ResumeResult, error) {
			resumeCalls = append(resumeCalls, req)
			return &coreai.ResumeResult{Resumed: true, Status: "rejected"}, nil
		},
		resumeStreamFn: func(context.Context, coreai.ResumeRequest, coreai.StreamEmitter) (*coreai.ResumeResult, error) {
			return nil, nil
		},
	}

	resp := performJSONRequest(t, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: approval.ID}}
		handler.RejectApproval(c)
	}, "POST", "/ai/approvals/"+approval.ID+"/reject", `{"reason":"风险过高"}`)

	var payload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal reject response: %v", err)
	}
	if payload.Data["status"] != "rejected" || payload.Data["reason"] != "风险过高" {
		t.Fatalf("reject payload = %#v", payload.Data)
	}
	if len(resumeCalls) != 1 || resumeCalls[0].CheckpointID != "cp-2" || resumeCalls[0].Approved {
		t.Fatalf("resume calls = %#v", resumeCalls)
	}

	var row model.AIApproval
	if err := suite.DB.First(&row, "id = ?", approval.ID).Error; err != nil {
		t.Fatalf("load rejected approval: %v", err)
	}
	if row.Status != "rejected" || row.Reason != "风险过高" {
		t.Fatalf("approval row = %#v", row)
	}
}

func performJSONRequest(t *testing.T, handlerFunc gin.HandlerFunc, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if body == "" {
		c.Request = httptest.NewRequest(method, target, nil)
	} else {
		c.Request = httptest.NewRequest(method, target, bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Set("uid", uint64(1))

	handlerFunc(c)

	if w.Code != 200 {
		t.Fatalf("%s %s status = %d, body=%s", method, target, w.Code, w.Body.String())
	}
	return w
}
