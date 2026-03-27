package logic

import (
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
)

// EvaluateOperationalEligibility 评估主机是否可参与运维操作。
//
// 检查主机是否满足参与自动化任务、部署等运维操作的条件。
// 不满足条件的主机包括：不存在、缺少 IP、维护中、离线、错误状态。
//
// 参数:
//   - host: 主机对象
//
// 返回:
//   - bool: 是否可参与操作
//   - string: 不可参与的原因（空字符串表示可参与）
func EvaluateOperationalEligibility(host *model.Node) (bool, string) {
	if host == nil {
		return false, "host not found"
	}
	if strings.TrimSpace(host.IP) == "" {
		return false, "host missing ip"
	}
	status := strings.ToLower(strings.TrimSpace(host.Status))
	switch status {
	case "maintenance":
		return false, buildMaintenanceReason(host)
	case "offline", "inactive", "error":
		return false, fmt.Sprintf("host unavailable: %s", status)
	}
	return true, ""
}

// buildMaintenanceReason 构建维护状态原因描述。
//
// 参数:
//   - host: 主机对象
//
// 返回: 维护原因描述字符串
func buildMaintenanceReason(host *model.Node) string {
	reason := strings.TrimSpace(host.MaintenanceReason)
	if reason == "" {
		reason = "maintenance mode"
	}
	if host.MaintenanceUntil != nil && !host.MaintenanceUntil.IsZero() {
		return fmt.Sprintf("%s (until %s)", reason, host.MaintenanceUntil.Format(time.RFC3339))
	}
	return reason
}
