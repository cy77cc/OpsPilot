package model

import (
	applicationmodel "github.com/cy77cc/OpsPilot/internal/modules/application/model"
	deploymentmodel "github.com/cy77cc/OpsPilot/internal/modules/deployment/model"
	projectmodel "github.com/cy77cc/OpsPilot/internal/modules/project/model"
)

type DeploymentTarget = deploymentmodel.DeploymentTarget
type DeploymentRelease = deploymentmodel.DeploymentRelease
type DeploymentReleaseApproval = deploymentmodel.DeploymentReleaseApproval
type Service = projectmodel.Service
type ServiceDeployTarget = applicationmodel.ServiceDeployTarget
