// Package model 定义数据库模型。
//
// 本文件包含定时任务相关的模型定义：
//   - Job: 任务调度定义
//   - JobExecution: 任务执行记录
//   - JobLog: 任务执行日志
package model

import "time"

// Job 是任务调度定义模型。
//
// 表名: jobs
// 用途: 存储定时任务的基本配置信息，包括执行命令、调度周期、目标主机等。
//
// 字段说明:
//   - ID: 任务唯一标识
//   - Name: 任务名称
//   - Type: 任务类型 (shell/script)
//   - Command: 要执行的命令或脚本
//   - HostIDs: 目标主机 ID 列表 (逗号分隔)
//   - Cron: Cron 表达式定义调度周期
//   - Status: 任务状态 (pending/running/success/failed)
//   - Timeout: 执行超时时间 (秒)
//   - Priority: 任务优先级
//   - Description: 任务描述
//   - LastRun: 上次执行时间
//   - NextRun: 下次预计执行时间
//   - CreatedBy: 创建者用户 ID
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
type Job struct {
	ID          uint       `gorm:"primaryKey" json:"id"`                                      // 任务 ID
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`                    // 任务名称
	Type        string     `gorm:"type:varchar(32);not null;default:'shell'" json:"type"`    // 任务类型 (shell, script)
	Command     string     `gorm:"type:text" json:"command"`                                  // 要执行的命令
	HostIDs     string     `gorm:"type:text" json:"host_ids"`                                 // 目标主机 ID 列表
	Cron        string     `gorm:"type:varchar(64)" json:"cron"`                              // Cron 调度表达式
	Status      string     `gorm:"type:varchar(32);not null;default:'pending'" json:"status"` // 任务状态 (pending, running, success, failed)
	Timeout     int        `gorm:"default:300" json:"timeout"`                                // 执行超时时间 (秒)
	Priority    int        `gorm:"default:0" json:"priority"`                                 // 任务优先级
	Description string     `gorm:"type:text" json:"description"`                              // 任务描述
	LastRun     *time.Time `json:"last_run"`                                                  // 上次执行时间
	NextRun     *time.Time `json:"next_run"`                                                  // 下次预计执行时间
	CreatedBy   uint       `json:"created_by"`                                                // 创建者用户 ID
	CreatedAt   time.Time  `json:"created_at"`                                                // 创建时间
	UpdatedAt   time.Time  `json:"updated_at"`                                                // 更新时间
}

// TableName 返回 Job 模型对应的数据库表名。
//
// 返回: 表名字符串 "jobs"
func (Job) TableName() string {
	return "jobs"
}

// JobExecution 是任务执行记录模型。
//
// 表名: job_executions
// 用途: 记录任务的每次执行情况，包括执行状态、输出结果、耗时等。
//
// 字段说明:
//   - ID: 执行记录唯一标识
//   - JobID: 关联的任务 ID
//   - HostID: 执行主机 ID
//   - HostIP: 执行主机 IP
//   - Status: 执行状态 (pending/running/success/failed)
//   - ExitCode: 命令退出码
//   - Output: 执行输出内容
//   - StartTime: 开始执行时间
//   - EndTime: 执行结束时间
//   - CreatedAt: 记录创建时间
type JobExecution struct {
	ID        uint       `gorm:"primaryKey" json:"id"`                                      // 执行记录 ID
	JobID     uint       `gorm:"not null;index" json:"job_id"`                              // 关联的任务 ID
	HostID    uint       `json:"host_id"`                                                   // 执行主机 ID
	HostIP    string     `gorm:"type:varchar(64)" json:"host_ip"`                           // 执行主机 IP
	Status    string     `gorm:"type:varchar(32);not null;default:'pending'" json:"status"` // 执行状态 (pending, running, success, failed)
	ExitCode  int        `json:"exit_code"`                                                 // 命令退出码
	Output    string     `gorm:"type:text" json:"output"`                                   // 执行输出内容
	StartTime time.Time  `json:"start_time"`                                                // 开始执行时间
	EndTime   *time.Time `json:"end_time"`                                                  // 执行结束时间
	CreatedAt time.Time  `json:"created_at"`                                                // 记录创建时间
}

// TableName 返回 JobExecution 模型对应的数据库表名。
//
// 返回: 表名字符串 "job_executions"
func (JobExecution) TableName() string {
	return "job_executions"
}

// JobLog 是任务执行日志模型。
//
// 表名: job_logs
// 用途: 记录任务执行过程中的详细日志信息，用于问题排查和审计。
//
// 字段说明:
//   - ID: 日志记录唯一标识
//   - JobID: 关联的任务 ID
//   - ExecutionID: 关联的执行记录 ID
//   - Level: 日志级别 (info/warn/error)
//   - Message: 日志消息内容
//   - CreatedAt: 日志创建时间
type JobLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`                            // 日志记录 ID
	JobID       uint      `gorm:"not null;index" json:"job_id"`                     // 关联的任务 ID
	ExecutionID uint      `gorm:"index" json:"execution_id"`                        // 关联的执行记录 ID
	Level       string    `gorm:"type:varchar(16);default:'info'" json:"level"`    // 日志级别 (info, warn, error)
	Message     string    `gorm:"type:text" json:"message"`                         // 日志消息内容
	CreatedAt   time.Time `json:"created_at"`                                       // 日志创建时间
}

// TableName 返回 JobLog 模型对应的数据库表名。
//
// 返回: 表名字符串 "job_logs"
func (JobLog) TableName() string {
	return "job_logs"
}
