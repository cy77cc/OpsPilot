package migration

import (
	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	applicationmodel "github.com/cy77cc/OpsPilot/internal/modules/application/model"
	automationmodel "github.com/cy77cc/OpsPilot/internal/modules/automation/model"
	cicdmodel "github.com/cy77cc/OpsPilot/internal/modules/cicd/model"
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	cmdbmodel "github.com/cy77cc/OpsPilot/internal/modules/cmdb/model"
	dashboardmodel "github.com/cy77cc/OpsPilot/internal/modules/dashboard/model"
	deploymentmodel "github.com/cy77cc/OpsPilot/internal/modules/deployment/model"
	governancemodel "github.com/cy77cc/OpsPilot/internal/modules/governance/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	jobsmodel "github.com/cy77cc/OpsPilot/internal/modules/jobs/model"
	llmprovidermodel "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	notificationmodel "github.com/cy77cc/OpsPilot/internal/modules/notification/model"
	projectmodel "github.com/cy77cc/OpsPilot/internal/modules/project/model"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"gorm.io/gorm"
)

// RunDevAutoMigrate is only for local development convenience.
func RunDevAutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&usermodel.User{},
		&usermodel.Role{},
		&usermodel.Permission{},
		&usermodel.UserRole{},
		&usermodel.RolePermission{},
		&usermodel.Department{},
		&usermodel.DepartmentMember{},
		&usermodel.DepartmentRole{},
		&hostmodel.Node{},
		&hostmodel.NodeEvent{},
		&hostmodel.SSHKey{},
		&hostmodel.HostCloudAccount{},
		&hostmodel.HostImportTask{},
		&hostmodel.HostVirtualizationTask{},
		&hostmodel.HostProbeSession{},
		&hostmodel.TrustedHostKey{},
		&hostmodel.HostHealthSnapshot{},
		&projectmodel.Project{},
		&projectmodel.Service{},
		&projectmodel.ServiceHelmRelease{},
		&projectmodel.ServiceRenderSnapshot{},
		&applicationmodel.ServiceRevision{},
		&applicationmodel.ServiceVariableSet{},
		&applicationmodel.ServiceDeployTarget{},
		&applicationmodel.ServiceReleaseRecord{},
		&deploymentmodel.DeploymentTarget{},
		&deploymentmodel.DeploymentTargetNode{},
		&deploymentmodel.DeploymentRelease{},
		&deploymentmodel.DeploymentReleaseApproval{},
		&deploymentmodel.DeploymentReleaseAudit{},
		&deploymentmodel.ServiceGovernancePolicy{},
		&deploymentmodel.AIOPSInspection{},
		&monitoringmodel.AlertEvent{},
		&monitoringmodel.AlertRule{},
		&monitoringmodel.AlertNotificationChannel{},
		&monitoringmodel.AlertRuleChannelBinding{},
		&monitoringmodel.AlertSeverityRoute{},
		&monitoringmodel.AlertNotificationDelivery{},
		&monitoringmodel.AlertSilence{},
		&monitoringmodel.ClusterBootstrapTask{},
		&clustermodel.Cluster{},
		&clustermodel.ClusterBootstrapProfile{},
		&clustermodel.ClusterNode{},
		&deploymentmodel.ClusterCredential{},
		&clustermodel.ClusterNamespaceBinding{},
		&clustermodel.ClusterReleaseRecord{},
		&clustermodel.ClusterHPAPolicy{},
		&clustermodel.ClusterQuotaPolicy{},
		&clustermodel.ClusterDeployApproval{},
		&governancemodel.AuditLog{},
		&dashboardmodel.ClusterResourceSnapshot{},
		&dashboardmodel.K8sWorkloadStats{},
		&dashboardmodel.K8sIssuePod{},
		&deploymentmodel.EnvironmentInstallJob{},
		&deploymentmodel.EnvironmentInstallJobStep{},
		&governancemodel.AuditLog{},
		&governancemodel.AuditLog{},
		&cmdbmodel.CMDBCI{},
		&cmdbmodel.CMDBRelation{},
		&cmdbmodel.CMDBSyncJob{},
		&cmdbmodel.CMDBSyncRecord{},
		&cmdbmodel.CMDBAudit{},
		&cicdmodel.CICDServiceCIConfig{},
		&cicdmodel.CICDServiceCIRun{},
		&cicdmodel.CICDDeploymentCDConfig{},
		&cicdmodel.CICDRelease{},
		&cicdmodel.CICDReleaseApproval{},
		&cicdmodel.CICDAuditEvent{},
		&automationmodel.AutomationInventory{},
		&automationmodel.AutomationPlaybook{},
		&automationmodel.AutomationRun{},
		&automationmodel.AutomationRunLog{},
		&automationmodel.AutomationExecutionAudit{},
		&automationmodel.TopologyAccessAudit{},
		&jobsmodel.Job{},
		&jobsmodel.JobExecution{},
		&jobsmodel.JobLog{},
		&notificationmodel.Notification{},
		&notificationmodel.UserNotification{},
		&aimodel.AIChatSession{},
		&aimodel.AIChatMessage{},
		&aimodel.AIRun{},
		&aimodel.AIRunEvent{},
		&aimodel.AIRunProjection{},
		&aimodel.AIRunContent{},
		&aimodel.AIDiagnosisReport{},
		&aimodel.AIScenePrompt{},
		&aimodel.AISceneConfig{},
		&aimodel.AITraceSpan{},
		&aimodel.AIUsageLog{},
		&aimodel.AICheckpoint{},
		&aimodel.AIToolRiskPolicy{},
		&aimodel.AIApprovalTask{},
		&aimodel.AIApprovalOutboxEvent{},
		&aimodel.AIAlertIngestEvent{},
		&aimodel.AIAlertHealJob{},
		&aimodel.AIAlertHealAttempt{},
		&llmprovidermodel.AILLMProvider{},
		&hostmodel.AIHostExecutionRecord{},
		&clustermodel.AdmissionPolicy{},
		&clustermodel.AdmissionExemption{},
		&clustermodel.RuntimeSecurityEvent{},
		&governancemodel.AuditLog{},
		&deploymentmodel.Policy{},
	)
}
