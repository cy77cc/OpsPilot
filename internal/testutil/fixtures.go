package testutil

import (
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/google/uuid"
)

// UserBuilder provides a fluent interface for creating test users.
type UserBuilder struct {
	user *model.User
}

// NewUserBuilder creates a new UserBuilder with default values.
func NewUserBuilder() *UserBuilder {
	return &UserBuilder{
		user: &model.User{
			Username:     "testuser-" + uuid.New().String()[:8],
			PasswordHash: "$2a$10$testhash", // placeholder hash
			Email:        "test@example.com",
			Phone:        "",
			Avatar:       "",
			Status:       1,
		},
	}
}

// WithUsername sets the username.
func (b *UserBuilder) WithUsername(username string) *UserBuilder {
	b.user.Username = username
	return b
}

// WithEmail sets the email.
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.Email = email
	return b
}

// WithStatus sets the status.
func (b *UserBuilder) WithStatus(status int8) *UserBuilder {
	b.user.Status = status
	return b
}

// Build returns the built user.
func (b *UserBuilder) Build() *model.User {
	return b.user
}

// NodeBuilder provides a fluent interface for creating test nodes.
type NodeBuilder struct {
	node *model.Node
}

// NewNodeBuilder creates a new NodeBuilder with default values.
func NewNodeBuilder() *NodeBuilder {
	return &NodeBuilder{
		node: &model.Node{
			Name:    "test-node-" + uuid.New().String()[:8],
			IP:      "10.0.0.1",
			Port:    22,
			SSHUser: "root",
			Status:  "active",
			OS:      "linux",
			Arch:    "amd64",
		},
	}
}

// WithName sets the node name.
func (b *NodeBuilder) WithName(name string) *NodeBuilder {
	b.node.Name = name
	return b
}

// WithIP sets the IP address.
func (b *NodeBuilder) WithIP(ip string) *NodeBuilder {
	b.node.IP = ip
	return b
}

// WithStatus sets the status.
func (b *NodeBuilder) WithStatus(status string) *NodeBuilder {
	b.node.Status = status
	return b
}

// WithClusterID sets the cluster ID.
func (b *NodeBuilder) WithClusterID(clusterID uint) *NodeBuilder {
	b.node.ClusterID = clusterID
	return b
}

// WithRole sets the node role.
func (b *NodeBuilder) WithRole(role string) *NodeBuilder {
	b.node.Role = role
	return b
}

// Build returns the built node.
func (b *NodeBuilder) Build() *model.Node {
	return b.node
}

// ClusterBuilder provides a fluent interface for creating test clusters.
type ClusterBuilder struct {
	cluster *model.Cluster
}

// NewClusterBuilder creates a new ClusterBuilder with default values.
func NewClusterBuilder() *ClusterBuilder {
	return &ClusterBuilder{
		cluster: &model.Cluster{
			Name:        "test-cluster-" + uuid.New().String()[:8],
			Endpoint:    "https://127.0.0.1:6443",
			Status:      "active",
			Type:        "kubernetes",
			AuthMethod:  "token",
		},
	}
}

// WithName sets the cluster name.
func (b *ClusterBuilder) WithName(name string) *ClusterBuilder {
	b.cluster.Name = name
	return b
}

// WithEndpoint sets the endpoint.
func (b *ClusterBuilder) WithEndpoint(endpoint string) *ClusterBuilder {
	b.cluster.Endpoint = endpoint
	return b
}

// WithStatus sets the status.
func (b *ClusterBuilder) WithStatus(status string) *ClusterBuilder {
	b.cluster.Status = status
	return b
}

// WithType sets the cluster type.
func (b *ClusterBuilder) WithType(clusterType string) *ClusterBuilder {
	b.cluster.Type = clusterType
	return b
}

// Build returns the built cluster.
func (b *ClusterBuilder) Build() *model.Cluster {
	return b.cluster
}

// ServiceBuilder provides a fluent interface for creating test services.
type ServiceBuilder struct {
	svc *model.Service
}

// NewServiceBuilder creates a new ServiceBuilder with default values.
func NewServiceBuilder() *ServiceBuilder {
	return &ServiceBuilder{
		svc: &model.Service{
			Name:        "test-service-" + uuid.New().String()[:8],
			Env:         "staging",
			YamlContent: "services:\n  app:\n    image: nginx:latest",
		},
	}
}

// WithName sets the service name.
func (b *ServiceBuilder) WithName(name string) *ServiceBuilder {
	b.svc.Name = name
	return b
}

// WithEnv sets the environment.
func (b *ServiceBuilder) WithEnv(env string) *ServiceBuilder {
	b.svc.Env = env
	return b
}

// WithYamlContent sets the YAML content.
func (b *ServiceBuilder) WithYamlContent(content string) *ServiceBuilder {
	b.svc.YamlContent = content
	return b
}

// Build returns the built service.
func (b *ServiceBuilder) Build() *model.Service {
	return b.svc
}

// DeploymentTargetBuilder provides a fluent interface for creating test deployment targets.
type DeploymentTargetBuilder struct {
	target *model.DeploymentTarget
}

// NewDeploymentTargetBuilder creates a new DeploymentTargetBuilder with default values.
func NewDeploymentTargetBuilder() *DeploymentTargetBuilder {
	return &DeploymentTargetBuilder{
		target: &model.DeploymentTarget{
			Name:       "test-target-" + uuid.New().String()[:8],
			TargetType: "k8s",
			Env:        "staging",
			Status:     "active",
		},
	}
}

