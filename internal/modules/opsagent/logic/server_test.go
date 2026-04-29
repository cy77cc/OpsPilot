package logic

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	pb "github.com/cy77cc/OpsPilot/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConnect_RegistrationMarksInstanceOnline(t *testing.T) {
	db := openOpsAgentTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}
	registry := NewSessionRegistry()
	server := NewServer(svcCtx, registry)

	stream := newFakeAgentServiceConnectServer(
		t,
		&pb.AgentMessage{
			Payload: &pb.AgentMessage_Registration{
				Registration: &pb.AgentRegistration{
					AgentId: "agent-host-1",
					Token:   "token-1",
				},
			},
		},
	)

	errCh := make(chan error, 1)
	go func() { errCh <- server.Connect(stream) }()

	var instance hostpluginmodel.HostPluginInstance
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Where("agent_id = ?", "agent-host-1").First(&instance).Error; err == nil &&
			instance.RuntimeStatus == "online" &&
			instance.LastSeenAt != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	session, ok := registry.GetByAgent("agent-host-1")
	if !ok {
		t.Fatal("expected session to be indexed by agent id")
	}
	if session.HostID != instance.HostID {
		t.Fatalf("expected session host id %d, got %d", instance.HostID, session.HostID)
	}
	if instance.RuntimeStatus != "online" {
		t.Fatalf("expected runtime_status online, got %q", instance.RuntimeStatus)
	}
	if instance.LastSeenAt == nil {
		t.Fatal("expected last_seen_at to be populated")
	}

	stream.cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("connect: %v", err)
	}
}

func TestConnect_BufconnRegistrationMarksInstanceOnline(t *testing.T) {
	db := openOpsAgentTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}
	registry := NewSessionRegistry()
	server := NewServer(svcCtx, registry)

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterAgentServiceServer(grpcServer, server)
	defer grpcServer.Stop()

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc new client: %v", err)
	}
	defer conn.Close()

	client := pb.NewAgentServiceClient(conn)
	stream, err := client.Connect(context.Background())
	if err != nil {
		t.Fatalf("connect client stream: %v", err)
	}
	if err := stream.Send(&pb.AgentMessage{
		Payload: &pb.AgentMessage_Registration{
			Registration: &pb.AgentRegistration{
				AgentId: "agent-host-1",
				Token:   "token-1",
			},
		},
	}); err != nil {
		t.Fatalf("send registration: %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("close send: %v", err)
	}

	var instance hostpluginmodel.HostPluginInstance
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Where("agent_id = ?", "agent-host-1").First(&instance).Error; err == nil &&
			instance.RuntimeStatus == "online" &&
			instance.LastSeenAt != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected runtime_status online after bufconn registration")
}

func TestSessionRegistry_DeleteByAgentStreamDoesNotDeleteNewerSession(t *testing.T) {
	registry := NewSessionRegistry()
	streamA := newFakeAgentServiceConnectServer(t)
	streamB := newFakeAgentServiceConnectServer(t)

	registry.Put(101, "agent-host-1", streamA)
	registry.Put(101, "agent-host-1", streamB)
	registry.DeleteByAgentStream("agent-host-1", streamA)

	session, ok := registry.GetByAgent("agent-host-1")
	if !ok {
		t.Fatal("expected newer session to remain registered")
	}
	if session.Stream != streamB {
		t.Fatal("expected newer stream to remain after stale cleanup")
	}
}

func TestHandleHeartbeat_UpdatesPluginInstanceRuntimeState(t *testing.T) {
	db := openOpsAgentTestDB(t)
	svc := NewServer(&svc.ServiceContext{DB: db}, NewSessionRegistry())

	err := svc.handleHeartbeat(context.Background(), &pb.Heartbeat{
		AgentId:     "agent-host-1",
		Status:      "online",
		TimestampMs: time.Now().Add(-time.Minute).UnixMilli(),
	})
	if err != nil {
		t.Fatalf("handle heartbeat: %v", err)
	}

	var instance hostpluginmodel.HostPluginInstance
	if err := db.Where("agent_id = ?", "agent-host-1").First(&instance).Error; err != nil {
		t.Fatalf("load instance: %v", err)
	}
	if instance.RuntimeStatus != "online" {
		t.Fatalf("expected runtime_status online, got %s", instance.RuntimeStatus)
	}
	if instance.HealthStatus != "healthy" {
		t.Fatalf("expected health_status healthy, got %s", instance.HealthStatus)
	}
	if instance.LastSeenAt == nil {
		t.Fatal("expected last_seen_at to be set")
	}
}

