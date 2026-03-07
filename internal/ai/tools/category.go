package tools

// Tool category builders - filter tools by category for scene-specific tool access

// buildCICDTools returns CI/CD and job related tools
func buildCICDTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByPrefix(all, "cicd_", "job_")
}

// buildConfigTools returns configuration management tools
func buildConfigTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByPrefix(all, "config_")
}

// buildDeploymentTools returns deployment and credential tools
func buildDeploymentTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByPrefix(all, "deployment_", "credential_")
}

// buildGovernanceTools returns governance related tools (topology, audit, user, role, permission)
func buildGovernanceTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByPrefix(all, "topology_", "audit_", "user_", "role_", "permission_")
}

// buildInventoryTools returns inventory listing tools
func buildInventoryTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByName(all,
		"host_list_inventory",
		"cluster_list_inventory",
		"service_list_inventory",
	)
}

// buildK8sTools returns Kubernetes and cluster related tools
func buildK8sTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByPrefix(all, "k8s_", "cluster_")
}

// buildMonitorTools returns monitoring tools
func buildMonitorTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByPrefix(all, "monitor_")
}

// buildOpsTools returns OS and host operation tools
func buildOpsTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByPrefix(all, "os_", "host_")
}

// buildServiceTools returns service management tools
func buildServiceTools(all []RegisteredTool) []RegisteredTool {
	return filterToolsByPrefix(all, "service_")
}
