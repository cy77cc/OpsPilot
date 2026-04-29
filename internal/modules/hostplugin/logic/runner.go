package logic

import (
	"context"
	"errors"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/gorm"
)

func (s *Service) RunPendingInstallTasksOnce(ctx context.Context) (bool, error) {
	db := s.db()
	if db == nil {
		return false, errors.New("hostplugin service: db is required")
	}

	var task hostpluginmodel.HostPluginTask
	err := db.WithContext(ctx).
		Where("operation = ? AND status = ?", "install", installStatusPending).
		Order("id ASC").
		First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	runErr := s.RunInstallTask(ctx, task.ID)
	if errors.Is(runErr, errInstallTaskNotPending) {
		return true, nil
	}
	return true, runErr
}
