package logic

import (
	"context"
	"io"
	"testing"
	"time"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
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
		&AgentMessage{
			Payload: &AgentMessageRegistration{
				Registration: &AgentRegistration{
					AgentID: "agent-host-1",
					Token:   "token-1",
				},
			},
		},
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Connect(stream)
	}()

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

func openOpsAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&hostpluginmodel.HostPlugin{},
		&hostpluginmodel.HostPluginInstance{},
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

	return db
}

type fakeAgentServiceConnectServer struct {
	t        *testing.T
	ctx      context.Context
	cancelFn context.CancelFunc
	messages []*AgentMessage
}

func newFakeAgentServiceConnectServer(t *testing.T, messages ...*AgentMessage) *fakeAgentServiceConnectServer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	return &fakeAgentServiceConnectServer{
		t:        t,
		ctx:      ctx,
		cancelFn: cancel,
		messages: messages,
	}
}

func (f *fakeAgentServiceConnectServer) Context() context.Context {
	return f.ctx
}

func (f *fakeAgentServiceConnectServer) Recv() (*AgentMessage, error) {
	if len(f.messages) == 0 {
		<-f.ctx.Done()
		return nil, io.EOF
	}
	msg := f.messages[0]
	f.messages = f.messages[1:]
	return msg, nil
}

func (f *fakeAgentServiceConnectServer) Send(*PlatformMessage) error {
	return nil
}

func (f *fakeAgentServiceConnectServer) cancel() {
	f.cancelFn()
}
