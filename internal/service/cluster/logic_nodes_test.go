package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

func TestExecuteHighRiskNodeOperation_FailedStateAndAuditForNodeActions(t *testing.T) {
	handler, db := newNodeOperationTestHandler(t)
	operatorID := seedRBACPermission(t, db, "node-approver", "cluster-approver", "k8s:approve")
	ctx := newNodeOperationGinContext(operatorID)

	cases := []struct {
		name   string
		action string
	}{
		{name: "Cordon", action: "node.cordon"},
		{name: "Uncordon", action: "node.uncordon"},
		{name: "Drain", action: "node.drain"},
		{name: "LabelUpdate", action: "node.labels"},
		{name: "LabelRemove", action: "node.labels.remove"},
		{name: "TaintUpdate", action: "node.taints"},
		{name: "TaintRemove", action: "node.taints.remove"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := handler.executeHighRiskNodeOperation(ctx, 42, "node-1", tc.action, "", func(context.Context, *model.Cluster, *model.ClusterNode, *kubernetes.Clientset) (map[string]any, error) {
				return nil, errors.New("token=secret-value denied")
			})
			if err != nil {
				t.Fatalf("execute node operation: %v", err)
			}
			if resp.State != OperationStateFailed {
				t.Fatalf("expected state %q, got %q", OperationStateFailed, resp.State)
			}
			if resp.Code != OperationCodeFailed {
				t.Fatalf("expected code %q, got %q", OperationCodeFailed, resp.Code)
			}
			if resp.AuditID == 0 {
				t.Fatalf("expected audit id to be recorded")
			}
			if strings.Contains(resp.Message, "secret-value") {
				t.Fatalf("expected response message to be sanitized, got %q", resp.Message)
			}
			if strings.Contains(resp.Diagnostics, "secret-value") {
				t.Fatalf("expected diagnostics to be sanitized, got %q", resp.Diagnostics)
			}

			assertAuditRecord(t, db, resp.AuditID, OperationStateFailed, governance.CodeInternalError)

			var audit model.OperationAudit
			if err := db.First(&audit, resp.AuditID).Error; err != nil {
				t.Fatalf("load audit: %v", err)
			}
			if strings.Contains(audit.Message, "secret-value") {
				t.Fatalf("expected audit message to be sanitized, got %q", audit.Message)
			}
		})
	}
}

func TestRemoveClusterNode_LastControlPlaneReturnsFailedOperationResponse(t *testing.T) {
	handler, db := newNodeOperationTestHandler(t)
	adminID := seedRBACPermission(t, db, "node-admin", "admin", "")

	node := model.ClusterNode{
		ClusterID: 42,
		Name:      "cp-1",
		IP:        "10.0.0.10",
		Role:      "control-plane",
		Status:    "ready",
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create control-plane node: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/clusters/42/nodes/cp-1", nil)
	ctx.Params = gin.Params{
		{Key: "id", Value: "42"},
		{Key: "name", Value: "cp-1"},
	}
	ctx.Set("uid", adminID)

	handler.RemoveClusterNode(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected http status %d, got %d", http.StatusOK, recorder.Code)
	}

	var body struct {
		Data ClusterOperationResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.State != OperationStateFailed {
		t.Fatalf("expected state %q, got %q", OperationStateFailed, body.Data.State)
	}
	if body.Data.Code != OperationCodeFailed {
		t.Fatalf("expected code %q, got %q", OperationCodeFailed, body.Data.Code)
	}
	if body.Data.AuditID == 0 {
		t.Fatalf("expected audit id for failed remove")
	}

	assertAuditRecord(t, db, body.Data.AuditID, OperationStateFailed, governance.CodeInternalError)
}

func newNodeOperationTestHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()

	handler, db := newHighRiskApprovalTestHandler(t)
	if err := db.AutoMigrate(&model.Cluster{}, &model.ClusterNode{}, &model.ClusterCredential{}); err != nil {
		t.Fatalf("migrate cluster node schema: %v", err)
	}

	previousKey := config.CFG.Security.EncryptionKey
	config.CFG.Security.EncryptionKey = "node-operation-test-key"
	t.Cleanup(func() {
		config.CFG.Security.EncryptionKey = previousKey
	})

	tokenEnc, err := utils.EncryptText("test-token", config.CFG.Security.EncryptionKey)
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}

	cluster := model.Cluster{
		ID:         42,
		Name:       "cluster-42",
		Status:     "active",
		Type:       "kubernetes",
		Source:     "platform_managed",
		EnvType:    "production",
		AuthMethod: "token",
		Endpoint:   "https://127.0.0.1",
	}
	if err := db.Create(&cluster).Error; err != nil {
		t.Fatalf("create cluster: %v", err)
	}

	node := model.ClusterNode{
		ClusterID: 42,
		Name:      "node-1",
		IP:        "10.0.0.1",
		Role:      "worker",
		Status:    "ready",
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create cluster node: %v", err)
	}

	cred := model.ClusterCredential{
		Name:        "cluster-42-cred",
		RuntimeType: "k8s",
		Source:      "platform_managed",
		ClusterID:   42,
		Endpoint:    "https://127.0.0.1",
		AuthMethod:  "token",
		TokenEnc:    tokenEnc,
		Status:      "active",
	}
	if err := db.Create(&cred).Error; err != nil {
		t.Fatalf("create credential: %v", err)
	}

	handler.repo = NewRepository(db)
	return handler, db
}

func newNodeOperationGinContext(userID uint64) *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/clusters/42/nodes/node-1", nil)
	ctx.Set("uid", userID)
	return ctx
}
