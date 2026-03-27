// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件定义审批事件迁移相关的功能开关，用于平滑过渡期控制行为。
package logic

import (
	"os"
	"strconv"
	"strings"
)

// approvalLegacyReplayEnvKey 环境变量键名。
const approvalLegacyReplayEnvKey = "OPS_AI_APPROVAL_LEGACY_REPLAY"

// ApprovalEventMigrationFlags 审批事件迁移功能开关。
//
// 用于控制迁移过渡期的行为，确保向后兼容。
type ApprovalEventMigrationFlags struct {
	// LegacyReplayEnabled 是否启用旧版重放逻辑
	LegacyReplayEnabled bool
}

// DefaultApprovalEventMigrationFlags 返回安全的过渡期默认配置。
func DefaultApprovalEventMigrationFlags() ApprovalEventMigrationFlags {
	return ApprovalEventMigrationFlags{
		LegacyReplayEnabled: false,
	}
}

// NewApprovalEventMigrationFlagsFromEnv loads migration gates from the environment.
func NewApprovalEventMigrationFlagsFromEnv() ApprovalEventMigrationFlags {
	flags := DefaultApprovalEventMigrationFlags()
	if raw, ok := os.LookupEnv(approvalLegacyReplayEnvKey); ok {
		if parsed, err := strconv.ParseBool(strings.TrimSpace(raw)); err == nil {
			flags.LegacyReplayEnabled = parsed
		}
	}
	return flags
}
