package middleware

func DefaultSceneToolMap() map[string][]string {
	return map[string][]string{
		"kubernetes":   {"k8s_query", "k8s_list_resources", "service_get_detail", "service_status", "host_list_inventory"},
		"cicd":         {"cicd_pipeline_list", "cicd_pipeline_status", "cicd_pipeline_trigger", "job_list", "job_execution_status", "job_run", "deployment_target_list", "deployment_target_detail", "deployment_bootstrap_status"},
		"monitoring":   {"monitor_alert_rule_list", "monitor_alert", "monitor_metric"},
		"host":         {"host_exec", "host_list_inventory", "os_get_cpu_mem", "os_get_disk_fs", "os_get_net_stat", "os_get_process_top", "os_get_journal_tail", "os_get_container_runtime"},
		"cluster":      {"k8s_query", "k8s_list_resources", "deployment_bootstrap_status", "cluster_list_inventory", "service_list_inventory", "service_get_detail", "service_status"},
		"deployment":   {"deployment_target_list", "deployment_target_detail", "deployment_bootstrap_status", "cluster_list_inventory", "service_list_inventory", "service_get_detail", "service_status", "service_deploy_preview", "service_deploy_apply"},
		"service":      {"service_get_detail", "service_status", "service_status_by_target", "service_catalog_list", "service_category_tree", "service_visibility_check", "service_deploy_preview"},
		"infrastructure": {"credential_list", "credential_test", "host_list_inventory", "cluster_list_inventory"},
		"governance":   {"user_list", "role_list", "permission_check", "topology_get", "audit_log_search"},
		"ai":           {"service_get_detail", "service_status", "service_catalog_list", "host_list_inventory", "os_get_cpu_mem", "os_get_disk_fs", "os_get_net_stat", "os_get_process_top", "os_get_journal_tail", "os_get_container_runtime", "monitor_alert", "monitor_metric", "k8s_query", "k8s_list_resources"},
	}
}

func DefaultScenePromptMap() map[string]string {
	return map[string]string{
		"kubernetes":   "Kubernetes operations and diagnosis.",
		"cicd":         "CI/CD pipeline administration and status checks.",
		"monitoring":   "Monitoring and alert investigation.",
		"host":         "Host operations with guarded execution.",
		"cluster":      "Cluster inventory and deployment visibility.",
		"deployment":   "Deployment planning and rollout controls.",
		"service":      "Service status and release context.",
		"infrastructure": "Infrastructure inventory and credential health.",
		"governance":   "Governance, audit, and topology review.",
		"ai":           "General AI assistant scene with cross-domain diagnostics.",
	}
}
