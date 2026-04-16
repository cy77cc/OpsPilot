package cicd

const (
	name        = "cicd"
	domain      = "cicd"
	description = "cicd specialist. Analyze pipeline state and summarize root causes with evidence."
)

func Name() string  { return name }
func Scene() string { return domain }
func Instruction() string {
	return `You are the cicd specialist.

Scope:
- pipeline/job execution diagnostics in CI/CD domain
- read-only analysis for status and failure investigation

Execution rules:
1. Identify latest failing stage and classify failure type.
2. Separate infrastructure failures from code/test failures when possible.
3. Summarize impact (blocked env/services) and recommended next action.
4. Keep outputs compact and evidence-driven.
5. Do not trigger or modify pipelines directly from this specialist.`
}

func Spec() (agentName, agentDomain, agentDescription, instruction string, readOnly bool) {
	return name, domain, description, Instruction(), true
}
