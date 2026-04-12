// Package logic 定义 CI/CD 服务的请求和响应类型别名。
//
// 本文件从 api/cicd/v1 包导入类型别名，简化业务逻辑层的类型引用。
// 类型定义位于 api/cicd/v1 包，确保前后端类型一致性。
package logic

import cicdv1 "github.com/cy77cc/OpsPilot/api/cicd/v1"

// UpsertServiceCIConfigReq 是创建或更新服务 CI 配置的请求类型。
type UpsertServiceCIConfigReq = cicdv1.UpsertServiceCIConfigReq

// TriggerCIRunReq 是触发 CI 构建的请求类型。
type TriggerCIRunReq = cicdv1.TriggerCIRunReq

// UpsertDeploymentCDConfigReq 是创建或更新部署 CD 配置的请求类型。
type UpsertDeploymentCDConfigReq = cicdv1.UpsertDeploymentCDConfigReq

// TriggerReleaseReq 是触发发布的请求类型。
type TriggerReleaseReq = cicdv1.TriggerReleaseReq

// ReleaseDecisionReq 是发布审批决策的请求类型。
type ReleaseDecisionReq = cicdv1.ReleaseDecisionReq

// RollbackReleaseReq 是回滚发布的请求类型。
type RollbackReleaseReq = cicdv1.RollbackReleaseReq
