package logic

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	golangssh "golang.org/x/crypto/ssh"
)

var (
	newHostPluginSSHClient  = sshclient.NewSSHClient
	runHostPluginSSHCommand = sshclient.RunCommand
	newHostPluginSFTPClient = sshclient.NewSFTPClient
	uploadHostPluginFile    = sshclient.UploadFile
)

type installPlan struct {
	workDir           string
	localPackagePath  string
	remotePackagePath string
	commands          []string
}

func (s *Service) ResolveVersionForHost(ctx context.Context, pluginKey, version, arch string) (*hostpluginmodel.HostPluginVersion, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var row hostpluginmodel.HostPluginVersion
	err := db.WithContext(ctx).
		Joins("JOIN host_plugins ON host_plugins.id = host_plugin_versions.plugin_id").
		Where("host_plugins.plugin_key = ? AND host_plugin_versions.version = ? AND host_plugin_versions.arch = ?", pluginKey, version, arch).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) RunInstallTask(ctx context.Context, instanceID uint64) (err error) {
	instance, host, version, err := s.loadInstallContext(ctx, instanceID)
	if err != nil {
		return err
	}

	task, err := s.startTask(ctx, instance.ID, "install")
	if err != nil {
		return err
	}
	defer func() {
		s.finishTask(ctx, task, instance.ID, version.Version, err)
	}()

	plan := s.buildInstallPlan(instance, version)
	return s.runInstallPlan(ctx, host, task, plan)
}

func (s *Service) loadInstallContext(ctx context.Context, instanceID uint64) (*hostpluginmodel.HostPluginInstance, *hostmodel.Node, *hostpluginmodel.HostPluginVersion, error) {
	db := s.db()
	if db == nil {
		return nil, nil, nil, errors.New("hostplugin service: db is required")
	}

	var instance hostpluginmodel.HostPluginInstance
	if err := db.WithContext(ctx).First(&instance, instanceID).Error; err != nil {
		return nil, nil, nil, err
	}

	var plugin hostpluginmodel.HostPlugin
	if err := db.WithContext(ctx).First(&plugin, instance.PluginID).Error; err != nil {
		return nil, nil, nil, err
	}

	var host hostmodel.Node
	if err := db.WithContext(ctx).First(&host, instance.HostID).Error; err != nil {
		return nil, nil, nil, err
	}

	version, err := s.ResolveVersionForHost(ctx, plugin.PluginKey, instance.DesiredVersion, strings.TrimSpace(host.Arch))
	if err != nil {
		return nil, nil, nil, err
	}

	return &instance, &host, version, nil
}

func (s *Service) renderInstallCommand(instance *hostpluginmodel.HostPluginInstance, version *hostpluginmodel.HostPluginVersion) string {
	workDir := fmt.Sprintf("/tmp/opspilot/plugins/%d", instance.ID)
	entry := strings.TrimSpace(version.InstallEntry)
	if entry == "" {
		entry = "install.sh"
	}
	if !strings.HasPrefix(entry, "./") && !strings.HasPrefix(entry, "/") {
		entry = "./" + entry
	}
	quotedEntry := shellQuote(entry)
	return fmt.Sprintf("cd %s && chmod +x %s && %s", shellQuote(workDir), quotedEntry, quotedEntry)
}

func (s *Service) buildInstallPlan(instance *hostpluginmodel.HostPluginInstance, version *hostpluginmodel.HostPluginVersion) installPlan {
	workDir := fmt.Sprintf("/tmp/opspilot/plugins/%d", instance.ID)
	localPackagePath := strings.TrimSpace(version.PackagePath)
	remotePackagePath := filepath.ToSlash(filepath.Join(workDir, filepath.Base(localPackagePath)))
	return installPlan{
		workDir:           workDir,
		localPackagePath:  localPackagePath,
		remotePackagePath: remotePackagePath,
		commands: []string{
			fmt.Sprintf("tar xzf %s -C %s", shellQuote(remotePackagePath), shellQuote(workDir)),
			s.renderInstallCommand(instance, version),
		},
	}
}

