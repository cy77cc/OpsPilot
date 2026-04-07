// Package governance 提供治理服务的公共类型和接口。
//
// 本文件重新导出 handler 和 logic 子包的类型，保持向后兼容性。
// 外部包可以继续从 governance 包导入类型。
//
// 目录结构:
//   - handler/  - HTTP 处理层（Service 接口、错误类型）
//   - logic/    - 业务逻辑层（领域类型）
//   - approval/, audit/, envelope/, policy/ - 子模块
//   - routes.go - 路由入口（待创建）
package governance

import (
	"github.com/cy77cc/OpsPilot/internal/service/governance/handler"
	"github.com/cy77cc/OpsPilot/internal/service/governance/logic"
)

// === logic 包类型重新导出 ===

type RiskLevel = logic.RiskLevel

const (
	RiskLow      RiskLevel = logic.RiskLow
	RiskMedium   RiskLevel = logic.RiskMedium
	RiskHigh     RiskLevel = logic.RiskHigh
	RiskCritical RiskLevel = logic.RiskCritical
)

type OperationState = logic.OperationState

const (
	StateCompleted        OperationState = logic.StateCompleted
	StateApprovalRequired OperationState = logic.StateApprovalRequired
	StateRejected         OperationState = logic.StateRejected
	StateFailed           OperationState = logic.StateFailed
)

const (
	TargetScopeCluster = logic.TargetScopeCluster
	TargetScopeProject = logic.TargetScopeProject
	TargetScopeTeam    = logic.TargetScopeTeam
	TargetScopeGlobal  = logic.TargetScopeGlobal
)

type Scope = logic.Scope
type OperationContext = logic.OperationContext
type OperationIntent = logic.OperationIntent
type ApprovalInfo = logic.ApprovalInfo
type Decision = logic.Decision
type FinalizeInput = logic.FinalizeInput
type FinalizeOutput = logic.FinalizeOutput
type Envelope = logic.Envelope
type Policy = logic.Policy

// logic 包函数重新导出

var (
	WithOperationContext    = logic.WithOperationContext
	OperationContextFromContext = logic.OperationContextFromContext
	MergeScopeFromContext   = logic.MergeScopeFromContext
)

// === handler 包类型和函数重新导出 ===

const (
	CodeSuccess               = handler.CodeSuccess
	CodeApprovalRequired      = handler.CodeApprovalRequired
	CodeApprovalRejected      = handler.CodeApprovalRejected
	CodeApprovalTokenInvalid  = handler.CodeApprovalTokenInvalid
	CodeApprovalTokenExpired  = handler.CodeApprovalTokenExpired
	CodeApprovalTokenReplay   = handler.CodeApprovalTokenReplay
	CodeApprovalScopeMismatch = handler.CodeApprovalScopeMismatch
	CodeApprovalNotApproved   = handler.CodeApprovalNotApproved
	CodePermissionDenied      = handler.CodePermissionDenied
	CodePolicyNotFound        = handler.CodePolicyNotFound
	CodeInternalError         = handler.CodeInternalError
)

type GovError = handler.GovError
type Service = handler.Service
type PolicyResolver = handler.PolicyResolver
type ApprovalService = handler.ApprovalService
type AuditService = handler.AuditService
type Redactor = handler.Redactor

var (
	NewGovError    = handler.NewGovError
	IsGovError     = handler.IsGovError
	NewService     = handler.NewService
	BuildEnvelope  = handler.BuildEnvelope
	IsCode         = handler.IsCode
)