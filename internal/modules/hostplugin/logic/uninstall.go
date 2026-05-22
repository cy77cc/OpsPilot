package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/gorm"
)

var executeHostPluginUninstallPlan = func(ctx context.Context, s *Service, host *hostmodel.Node, task *hostpluginmodel.HostPluginTask) error {
	return s.runUninstallPlan(ctx, host, task)
}

// UninstallOnHost uninstalls a plugin from an existing host.
func (s *Service) UninstallOnHost(ctx context.Context, hostID, instanceID uint64) (uint64, error) {
	db := s.db()
	if db == nil {
		return 0, errors.New("hostplugin service: db is required")
	}

	// Validate instance belongs to host
	var instance hostpluginmodel.HostPluginInstance
	if err := db.WithContext(ctx).Where("id = ? AND host_id = ?", instanceID, hostID).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("plugin instance not found for this host")
		}
		return 0, err
	}

	if instance.RuntimeStatus == "uninstalled" {
		return 0, errors.New("plugin is already uninstalled")
	}

	var taskID uint64
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Mark instance as draining
		if err := tx.Model(&hostpluginmodel.HostPluginInstance{}).
			Where("id = ?", instanceID).
			Updates(map[string]any{
				"runtime_status": "draining",
				"last_error":     "",
			}).Error; err != nil {
			return err
		}

		// Create uninstall task
		task := hostpluginmodel.HostPluginTask{
			InstanceID:   instanceID,
			Operation:    "uninstall",
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

	// Execute uninstall asynchronously (in background goroutine)
	go func() {
		runErr := s.RunUninstallTask(context.Background(), taskID)
		if runErr != nil {
			_ = s.appendTaskLog(context.Background(), taskID, "stderr", runErr.Error())
		}
	}()

	return taskID, nil
}

// RunUninstallTask executes an uninstall task.
func (s *Service) RunUninstallTask(ctx context.Context, taskID uint64) (err error) {
	task, err := s.startUninstallTask(ctx, taskID)
	if err != nil {
		return err
	}
	defer func() {
		s.finishUninstallTask(ctx, task, err)
	}()

	// Load host
	var instance hostpluginmodel.HostPluginInstance
	if err := s.db().WithContext(ctx).First(&instance, task.InstanceID).Error; err != nil {
		return err
	}

	var host hostmodel.Node
	if err := s.db().WithContext(ctx).First(&host, instance.HostID).Error; err != nil {
		return err
	}

	return executeHostPluginUninstallPlan(ctx, s, &host, task)
}

func (s *Service) startUninstallTask(ctx context.Context, taskID uint64) (*hostpluginmodel.HostPluginTask, error) {
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
		result := tx.Model(&hostpluginmodel.HostPluginTask{}).
			Where("id = ? AND status = ?", taskID, installStatusPending).
			Updates(map[string]any{
				"status":     installStatusRunning,
				"started_at": &now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("uninstall task not pending: %d", taskID)
		}
		task.Status = installStatusRunning
		task.StartedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Service) finishUninstallTask(ctx context.Context, task *hostpluginmodel.HostPluginTask, runErr error) {
	db := s.db()
	if db == nil || task == nil {
		return
	}

	now := time.Now().UTC()
	task.FinishedAt = &now

	instanceUpdates := map[string]any{
		"updated_at": now,
	}

	if runErr != nil {
		task.Status = installStatusFailed
		task.ErrorMessage = strings.TrimSpace(runErr.Error())
		instanceUpdates["runtime_status"] = "online" // revert draining
		instanceUpdates["last_error"] = strings.TrimSpace(runErr.Error())
	} else {
		task.Status = installStatusSucceeded
		task.ErrorMessage = ""
		instanceUpdates["install_status"] = "uninstalled"
		instanceUpdates["runtime_status"] = "uninstalled"
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

func (s *Service) runUninstallPlan(ctx context.Context, host *hostmodel.Node, task *hostpluginmodel.HostPluginTask) error {
	privateKey, passphrase, err := s.loadNodePrivateKey(ctx, host)
	if err != nil {
		return err
	}

	password, err := s.resolveNodeSSHPassword(host)
	if err != nil {
		return err
	}
	password = strings.TrimSpace(password)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}

	cli, err := newHostPluginSSHClient(host.SSHUser, password, host.IP, host.Port, privateKey, passphrase)
	if err != nil {
		return err
	}
	defer cli.Close()

	commands := []string{
		"systemctl stop opsagent || true",
		"systemctl disable opsagent || true",
		"rm -rf /etc/opsagent",
		"rm -rf /tmp/opsagent",
		"rm -f /usr/local/bin/opsagent",
		"rm -f /etc/systemd/system/opsagent.service",
		"systemctl daemon-reload || true",
	}

	for _, cmd := range commands {
		if logErr := s.appendTaskLog(ctx, task.ID, "stdout", "$ "+cmd); logErr != nil {
			return logErr
		}
		out, runErr := runHostPluginSSHCommand(cli, cmd)
		if strings.TrimSpace(out) != "" {
			_ = s.appendTaskLog(ctx, task.ID, "stdout", out)
		}
		if runErr != nil {
			_ = s.appendTaskLog(ctx, task.ID, "stderr", runErr.Error())
			// Continue uninstall even if individual commands fail
		}
	}

	// Revoke certificate
	if s.svcCtx != nil && s.svcCtx.DB != nil {
		certStore := &certStoreOp{db: s.svcCtx.DB}
		_ = certStore.RevokeCert(task.InstanceID)
	}

	return nil
}

// certStoreOp is a thin wrapper for cert revocation operations.
// Declared here to keep the logic self-contained and avoid coupling
// to the PKI store package.
type certStoreOp struct {
	db *gorm.DB
}

func (cs *certStoreOp) RevokeCert(instanceID uint64) error {
	result := cs.db.Model(&hostpluginmodel.OpsAgentHostCert{}).
		Where("instance_id = ? AND revoked = ?", instanceID, false).
		Update("revoked", true)
	return result.Error
}
