package logic

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
)

// MatchRiskPolicy selects the best matching policy using specificity precedence.
func MatchRiskPolicy(rules []model.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (*model.AIToolRiskPolicy, bool) {
	var (
		best         *model.AIToolRiskPolicy
		bestScore    int
		bestPriority int
	)

	for i := range rules {
		rule := &rules[i]
		score, ok := matchRiskPolicyScore(rule, scene, commandClass, args)
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

func matchRiskPolicyScore(rule *model.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (int, bool) {
	if rule == nil {
		return 0, false
	}

	if rule.Scene != nil && strings.TrimSpace(*rule.Scene) != "" && !strings.EqualFold(strings.TrimSpace(*rule.Scene), strings.TrimSpace(scene)) {
		return 0, false
	}
	if rule.CommandClass != nil && strings.TrimSpace(*rule.CommandClass) != "" && !strings.EqualFold(strings.TrimSpace(*rule.CommandClass), strings.TrimSpace(commandClass)) {
		return 0, false
	}
	if !matchesArgumentRules(rule.ArgumentRulesJSON, args) {
		return 0, false
	}

	score := 0
	if hasArgumentRules(rule.ArgumentRulesJSON) {
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

func hasArgumentRules(raw *string) bool {
	if raw == nil {
		return false
	}
	return strings.TrimSpace(*raw) != "" && strings.TrimSpace(*raw) != "{}"
}

func matchesArgumentRules(raw *string, args map[string]any) bool {
	if !hasArgumentRules(raw) {
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
		if !argumentValueMatches(expected, actual) {
			return false
		}
	}
	return true
}

func argumentValueMatches(expected, actual any) bool {
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
	expectedFloat, expectedOK := toFloat64(expected)
	actualFloat, actualOK := toFloat64(actual)
	if !expectedOK || !actualOK {
		return false
	}
	return expectedFloat == actualFloat
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}
