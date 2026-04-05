package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestExecuteHighRiskWorkloadOperation_MissingApprovalTokenCreatesApprovalAndAudit(t *testing.T) {
	handler, db := newWorkloadOperationTestHandler(t)
	ctx := newWorkloadOperationGinContext(1001)
	client := k8sfake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
	})

	resp, err := handler.executeHighRiskWorkloadOperationWithClient(
		ctx,
		42,
		workloadOperationTarget{
			Namespace: "default",
			Resource:  "deployment",
			Name:      "web",
			Action:    "deployment.restart",
		},
		"",
		client,
		func(context.Context, kubernetesWorkloadClient) (map[string]any, error) {
			t.Fatalf("workload operation should not execute before approval")
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("execute workload operation: %v", err)
	}
	if resp.State != OperationStateApprovalRequired {
		t.Fatalf("expected state %q, got %q", OperationStateApprovalRequired, resp.State)
	}
	if resp.Code != OperationCodeApprovalRequired {
		t.Fatalf("expected code %q, got %q", OperationCodeApprovalRequired, resp.Code)
	}
	if resp.Approval == nil || resp.Approval.Ticket == "" {
		t.Fatalf("expected approval ticket to be returned")
	}
	assertAuditRecord(t, db, resp.AuditID, OperationStateApprovalRequired, OperationCodeApprovalRequired)
}

func TestExecuteHighRiskWorkloadOperation_ApprovedDeploymentRestartCompletesAndAudits(t *testing.T) {
	handler, db := newWorkloadOperationTestHandler(t)
	ctx := newWorkloadOperationGinContext(1001)
	scope := ApprovalScope{
		ClusterID:  42,
		Namespace:  "default",
		Action:     "deployment.restart",
		Resource:   "deployment",
		ResourceID: "web",
	}
	token := issueApprovalTicket(t, db, scope, 1001, time.Now().UTC().Add(5*time.Minute), true)
	client := k8sfake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
	})

	resp, err := handler.executeHighRiskWorkloadOperationWithClient(
		ctx,
		42,
		workloadOperationTarget{
			Namespace: "default",
			Resource:  "deployment",
			Name:      "web",
			Action:    "deployment.restart",
		},
		token,
		client,
		func(ctx context.Context, kubeClient kubernetesWorkloadClient) (map[string]any, error) {
			return handler.restartDeployment(ctx, kubeClient, "default", "web")
		},
	)
	if err != nil {
		t.Fatalf("execute workload operation: %v", err)
	}
	if resp.State != OperationStateCompleted {
		t.Fatalf("expected state %q, got %q", OperationStateCompleted, resp.State)
	}
	if resp.Code != OperationCodeSuccess {
		t.Fatalf("expected code %q, got %q", OperationCodeSuccess, resp.Code)
	}
	assertAuditRecord(t, db, resp.AuditID, OperationStateCompleted, OperationCodeSuccess)

	deployment, err := client.AppsV1().Deployments("default").Get(context.Background(), "web", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("load deployment: %v", err)
	}
	if deployment.Spec.Template.Annotations[workloadRestartedAtAnnotation] == "" {
		t.Fatalf("expected restart annotation to be written")
	}
}

func TestWorkloadMutationHelpers_ScaleStatefulSet(t *testing.T) {
	handler, _ := newWorkloadOperationTestHandler(t)
	client := k8sfake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "default"},
		Spec:       appsv1.StatefulSetSpec{Replicas: int32Ptr(2)},
	})

	details, err := handler.scaleStatefulSet(context.Background(), client, "default", "db", 5)
	if err != nil {
		t.Fatalf("scale statefulset: %v", err)
	}
	if got := details["replicas"]; got != int32(5) {
		t.Fatalf("expected replicas 5, got %#v", got)
	}

	statefulset, err := client.AppsV1().StatefulSets("default").Get(context.Background(), "db", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("load statefulset: %v", err)
	}
	if statefulset.Spec.Replicas == nil || *statefulset.Spec.Replicas != 5 {
		t.Fatalf("expected replicas 5, got %#v", statefulset.Spec.Replicas)
	}
}

func TestWorkloadMutationHelpers_DeletePod(t *testing.T) {
	handler, _ := newWorkloadOperationTestHandler(t)
	client := k8sfake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-123", Namespace: "default"},
	})

	details, err := handler.deletePod(context.Background(), client, "default", "web-123")
	if err != nil {
		t.Fatalf("delete pod: %v", err)
	}
	if got := details["deleted"]; got != true {
		t.Fatalf("expected delete flag true, got %#v", got)
	}
	if _, err := client.CoreV1().Pods("default").Get(context.Background(), "web-123", metav1.GetOptions{}); err == nil {
		t.Fatalf("expected pod to be deleted")
	}
}

func newWorkloadOperationTestHandler(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	return newNodeOperationTestHandler(t)
}

func newWorkloadOperationGinContext(userID uint64) *gin.Context {
	return newNodeOperationGinContext(userID)
}

func int32Ptr(v int32) *int32 {
	return &v
}
