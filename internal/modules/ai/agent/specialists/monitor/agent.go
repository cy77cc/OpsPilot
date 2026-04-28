package monitor

import "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"

const (
	name        = "monitor"
	domain      = "monitoring"
	description = "monitor specialist. Diagnose metric anomalies and produce concise risk-oriented summaries."
)

func Name() string  { return name }
func Scene() string { return domain }
func Instruction() string {
	return `You are the monitor specialist.

Scope:
- only monitor/metric and alert diagnostics
- read-only analysis

Skills:
- Use the 'ops-triage-workflow' skill for structured incident investigation across domains.

Execution rules:
1. Start from active alerts and key metrics for the requested service or target.
2. If metric output is large, reduce it into findings instead of echoing raw payloads.
3. Highlight severity, blast radius, and confidence.
4. Return compact summary with evidence pointers and next diagnostic step.
5. Do not propose or execute write actions directly.`
}

func Spec() (agentName, agentDomain, agentDescription, instruction string, readOnly bool) {
	return name, domain, description, Instruction(), true
}

func ShouldDelegateToIsolationWorker(scope contracts.Scope, pointCount int) bool {
	if pointCount > 500 {
		return true
	}

	switch scope.TimeRange {
	case "6h", "12h", "24h", "48h", "7d":
		return pointCount > 100
	default:
		return false
	}
}
