package runtime

type a2uiProjectionState struct {
	totalPlanSteps int
	lastIterations int
}

func mapAgentNameToIntentType(agentName string) string {
	switch agentName {
	case "QAAgent":
		return "qa"
	case "DiagnosisAgent":
		return "diagnosis"
	case "ChangeAgent":
		return "change"
	default:
		return "unknown"
	}
}