// WithName sets the target name.
func (b *DeploymentTargetBuilder) WithName(name string) *DeploymentTargetBuilder {
	b.target.Name = name
	return b
}

// WithTargetType sets the target type (k8s or compose).
func (b *DeploymentTargetBuilder) WithTargetType(targetType string) *DeploymentTargetBuilder {
	b.target.TargetType = targetType
	return b
}

// WithEnv sets the environment.
func (b *DeploymentTargetBuilder) WithEnv(env string) *DeploymentTargetBuilder {
	b.target.Env = env
	return b
}

// WithClusterID sets the cluster ID.
func (b *DeploymentTargetBuilder) WithClusterID(clusterID uint) *DeploymentTargetBuilder {
	b.target.ClusterID = clusterID
	return b
}

// WithStatus sets the status.
func (b *DeploymentTargetBuilder) WithStatus(status string) *DeploymentTargetBuilder {
	b.target.Status = status
	return b
}

// Build returns the built target.
func (b *DeploymentTargetBuilder) Build() *model.DeploymentTarget {
	return b.target
}

// RoleBuilder provides a fluent interface for creating test roles.
type RoleBuilder struct {
	role *model.Role
}

// NewRoleBuilder creates a new RoleBuilder with default values.
func NewRoleBuilder() *RoleBuilder {
	return &RoleBuilder{
		role: &model.Role{
			Name:   "test-role-" + uuid.New().String()[:8],
			Code:   "test_role_" + uuid.New().String()[:8],
			Status: 1,
		},
	}
}

// WithName sets the role name.
func (b *RoleBuilder) WithName(name string) *RoleBuilder {
	b.role.Name = name
	return b
}

// WithCode sets the role code.
func (b *RoleBuilder) WithCode(code string) *RoleBuilder {
	b.role.Code = code
	return b
}

// Build returns the built role.
func (b *RoleBuilder) Build() *model.Role {
	return b.role
}

// PermissionBuilder provides a fluent interface for creating test permissions.
type PermissionBuilder struct {
	perm *model.Permission
}

// NewPermissionBuilder creates a new PermissionBuilder with default values.
func NewPermissionBuilder() *PermissionBuilder {
	return &PermissionBuilder{
		perm: &model.Permission{
			Name:     "test-permission-" + uuid.New().String()[:8],
			Code:     "test_perm_" + uuid.New().String()[:8],
			Resource: "/api/test",
			Action:   "GET",
			Status:   1,
		},
	}
}

// WithResource sets the resource.
func (b *PermissionBuilder) WithResource(resource string) *PermissionBuilder {
	b.perm.Resource = resource
	return b
}

// WithAction sets the action.
func (b *PermissionBuilder) WithAction(action string) *PermissionBuilder {
	b.perm.Action = action
	return b
}

// WithCode sets the code.
func (b *PermissionBuilder) WithCode(code string) *PermissionBuilder {
	b.perm.Code = code
	return b
}

// Build returns the built permission.
func (b *PermissionBuilder) Build() *model.Permission {
	return b.perm
}

// CICDServiceCIConfigBuilder provides a fluent interface for creating test CICD configs.
type CICDServiceCIConfigBuilder struct {
	config *model.CICDServiceCIConfig
}

// NewCICDServiceCIConfigBuilder creates a new CICDServiceCIConfigBuilder with default values.
func NewCICDServiceCIConfigBuilder() *CICDServiceCIConfigBuilder {
	return &CICDServiceCIConfigBuilder{
		config: &model.CICDServiceCIConfig{
			RepoURL:        "https://github.com/test/repo.git",
			Branch:         "main",
			ArtifactTarget: "test-target",
			TriggerMode:    "manual",
			Status:         "active",
		},
	}
}

// WithRepoURL sets the repository URL.
func (b *CICDServiceCIConfigBuilder) WithRepoURL(url string) *CICDServiceCIConfigBuilder {
	b.config.RepoURL = url
	return b
}

// WithServiceID sets the service ID.
func (b *CICDServiceCIConfigBuilder) WithServiceID(id uint) *CICDServiceCIConfigBuilder {
	b.config.ServiceID = id
	return b
}

// Build returns the built config.
func (b *CICDServiceCIConfigBuilder) Build() *model.CICDServiceCIConfig {
	return b.config
}

// CICDReleaseBuilder provides a fluent interface for creating test CICD releases.
type CICDReleaseBuilder struct {
	release *model.CICDRelease
}

// NewCICDReleaseBuilder creates a new CICDReleaseBuilder with default values.
func NewCICDReleaseBuilder() *CICDReleaseBuilder {
	return &CICDReleaseBuilder{
		release: &model.CICDRelease{
			Env:         "staging",
			RuntimeType: "k8s",
			Version:     "v1.0.0",
			Strategy:    "rolling",
			Status:      "pending_approval",
		},
	}
}

// WithServiceID sets the service ID.
func (b *CICDReleaseBuilder) WithServiceID(id uint) *CICDReleaseBuilder {
	b.release.ServiceID = id
	return b
}

// WithStatus sets the status.
func (b *CICDReleaseBuilder) WithStatus(status string) *CICDReleaseBuilder {
	b.release.Status = status
	return b
}

// Build returns the built release.
func (b *CICDReleaseBuilder) Build() *model.CICDRelease {
	return b.release
}
