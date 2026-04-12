package ai

import "testing"

func TestAILLMProvider_TableName(t *testing.T) {
	if got := (AILLMProvider{}).TableName(); got != "ai_llm_providers" {
		t.Fatalf("expected table name ai_llm_providers, got %q", got)
	}
}
