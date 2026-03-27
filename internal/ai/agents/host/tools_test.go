package host

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

func TestHostExec_RejectsWhenCommandAndScriptBothProvided(t *testing.T) {
	ctx := runtimectx.WithServices(context.Background(), nil)
	hostExec := HostExec(ctx)

	_, err := hostExec.InvokableRun(ctx, `{"host_id":1,"command":"date","script":"whoami"}`)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "provide exactly one of command or script") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestHostExec_RejectsWhenCommandAndScriptBothEmpty(t *testing.T) {
	ctx := runtimectx.WithServices(context.Background(), nil)
	hostExec := HostExec(ctx)

	_, err := hostExec.InvokableRun(ctx, `{"host_id":1}`)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "provide exactly one of command or script") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestHostExec_RejectsWhenHostIDInvalid(t *testing.T) {
	ctx := runtimectx.WithServices(context.Background(), nil)
	hostExec := HostExec(ctx)

	_, err := hostExec.InvokableRun(ctx, `{"host_id":0,"command":"date"}`)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "host_id is required") {
		t.Fatalf("expected host_id validation error, got %v", err)
	}
}

func TestNewHostTools_DoesNotExposeLegacyExecToolNames(t *testing.T) {
	names := toolNames(t, NewHostTools(context.Background()))
	if !containsTool(names, "host_exec") {
		t.Fatalf("expected host_exec in tools, got %v", names)
	}
	for _, legacy := range []string{"host_exec_readonly", "host_exec_change", "host_exec_by_target", "host_ssh_exec_readonly"} {
		if containsTool(names, legacy) {
			t.Fatalf("did not expect %s in tools, got %v", legacy, names)
		}
	}
}

func TestLegacyHostExec_UsesPolicyEngine(t *testing.T) {
	ctx := runtimectx.WithServices(context.Background(), nil)
	hostExec := HostExec(ctx)

	_, err := hostExec.InvokableRun(ctx, `{"host_id":1,"command":"systemctl status nginx"}`)
	if err == nil {
		t.Fatal("expected approval error contract")
	}
	if !strings.Contains(err.Error(), "decision=require_approval_interrupt") {
		t.Fatalf("expected policy decision in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "violations=[") {
		t.Fatalf("expected policy violations in error, got %v", err)
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