func (s *Service) runInstallPlan(ctx context.Context, host *hostmodel.Node, task *hostpluginmodel.HostPluginTask, plan installPlan) error {
	if host == nil {
		return errors.New("hostplugin service: host is required")
	}
	if strings.TrimSpace(plan.localPackagePath) == "" {
		return errors.New("hostplugin service: package path is required")
	}

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

	if err := s.runLoggedSSHCommand(ctx, cli, task.ID, fmt.Sprintf("mkdir -p %s", shellQuote(plan.workDir))); err != nil {
		return err
	}
	if err := s.uploadInstallPackage(ctx, cli, task.ID, plan.localPackagePath, plan.remotePackagePath); err != nil {
		return err
	}
	for _, cmd := range plan.commands {
		if err := s.runLoggedSSHCommand(ctx, cli, task.ID, cmd); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) runLoggedSSHCommand(ctx context.Context, cli *golangssh.Client, taskID uint64, cmd string) error {
	if logErr := s.appendTaskLog(ctx, taskID, "stdout", "$ "+cmd); logErr != nil {
		return logErr
	}

	out, runErr := runHostPluginSSHCommand(cli, cmd)
	if strings.TrimSpace(out) != "" {
		if logErr := s.appendTaskLog(ctx, taskID, "stdout", out); logErr != nil {
			return logErr
		}
	}
	if runErr != nil {
		_ = s.appendTaskLog(ctx, taskID, "stderr", runErr.Error())
		return fmt.Errorf("run install command %q: %w", cmd, runErr)
	}
	return nil
}

func (s *Service) uploadInstallPackage(ctx context.Context, cli *golangssh.Client, taskID uint64, localPath, remotePath string) error {
	if logErr := s.appendTaskLog(ctx, taskID, "stdout", fmt.Sprintf("upload %s -> %s", localPath, remotePath)); logErr != nil {
		return logErr
	}

	sftpClient, err := newHostPluginSFTPClient(cli)
	if err != nil {
		_ = s.appendTaskLog(ctx, taskID, "stderr", err.Error())
		return err
	}
	defer sftpClient.Close()

	if err := uploadHostPluginFile(sftpClient, localPath, remotePath); err != nil {
		_ = s.appendTaskLog(ctx, taskID, "stderr", err.Error())
		return fmt.Errorf("upload install package to %s: %w", remotePath, err)
	}
	return nil
}

func (s *Service) runSSHCommands(ctx context.Context, host *hostmodel.Node, task *hostpluginmodel.HostPluginTask, cmds []string) error {
	plan := installPlan{
		workDir:  "",
		commands: cmds,
	}

	if host == nil {
		return errors.New("hostplugin service: host is required")
	}

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

	for _, cmd := range plan.commands {
		if logErr := s.appendTaskLog(ctx, task.ID, "stdout", "$ "+cmd); logErr != nil {
			return logErr
		}

		out, runErr := runHostPluginSSHCommand(cli, cmd)
		if strings.TrimSpace(out) != "" {
			if logErr := s.appendTaskLog(ctx, task.ID, "stdout", out); logErr != nil {
				return logErr
			}
		}
		if runErr != nil {
			_ = s.appendTaskLog(ctx, task.ID, "stderr", runErr.Error())
			return fmt.Errorf("run install command %q: %w", cmd, runErr)
		}
	}

	return nil
}

func (s *Service) loadNodePrivateKey(ctx context.Context, host *hostmodel.Node) (string, string, error) {
	if host == nil || host.SSHKeyID == nil {
		return "", "", nil
	}

	var key hostmodel.SSHKey
	if err := s.db().WithContext(ctx).
		Select("id", "private_key", "passphrase", "encrypted").
		Where("id = ?", uint64(*host.SSHKeyID)).
		First(&key).Error; err != nil {
		return "", "", err
	}

	passphrase := strings.TrimSpace(key.Passphrase)
	if !key.Encrypted {
		return strings.TrimSpace(key.PrivateKey), passphrase, nil
	}

	privateKey, err := utils.DecryptText(strings.TrimSpace(key.PrivateKey), config.CFG.Security.EncryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("decrypt private key: %w", err)
	}
	return privateKey, passphrase, nil
}

func (s *Service) resolveNodeSSHPassword(host *hostmodel.Node) (string, error) {
	if host == nil {
		return "", nil
	}

	password := strings.TrimSpace(host.SSHPassword)
	if password == "" {
		return "", nil
	}

	key := strings.TrimSpace(config.CFG.Security.EncryptionKey)
	if key == "" {
		return "", fmt.Errorf("security.encryption_key is not configured")
	}
	plain, err := utils.DecryptText(password, key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt SSH password: %w", err)
	}
	return plain, nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
