package assist

import (
	"context"
	"strings"
	"testing"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
)

func TestNormalizeFormAssistOutput(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "plain text",
			raw:  "  hello world  ",
			want: "hello world",
		},
		{
			name: "markdown fenced json",
			raw:  "```json\n{\"foo\": \"bar\"}\n```",
			want: "{\"foo\": \"bar\"}",
		},
		{
			name: "markdown fenced plain",
			raw:  "```\njust content\n```",
			want: "just content",
		},
		{
			name: "one-line lead-in with colon",
			raw:  "Here is the query: \nSELECT * FROM table",
			want: "SELECT * FROM table",
		},
		{
			name: "lead-in and markdown fence",
			raw:  "Here is the result:\n```sql\nSELECT * FROM table\n```",
			want: "SELECT * FROM table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeFormAssistOutput(tt.raw); got != tt.want {
				t.Errorf("NormalizeFormAssistOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeFormContext(t *testing.T) {
	input := map[string]any{
		"name":       "ops-pilot",
		"password":   "secret123",
		"secret":     "mysecret",
		"token":      "abc-def",
		"api_key":    "key-123",
		"access_key": "access-456",
		"port":       8080,
	}

	got := SanitizeFormContext(input)

	if _, ok := got["password"]; ok {
		t.Errorf("SanitizeFormContext() did not remove 'password'")
	}
	if _, ok := got["secret"]; ok {
		t.Errorf("SanitizeFormContext() did not remove 'secret'")
	}
	if _, ok := got["token"]; ok {
		t.Errorf("SanitizeFormContext() did not remove 'token'")
	}
	if _, ok := got["api_key"]; ok {
		t.Errorf("SanitizeFormContext() did not remove 'api_key'")
	}
	if _, ok := got["access_key"]; ok {
		t.Errorf("SanitizeFormContext() did not remove 'access_key'")
	}

	if got["name"] != "ops-pilot" {
		t.Errorf("SanitizeFormContext() corrupted 'name': got %v", got["name"])
	}
	if got["port"] != 8080 {
		t.Errorf("SanitizeFormContext() corrupted 'port': got %v", got["port"])
	}

	// Verify original is not mutated
	if input["password"] != "secret123" {
		t.Errorf("SanitizeFormContext() mutated original map")
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	meta := aiv1.FieldMeta{
		Label:   "Config JSON",
		Purpose: "Generate valid channel configuration JSON",
		Rules:   "Return valid JSON only.",
	}

	got := BuildSystemPrompt(meta)

	if !strings.Contains(got, "Professional Ops Assistant") {
		t.Errorf("BuildSystemPrompt() missing role: %q", got)
	}
	if !strings.Contains(got, "Field: Config JSON") {
		t.Errorf("BuildSystemPrompt() missing label: %q", got)
	}
	if !strings.Contains(got, "Purpose: Generate valid channel configuration JSON") {
		t.Errorf("BuildSystemPrompt() missing purpose: %q", got)
	}
	if !strings.Contains(got, "Rules: Return valid JSON only.") {
		t.Errorf("BuildSystemPrompt() missing rules: %q", got)
	}
	if !strings.Contains(got, "Constraint: Output ONLY the value") {
		t.Errorf("BuildSystemPrompt() missing constraint: %q", got)
	}
}

func TestNewService(t *testing.T) {
	l := &logic.Logic{}
	s := NewService(l)
	if s == nil {
		t.Fatal("NewService returned nil")
	}
	if s.logic != l {
		t.Fatal("NewService did not set logic correctly")
	}
}

func TestService_StreamAssist_Error(t *testing.T) {
	// This test will fail at model acquisition because we don't have a configured environment.
	// It verifies that StreamAssist handles errors from dependencies correctly.
	s := NewService(&logic.Logic{})
	ctx := context.Background()
	input := logic.FormAssistInput{
		UserPrompt: "help",
	}
	emit := func(event string, data any) {}

	err := s.StreamAssist(ctx, input, emit)
	if err == nil {
		t.Error("expected error from StreamAssist in unconfigured test environment, got nil")
	}
}
