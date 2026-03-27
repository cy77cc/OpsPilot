// Package logic 提供自动化运维服务的业务逻辑层。
//
// 本文件定义自动化运维服务的数据传输对象（DTO），用于 Handler 和 Logic 层之间的数据传递。
package logic

// PreviewRunReq 是预览执行请求参数。
//
// 用于预览自动化任务的执行配置，生成预览令牌。
type PreviewRunReq struct {
	Action string         `json:"action"` // 执行动作类型
	Params map[string]any `json:"params"` // 执行参数
}

// ExecuteRunReq 是执行自动化任务请求参数。
//
// 用于触发实际的自动化任务执行。
type ExecuteRunReq struct {
	ApprovalToken string         `json:"approval_token"` // 审批令牌，由预览接口生成
	Action        string         `json:"action"`         // 执行动作类型
	Params        map[string]any `json:"params"`         // 执行参数
}

// CreateInventoryReq 是创建主机清单请求参数。
//
// 用于创建新的自动化主机清单。
type CreateInventoryReq struct {
	Name      string `json:"name"`       // 清单名称，不能为空
	HostsJSON string `json:"hosts_json"` // 主机配置 JSON 字符串
}

// CreatePlaybookReq 是创建 Playbook 请求参数。
//
// 用于创建新的自动化 Playbook。
type CreatePlaybookReq struct {
	Name       string `json:"name"`        // Playbook 名称，不能为空
	ContentYML string `json:"content_yml"` // YAML 格式的 Playbook 内容
	RiskLevel  string `json:"risk_level"`  // 风险等级（low/medium/high），默认 medium
}
