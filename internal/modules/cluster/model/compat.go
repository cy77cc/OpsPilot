package model

import (
	deploymentmodel "github.com/cy77cc/OpsPilot/internal/modules/deployment/model"
	governancemodel "github.com/cy77cc/OpsPilot/internal/modules/governance/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	projectmodel "github.com/cy77cc/OpsPilot/internal/modules/project/model"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
)

type Node = hostmodel.Node
type SSHKey = hostmodel.SSHKey

type ClusterBootstrapTask = monitoringmodel.ClusterBootstrapTask
type ClusterCredential = deploymentmodel.ClusterCredential
type DeploymentRelease = deploymentmodel.DeploymentRelease
type DeploymentTarget = deploymentmodel.DeploymentTarget

type OperationApproval = governancemodel.AuditLog
type OperationAudit = governancemodel.AuditLog
type ClusterOperationAuditRecord = governancemodel.AuditLog

type Project = projectmodel.Project
type Service = projectmodel.Service

type User = usermodel.User
type Role = usermodel.Role
type Permission = usermodel.Permission
type UserRole = usermodel.UserRole
type RolePermission = usermodel.RolePermission
