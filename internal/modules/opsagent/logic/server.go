package logic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	pb "github.com/cy77cc/OpsPilot/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
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

	if revision, revErr := s.lookupPendingConfigRevision(stream.Context(), instance.ID); revErr == nil && revision != nil {
		if sendErr := s.sendConfigUpdate(stream.Context(), s.registry.MustGetByAgent(agentID), *revision); sendErr != nil {
			return sendErr
		}
	}

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
		msg, err := stream.Recv()
		switch {
		case err == nil:
			if handleErr := s.handleAgentMessage(stream.Context(), stream, msg); handleErr != nil {
				return handleErr
			}
		case errors.Is(err, io.EOF):
			return nil
		default:
			return err
		}
	}
}

func (s *Server) handleAgentMessage(ctx context.Context, stream grpc.BidiStreamingServer[pb.AgentMessage, pb.PlatformMessage], msg *pb.AgentMessage) error {
	if msg == nil {
		return nil
	}
	switch payload := msg.GetPayload().(type) {
	case *pb.AgentMessage_Heartbeat:
		return s.handleHeartbeat(ctx, payload.Heartbeat)
	case *pb.AgentMessage_Metrics:
		return s.handleMetrics(ctx, stream, payload.Metrics)
	case *pb.AgentMessage_ExecOutput:
		handleExecOutput(payload.ExecOutput)
		return nil
	case *pb.AgentMessage_ExecResult:
		handleExecResult(payload.ExecResult)
		return nil
	case *pb.AgentMessage_Ack:
		return s.handleAck(ctx, payload.Ack)
	default:
		return nil
	}
}

func (s *Server) handleHeartbeat(ctx context.Context, hb *pb.Heartbeat) error {
	if hb == nil {
		return nil
	}
	db := s.db()
	if db == nil {
		return status.Error(codes.FailedPrecondition, "opsagent service context requires db")
	}
	agentID := strings.TrimSpace(hb.GetAgentId())
	if agentID == "" {
		return status.Error(codes.InvalidArgument, "heartbeat agent_id is required")
	}
	now := time.Now()
	if hb.GetTimestampMs() > 0 {
		now = time.UnixMilli(hb.GetTimestampMs())
	}
	updates := map[string]any{
		"runtime_status": "online",
		"health_status":  "healthy",
		"last_seen_at":   &now,
	}
	return db.WithContext(ctx).
		Model(&hostpluginmodel.HostPluginInstance{}).
		Where("agent_id = ?", agentID).
		Updates(updates).Error
}

func (s *Server) handleMetrics(ctx context.Context, stream grpc.BidiStreamingServer[pb.AgentMessage, pb.PlatformMessage], batch *pb.MetricBatch) error {
	if batch == nil {
		return nil
	}
	instance, err := s.lookupInstanceByStream(ctx, stream)
	if err != nil {
		return err
	}
	return s.persistMetricBatch(ctx, instance, batch)
}

func (s *Server) handleAck(ctx context.Context, ack *pb.Ack) error {
	if ack == nil {
		return nil
	}
	db := s.db()
	if db == nil {
		return status.Error(codes.FailedPrecondition, "opsagent service context requires db")
	}
	refID := strings.TrimSpace(ack.GetRefId())
	if !strings.HasPrefix(refID, "config_revision:") {
		return nil
	}
	revisionID := strings.TrimPrefix(refID, "config_revision:")
	statusValue := "failed"
	if ack.GetSuccess() {
		statusValue = "delivered"
	}
	return db.WithContext(ctx).
		Model(&hostpluginmodel.HostPluginConfigRevision{}).
		Where("id = ?", revisionID).
		Update("delivery_status", statusValue).Error
}

func (s *Server) sendConfigUpdate(ctx context.Context, session *SessionHandle, revision hostpluginmodel.HostPluginConfigRevision) error {
	if session == nil || session.Stream == nil {
		return status.Error(codes.FailedPrecondition, "opsagent session stream unavailable")
	}
	stream, ok := session.Stream.(interface {
		Send(*pb.PlatformMessage) error
	})
	if !ok {
		return status.Error(codes.FailedPrecondition, "opsagent session stream does not support platform messages")
	}
	version, _ := parseRevisionVersion(revision.Version)
	return stream.Send(&pb.PlatformMessage{
		Payload: &pb.PlatformMessage_ConfigUpdate{
			ConfigUpdate: &pb.ConfigUpdate{
				ConfigYaml: []byte(revision.ConfigYAML),
				Version:    version,
			},
		},
	})
}

func (s *Server) lookupPendingConfigRevision(ctx context.Context, instanceID uint64) (*hostpluginmodel.HostPluginConfigRevision, error) {
	db := s.db()
	if db == nil {
		return nil, status.Error(codes.FailedPrecondition, "opsagent service context requires db")
	}
	var revision hostpluginmodel.HostPluginConfigRevision
	err := db.WithContext(ctx).
		Where("instance_id = ? AND delivery_status = ?", instanceID, "pending").
		Order("id DESC").
		First(&revision).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &revision, nil
}

func (s *Server) lookupInstanceByStream(ctx context.Context, stream grpc.BidiStreamingServer[pb.AgentMessage, pb.PlatformMessage]) (*hostpluginmodel.HostPluginInstance, error) {
	db := s.db()
	if db == nil {
		return nil, status.Error(codes.FailedPrecondition, "opsagent service context requires db")
	}
	peerAgentID, err := s.registry.AgentIDByStream(stream)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	var instance hostpluginmodel.HostPluginInstance
	if err := db.WithContext(ctx).
		Where("agent_id = ?", peerAgentID).
		First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "opsagent instance not found")
		}
		return nil, status.Errorf(codes.Internal, "load opsagent instance: %v", err)
	}
	return &instance, nil
}

// buildServerCredentials creates gRPC server TLS credentials from config.
func buildServerCredentials(tlsCfg config.OpsAgentTLS) (credentials.TransportCredentials, error) {
	certPEM, err := os.ReadFile(tlsCfg.ServerCert)
	if err != nil {
		return nil, fmt.Errorf("read server cert: %w", err)
	}
	keyPEM, err := os.ReadFile(tlsCfg.ServerKey)
	if err != nil {
		return nil, fmt.Errorf("read server key: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load server keypair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if tlsCfg.CACert != "" {
		caPEM, err := os.ReadFile(tlsCfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("failed to parse CA cert")
		}
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = certPool
	}

	return credentials.NewTLS(tlsConfig), nil
}

func StartGRPCServer(ctx context.Context, svcCtx *svc.ServiceContext) error {
	addr := fmt.Sprintf("%s:%d", config.CFG.OpsAgent.Host, config.CFG.OpsAgent.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	var opts []grpc.ServerOption

	if config.CFG.OpsAgent.TLS.Enabled {
		creds, err := buildServerCredentials(config.CFG.OpsAgent.TLS)
		if err != nil {
			return fmt.Errorf("opsagent grpc: build TLS: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)
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
