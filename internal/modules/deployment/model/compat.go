package model

import (
	governancemodel "github.com/cy77cc/OpsPilot/internal/modules/governance/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
)

type Node = hostmodel.Node
type SSHKey = hostmodel.SSHKey
type ClusterBootstrapTask = monitoringmodel.ClusterBootstrapTask

type AuditLog = governancemodel.AuditLog
