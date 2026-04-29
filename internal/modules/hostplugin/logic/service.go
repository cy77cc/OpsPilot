package logic

import (
	"context"
	"errors"
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

func (s *Service) CreatePendingInstance(ctx context.Context, tx *gorm.DB, hostID uint64, pluginKey, version string, _ uint64) error {
	if tx == nil {
		return errors.New("hostplugin service: tx is required")
	}

	pluginKey = strings.TrimSpace(pluginKey)
	if pluginKey == "" {
		return errors.New("hostplugin service: plugin key is required")
	}

	seed, ok := findDefaultCatalogPlugin(pluginKey)
	if !ok {
		return errors.New("hostplugin service: unknown plugin key")
	}

	plugin := seed
	if err := tx.WithContext(ctx).Where("plugin_key = ?", pluginKey).FirstOrCreate(&plugin).Error; err != nil {
		return err
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
	return tx.WithContext(ctx).Create(&instance).Error
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
