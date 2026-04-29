package logic

import (
	"context"
	"testing"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	pb "github.com/cy77cc/OpsPilot/proto"
)

func TestDispatcherExecuteCommand_UsesSessionAndResultChannels(t *testing.T) {
	registry := NewSessionRegistry()
	stream := newFakeAgentServiceConnectServer(t)
	registry.Put(101, "agent-host-1", stream)
	dispatcher := &Dispatcher{svcCtx: &svc.ServiceContext{}, registry: registry}
	instance := &hostpluginmodel.HostPluginInstance{HostID: 101}

	done := make(chan struct{})
	go func() {
		msg := <-stream.sent
		cmd := msg.GetExecCommand()
		if cmd == nil {
			t.Errorf("expected exec command payload")
			close(done)
			return
		}
		handleExecOutput(&pb.ExecOutput{TaskId: cmd.GetTaskId(), Stream: "stdout", Data: []byte("ok")})
		handleExecResult(&pb.ExecResult{TaskId: cmd.GetTaskId(), ExitCode: 0, DurationMs: 12})
		close(done)
	}()

	result, err := dispatcher.ExecuteCommand(context.Background(), instance, "uptime")
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}
	<-done
	if result.Stdout != "ok" {
		t.Fatalf("expected stdout ok, got %q", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
}
