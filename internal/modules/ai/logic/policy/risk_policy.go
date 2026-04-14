// Package policy 实现风险策略匹配逻辑。
package policy

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"

	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

// Match 选择最佳匹配的风险策略。
func Match(rules []ai.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (*ai.AIToolRiskPolicy, bool) {
	var (
		best         *ai.AIToolRiskPolicy
		bestScore    int
		bestPriority int
	)
	for i := range rules {
		rule := &rules[i]
		score, ok := matchScore(rule, scene, commandClass, args)
		if !ok {
			continue
		}
		if best == nil || rule.Priority > bestPriority || (rule.Priority == bestPriority && score > bestScore) {
			best = rule
			bestScore = score
			bestPriority = rule.Priority
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

func matchScore(rule *ai.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (int, bool) {
	if rule == nil {
		return 0, false
	}
	if rule.Scene != nil && strings.TrimSpace(*rule.Scene) != "" && !strings.EqualFold(strings.TrimSpace(*rule.Scene), strings.TrimSpace(scene)) {
		return 0, false
	}
	if rule.CommandClass != nil && strings.TrimSpace(*rule.CommandClass) != "" && !strings.EqualFold(strings.TrimSpace(*rule.CommandClass), strings.TrimSpace(commandClass)) {
		return 0, false
	}
	if !matchesArgRules(rule.ArgumentRulesJSON, args) {
		return 0, false
	}
	score := 0
	if hasArgRules(rule.ArgumentRulesJSON) {
		score += 4
	}
	if rule.CommandClass != nil && strings.TrimSpace(*rule.CommandClass) != "" {
		score += 2
	}
	if rule.Scene != nil && strings.TrimSpace(*rule.Scene) != "" {
		score += 1
	}
	return score, true
}

func hasArgRules(raw *string) bool {
	if raw == nil {
		return false
	}
	return strings.TrimSpace(*raw) != "" && strings.TrimSpace(*raw) != "{}"
}

func matchesArgRules(raw *string, args map[string]any) bool {
	if !hasArgRules(raw) {
		return true
	}
	if len(args) == 0 {
		return false
	}
	var ruleMap map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(*raw)), &ruleMap); err != nil {
		return false
	}
	if len(ruleMap) == 0 {
		return true
	}
	for key, expected := range ruleMap {
		actual, ok := args[key]
		if !ok {
			return false
		}
		if !argValueMatches(expected, actual) {
			return false
		}
	}
	return true
}

func argValueMatches(expected, actual any) bool {
	if expected == nil {
		return actual == nil
	}
	if expectedMap, ok := expected.(map[string]any); ok {
		if regexPattern, ok := expectedMap["regex"].(string); ok && regexPattern != "" {
			actualString, ok := actual.(string)
			if !ok {
				return false
			}
			re, err := regexp.Compile(regexPattern)
			if err != nil {
				return false
			}
			return re.MatchString(actualString)
		}
	}
	if reflect.DeepEqual(expected, actual) {
		return true
	}
	return numericEqual(expected, actual)
}

func numericEqual(expected, actual any) bool {
	e, ok1 := toFloat64(expected)
	a, ok2 := toFloat64(actual)
	if !ok1 || !ok2 {
		return false
	}
	return e == a
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(v).Convert(reflect.TypeOf(float64(0))).Float()), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}
