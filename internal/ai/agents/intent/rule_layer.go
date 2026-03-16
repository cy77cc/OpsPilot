package intent

import "strings"

type RuleLayer struct{}

func (r RuleLayer) Classify(message string) (Decision, bool) {
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		return Decision{}, false
	}

	diagnosisHints := []string{
		"diagnose",
		"diagnosis",
		"root cause",
		"investigate",
		"why is",
		"why are",
		"failing",
		"incident",
		"crashloop",
		"quota",
	}
	for _, hint := range diagnosisHints {
		if strings.Contains(text, hint) {
			return Decision{
				IntentType:    IntentTypeDiagnosis,
				AssistantType: AssistantTypeDiagnosis,
				RiskLevel:     RiskLevelMedium,
			}, true
		}
	}
	return Decision{}, false
}
