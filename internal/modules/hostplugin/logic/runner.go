package logic

import (
	"context"
	"errors"
	"time"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/gorm"
)

const staleInstallTaskTimeout = 2 * time.Minute

func (s *Service) RunPendingInstallTasksOnce(ctx context.Context) (bool, error) {
	db := s.db()
	if db == nil {
		return false, errors.New("hostplugin service: db is required")
	}

	recovered, err := s.recoverStaleRunningInstallTask(ctx, staleInstallTaskTimeout)
	if err != nil {
		return false, err
	}

	var task hostpluginmodel.HostPluginTask
	err = db.WithContext(ctx).
		Where("operation = ? AND status = ?", "install", installStatusPending).
		Order("id ASC").
		First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return recovered, nil
	}
	if err != nil {
		return false, err
	}

	runErr := s.RunInstallTask(ctx, task.ID)
	if errors.Is(runErr, ErrInstallTaskNotPending) {
		return true, nil
	}
	return true, runErr
}

func (s *Service) recoverStaleRunningInstallTask(ctx context.Context, timeout time.Duration) (bool, error) {
	db := s.db()
	if db == nil {
		return false, errors.New("hostplugin service: db is required")
	}

	cutoff := time.Now().UTC().Add(-timeout)
	var task hostpluginmodel.HostPluginTask
	err := db.WithContext(ctx).
		Where("operation = ? AND status = ? AND started_at IS NOT NULL AND started_at < ?", "install", installStatusRunning, cutoff).
		Order("started_at ASC, id ASC").
		First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	recoveryMessage := "recovered stale running install task for retry"
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&hostpluginmodel.HostPluginTask{}).
			Where("id = ? AND status = ?", task.ID, installStatusRunning).
			Updates(map[string]any{
				"status":        installStatusPending,
				"started_at":    nil,
				"error_message": recoveryMessage,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&hostpluginmodel.HostPluginInstance{}).
			Where("id = ?", task.InstanceID).
			Updates(map[string]any{
				"install_status": installStatusPending,
				"last_error":     recoveryMessage,
			}).Error
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
