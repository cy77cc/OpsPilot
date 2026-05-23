package logic

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gateway "github.com/cy77cc/OpsPilot/internal/modules/gateway/logic"
	gatewaymodel "github.com/cy77cc/OpsPilot/internal/modules/gateway/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	pb "github.com/cy77cc/OpsPilot/proto"
	"google.golang.org/protobuf/proto"
)

type DispatchResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMS int64
	TimedOut   bool
	Truncated  bool
	Killed     bool
}

type Dispatcher struct {
	svcCtx   *svc.ServiceContext
	registry *SessionRegistry
}

type execWaiter struct {
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	resultCh chan *pb.ExecResult
}

var (
	execWaiters   sync.Map
	dispatchSeqID atomic.Uint64
)

func NewDispatcher(svcCtx *svc.ServiceContext) *Dispatcher {
	var registry *SessionRegistry
	if svcCtx != nil {
		registry = WrapSessionRegistry(svcCtx.OpsAgentRegistry)
	}
	return &Dispatcher{svcCtx: svcCtx, registry: registry}
}

func (d *Dispatcher) ExecuteCommand(ctx context.Context, instance *hostpluginmodel.HostPluginInstance, command string) (*DispatchResult, error) {
	// Check if this host routes through a gateway
	if d.svcCtx != nil && d.svcCtx.RouteTable != nil {
		if rt, ok := d.svcCtx.RouteTable.(*gateway.RouteTable); ok {
			route := rt.Get(instance.HostID)
			if route != nil && !route.Direct {
				return d.executeViaGateway(ctx, instance, command, route)
			}
		}
	}

	session, taskID, waiter, err := d.prepareDispatch(instance)
	if err != nil {
		return nil, err
	}
	defer execWaiters.Delete(taskID)

	msg := &pb.PlatformMessage{
		Payload: &pb.PlatformMessage_ExecCommand{
			ExecCommand: &pb.ExecuteCommand{
				TaskId:         taskID,
				Command:        "sh",
				Args:           []string{"-c", command},
				TimeoutSeconds: resolveTimeoutSeconds(ctx, 30),
			},
		},
	}
	if err := sessionSend(session, msg); err != nil {
		return nil, err
	}
	return waitForDispatchResult(ctx, waiter)
}

func (d *Dispatcher) ExecuteScript(ctx context.Context, instance *hostpluginmodel.HostPluginInstance, interpreter, script string) (*DispatchResult, error) {
	session, taskID, waiter, err := d.prepareDispatch(instance)
	if err != nil {
		return nil, err
	}
	defer execWaiters.Delete(taskID)

	msg := &pb.PlatformMessage{
		Payload: &pb.PlatformMessage_ExecScript{
			ExecScript: &pb.ExecuteScript{
				TaskId:         taskID,
				Interpreter:    interpreter,
				Script:         script,
				TimeoutSeconds: resolveTimeoutSeconds(ctx, 30),
			},
		},
	}
	if err := sessionSend(session, msg); err != nil {
		return nil, err
	}
	return waitForDispatchResult(ctx, waiter)
}