func TestHandleMetrics_PersistsHealthSnapshot(t *testing.T) {
	db := openOpsAgentTestDB(t)
	registry := NewSessionRegistry()
	stream := newFakeAgentServiceConnectServer(t)
	registry.Put(101, "agent-host-1", stream)
	svc := NewServer(&svc.ServiceContext{DB: db}, registry)

	batch := &pb.MetricBatch{
		Metrics: []*pb.Metric{
			{
				Name: "cpu",
				Fields: []*pb.Field{{
					Key:   "usage_percent",
					Value: &pb.Field_DoubleValue{DoubleValue: 50},
				}},
				TimestampMs: time.Now().UnixMilli(),
			},
			{
				Name: "memory",
				Fields: []*pb.Field{
					{Key: "total_bytes", Value: &pb.Field_IntValue{IntValue: 8 * 1024 * 1024 * 1024}},
					{Key: "used_percent", Value: &pb.Field_DoubleValue{DoubleValue: 25}},
				},
				TimestampMs: time.Now().UnixMilli(),
			},
			{
				Name: "disk",
				Fields: []*pb.Field{
					{Key: "used_percent", Value: &pb.Field_DoubleValue{DoubleValue: 70}},
					{Key: "io_iops", Value: &pb.Field_DoubleValue{DoubleValue: 12}},
				},
			},
			{
				Name: "net",
				Fields: []*pb.Field{
					{Key: "rx_bytes", Value: &pb.Field_IntValue{IntValue: 111}},
					{Key: "tx_bytes", Value: &pb.Field_IntValue{IntValue: 222}},
				},
			},
		},
	}

	if err := svc.handleMetrics(context.Background(), stream, batch); err != nil {
		t.Fatalf("handle metrics: %v", err)
	}

	var snapshot hostmodel.HostHealthSnapshot
	if err := db.Where("host_id = ?", 101).First(&snapshot).Error; err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if snapshot.MemoryTotalMB != 8192 {
		t.Fatalf("expected memory total 8192, got %d", snapshot.MemoryTotalMB)
	}
	if snapshot.MemoryUsedMB != 2048 {
		t.Fatalf("expected memory used 2048, got %d", snapshot.MemoryUsedMB)
	}
	if snapshot.DiskUsedPct != 70 {
		t.Fatalf("expected disk used pct 70, got %v", snapshot.DiskUsedPct)
	}
	var summary map[string]any
	if err := json.Unmarshal([]byte(snapshot.SummaryJSON), &summary); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if summary["net_rx_bytes"].(float64) != 111 {
		t.Fatalf("expected net_rx_bytes 111, got %v", summary["net_rx_bytes"])
	}
}

func TestHandleAck_MarksConfigRevisionDelivered(t *testing.T) {
	db := openOpsAgentTestDB(t)
	svc := NewServer(&svc.ServiceContext{DB: db}, NewSessionRegistry())

	revision := hostpluginmodel.HostPluginConfigRevision{
		InstanceID:     1,
		Version:        "1",
		ConfigYAML:     "agent: {}",
		Checksum:       "sha256",
		DeliveryStatus: "pending",
	}
	if err := db.Create(&revision).Error; err != nil {
		t.Fatalf("create config revision: %v", err)
	}

	if err := svc.handleAck(context.Background(), &pb.Ack{
		RefId:   "config_revision:" + strconv.FormatUint(revision.ID, 10),
		Success: true,
	}); err != nil {
		t.Fatalf("handle ack: %v", err)
	}

	var reloaded hostpluginmodel.HostPluginConfigRevision
	if err := db.First(&reloaded, revision.ID).Error; err != nil {
		t.Fatalf("load revision: %v", err)
	}
	if reloaded.DeliveryStatus != "delivered" {
		t.Fatalf("expected delivered status, got %s", reloaded.DeliveryStatus)
	}
}

func openOpsAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&hostmodel.Node{},
		&hostmodel.HostHealthSnapshot{},
		&hostpluginmodel.HostPlugin{},
		&hostpluginmodel.HostPluginInstance{},
		&hostpluginmodel.HostPluginConfigRevision{},
	); err != nil {
		t.Fatalf("auto migrate opsagent tables: %v", err)
	}

	plugin := hostpluginmodel.HostPlugin{
		PluginKey:      "opsagent",
		Name:           "OpsAgent",
		Category:       "runtime",
		Description:    "agent runtime",
		DefaultVersion: "v1.0.0",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	instance := hostpluginmodel.HostPluginInstance{
		HostID:           101,
		PluginID:         plugin.ID,
		DesiredVersion:   "v1.0.0",
		InstalledVersion: "v1.0.0",
		InstallStatus:    "succeeded",
		RuntimeStatus:    "pending_online",
		AgentID:          "agent-host-1",
		CapabilitiesJSON: "[]",
		LastError:        "",
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	node := hostmodel.Node{
		ID:          hostmodel.NodeID(101),
		Name:        "host-a",
		IP:          "10.0.0.8",
		Port:        22,
		SSHUser:     "root",
		Status:      "online",
		HealthState: "unknown",
		Source:      "manual_ssh",
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	return db
}

type fakeAgentServiceConnectServer struct {
	t        *testing.T
	ctx      context.Context
	cancelFn context.CancelFunc
	messages []*pb.AgentMessage
	sent     chan *pb.PlatformMessage
}

func newFakeAgentServiceConnectServer(t *testing.T, messages ...*pb.AgentMessage) *fakeAgentServiceConnectServer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	return &fakeAgentServiceConnectServer{
		t:        t,
		ctx:      ctx,
		cancelFn: cancel,
		messages: messages,
		sent:     make(chan *pb.PlatformMessage, 8),
	}
}

func (f *fakeAgentServiceConnectServer) SetHeader(metadata.MD) error  { return nil }
func (f *fakeAgentServiceConnectServer) SendHeader(metadata.MD) error { return nil }
func (f *fakeAgentServiceConnectServer) SetTrailer(metadata.MD)       {}
func (f *fakeAgentServiceConnectServer) Context() context.Context     { return f.ctx }
func (f *fakeAgentServiceConnectServer) SendMsg(any) error            { return nil }
func (f *fakeAgentServiceConnectServer) RecvMsg(any) error            { return nil }

func (f *fakeAgentServiceConnectServer) Recv() (*pb.AgentMessage, error) {
	if len(f.messages) == 0 {
		<-f.ctx.Done()
		return nil, io.EOF
	}
	msg := f.messages[0]
	f.messages = f.messages[1:]
	return msg, nil
}

func (f *fakeAgentServiceConnectServer) Send(msg *pb.PlatformMessage) error {
	f.sent <- msg
	return nil
}

func (f *fakeAgentServiceConnectServer) cancel() { f.cancelFn() }
