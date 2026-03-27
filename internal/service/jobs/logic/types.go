// Package logic 提供任务管理的业务逻辑层。
//
// 本文件定义任务管理相关的请求和响应数据结构。
package logic

// CreateJobReq 是创建任务的请求参数。
//
// 字段说明:
//   - Name: 任务名称 (必填)
//   - Type: 任务类型 (shell/script)，默认 shell
//   - Command: 要执行的命令或脚本
//   - HostIDs: 目标主机 ID 列表 (逗号分隔)
//   - Cron: Cron 调度表达式
//   - Timeout: 执行超时时间 (秒)，默认 300
//   - Priority: 任务优先级
//   - Description: 任务描述
type CreateJobReq struct {
	Name        string `json:"name" binding:"required"` // 任务名称 (必填)
	Type        string `json:"type"`                    // 任务类型 (shell/script)
	Command     string `json:"command"`                 // 要执行的命令
	HostIDs     string `json:"host_ids"`                // 目标主机 ID 列表
	Cron        string `json:"cron"`                    // Cron 调度表达式
	Timeout     int    `json:"timeout"`                 // 执行超时时间 (秒)
	Priority    int    `json:"priority"`                // 任务优先级
	Description string `json:"description"`             // 任务描述
}

// UpdateJobReq 是更新任务的请求参数。
//
// 字段说明:
//   - Name: 任务名称
//   - Type: 任务类型 (shell/script)
//   - Command: 要执行的命令或脚本
//   - HostIDs: 目标主机 ID 列表 (逗号分隔)
//   - Cron: Cron 调度表达式
//   - Status: 任务状态 (pending/running/success/failed/stopped)
//   - Timeout: 执行超时时间 (秒)
//   - Priority: 任务优先级
//   - Description: 任务描述
//
// 注意: 仅更新非零值字段
type UpdateJobReq struct {
	Name        string `json:"name"`        // 任务名称
	Type        string `json:"type"`        // 任务类型 (shell/script)
	Command     string `json:"command"`     // 要执行的命令
	HostIDs     string `json:"host_ids"`    // 目标主机 ID 列表
	Cron        string `json:"cron"`        // Cron 调度表达式
	Status      string `json:"status"`      // 任务状态 (pending/running/success/failed/stopped)
	Timeout     int    `json:"timeout"`     // 执行超时时间 (秒)
	Priority    int    `json:"priority"`    // 任务优先级
	Description string `json:"description"` // 任务描述
}

// ListJobsReq 是获取任务列表的请求参数。
//
// 字段说明:
//   - Page: 页码，从 1 开始，最小值 1
//   - PageSize: 每页数量，最小值 1，最大值 100
type ListJobsReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`       // 页码 (从 1 开始)
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"` // 每页数量 (1-100)
}
