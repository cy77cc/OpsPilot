package testutil

import (
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// IntegrationSuite provides a complete test environment with in-memory database
// and mock services for integration testing.
type IntegrationSuite struct {
	DB      *gorm.DB
	SvcCtx  *svc.ServiceContext
	MockSSH *MockSSHClient
	MockK8s *MockK8sClient
	t       *testing.T
}

// NewIntegrationSuite creates a new integration test suite with SQLite in-memory database.
func NewIntegrationSuite(t *testing.T) *IntegrationSuite {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	// Auto migrate all models
	if err := autoMigrateAll(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	suite := &IntegrationSuite{
		DB:      db,
		MockSSH: NewMockSSHClient(),
		MockK8s: NewMockK8sClient(),
		t:       t,
	}

	// Create minimal ServiceContext
	suite.SvcCtx = &svc.ServiceContext{
		DB: db,
	}

	return suite
}

// autoMigrateAll migrates all models used in the application.
func autoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(
		// User & RBAC
		&model.User{},
		&model.UserRole{},
		&model.Role{},
		&model.RolePermission{},
		&model.Permission{},

		// Infrastructure
		&model.Node{},
		&model.SSHKey{},
		&model.NodeEvent{},
		&model.HostCloudAccount{},
		&model.HostImportTask{},
		&model.HostVirtualizationTask{},
		&model.Cluster{},
		&model.ClusterCredential{},
		&model.ClusterNode{},

		// Deployment
		&model.DeploymentTarget{},
		&model.DeploymentTargetNode{},
		&model.DeploymentRelease{},
		&model.DeploymentReleaseApproval{},
		&model.DeploymentReleaseAudit{},
		&model.ServiceGovernancePolicy{},
		&model.AIOPSInspection{},
		&model.Service{},
		&model.EnvironmentInstallJob{},
		&model.EnvironmentInstallJobStep{},

		// Project
		&model.Project{},

		// CICD
		&model.CICDServiceCIConfig{},
		&model.CICDServiceCIRun{},
		&model.CICDDeploymentCDConfig{},
		&model.CICDRelease{},
		&model.CICDReleaseApproval{},
		&model.CICDAuditEvent{},

		// CMDB
		&model.CMDBCI{},
		&model.CMDBRelation{},
		&model.CMDBSyncJob{},
		&model.CMDBSyncRecord{},
		&model.CMDBAudit{},

		// Notification
		&model.Notification{},
		&model.UserNotification{},
		&model.AlertEvent{},
		&model.AlertRule{},
		&model.AlertNotificationChannel{},
		&model.AlertNotificationDelivery{},

		// AI
		&model.AIChatSession{},
		&model.AIChatMessage{},
		&model.AICommandExecution{},

		// Jobs
		&model.Job{},
		&model.JobExecution{},
		&model.JobLog{},

		// Audit
		&model.AuditLog{},

		// Policy
		&model.Policy{},

		// AIOps
		&model.RiskFinding{},
		&model.Anomaly{},
		&model.Suggestion{},
	)
}

// Cleanup cleans up the test database. Call this in t.Cleanup or defer.
func (s *IntegrationSuite) Cleanup() {
	if s.DB != nil {
		sqlDB, _ := s.DB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}

// SeedUser creates a test user with optional overrides.
func (s *IntegrationSuite) SeedUser(overrides ...func(*model.User)) *model.User {
	s.t.Helper()
	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Email:        "test@example.com",
		Status:       1,
	}
	for _, fn := range overrides {
		fn(user)
	}
	if err := s.DB.Create(user).Error; err != nil {
		s.t.Fatalf("failed to seed user: %v", err)
	}
	return user
}

// SeedNode creates a test node with optional overrides.
func (s *IntegrationSuite) SeedNode(overrides ...func(*model.Node)) *model.Node {
	s.t.Helper()
	node := &model.Node{
		Name:    "test-node",
		IP:      "10.0.0.1",
		SSHUser: "root",
		Status:  "active",
	}
	for _, fn := range overrides {
		fn(node)
	}
	if err := s.DB.Create(node).Error; err != nil {
		s.t.Fatalf("failed to seed node: %v", err)
	}
	return node
}

// SeedCluster creates a test cluster with optional overrides.
func (s *IntegrationSuite) SeedCluster(overrides ...func(*model.Cluster)) *model.Cluster {
	s.t.Helper()
	cluster := &model.Cluster{
		Name:     "test-cluster",
		Endpoint: "https://127.0.0.1:6443",
		Status:   "active",
		Type:     "kubernetes",
	}
	for _, fn := range overrides {
		fn(cluster)
	}
	if err := s.DB.Create(cluster).Error; err != nil {
		s.t.Fatalf("failed to seed cluster: %v", err)
	}
	return cluster
}

// SeedService creates a test service with optional overrides.
func (s *IntegrationSuite) SeedService(overrides ...func(*model.Service)) *model.Service {
	s.t.Helper()
	svc := &model.Service{
		Name:        "test-service",
		Env:         "staging",
		YamlContent: "services:\n  app:\n    image: nginx:latest",
	}
	for _, fn := range overrides {
		fn(svc)
	}
	if err := s.DB.Create(svc).Error; err != nil {
		s.t.Fatalf("failed to seed service: %v", err)
	}
	return svc
}

// SeedDeploymentTarget creates a test deployment target with optional overrides.
func (s *IntegrationSuite) SeedDeploymentTarget(overrides ...func(*model.DeploymentTarget)) *model.DeploymentTarget {
	s.t.Helper()
	target := &model.DeploymentTarget{
		Name:       "test-target",
		TargetType: "k8s",
		Env:        "staging",
		Status:     "active",
	}
	for _, fn := range overrides {
		fn(target)
	}
	if err := s.DB.Create(target).Error; err != nil {
		s.t.Fatalf("failed to seed deployment target: %v", err)
	}
	return target
}

// SeedRole creates a test role with optional overrides.
func (s *IntegrationSuite) SeedRole(overrides ...func(*model.Role)) *model.Role {
	s.t.Helper()
	role := &model.Role{
		Name:   "test-role",
		Code:   "test_role",
		Status: 1,
	}
	for _, fn := range overrides {
		fn(role)
	}
	if err := s.DB.Create(role).Error; err != nil {
		s.t.Fatalf("failed to seed role: %v", err)
	}
	return role
}

// SeedPermission creates a test permission with optional overrides.
func (s *IntegrationSuite) SeedPermission(overrides ...func(*model.Permission)) *model.Permission {
	s.t.Helper()
	perm := &model.Permission{
		Name:     "test-permission",
		Code:     "test_permission",
		Resource: "/api/test",
		Action:   "GET",
		Status:   1,
	}
	for _, fn := range overrides {
		fn(perm)
	}
	if err := s.DB.Create(perm).Error; err != nil {
		s.t.Fatalf("failed to seed permission: %v", err)
	}
	return perm
}

// AssertRecordExists asserts that a record exists in the database.
func (s *IntegrationSuite) AssertRecordExists(model interface{}, conditions ...interface{}) {
	s.t.Helper()
	var count int64
	if err := s.DB.Model(model).Where(conditions[0], conditions[1:]...).Count(&count).Error; err != nil {
		s.t.Fatalf("failed to check record existence: %v", err)
	}
	if count == 0 {
		s.t.Fatalf("expected record to exist, but it was not found")
	}
}

// AssertRecordCount asserts the number of records matching conditions.
func (s *IntegrationSuite) AssertRecordCount(expected int64, model interface{}, conditions ...interface{}) {
	s.t.Helper()
	var count int64
	if err := s.DB.Model(model).Where(conditions[0], conditions[1:]...).Count(&count).Error; err != nil {
		s.t.Fatalf("failed to count records: %v", err)
	}
	if count != expected {
		s.t.Fatalf("expected %d records, got %d", expected, count)
	}
}
