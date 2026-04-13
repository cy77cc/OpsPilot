package model

import (
	applicationmodel "github.com/cy77cc/OpsPilot/internal/modules/application/model"
	cicdmodel "github.com/cy77cc/OpsPilot/internal/modules/cicd/model"
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	deploymentmodel "github.com/cy77cc/OpsPilot/internal/modules/deployment/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	projectmodel "github.com/cy77cc/OpsPilot/internal/modules/project/model"
)

type Node = hostmodel.Node
type NodeEvent = hostmodel.NodeEvent
type Cluster = clustermodel.Cluster
type Service = projectmodel.Service
type ServiceReleaseRecord = applicationmodel.ServiceReleaseRecord
type DeploymentRelease = deploymentmodel.DeploymentRelease
type AlertEvent = monitoringmodel.AlertEvent
type CICDServiceCIRun = cicdmodel.CICDServiceCIRun
