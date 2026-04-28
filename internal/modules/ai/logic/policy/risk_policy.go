// Package policy 实现风险策略匹配逻辑。
package policy

import (
	agentapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

// Match 选择最佳匹配的风险策略。
// Delegates to the shared implementation in agent/shared/approval.
func Match(rules []ai.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (*ai.AIToolRiskPolicy, bool) {
	return agentapproval.MatchRiskPolicy(rules, scene, commandClass, args)
}
