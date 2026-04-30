package logic

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

type Service struct {
	svcCtx *svc.ServiceContext
}

func NewService(svcCtx *svc.ServiceContext) *Service {
	return &Service{svcCtx: svcCtx}
}

func (s *Service) EnsureDefaultCatalog(ctx context.Context) error {
	db := s.db()
	if db == nil {
		return errors.New("hostplugin service: db is required")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, seed := range defaultCatalog {
			plugin := seed
			if err := tx.Where("plugin_key = ?", plugin.PluginKey).FirstOrCreate(&plugin).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) ListCatalog(ctx context.Context) ([]hostpluginmodel.HostPlugin, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	if err := s.EnsureDefaultCatalog(ctx); err != nil {
		return nil, err
	}

	var plugins []hostpluginmodel.HostPlugin
	err := db.WithContext(ctx).Order("id ASC").Find(&plugins).Error
	return plugins, err
}

func (s *Service) RequireOnlineCapability(ctx context.Context, hostID uint64, capability string) (*hostpluginmodel.HostPluginInstance, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var instance hostpluginmodel.HostPluginInstance
	err := db.WithContext(ctx).
		Table("host_plugin_instances AS hpi").
		Select("hpi.*").
		Joins("JOIN host_plugins hp ON hp.id = hpi.plugin_id").
		Where("hp.plugin_key = ?", "opsagent").
		Where("hpi.host_id = ?", hostID).
		Where("hpi.install_status = ?", "succeeded").
		Where("hpi.runtime_status = ?", "online").
		First(&instance).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("online opsagent plugin instance not found")
		}
		return nil, err
	}

	if strings.TrimSpace(capability) == "" {
		return &instance, nil
	}

	var capabilities []string
	if strings.TrimSpace(instance.CapabilitiesJSON) != "" {
		_ = json.Unmarshal([]byte(instance.CapabilitiesJSON), &capabilities)
	}
	for _, item := range capabilities {
		if item == capability {
			return &instance, nil
		}
	}
	return nil, errors.New("required plugin capability is unavailable")
}

func (s *Service) CreatePendingInstance(ctx context.Context, tx *gorm.DB, hostID uint64, pluginKey, version string, _ uint64) error {
	_, err := s.CreatePendingInstanceWithTask(ctx, tx, hostID, pluginKey, version, 0)
	return err
}

func (s *Service) CreatePendingInstanceWithTask(ctx context.Context, tx *gorm.DB, hostID uint64, pluginKey, version string, _ uint64) (uint64, error) {
	if tx == nil {
		return 0, errors.New("hostplugin service: tx is required")
	}

	pluginKey = strings.TrimSpace(pluginKey)
	if pluginKey == "" {
		return 0, errors.New("hostplugin service: plugin key is required")
	}

	seed, ok := findDefaultCatalogPlugin(pluginKey)
	if !ok {
		return 0, errors.New("hostplugin service: unknown plugin key")
	}

	plugin := seed
	if err := tx.WithContext(ctx).Where("plugin_key = ?", pluginKey).FirstOrCreate(&plugin).Error; err != nil {
		return 0, err
	}

	desiredVersion := strings.TrimSpace(version)
	if desiredVersion == "" {
		desiredVersion = plugin.DefaultVersion
	}

	instance := hostpluginmodel.HostPluginInstance{
		HostID:           hostID,
		PluginID:         plugin.ID,
		DesiredVersion:   desiredVersion,
		CapabilitiesJSON: "[]",
		LastError:        "",
	}
	if err := tx.WithContext(ctx).Create(&instance).Error; err != nil {
		return 0, err
	}
	instance.AgentID = buildOpsAgentID(hostID, instance.ID)
	if err := tx.WithContext(ctx).Model(&hostpluginmodel.HostPluginInstance{}).
		Where("id = ?", instance.ID).
		Update("agent_id", instance.AgentID).Error; err != nil {
		return 0, err
	}

	configYAML := renderOpsAgentConfig(instance.AgentID)
	revision := hostpluginmodel.HostPluginConfigRevision{
		InstanceID:     instance.ID,
		Version:        "1",
		ConfigYAML:     configYAML,
		Checksum:       sha256Hex(configYAML),
		DeliveryStatus: "pending",
		CreatedBy:      0,
	}
	if err := tx.WithContext(ctx).Create(&revision).Error; err != nil {
		return 0, err
	}

	task := hostpluginmodel.HostPluginTask{
		InstanceID:   instance.ID,
		Operation:    "install",
		Status:       installStatusPending,
		RequestedBy:  0,
		ErrorMessage: "",
	}
	if err := tx.WithContext(ctx).Create(&task).Error; err != nil {
		return 0, err
	}

	return task.ID, nil
}

func (s *Service) db() *gorm.DB {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return nil
	}
	return s.svcCtx.DB
}

func findDefaultCatalogPlugin(pluginKey string) (hostpluginmodel.HostPlugin, bool) {
	for _, item := range defaultCatalog {
		if item.PluginKey == pluginKey {
			return item, true
		}
	}
	return hostpluginmodel.HostPlugin{}, false
}

func buildOpsAgentID(hostID, instanceID uint64) string {
	return fmt.Sprintf("opsagent-host-%d-instance-%d", hostID, instanceID)
}

func renderOpsAgentConfig(agentID string) string {
	return fmt.Sprintf(`agent:
  id: "%s"
  name: "%s"
  interval_seconds: 10

grpc:
  enroll_token: "%s"
`, agentID, agentID, agentID)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:])
}
