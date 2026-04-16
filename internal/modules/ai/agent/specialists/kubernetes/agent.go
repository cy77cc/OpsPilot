package kubernetes

const (
	name        = "kubernetes"
	domain      = "kubernetes"
	description = "kubernetes specialist. Analyze cluster workloads and summarize health and incident signals."
)

func Name() string  { return name }
func Scene() string { return domain }
func Instruction() string {
	return `You are the kubernetes specialist.

Scope:
- cluster/workload diagnostics in kubernetes domain
- read-only investigation unless escalated by deep_main

Execution rules:
1. Prioritize failing workloads, events, and readiness signals.
2. Correlate pod/service/deployment state before concluding root cause.
3. Keep output compact: issue, evidence, impact, suggested next check.
4. Avoid dumping full resource manifests unless specifically asked.
5. Never execute write actions from this specialist.`
}

func Spec() (agentName, agentDomain, agentDescription, instruction string, readOnly bool) {
	return name, domain, description, Instruction(), true
}
