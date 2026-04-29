package logic

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newHostPluginServiceForTest(t *testing.T) *Service {
	svc, _ := newHostPluginServiceAndDBForTest(t)
	return svc
}

func newHostPluginServiceAndDBForTest(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&hostmodel.Node{},
		&hostmodel.SSHKey{},
		&hostpluginmodel.HostPlugin{},
		&hostpluginmodel.HostPluginVersion{},
		&hostpluginmodel.HostPluginInstance{},
		&hostpluginmodel.HostPluginTask{},
		&hostpluginmodel.HostPluginTaskLog{},
	); err != nil {
		t.Fatalf("auto migrate hostplugin install models: %v", err)
	}

	plugin := hostpluginmodel.HostPlugin{
		PluginKey:      "opsagent",
		Name:           "OpsAgent",
		Category:       "host-observability",
		Description:    "agent runtime",
		DefaultVersion: "nodeagentx-dc57fbc-dirty",
		Status:         "active",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	versions := []hostpluginmodel.HostPluginVersion{
		{
			PluginID:         plugin.ID,
			Version:          "nodeagentx-dc57fbc-dirty",
			Arch:             "amd64",
			PackagePath:      "/tmp/releases/nodeagentx-linux-amd64.tar.gz",
			InstallEntry:     "install.sh",
			UpgradeEntry:     "upgrade.sh",
			UninstallEntry:   "uninstall.sh",
			Checksum:         "sha256-amd64",
			CapabilitiesJSON: "[]",
			ConfigSchemaJSON: "{}",
		},
		{
			PluginID:         plugin.ID,
			Version:          "nodeagentx-dc57fbc-dirty",
			Arch:             "arm64",
			PackagePath:      "/tmp/releases/nodeagentx-linux-arm64.tar.gz",
			InstallEntry:     "install.sh",
			UpgradeEntry:     "upgrade.sh",
			UninstallEntry:   "uninstall.sh",
			Checksum:         "sha256-arm64",
			CapabilitiesJSON: "[]",
			ConfigSchemaJSON: "{}",
		},
	}
	if err := db.Create(&versions).Error; err != nil {
		t.Fatalf("create plugin versions: %v", err)
	}

	return NewService(&svc.ServiceContext{DB: db}), db
}

func TestResolvePackageForHost_SelectsByArchitecture(t *testing.T) {
	svc := newHostPluginServiceForTest(t)
	version, err := svc.ResolveVersionForHost(context.Background(), "opsagent", "nodeagentx-dc57fbc-dirty", "amd64")
	if err != nil {
		t.Fatalf("resolve package: %v", err)
	}
	if !strings.Contains(version.PackagePath, "linux-amd64.tar.gz") {
		t.Fatalf("expected amd64 package path, got %s", version.PackagePath)
	}
}

func TestResolvePackageForHost_NormalizesArchitectureAliases(t *testing.T) {
	svc := newHostPluginServiceForTest(t)

	cases := []struct {
		arch        string
		wantPackage string
	}{
		{arch: "x86_64", wantPackage: "linux-amd64.tar.gz"},
		{arch: "amd64", wantPackage: "linux-amd64.tar.gz"},
		{arch: "aarch64", wantPackage: "linux-arm64.tar.gz"},
		{arch: "arm64", wantPackage: "linux-arm64.tar.gz"},
	}

	for _, tc := range cases {
		version, err := svc.ResolveVersionForHost(context.Background(), "opsagent", "nodeagentx-dc57fbc-dirty", tc.arch)
		if err != nil {
			t.Fatalf("resolve package for arch %s: %v", tc.arch, err)
		}
		if !strings.Contains(version.PackagePath, tc.wantPackage) {
			t.Fatalf("expected %s for arch %s, got %s", tc.wantPackage, tc.arch, version.PackagePath)
		}
	}
}

func TestBuildInstallPlan_UsesRemoteTarballPath(t *testing.T) {
	svc := newHostPluginServiceForTest(t)
	instance := &hostpluginmodel.HostPluginInstance{ID: 42}
	version := &hostpluginmodel.HostPluginVersion{
		PackagePath:  "/controller/releases/nodeagentx-linux-amd64.tar.gz",
		InstallEntry: "install.sh",
	}

	plan := svc.buildInstallPlan(instance, version)

	if plan.remotePackagePath != "/tmp/opspilot/plugins/42/nodeagentx-linux-amd64.tar.gz" {
		t.Fatalf("unexpected remote package path: %s", plan.remotePackagePath)
	}
	if got := filepath.Base(plan.localPackagePath); got != "nodeagentx-linux-amd64.tar.gz" {
		t.Fatalf("expected local package basename to be preserved, got %s", got)
	}
	if len(plan.commands) != 2 {
		t.Fatalf("expected 2 post-upload commands, got %d", len(plan.commands))
	}
	if strings.Contains(plan.commands[0], "/controller/releases/nodeagentx-linux-amd64.tar.gz") {
		t.Fatalf("untar command should not reference controller-local path: %s", plan.commands[0])
	}
	if !strings.Contains(plan.commands[0], plan.remotePackagePath) {
		t.Fatalf("untar command should reference remote package path: %s", plan.commands[0])
	}
	if !strings.Contains(plan.commands[1], "cd '/tmp/opspilot/plugins/42'") {
		t.Fatalf("install command should run inside remote work dir: %s", plan.commands[1])
	}
}

