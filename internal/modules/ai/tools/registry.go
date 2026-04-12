package tools

// AllCatalogEntries returns the canonical tool catalog across AI domains.
func AllCatalogEntries() []ToolMetadata {
	entries := make([]ToolMetadata, 0, 48)
	entries = append(entries, staticCatalogEntries()...)
	return entries
}

func staticCatalogEntries() []ToolMetadata {
	return []ToolMetadata{
		{ToolName: "service_get_detail", Domain: "service", Capability: "query", RiskLevel: "low", OutputMode: "inline", Description: "get detailed service information", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_status", Domain: "service", Capability: "query", RiskLevel: "low", OutputMode: "inline", Description: "get current service runtime status", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_status_by_target", Domain: "service", Capability: "query", RiskLevel: "low", OutputMode: "inline", Description: "resolve and get service status by id or name", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_deploy_preview", Domain: "service", Capability: "preview", RiskLevel: "medium", OutputMode: "inline", Description: "preview service deployment impact", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_deploy_apply", Domain: "service", Capability: "mutation", RiskLevel: "high", OutputMode: "inline", Description: "apply service deployment", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_deploy", Domain: "service", Capability: "mutation", RiskLevel: "high", OutputMode: "inline", Description: "preview or apply service deployment", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_catalog_list", Domain: "service", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list services by catalog filters", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_category_tree", Domain: "service", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list service category hierarchy", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_visibility_check", Domain: "service", Capability: "query", RiskLevel: "low", OutputMode: "inline", Description: "check service visibility before deployment", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "deployment_target_list", Domain: "deployment", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list deployment targets", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "deployment_target_detail", Domain: "deployment", Capability: "query", RiskLevel: "low", OutputMode: "inline", Description: "get deployment target detail", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "deployment_bootstrap_status", Domain: "deployment", Capability: "query", RiskLevel: "medium", OutputMode: "inline", Description: "get target bootstrap and readiness status", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "cluster_list_inventory", Domain: "deployment", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list clusters in deployment inventory", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "service_list_inventory", Domain: "deployment", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list services by runtime and environment", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "cicd_pipeline_list", Domain: "cicd", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list CI pipeline configurations", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "cicd_pipeline_status", Domain: "cicd", Capability: "query", RiskLevel: "low", OutputMode: "inline", Description: "get pipeline configuration and recent runs", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "cicd_pipeline_trigger", Domain: "cicd", Capability: "mutation", RiskLevel: "high", OutputMode: "inline", Description: "trigger a CI pipeline run", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "job_list", Domain: "cicd", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list jobs in CI/CD inventory", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "job_execution_status", Domain: "cicd", Capability: "query", RiskLevel: "low", OutputMode: "inline", Description: "get job execution status", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "job_run", Domain: "cicd", Capability: "mutation", RiskLevel: "high", OutputMode: "inline", Description: "run a job manually", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "credential_list", Domain: "infrastructure", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list infrastructure credentials", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "credential_test", Domain: "infrastructure", Capability: "query", RiskLevel: "medium", OutputMode: "inline", Description: "get credential connectivity test result", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "user_list", Domain: "governance", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list platform users", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "role_list", Domain: "governance", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list platform roles", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "permission_check", Domain: "governance", Capability: "query", RiskLevel: "medium", OutputMode: "inline", Description: "check a user's permission on a resource", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "topology_get", Domain: "governance", Capability: "query", RiskLevel: "medium", OutputMode: "inline", Description: "get service dependency topology", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "audit_log_search", Domain: "governance", Capability: "query", RiskLevel: "medium", OutputMode: "inline", Description: "search governance audit logs", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
	}
}
