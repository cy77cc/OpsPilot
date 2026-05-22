package logic

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
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

// InstallOnHost installs a plugin on an existing host. It generates mTLS certs,
// creates the instance, config, and install task, and returns the task ID.
func (s *Service) InstallOnHost(ctx context.Context, hostID uint64, pluginKey, version string, hostIP string) (uint64, error) {
	db := s.db()
	if db == nil {
		return 0, errors.New("hostplugin service: db is required")
	}

	// Check for existing instance
	var existing hostpluginmodel.HostPluginInstance
	err := db.WithContext(ctx).
		Joins("JOIN host_plugins ON host_plugins.id = host_plugin_instances.plugin_id").
		Where("host_plugins.plugin_key = ? AND host_plugin_instances.host_id = ?", pluginKey, hostID).
		First(&existing).Error
	if err == nil {
		return 0, errors.New("plugin instance already exists on this host")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// Ensure CA is initialized
	if s.svcCtx != nil && s.svcCtx.CAManager != nil {
		if _, _, err := s.svcCtx.CAManager.EnsureCA(); err != nil {
			return 0, fmt.Errorf("ensure CA: %w", err)
		}
	}

	var taskID uint64
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create instance
		seed, ok := findDefaultCatalogPlugin(pluginKey)
		if !ok {
			return errors.New("unknown plugin key")
		}

		plugin := seed
		if err := tx.Where("plugin_key = ?", pluginKey).FirstOrCreate(&plugin).Error; err != nil {
			return err
		}

		desiredVersion := version
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
		if err := tx.Create(&instance).Error; err != nil {
			return err
		}

		instance.AgentID = buildOpsAgentID(hostID, instance.ID)
		if err := tx.Model(&hostpluginmodel.HostPluginInstance{}).
			Where("id = ?", instance.ID).
			Update("agent_id", instance.AgentID).Error; err != nil {
			return err
		}

		// Generate mTLS client cert
		if s.svcCtx != nil && s.svcCtx.CAManager != nil {
			certPEM, keyPEM, err := s.svcCtx.CAManager.IssueClientCert(
				instance.AgentID, net.ParseIP(hostIP),
			)
			if err != nil {
				return fmt.Errorf("issue client cert: %w", err)
			}

			// Encrypt key PEM
			encryptedKey, err := utils.EncryptText(string(keyPEM), config.CFG.Security.EncryptionKey)
			if err != nil {
				return fmt.Errorf("encrypt key: %w", err)
			}

			certRecord := hostpluginmodel.OpsAgentHostCert{
				HostID:       hostID,
				InstanceID:   instance.ID,
				SerialNumber: instance.AgentID,
				CertPEM:      string(certPEM),
				KeyPEM:       encryptedKey,
				NotAfter:     time.Now().AddDate(1, 0, 0),
				Revoked:      false,
			}
			if err := tx.Create(&certRecord).Error; err != nil {
				return fmt.Errorf("persist cert: %w", err)
			}
		}

		// Generate full config
		configYAML := renderFullOpsAgentConfig(instance.AgentID, s.resolveGRPCServerAddr())
		revision := hostpluginmodel.HostPluginConfigRevision{
			InstanceID:     instance.ID,
			Version:        "1",
			ConfigYAML:     configYAML,
			Checksum:       sha256Hex(configYAML),
			DeliveryStatus: "pending",
			CreatedBy:      0,
		}
		if err := tx.Create(&revision).Error; err != nil {
			return err
		}

		// Create install task
		task := hostpluginmodel.HostPluginTask{
			InstanceID:   instance.ID,
			Operation:    "install",
			Status:       installStatusPending,
			RequestedBy:  0,
			ErrorMessage: "",
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}

		taskID = task.ID
		return nil
	})
	if err != nil {
		return 0, err
	}
	return taskID, nil
}

// resolveGRPCServerAddr determines the gRPC server address for agent config.
func (s *Service) resolveGRPCServerAddr() string {
	host := config.CFG.OpsAgent.Host
	port := config.CFG.OpsAgent.Port
	if host == "0.0.0.0" || host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// renderFullOpsAgentConfig generates a complete config with TLS paths and gRPC address.
func renderFullOpsAgentConfig(agentID, grpcServerAddr string) string {
	bearerToken, _ := generateRandomToken(32)
	return fmt.Sprintf(`agent:
  id: "%s"
  name: "%s"
  interval_seconds: 15
  shutdown_timeout_seconds: 30

server:
  listen_addr: "127.0.0.1:18080"

grpc:
  server_addr: "%s"
  enroll_token: "%s"
  heartbeat_interval: 15
  mtls:
    cert_file: "/etc/opsagent/certs/client.crt"
    key_file: "/etc/opsagent/certs/client.key"
    ca_file: "/etc/opsagent/certs/ca.crt"

auth:
  enabled: true
  bearer_token: "%s"

collector:
  inputs:
    - type: "cpu"
    - type: "memory"
    - type: "disk"
    - type: "net"

sandbox:
  enabled: true
  nsjail_path: "/usr/bin/nsjail"
  base_workdir: "/tmp/opsagent/sandbox"
`, agentID, agentID, grpcServerAddr, agentID, bearerToken)
}

// generateRandomToken generates a random alphanumeric string of the given length.
func generateRandomToken(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b), nil
}

// ListInstancesByHost returns all plugin instances for a host.
func (s *Service) ListInstancesByHost(ctx context.Context, hostID uint64) ([]hostpluginmodel.HostPluginInstance, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var instances []hostpluginmodel.HostPluginInstance
	err := db.WithContext(ctx).Where("host_id = ?", hostID).Order("id ASC").Find(&instances).Error
	return instances, err
}

// GetTask returns a task by ID.
func (s *Service) GetTask(ctx context.Context, taskID uint64) (*hostpluginmodel.HostPluginTask, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var task hostpluginmodel.HostPluginTask
	if err := db.WithContext(ctx).First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTaskLogs returns all logs for a task.
func (s *Service) ListTaskLogs(ctx context.Context, taskID uint64) ([]hostpluginmodel.HostPluginTaskLog, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var logs []hostpluginmodel.HostPluginTaskLog
	err := db.WithContext(ctx).Where("task_id = ?", taskID).Order("id ASC").Find(&logs).Error
	return logs, err
}

// GetHost returns a host by ID (thin wrapper for handler use).
func (s *Service) GetHost(ctx context.Context, hostID uint64) (*hostmodel.Node, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var host hostmodel.Node
	if err := db.WithContext(ctx).First(&host, hostID).Error; err != nil {
		return nil, err
	}
	return &host, nil
}

// HostPluginPackageInput is the input for creating a package.
type HostPluginPackageInput struct {
	PluginKey   string
	Version     string
	Arch        string
	Filename    string
	StoragePath string
	Checksum    string
	SizeBytes   int64
}

// CreatePackageFromInput creates a package from input.
func (s *Service) CreatePackageFromInput(ctx context.Context, input *HostPluginPackageInput) error {
	pkg := &hostpluginmodel.HostPluginPackage{
		PluginKey:   input.PluginKey,
		Version:     input.Version,
		Arch:        input.Arch,
		Filename:    input.Filename,
		StoragePath: input.StoragePath,
		Checksum:    input.Checksum,
		SizeBytes:   input.SizeBytes,
	}
	return s.CreatePackage(ctx, pkg)
}
