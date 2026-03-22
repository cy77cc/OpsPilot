package host

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

func TestNewHostReadonlyTools_ContainsHostExecReadonlyOnly(t *testing.T) {
	names := toolNames(t, NewHostReadonlyTools(context.Background()))
	if !containsTool(names, "host_exec_readonly") {
		t.Fatalf("expected host_exec_readonly in readonly tools, got %v", names)
	}
	if containsTool(names, "host_exec_change") {
		t.Fatalf("did not expect host_exec_change in readonly tools, got %v", names)
	}
}

func TestHostExecReadonly_InterruptsWhenValidationFails(t *testing.T) {
	ctx := runtimectx.WithServices(context.Background(), nil)
	hostExec := HostExecReadonly(ctx)

	out, err := hostExec.InvokableRun(ctx, `{"target":"localhost","command":"systemctl status nginx"}`)
	if err != nil {
		t.Fatalf("expected suspended payload, got error: %v", err)
	}
	if !strings.Contains(out, `"status":"suspended"`) {
		t.Fatalf("expected suspended status, got %s", out)
	}
	if !strings.Contains(out, `"approval_required":true`) {
		t.Fatalf("expected approval_required=true, got %s", out)
	}
}

func TestHostExecChange_AlwaysRequestsApprovalBeforeExecution(t *testing.T) {
	ctx := runtimectx.WithServices(context.Background(), nil)
	hostExec := HostExecChange(ctx)

	out, err := hostExec.InvokableRun(ctx, `{"target":"localhost","command":"cat /etc/hosts"}`)
	if err != nil {
		t.Fatalf("expected suspended payload, got error: %v", err)
	}
	if !strings.Contains(out, `"status":"suspended"`) {
		t.Fatalf("expected suspended status, got %s", out)
	}
	if !strings.Contains(out, `"approval_required":true`) {
		t.Fatalf("expected approval_required=true, got %s", out)
	}
}

func TestLegacyHostExec_UsesPolicyEngine(t *testing.T) {
	ctx := runtimectx.WithServices(context.Background(), nil)
	hostExec := HostExec(ctx)

	out, err := hostExec.InvokableRun(ctx, `{"host_id":1,"command":"systemctl status nginx"}`)
	if err != nil {
		t.Fatalf("expected suspended payload, got error: %v", err)
	}
	if !strings.Contains(out, `"status":"suspended"`) {
		t.Fatalf("expected suspended status, got %s", out)
	}
	if !strings.Contains(out, `"policy_decision":"require_approval_interrupt"`) {
		t.Fatalf("expected policy decision in legacy output, got %s", out)
	}
}

func TestLegacyHostExecByTarget_LocalhostCannotBypassPolicy(t *testing.T) {
	ctx := runtimectx.WithServices(context.Background(), nil)
	hostExec := HostExecByTarget(ctx)

	out, err := hostExec.InvokableRun(ctx, `{"target":"localhost","command":"systemctl status nginx"}`)
	if err != nil {
		t.Fatalf("expected suspended payload, got error: %v", err)
	}
	if !strings.Contains(out, `"status":"suspended"`) {
		t.Fatalf("expected suspended status, got %s", out)
	}
	if !strings.Contains(out, `"policy_decision":"require_approval_interrupt"`) {
		t.Fatalf("expected policy decision in legacy output, got %s", out)
	}
}

func toolNames(t *testing.T, tools []tool.InvokableTool) []string {
	t.Helper()
	result := make([]string, 0, len(tools))
	for _, item := range tools {
		info, err := item.Info(t.Context())
		if err != nil {
			t.Fatalf("get tool info: %v", err)
		}
		result = append(result, info.Name)
	}
	return result
}

func containsTool(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