func (d *Dispatcher) executeViaGateway(ctx context.Context, instance *hostpluginmodel.HostPluginInstance, command string, route *gatewaymodel.HostRoute) (*DispatchResult, error) {
	if route.Mode == "tunnel" {
		// Tunnel mode: forward through tunnel manager
		if tm, ok := d.svcCtx.TunnelManager.(*gateway.TunnelManager); ok {
			stream, ok := tm.GetStream(route.TunnelID)
			if !ok {
				return nil, fmt.Errorf("tunnel %s stream not found", route.TunnelID)
			}

			taskID := fmt.Sprintf("opsagent-dispatch-%d", dispatchSeqID.Add(1))
			waiter := &execWaiter{resultCh: make(chan *pb.ExecResult, 1)}
			execWaiters.Store(taskID, waiter)
			defer execWaiters.Delete(taskID)

			innerMsg := &pb.PlatformMessage{
				Payload: &pb.PlatformMessage_ExecCommand{
					ExecCommand: &pb.ExecuteCommand{
						TaskId:         taskID,
						Command:        "sh",
						Args:           []string{"-c", command},
						TimeoutSeconds: resolveTimeoutSeconds(ctx, 30),
					},
				},
			}
			payload, _ := proto.Marshal(innerMsg)

			tunnelMsg := &pb.PlatformMessage{
				Payload: &pb.PlatformMessage_TunnelData{
					TunnelData: &pb.TunnelData{
						TunnelId: route.TunnelID,
						Payload:  payload,
					},
				},
			}
			if err := stream.Send(tunnelMsg); err != nil {
				return nil, fmt.Errorf("send via tunnel: %w", err)
			}
			return waitForDispatchResult(ctx, waiter)
		}
		return nil, fmt.Errorf("tunnel manager unavailable")
	}

	// Proxy mode: send ProxyCommandRequest via gateway session
	gatewaySession, ok := d.registry.GetByHostID(route.GatewayID)
	if !ok || gatewaySession == nil {
		return nil, fmt.Errorf("gateway session not found for host %d", route.GatewayID)
	}

	taskID := fmt.Sprintf("proxy-%d-%s", instance.HostID, command)
	waiter := &execWaiter{resultCh: make(chan *pb.ExecResult, 1)}
	execWaiters.Store(taskID, waiter)
	defer execWaiters.Delete(taskID)

	proxyMsg := &pb.PlatformMessage{
		Payload: &pb.PlatformMessage_ProxyCommand{
			ProxyCommand: &pb.ProxyCommandRequest{
				HostId:         fmt.Sprintf("%d", instance.HostID),
				Command:        command,
				TimeoutSeconds: resolveTimeoutSeconds(ctx, 30),
			},
		},
	}
	if err := sessionSend(gatewaySession, proxyMsg); err != nil {
		return nil, err
	}
	return waitForDispatchResult(ctx, waiter)
}

func (d *Dispatcher) prepareDispatch(instance *hostpluginmodel.HostPluginInstance) (*SessionHandle, string, *execWaiter, error) {
	if d == nil || d.registry == nil || instance == nil {
		return nil, "", nil, fmt.Errorf("opsagent dispatcher is unavailable")
	}
	session, ok := d.registry.GetByHostID(instance.HostID)
	if !ok || session == nil {
		return nil, "", nil, fmt.Errorf("opsagent session not found for host %d", instance.HostID)
	}
	taskID := fmt.Sprintf("opsagent-dispatch-%d", dispatchSeqID.Add(1))
	waiter := &execWaiter{resultCh: make(chan *pb.ExecResult, 1)}
	execWaiters.Store(taskID, waiter)
	return session, taskID, waiter, nil
}

func sessionSend(session *SessionHandle, msg *pb.PlatformMessage) error {
	if session == nil || session.Stream == nil {
		return fmt.Errorf("opsagent session stream unavailable")
	}
	stream, ok := session.Stream.(interface {
		Send(*pb.PlatformMessage) error
	})
	if !ok {
		return fmt.Errorf("opsagent session stream does not support platform messages")
	}
	return stream.Send(msg)
}

func waitForDispatchResult(ctx context.Context, waiter *execWaiter) (*DispatchResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-waiter.resultCh:
		if result == nil {
			return nil, fmt.Errorf("opsagent execution result missing")
		}
		return &DispatchResult{
			Stdout:     strings.TrimSpace(waiter.stdout.String()),
			Stderr:     strings.TrimSpace(waiter.stderr.String()),
			ExitCode:   int(result.GetExitCode()),
			DurationMS: result.GetDurationMs(),
			TimedOut:   result.GetTimedOut(),
			Truncated:  result.GetTruncated(),
			Killed:     result.GetKilled(),
		}, nil
	}
}

func handleExecOutput(output *pb.ExecOutput) {
	if output == nil {
		return
	}
	value, ok := execWaiters.Load(output.GetTaskId())
	if !ok {
		return
	}
	waiter := value.(*execWaiter)
	switch output.GetStream() {
	case "stderr":
		waiter.stderr.Write(output.GetData())
	default:
		waiter.stdout.Write(output.GetData())
	}
}

func handleExecResult(result *pb.ExecResult) {
	if result == nil {
		return
	}
	value, ok := execWaiters.Load(result.GetTaskId())
	if !ok {
		return
	}
	waiter := value.(*execWaiter)
	select {
	case waiter.resultCh <- result:
	default:
	}
}

func resolveTimeoutSeconds(ctx context.Context, fallback int32) int32 {
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			seconds := int32(remaining.Round(time.Second) / time.Second)
			if seconds > 0 {
				return seconds
			}
		}
	}
	return fallback
}
