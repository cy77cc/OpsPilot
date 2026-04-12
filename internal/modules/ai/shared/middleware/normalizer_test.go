package middleware

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	einoschema "github.com/cloudwego/eino/schema"
)

func TestNormalizeToolArgs_HostExec_BareIPv4HostIDToTarget(t *testing.T) {
	t.Parallel()

	params := hostExecParams(t)
	raw := `{"host_id": 115.190.245.134, "command":"uptime"}`
	result, err := NormalizeToolArgs("host_exec", raw, params)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}

	normalized := decodeJSONObject(t, result.NormalizedJSON)
	if got := normalized["target"]; got != "115.190.245.134" {
		t.Fatalf("expected target from host_id IPv4, got %#v", got)
	}
	if _, ok := normalized["host_id"]; ok {
		t.Fatalf("expected host_id to be removed, got %#v", normalized["host_id"])
	}
	if got := normalized["command"]; got != "uptime" {
		t.Fatalf("expected command to remain unchanged, got %#v", got)
	}
	if !containsCoercion(result.Metadata.Coercions, "host_id:bare-ipv4->quoted-string") {
		t.Fatalf("expected bare-ipv4 coercion metadata, got %#v", result.Metadata.Coercions)
	}
	if !containsCoercion(result.Metadata.Coercions, "host_id:ipv4->target") {
		t.Fatalf("expected ipv4->target coercion metadata, got %#v", result.Metadata.Coercions)
	}
}

func TestNormalizeToolArgs_HostExec_RemovesIPv4HostIDWhenTargetProvided(t *testing.T) {
	t.Parallel()

	params := hostExecParams(t)
	raw := `{"target":"10.0.0.1","host_id":"115.190.245.134","command":"uptime"}`
	result, err := NormalizeToolArgs("host_exec", raw, params)
	if err != nil {
		t.Fatalf("normalize args: %v", err)
	}

	normalized := decodeJSONObject(t, result.NormalizedJSON)
	if got := normalized["target"]; got != "10.0.0.1" {
		t.Fatalf("expected explicit target to be preserved, got %#v", got)
	}
	if _, ok := normalized["host_id"]; ok {
		t.Fatalf("expected host_id to be removed when it is IPv4 string, got %#v", normalized["host_id"])
	}
}

func hostExecParams(t *testing.T) *einoschema.ParamsOneOf {
	t.Helper()

	type hostExecInputSchema struct {
		HostID  int    `json:"host_id"`
		Target  string `json:"target,omitempty"`
		Command string `json:"command,omitempty"`
		Script  string `json:"script,omitempty"`
	}
	testTool, err := einoutils.InferOptionableTool(
		"host_exec",
		"test schema",
		func(ctx context.Context, input *hostExecInputSchema, opts ...tool.Option) (string, error) {
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("build host_exec test tool: %v", err)
	}
	info, err := testTool.Info(context.Background())
	if err != nil {
		t.Fatalf("host_exec info: %v", err)
	}
	if info == nil || info.ParamsOneOf == nil {
		t.Fatal("host_exec params schema unavailable")
	}
	return info.ParamsOneOf
}

func decodeJSONObject(t *testing.T, raw string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode normalized json: %v\nraw=%s", err, raw)
	}
	return out
}

func containsCoercion(values []string, target string) bool {
	for _, item := range values {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}
