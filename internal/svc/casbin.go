package svc

import (
	"github.com/casbin/casbin/v2"
	casbinadapter "github.com/cy77cc/OpsPilot/internal/component/casbin"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	"gorm.io/gorm"
)

func newCasbinEnforcer(db *gorm.DB) *casbin.Enforcer {
	adapter := casbinadapter.NewAdapter(db)
	enforcer, err := casbin.NewEnforcer("resource/casbin/rbac_model.conf", adapter)
	if err != nil {
		logger.L().Error("Failed to initialize Casbin Enforcer", logger.Error(err))
		return nil
	}
	if err := enforcer.LoadPolicy(); err != nil {
		logger.L().Error("Failed to load Casbin policy", logger.Error(err))
	}
	return enforcer
}