func TestEnqueueInstallTask_PersistsPendingTaskBeforeExecution(t *testing.T) {
	svc, db := newHostPluginServiceAndDBForTest(t)
	instanceID := seedInstallInstanceForTest(t, db, "amd64", "nodeagentx-dc57fbc-dirty")

	task, err := svc.EnqueueInstallTask(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("enqueue install task: %v", err)
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending task status, got %s", task.Status)
	}

	var storedTask hostpluginmodel.HostPluginTask
	if err := db.First(&storedTask, task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != "pending" {
		t.Fatalf("expected persisted task status pending, got %s", storedTask.Status)
	}
	if storedTask.StartedAt != nil {
		t.Fatalf("expected queued task to have nil started_at")
	}

	var storedInstance hostpluginmodel.HostPluginInstance
	if err := db.First(&storedInstance, instanceID).Error; err != nil {
		t.Fatalf("load instance: %v", err)
	}
	if storedInstance.InstallStatus != "pending" {
		t.Fatalf("expected instance to remain pending before execution, got %s", storedInstance.InstallStatus)
	}
}

func TestRunInstallTask_PreResolutionFailureMarksTaskAndInstanceFailed(t *testing.T) {
	svc, db := newHostPluginServiceAndDBForTest(t)
	instanceID := seedInstallInstanceForTest(t, db, "", "nodeagentx-dc57fbc-dirty")

	task, err := svc.EnqueueInstallTask(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("enqueue install task: %v", err)
	}

	runErr := svc.RunInstallTask(context.Background(), task.ID)
	if runErr == nil {
		t.Fatalf("expected run install task to fail without host arch")
	}

	var storedTask hostpluginmodel.HostPluginTask
	if err := db.First(&storedTask, task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != "failed" {
		t.Fatalf("expected task status failed, got %s", storedTask.Status)
	}
	if strings.TrimSpace(storedTask.ErrorMessage) == "" {
		t.Fatalf("expected task error message")
	}
	if storedTask.StartedAt == nil || storedTask.FinishedAt == nil {
		t.Fatalf("expected task to record started_at and finished_at")
	}

	var storedInstance hostpluginmodel.HostPluginInstance
	if err := db.First(&storedInstance, instanceID).Error; err != nil {
		t.Fatalf("load instance: %v", err)
	}
	if storedInstance.InstallStatus != "failed" {
		t.Fatalf("expected instance status failed, got %s", storedInstance.InstallStatus)
	}
	if strings.TrimSpace(storedInstance.LastError) == "" {
		t.Fatalf("expected instance last_error to be populated")
	}
}

func TestTaskStatusTransitions_UseContractValues(t *testing.T) {
	svc, db := newHostPluginServiceAndDBForTest(t)
	instanceID := seedInstallInstanceForTest(t, db, "amd64", "nodeagentx-dc57fbc-dirty")

	task, err := svc.EnqueueInstallTask(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("enqueue install task: %v", err)
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending status, got %s", task.Status)
	}

	task, err = svc.startTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if task.Status != "running" {
		t.Fatalf("expected running status, got %s", task.Status)
	}

	svc.finishTask(context.Background(), task, "nodeagentx-dc57fbc-dirty", nil)

	var storedTask hostpluginmodel.HostPluginTask
	if err := db.First(&storedTask, task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != "succeeded" {
		t.Fatalf("expected task status succeeded, got %s", storedTask.Status)
	}

	var storedInstance hostpluginmodel.HostPluginInstance
	if err := db.First(&storedInstance, instanceID).Error; err != nil {
		t.Fatalf("load instance: %v", err)
	}
	if storedInstance.InstallStatus != "succeeded" {
		t.Fatalf("expected instance status succeeded, got %s", storedInstance.InstallStatus)
	}
	if storedInstance.InstalledVersion != "nodeagentx-dc57fbc-dirty" {
		t.Fatalf("expected installed version to be persisted, got %s", storedInstance.InstalledVersion)
	}
}

func TestStartTask_RejectsAlreadyClaimedTask(t *testing.T) {
	svc, db := newHostPluginServiceAndDBForTest(t)
	instanceID := seedInstallInstanceForTest(t, db, "amd64", "nodeagentx-dc57fbc-dirty")

	task, err := svc.EnqueueInstallTask(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("enqueue install task: %v", err)
	}
	if _, err := svc.startTask(context.Background(), task.ID); err != nil {
		t.Fatalf("first claim should succeed: %v", err)
	}
	if _, err := svc.startTask(context.Background(), task.ID); !errors.Is(err, errInstallTaskNotPending) {
		t.Fatalf("expected errInstallTaskNotPending on duplicate claim, got %v", err)
	}
}

func seedInstallInstanceForTest(t *testing.T, db *gorm.DB, arch, desiredVersion string) uint64 {
	t.Helper()

	host := hostmodel.Node{
		Name:    "host-install-test",
		IP:      "10.0.0.10",
		Port:    22,
		SSHUser: "root",
		Arch:    arch,
		Status:  "online",
		Source:  "manual_ssh",
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("create host: %v", err)
	}

	var plugin hostpluginmodel.HostPlugin
	if err := db.Where("plugin_key = ?", "opsagent").First(&plugin).Error; err != nil {
		t.Fatalf("load plugin: %v", err)
	}

	instance := hostpluginmodel.HostPluginInstance{
		HostID:           uint64(host.ID),
		PluginID:         plugin.ID,
		DesiredVersion:   desiredVersion,
		InstallStatus:    "pending",
		RuntimeStatus:    "pending_online",
		HealthStatus:     "unknown",
		CapabilitiesJSON: "[]",
		LastError:        "",
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	return instance.ID
}

func TestRunInstallTask_CallsExecutorAfterQueueClaim(t *testing.T) {
	svc, db := newHostPluginServiceAndDBForTest(t)
	instanceID := seedInstallInstanceForTest(t, db, "amd64", "nodeagentx-dc57fbc-dirty")
	task, err := svc.EnqueueInstallTask(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("enqueue install task: %v", err)
	}

	originalExecutor := executeHostPluginInstallPlan
	defer func() { executeHostPluginInstallPlan = originalExecutor }()
	executeHostPluginInstallPlan = func(ctx context.Context, s *Service, host *hostmodel.Node, task *hostpluginmodel.HostPluginTask, plan installPlan) error {
		var reloadedTask hostpluginmodel.HostPluginTask
		if err := db.First(&reloadedTask, task.ID).Error; err != nil {
			t.Fatalf("reload task in executor: %v", err)
		}
		if reloadedTask.Status != "running" {
			t.Fatalf("expected task to be running before executor, got %s", reloadedTask.Status)
		}
		var reloadedInstance hostpluginmodel.HostPluginInstance
		if err := db.First(&reloadedInstance, instanceID).Error; err != nil {
			t.Fatalf("reload instance in executor: %v", err)
		}
		if reloadedInstance.InstallStatus != "running" {
			t.Fatalf("expected instance to be running before executor, got %s", reloadedInstance.InstallStatus)
		}
		return errors.New("executor stop")
	}

	if err := svc.RunInstallTask(context.Background(), task.ID); err == nil {
		t.Fatalf("expected executor stop error")
	}
}

func TestRunPendingInstallTasksOnce_ClaimsQueuedTask(t *testing.T) {
	svc, db := newHostPluginServiceAndDBForTest(t)
	instanceID := seedInstallInstanceForTest(t, db, "x86_64", "nodeagentx-dc57fbc-dirty")
	task, err := svc.EnqueueInstallTask(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("enqueue install task: %v", err)
	}

	originalExecutor := executeHostPluginInstallPlan
	defer func() { executeHostPluginInstallPlan = originalExecutor }()
	executeHostPluginInstallPlan = func(ctx context.Context, s *Service, host *hostmodel.Node, task *hostpluginmodel.HostPluginTask, plan installPlan) error {
		return errors.New("runner stop")
	}

	claimed, err := svc.RunPendingInstallTasksOnce(context.Background())
	if !claimed {
		t.Fatalf("expected runner to claim pending task")
	}
	if err == nil || !strings.Contains(err.Error(), "runner stop") {
		t.Fatalf("expected runner stop error, got %v", err)
	}

	var storedTask hostpluginmodel.HostPluginTask
	if err := db.First(&storedTask, task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != "failed" {
		t.Fatalf("expected failed task status after runner error, got %s", storedTask.Status)
	}
}
