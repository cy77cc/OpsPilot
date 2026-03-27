package logic

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestMatchRiskPolicy_PrecedenceOrder(t *testing.T) {
	scene := "cluster"
	commandClass := "write"
	args := map[string]any{"namespace": "prod"}

	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl"},
		{ID: 2, ToolName: "kubectl", Scene: stringPtr("cluster")},
		{ID: 3, ToolName: "kubectl", CommandClass: stringPtr("write")},
		{ID: 4, ToolName: "kubectl", ArgumentRulesJSON: stringPtr(`{"namespace":"prod"}`)},
	}

	matched, ok := MatchRiskPolicy(rules, scene, commandClass, args)
	if !ok {
		t.Fatal("expected a matching policy")
	}
	if matched == nil || matched.ID != 4 {
		t.Fatalf("expected argument-aware rule to win, got %#v", matched)
	}
}

func TestMatchRiskPolicy_ArgumentAwareDoesNotMatchWhenArgsDiffer(t *testing.T) {
	scene := "cluster"
	commandClass := "write"

	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl"},
		{ID: 2, ToolName: "kubectl", Scene: stringPtr("cluster")},
		{ID: 3, ToolName: "kubectl", CommandClass: stringPtr("write")},
		{ID: 4, ToolName: "kubectl", ArgumentRulesJSON: stringPtr(`{"namespace":"prod"}`)},
	}

	matched, ok := MatchRiskPolicy(rules, scene, commandClass, map[string]any{"namespace": "dev"})
	if !ok {
		t.Fatal("expected a matching policy")
	}
	if matched == nil || matched.ID != 3 {
		t.Fatalf("expected command_class rule to win when args do not match, got %#v", matched)
	}
}

func TestMatchRiskPolicy_SceneOnlyBeatsToolOnly(t *testing.T) {
	scene := "cluster"
	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl", Priority: 1},
		{ID: 2, ToolName: "kubectl", Scene: stringPtr(scene), Priority: 1},
	}

	matched, ok := MatchRiskPolicy(rules, scene, "", nil)
	if !ok {
		t.Fatal("expected a matching policy")
	}
	if matched == nil || matched.ID != 2 {
		t.Fatalf("expected scene-only rule to win over tool-only rule, got %#v", matched)
	}
}

func TestMatchRiskPolicy_PriorityBreaksTieForSameSpecificity(t *testing.T) {
	scene := "cluster"
	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl", Scene: stringPtr(scene), Priority: 5},
		{ID: 2, ToolName: "kubectl", Scene: stringPtr(scene), Priority: 10},
	}

	matched, ok := MatchRiskPolicy(rules, scene, "", nil)
	if !ok {
		t.Fatal("expected a matching policy")
	}
	if matched == nil || matched.ID != 2 {
		t.Fatalf("expected higher priority scene rule to win, got %#v", matched)
	}
}

func TestMatchRiskPolicy_HigherPriorityBeatsMoreSpecificLowerPriority(t *testing.T) {
	scene := "cluster"
	commandClass := "write"
	args := map[string]any{"namespace": "prod"}

	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl", ArgumentRulesJSON: stringPtr(`{"namespace":"prod"}`), Priority: 5},
		{ID: 2, ToolName: "kubectl", CommandClass: stringPtr(commandClass), Priority: 10},
		{ID: 3, ToolName: "kubectl", Scene: stringPtr(scene), Priority: 8},
	}

	matched, ok := MatchRiskPolicy(rules, scene, commandClass, args)
	if !ok {
		t.Fatal("expected a matching policy")
	}
	if matched == nil || matched.ID != 2 {
		t.Fatalf("expected higher priority command_class rule to win over lower priority argument-aware rule, got %#v", matched)
	}
}

func TestMatchRiskPolicy_SamePrioritySceneAndCommandClassBeatsCommandClassOnly(t *testing.T) {
	scene := "cluster"
	commandClass := "write"
	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl", CommandClass: stringPtr(commandClass), Priority: 10},
		{ID: 2, ToolName: "kubectl", Scene: stringPtr(scene), CommandClass: stringPtr(commandClass), Priority: 10},
	}

	matched, ok := MatchRiskPolicy(rules, scene, commandClass, nil)
	if !ok {
		t.Fatal("expected a matching policy")
	}
	if matched == nil || matched.ID != 2 {
		t.Fatalf("expected scene+command_class rule to win over command_class-only rule, got %#v", matched)
	}
}

func TestMatchRiskPolicy_SamePriorityArgumentAndSceneBeatsArgumentOnly(t *testing.T) {
	scene := "cluster"
	commandClass := "write"
	args := map[string]any{"namespace": "prod"}
	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl", ArgumentRulesJSON: stringPtr(`{"namespace":"prod"}`), Priority: 10},
		{ID: 2, ToolName: "kubectl", Scene: stringPtr(scene), ArgumentRulesJSON: stringPtr(`{"namespace":"prod"}`), Priority: 10},
	}

	matched, ok := MatchRiskPolicy(rules, scene, commandClass, args)
	if !ok {
		t.Fatal("expected a matching policy")
	}
	if matched == nil || matched.ID != 2 {
		t.Fatalf("expected argument+scene rule to win over argument-only rule, got %#v", matched)
	}
}

func TestMatchRiskPolicy_RejectsCrossTypeExactMismatch(t *testing.T) {
	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl", ArgumentRulesJSON: stringPtr(`{"replicas":3,"enabled":true}`), Priority: 10},
	}

	matched, ok := MatchRiskPolicy(rules, "", "", map[string]any{
		"replicas": "3",
		"enabled":  true,
	})
	if ok {
		t.Fatalf("expected no match for cross-type mismatch, got %#v", matched)
	}

	matched, ok = MatchRiskPolicy(rules, "", "", map[string]any{
		"replicas": 3,
		"enabled":  "true",
	})
	if ok {
		t.Fatalf("expected no match for cross-type mismatch, got %#v", matched)
	}
}

func TestMatchRiskPolicy_AllowsExplicitNumericNormalization(t *testing.T) {
	rules := []model.AIToolRiskPolicy{
		{ID: 1, ToolName: "kubectl", ArgumentRulesJSON: stringPtr(`{"replicas":3}`), Priority: 10},
	}

	matched, ok := MatchRiskPolicy(rules, "", "", map[string]any{"replicas": 3.0})
	if !ok {
		t.Fatal("expected a matching policy")
	}
	if matched == nil || matched.ID != 1 {
		t.Fatalf("expected numeric-normalized match, got %#v", matched)
	}
}

func stringPtr(s string) *string { return &s }
