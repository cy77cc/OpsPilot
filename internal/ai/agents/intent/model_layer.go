package intent

import "context"

type ModelLayer struct{}

func (m ModelLayer) Classify(_ context.Context, _ string) (Decision, error) {
	return Decision{
		IntentType:    IntentTypeQA,
		AssistantType: AssistantTypeQA,
		RiskLevel:     RiskLevelLow,
	}, nil
}
