package model

import (
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	deploymentmodel "github.com/cy77cc/OpsPilot/internal/modules/deployment/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	projectmodel "github.com/cy77cc/OpsPilot/internal/modules/project/model"
)

type Cluster = clustermodel.Cluster
type Node = hostmodel.Node
type SSHKey = hostmodel.SSHKey
type Service = projectmodel.Service
type ServiceHelmRelease = projectmodel.ServiceHelmRelease
type DeploymentTarget = deploymentmodel.DeploymentTarget
type DeploymentTargetNode = deploymentmodel.DeploymentTargetNode
type DeploymentRelease = deploymentmodel.DeploymentRelease
