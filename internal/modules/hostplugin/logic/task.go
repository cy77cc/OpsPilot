package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/gorm"
)

const (
	installStatusPending   = "pending"
	installStatusRunning   = "running"
	installStatusSucceeded = "succeeded"
	installStatusFailed    = "failed"
)

func (s *Service) ListInstanceIDsByHost(ctx context.Context, hostID uint64) ([]uint64, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var ids []uint64
	err := db.WithContext(ctx).
		Model(&hostpluginmodel.HostPluginInstance{}).
		Where("host_id = ?", hostID).
		Order("id ASC").
		Pluck("id", &ids).Error
	return ids, err
}

func (s *Service) EnqueueInstallTask(ctx context.Context, instanceID uint64) (*hostpluginmodel.HostPluginTask, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	task := &hostpluginmodel.HostPluginTask{
		InstanceID:   instanceID,
		Operation:    "install",
		Status:       installStatusPending,
		RequestedBy:  0,
		ErrorMessage: "",
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		return tx.Model(&hostpluginmodel.HostPluginInstance{}).
			Where("id = ?", instanceID).
			Updates(map[string]any{
				"install_status": installStatusPending,
				"last_error":     "",
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) startTask(ctx context.Context, taskID uint64) (*hostpluginmodel.HostPluginTask, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var task hostpluginmodel.HostPluginTask
	now := time.Now().UTC()
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		if err := tx.Model(&hostpluginmodel.HostPluginTask{}).
			Where("id = ?", taskID).
			Updates(map[string]any{
				"status":        installStatusRunning,
				"started_at":    &now,
				"error_message": "",
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&hostpluginmodel.HostPluginInstance{}).
			Where("id = ?", task.InstanceID).
			Updates(map[string]any{
				"install_status": installStatusRunning,
				"last_error":     "",
			}).Error; err != nil {
			return err
		}
		task.Status = installStatusRunning
		task.StartedAt = &now
		task.ErrorMessage = ""
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Service) finishTask(ctx context.Context, task *hostpluginmodel.HostPluginTask, installedVersion string, runErr error) {
	db := s.db()
	if db == nil || task == nil {
		return
	}

	now := time.Now().UTC()
	task.FinishedAt = &now
	task.ErrorMessage = ""

	instanceUpdates := map[string]any{
		"updated_at": now,
	}
	if runErr != nil {
		task.Status = installStatusFailed
		task.ErrorMessage = strings.TrimSpace(runErr.Error())
		instanceUpdates["install_status"] = installStatusFailed
		instanceUpdates["last_error"] = strings.TrimSpace(runErr.Error())
	} else {
		task.Status = installStatusSucceeded
		instanceUpdates["install_status"] = installStatusSucceeded
		instanceUpdates["installed_version"] = strings.TrimSpace(installedVersion)
		instanceUpdates["last_error"] = ""
	}

	_ = db.WithContext(ctx).Model(&hostpluginmodel.HostPluginTask{}).
		Where("id = ?", task.ID).
		Updates(map[string]any{
			"status":        task.Status,
			"finished_at":   task.FinishedAt,
			"error_message": task.ErrorMessage,
		}).Error

	_ = db.WithContext(ctx).Model(&hostpluginmodel.HostPluginInstance{}).
		Where("id = ?", task.InstanceID).
		Updates(instanceUpdates).Error
}

func (s *Service) appendTaskLog(ctx context.Context, taskID uint64, stream, content string) error {
	db := s.db()
	if db == nil {
		return errors.New("hostplugin service: db is required")
	}

	return db.WithContext(ctx).Create(&hostpluginmodel.HostPluginTaskLog{
		TaskID:  taskID,
		Stream:  strings.TrimSpace(stream),
		Content: strings.TrimSpace(content),
	}).Error
}
