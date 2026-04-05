package cluster

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestValidateServiceMutationReq_RejectsMissingPorts(t *testing.T) {
	err := validateServiceMutationReq(ServiceMutationReq{
		Name:     "web",
		Type:     string(corev1.ServiceTypeClusterIP),
		Selector: map[string]string{"app": "web"},
	})
	if err == nil || err.Error() != "ports required" {
		t.Fatalf("expected ports required validation error, got %v", err)
	}
}

func TestExecuteServiceIngressMutation_MissingApprovalTokenCreatesApprovalAndAuditForHighRiskDelete(t *testing.T) {
	handler, db := newWorkloadOperationTestHandler(t)
	ctx := newWorkloadOperationGinContext(1001)
	client := k8sfake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{"app": "edge"},
			Ports: []corev1.ServicePort{{
				Port:       80,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstrFromInt(8080),
			}},
		},
	})

	resp, err := handler.executeServiceIngressMutationWithClient(
		ctx,
		42,
		serviceIngressMutationTarget{
			Namespace:       "default",
			Resource:        "service",
			Name:            "edge",
			Action:          "service.delete",
			RequireApproval: true,
		},
		"",
		client,
		func(context.Context, kubernetesServiceIngressClient) (map[string]any, error) {
			t.Fatalf("mutation should not execute before approval")
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("execute service mutation: %v", err)
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

func TestExecuteServiceIngressMutation_LowRiskServiceCreateCompletesAndAudits(t *testing.T) {
	handler, db := newWorkloadOperationTestHandler(t)
	ctx := newWorkloadOperationGinContext(1001)
	client := k8sfake.NewSimpleClientset()
	req := ServiceMutationReq{
		Name:     "internal-api",
		Type:     string(corev1.ServiceTypeClusterIP),
		Selector: map[string]string{"app": "internal-api"},
		Ports: []ServiceMutationPort{{
			Name:       "http",
			Port:       80,
			TargetPort: "8080",
			Protocol:   string(corev1.ProtocolTCP),
		}},
	}

	resp, err := handler.executeServiceIngressMutationWithClient(
		ctx,
		42,
		serviceIngressMutationTarget{
			Namespace: "default",
			Resource:  "service",
			Name:      "internal-api",
			Action:    "service.create",
		},
		"",
		client,
		func(ctx context.Context, kubeClient kubernetesServiceIngressClient) (map[string]any, error) {
			return handler.upsertService(ctx, kubeClient, "default", req)
		},
	)
	if err != nil {
		t.Fatalf("execute service create: %v", err)
	}
	if resp.State != OperationStateCompleted {
		t.Fatalf("expected state %q, got %q", OperationStateCompleted, resp.State)
	}
	if resp.Code != OperationCodeSuccess {
		t.Fatalf("expected code %q, got %q", OperationCodeSuccess, resp.Code)
	}
	assertAuditRecord(t, db, resp.AuditID, OperationStateCompleted, OperationCodeSuccess)

	svc, err := client.CoreV1().Services("default").Get(context.Background(), "internal-api", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("load service: %v", err)
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Fatalf("expected ClusterIP service, got %s", svc.Spec.Type)
	}
}

func TestExecuteServiceIngressMutation_ApprovedIngressUpdateCompletesAndAudits(t *testing.T) {
	handler, db := newWorkloadOperationTestHandler(t)
	ctx := newWorkloadOperationGinContext(1001)
	scope := ApprovalScope{
		ClusterID:  42,
		Namespace:  "default",
		Action:     "ingress.update",
		Resource:   "ingress",
		ResourceID: "web",
	}
	token := issueApprovalTicket(t, db, scope, 1001, time.Now().UTC().Add(5*time.Minute), true)
	client := k8sfake.NewSimpleClientset(&networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "old.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: pathTypePtr(networkingv1.PathTypePrefix),
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "web",
									Port: networkingv1.ServiceBackendPort{Number: 80},
								},
							},
						}},
					},
				},
			}},
		},
	})
	req := IngressMutationReq{
		Name: "web",
		Rules: []IngressMutationRule{{
			Host: "app.example.com",
			Paths: []IngressMutationPath{{
				Path:        "/",
				PathType:    string(networkingv1.PathTypePrefix),
				ServiceName: "web",
				ServicePort: 8080,
			}},
		}},
	}

	resp, err := handler.executeServiceIngressMutationWithClient(
		ctx,
		42,
		serviceIngressMutationTarget{
			Namespace:       "default",
			Resource:        "ingress",
			Name:            "web",
			Action:          "ingress.update",
			RequireApproval: true,
		},
		token,
		client,
		func(ctx context.Context, kubeClient kubernetesServiceIngressClient) (map[string]any, error) {
			return handler.upsertIngress(ctx, kubeClient, "default", req)
		},
	)
	if err != nil {
		t.Fatalf("execute ingress update: %v", err)
	}
	if resp.State != OperationStateCompleted {
		t.Fatalf("expected state %q, got %q", OperationStateCompleted, resp.State)
	}
	if resp.Code != OperationCodeSuccess {
		t.Fatalf("expected code %q, got %q", OperationCodeSuccess, resp.Code)
	}
	assertAuditRecord(t, db, resp.AuditID, OperationStateCompleted, OperationCodeSuccess)

	ing, err := client.NetworkingV1().Ingresses("default").Get(context.Background(), "web", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("load ingress: %v", err)
	}
	if len(ing.Spec.Rules) != 1 || ing.Spec.Rules[0].Host != "app.example.com" {
		t.Fatalf("expected ingress host to be updated, got %+v", ing.Spec.Rules)
	}
}

func intstrFromInt(v int) intstr.IntOrString {
	return intstr.FromInt(v)
}

func pathTypePtr(v networkingv1.PathType) *networkingv1.PathType {
	return &v
}
