package intent

import (
	"context"
	"testing"
)

func TestRouter_ObviousQAPromptRoutesToQA(t *testing.T) {
	router := NewRouter()

	got, err := router.Route(context.Background(), "What does a Kubernetes namespace do?")
	if err != nil {
		t.Fatalf("route qa prompt: %v", err)
	}
	if got.IntentType != IntentTypeQA || got.AssistantType != AssistantTypeQA {
		t.Fatalf("expected QA routing, got %#v", got)
	}
}

func TestRouter_ObviousDiagnosisPromptRoutesToDiagnosis(t *testing.T) {
	router := NewRouter()

	got, err := router.Route(context.Background(), "Diagnose why the rollout is failing and find the root cause")
	if err != nil {
		t.Fatalf("route diagnosis prompt: %v", err)
	}
	if got.IntentType != IntentTypeDiagnosis || got.AssistantType != AssistantTypeDiagnosis {
		t.Fatalf("expected diagnosis routing, got %#v", got)
	}
}

func TestRouter_AmbiguousPromptFallsBackToQA(t *testing.T) {
	router := NewRouter()

	got, err := router.Route(context.Background(), "Can you help with this deployment?")
	if err != nil {
		t.Fatalf("route ambiguous prompt: %v", err)
	}
	if got.IntentType != IntentTypeQA || got.AssistantType != AssistantTypeQA {
		t.Fatalf("expected QA fallback, got %#v", got)
	}
}

func TestRouter_ResultIncludesRiskLevel(t *testing.T) {
	router := NewRouter()

	got, err := router.Route(context.Background(), "Please investigate the failing pods and diagnose the incident")
	if err != nil {
		t.Fatalf("route diagnosis prompt: %v", err)
	}
	if got.RiskLevel == "" {
		t.Fatalf("expected risk level to be set, got %#v", got)
	}
}
