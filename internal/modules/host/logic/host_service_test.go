package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/gorm"
)

func TestCreateWithProbe_CreatesHostAndPluginInstance(t *testing.T) {
	svc, db := newHostLogicTestService(t)
	if err := db.AutoMigrate(&hostpluginmodel.HostPlugin{}, &hostpluginmodel.HostPluginInstance{}, &hostpluginmodel.HostPluginTask{}); err != nil {
		t.Fatalf("auto migrate hostplugin tables: %v", err)
	}
	const probeToken = "host-plugin-probe-token"
	if err := db.Create(&model.HostProbeSession{
		TokenHash:      hashToken(probeToken),
		Name:           "host-a",
		IP:             "10.0.0.8",
		Port:           22,
		AuthType:       "password",
		Username:       "root",
		PasswordCipher: "",
		Reachable:      true,
		FactsJSON:      `{"arch":"amd64"}`,
		WarningsJSON:   `[]`,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		CreatedBy:      1,
	}).Error; err != nil {
		t.Fatalf("seed probe session: %v", err)
	}
	req := CreateReq{
		ProbeToken: probeToken,
		Name:       "host-a",
		PluginInstalls: []PluginInstallReq{{
			PluginKey: "opsagent",
			Version:   "nodeagentx-dc57fbc-dirty",
		}},
	}

	_, err := svc.CreateWithProbe(context.Background(), 1, true, req)
	if err != nil {
		t.Fatalf("create host with plugin: %v", err)
	}

	var count int64
	db.Table("host_plugin_instances").Where("host_id > 0").Count(&count)
	if count != 1 {
		t.Fatalf("expected one plugin instance, got %d", count)
	}
	db.Table("host_plugin_tasks").Count(&count)
	if count != 1 {
		t.Fatalf("expected one install task row, got %d", count)
	}

	var task hostpluginmodel.HostPluginTask
	if err := db.First(&task).Error; err != nil {
		t.Fatalf("load install task: %v", err)
	}
	if task.Status != "pending" {
		t.Fatalf("expected install task to persist as pending, got %s", task.Status)
	}
}

func TestCreateWithProbe_LegacyRejectsPluginInstallsWithoutProbe(t *testing.T) {
	svc, db := newHostLogicTestService(t)
	if err := db.AutoMigrate(&hostpluginmodel.HostPlugin{}, &hostpluginmodel.HostPluginInstance{}); err != nil {
		t.Fatalf("auto migrate hostplugin tables: %v", err)
	}

	_, err := svc.CreateWithProbe(context.Background(), 1, true, CreateReq{
		Name: "legacy-host",
		IP:   "10.0.0.9",
		PluginInstalls: []PluginInstallReq{{
			PluginKey: "opsagent",
			Version:   "nodeagentx-dc57fbc-dirty",
		}},
	})
	if err == nil {
		t.Fatalf("expected legacy host creation with plugin installs to fail")
	}
	if !strings.Contains(err.Error(), "plugin_installs requires probe-based host creation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConsumeProbe_ConcurrentOnlyOneSucceeds(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)

	const userID = uint64(9001)
	probeToken := fmt.Sprintf("task8-concurrent-probe-token-%d", time.Now().UnixNano())
	if err := db.WithContext(context.Background()).Create(&model.HostProbeSession{
		TokenHash:      hashToken(probeToken),
		Name:           "task8-node",
		IP:             "10.10.0.8",
		Port:           22,
		AuthType:       "password",
		Username:       "root",
		PasswordCipher: "task8-cipher",
		Reachable:      true,
		FactsJSON:      `{}`,
		WarningsJSON:   `[]`,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		CreatedBy:      userID,
	}).Error; err != nil {
		t.Fatalf("seed probe session: %v", err)
	}

	var queried int32
	bothQueried := make(chan struct{})
	queryCallbackName := "test:task8_consume_probe_query_barrier"
	if err := db.Callback().Query().After("gorm:query").Register(queryCallbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "host_probe_sessions" {
			return
		}
		if atomic.AddInt32(&queried, 1) == 2 {
			close(bothQueried)
		}
		select {
		case <-bothQueried:
		case <-time.After(2 * time.Second):
			tx.AddError(errors.New("timeout waiting concurrent consumeProbe queries"))
		}
	}); err != nil {
		t.Fatalf("register query barrier callback: %v", err)
	}

	var updates int32
	updateCallbackName := "test:task8_consume_probe_update_delay_second"
	if err := db.Callback().Update().Before("gorm:update").Register(updateCallbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "host_probe_sessions" {
			return
		}
		if atomic.AddInt32(&updates, 1) == 2 {
			time.Sleep(10 * time.Millisecond)
		}
	}); err != nil {
		t.Fatalf("register update ordering callback: %v", err)
	}

	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, consumeErr := hostSvc.consumeProbe(context.Background(), userID, probeToken)
			errCh <- consumeErr
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	successes := 0
	notFound := 0
	for consumeErr := range errCh {
		switch {
		case consumeErr == nil:
			successes++
		case consumeErr.Error() == "probe_not_found":
			notFound++
		default:
			t.Fatalf("consume probe returned unexpected error: %v", consumeErr)
		}
	}

	if successes != 1 || notFound != 1 {
		t.Fatalf("expected one success and one probe_not_found, got success=%d probe_not_found=%d", successes, notFound)
	}
}

func TestFixCreateMultipleManualHosts(t *testing.T) {
	s, db := newHostLogicTestService(t)
	// 模拟迁移逻辑
	if err := db.Exec("DROP INDEX IF EXISTS idx_provider_instance").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_provider_instance ON nodes(provider, provider_instance_id) WHERE provider IS NOT NULL AND provider != ''`).Error; err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// 1. 创建第一个手动主机
	req1 := CreateReq{
		Name:     "Host1",
		IP:       "192.168.1.1",
		Port:     22,
		Username: "root",
		Source:   "manual_ssh",
	}
	_, err := s.CreateWithProbe(ctx, 1, true, req1)
	if err != nil {
		t.Fatalf("Failed to create first manual host: %v", err)
	}

	// 2. 创建第二个手动主机 (之前会报错)
	req2 := CreateReq{
		Name:     "Host2",
		IP:       "192.168.1.2",
		Port:     22,
		Username: "root",
		Source:   "manual_ssh",
	}
	_, err = s.CreateWithProbe(ctx, 1, true, req2)
	if err != nil {
		t.Fatalf("Failed to create second manual host: %v", err)
	}

	// 3. 验证云主机依然有约束
	provider := "aliyun"
	insID := "i-12345"
	node1 := &model.Node{
		Name:       "Cloud1",
		IP:         "1.1.1.1",
		Provider:   &provider,
		ProviderID: &insID,
		Status:     "online",
	}
	if err := db.Create(node1).Error; err != nil {
		t.Fatal(err)
	}

	node2 := &model.Node{
		Name:       "Cloud2",
		IP:         "1.1.1.2",
		Provider:   &provider,
		ProviderID: &insID,
		Status:     "online",
	}
	err = db.Create(node2).Error
	if err == nil {
		t.Fatal("Expected unique constraint violation for duplicate cloud hosts")
	}
}
