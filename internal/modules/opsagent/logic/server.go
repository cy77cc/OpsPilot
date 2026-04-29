package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	pb "github.com/cy77cc/OpsPilot/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Server struct {
	pb.UnimplementedAgentServiceServer
	svcCtx   *svc.ServiceContext
	registry *SessionRegistry
}

func NewServer(svcCtx *svc.ServiceContext, registry *SessionRegistry) *Server {
	if registry == nil {
		registry = NewSessionRegistry()
	}
	return &Server{svcCtx: svcCtx, registry: registry}
}

func (s *Server) Connect(stream grpc.BidiStreamingServer[pb.AgentMessage, pb.PlatformMessage]) error {
	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	reg := msg.GetRegistration()
	if reg == nil {
		return status.Error(codes.InvalidArgument, "first message must be registration")
	}
	agentID := strings.TrimSpace(reg.GetAgentId())

	instance, err := s.bindRegistration(stream.Context(), reg)
	if err != nil {
		return err
	}

	s.registry.Put(instance.HostID, agentID, stream)
	defer s.registry.DeleteByAgentStream(agentID, stream)

	return s.consumeMessages(stream.Context(), instance, stream)
}

func (s *Server) bindRegistration(ctx context.Context, reg *pb.AgentRegistration) (*hostpluginmodel.HostPluginInstance, error) {
	db := s.db()
	if db == nil {
		return nil, status.Error(codes.FailedPrecondition, "opsagent service context requires db")
	}

	agentID := strings.TrimSpace(reg.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if strings.TrimSpace(reg.GetToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	var instance hostpluginmodel.HostPluginInstance
	err := db.WithContext(ctx).
		Table("host_plugin_instances AS hpi").
		Select("hpi.*").
		Joins("JOIN host_plugins hp ON hp.id = hpi.plugin_id").
		Where("hp.plugin_key = ?", "opsagent").
		Where("hpi.agent_id = ?", agentID).
		Where("hpi.install_status = ?", "succeeded").
		First(&instance).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "opsagent instance not found")
		}
		return nil, status.Errorf(codes.Internal, "load opsagent instance: %v", err)
	}

	now := time.Now()
	updates := map[string]any{
		"runtime_status": "online",
		"last_seen_at":   &now,
	}
	if capabilities, marshalErr := json.Marshal(reg.GetCapabilities()); marshalErr == nil && len(reg.GetCapabilities()) > 0 {
		updates["capabilities_json"] = string(capabilities)
	}

	if err := db.WithContext(ctx).
		Model(&hostpluginmodel.HostPluginInstance{}).
		Where("id = ?", instance.ID).
		Updates(updates).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "update opsagent instance: %v", err)
	}

	instance.RuntimeStatus = "online"
	instance.LastSeenAt = &now
	if encoded, ok := updates["capabilities_json"].(string); ok {
		instance.CapabilitiesJSON = encoded
	}
	return &instance, nil
}

func (s *Server) consumeMessages(_ context.Context, _ *hostpluginmodel.HostPluginInstance, stream grpc.BidiStreamingServer[pb.AgentMessage, pb.PlatformMessage]) error {
	for {
		_, err := stream.Recv()
		switch {
		case err == nil:
			continue
		case errors.Is(err, io.EOF):
			return nil
		default:
			return err
		}
	}
}

func StartGRPCServer(ctx context.Context, svcCtx *svc.ServiceContext) error {
	addr := fmt.Sprintf("%s:%d", config.CFG.OpsAgent.Host, config.CFG.OpsAgent.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	grpcServer := grpc.NewServer()
	pb.RegisterAgentServiceServer(grpcServer, NewServer(svcCtx, WrapSessionRegistry(svcCtx.OpsAgentRegistry)))

	go func() {
		<-ctx.Done()
		done := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			grpcServer.Stop()
		}
	}()

	logger.L().Info("opsagent grpc server started", logger.String("addr", addr))
	return grpcServer.Serve(listener)
}

func (s *Server) db() *gorm.DB {
	if s == nil || s.svcCtx == nil {
		return nil
	}
	return s.svcCtx.DB
}
