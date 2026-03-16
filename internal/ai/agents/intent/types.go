package intent

const (
	IntentTypeQA        = "qa"
	IntentTypeDiagnosis = "diagnosis"

	AssistantTypeQA        = "qa"
	AssistantTypeDiagnosis = "diagnosis"

	RiskLevelLow    = "low"
	RiskLevelMedium = "medium"
)

type Decision struct {
	IntentType    string `json:"intent_type"`
	AssistantType string `json:"assistant_type"`
	RiskLevel     string `json:"risk_level"`
}
