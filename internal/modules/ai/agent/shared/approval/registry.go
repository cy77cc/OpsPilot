package approval

import (
	"sort"
	"strings"
)

// RiskPolicy describes the code-default approval behavior for a tool.
type RiskPolicy struct {
	NeedsApproval             bool
	ConditionalByCommandClass bool
}

// RiskRegistry stores the default approval policy for tools.
type RiskRegistry struct {
	policies map[string]RiskPolicy
}

var defaultRiskRegistry = NewRiskRegistry(map[string]RiskPolicy{
	"host_batch":               {NeedsApproval: true},
	"host_batch_exec_apply":    {NeedsApproval: true},
	"host_batch_status_update": {NeedsApproval: true},

	"host_exec":               {NeedsApproval: true, ConditionalByCommandClass: true},
	"host_batch_exec_preview": {NeedsApproval: true, ConditionalByCommandClass: true},

	"k8s_scale_deployment":    {NeedsApproval: true},
	"k8s_restart_deployment":  {NeedsApproval: true},
	"k8s_delete_pod":          {NeedsApproval: true},
	"k8s_rollback_deployment": {NeedsApproval: true},
	"k8s_delete_deployment":   {NeedsApproval: true},

	"cicd_trigger_pipeline": {NeedsApproval: true},
	"cicd_cancel_pipeline":  {NeedsApproval: true},

	"service_restart":       {NeedsApproval: true},
	"service_scale":         {NeedsApproval: true},
	"service_update_config": {NeedsApproval: true},
})

// NewRiskRegistry creates a registry from the provided policies.
func NewRiskRegistry(policies map[string]RiskPolicy) *RiskRegistry {
	registry := &RiskRegistry{policies: make(map[string]RiskPolicy, len(policies))}
	for toolName, policy := range policies {
		registry.policies[normalizeToolName(toolName)] = policy
	}
	return registry
}

// DefaultRiskRegistry returns the built-in approval registry.
func DefaultRiskRegistry() *RiskRegistry {
	return defaultRiskRegistry
}

// Lookup returns the policy for a tool if it exists.
func (r *RiskRegistry) Lookup(toolName string) (RiskPolicy, bool) {
	if r == nil {
		return RiskPolicy{}, false
	}
	policy, ok := r.policies[normalizeToolName(toolName)]
	return policy, ok
}

// RequiresApproval resolves the final approval decision for a tool call.
func (r *RiskRegistry) RequiresApproval(toolName, commandClass string) bool {
	policy, ok := r.Lookup(toolName)
	if !ok {
		return false
	}
	return policy.RequiresApproval(commandClass)
}

// ToolNames returns the registered tool names in sorted order.
func (r *RiskRegistry) ToolNames() []string {
	if r == nil || len(r.policies) == 0 {
		return nil
	}
	names := make([]string, 0, len(r.policies))
	for toolName := range r.policies {
		names = append(names, toolName)
	}
	sort.Strings(names)
	return names
}

// RequiresApproval resolves the policy for a command class.
func (p RiskPolicy) RequiresApproval(commandClass string) bool {
	if !p.ConditionalByCommandClass {
		return p.NeedsApproval
	}
	if isReadonlyCommandClass(commandClass) {
		return false
	}
	return p.NeedsApproval
}

func normalizeToolName(toolName string) string {
	return strings.ToLower(strings.TrimSpace(toolName))
}

func isReadonlyCommandClass(commandClass string) bool {
	return strings.EqualFold(strings.TrimSpace(commandClass), "readonly")
}
