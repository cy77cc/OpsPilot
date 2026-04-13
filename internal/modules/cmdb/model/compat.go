package model

import (
	applicationmodel "github.com/cy77cc/OpsPilot/internal/modules/application/model"
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	deploymentmodel "github.com/cy77cc/OpsPilot/internal/modules/deployment/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	projectmodel "github.com/cy77cc/OpsPilot/internal/modules/project/model"
)

type Node = hostmodel.Node
type Cluster = clustermodel.Cluster
type Service = projectmodel.Service
type DeploymentTarget = deploymentmodel.DeploymentTarget
type ServiceDeployTarget = applicationmodel.ServiceDeployTarget
