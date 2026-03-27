// Package host 提供主机运维相关的工具实现。
//
// 本文件实现主机操作工具集，包括：
//   - SSH 命令执行（只读和批量）
//   - 主机清单查询
//   - 系统诊断（CPU、内存、磁盘、网络、进程）
//   - 日志和容器运行时查询
package host

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	common "github.com/cy77cc/OpsPilot/internal/ai/common/approval"
	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/utils"
)

func serviceContextFromRuntime(ctx context.Context) *svc.ServiceContext {
	svcCtx, _ := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
	return svcCtx
}

// =============================================================================
// 输入类型定义
// =============================================================================

// HostExecInput 主机命令执行输入。
type HostExecInput struct {
	HostID  int    `json:"host_id" jsonschema_description:"required,host id"`
	Command string `json:"command,omitempty" jsonschema_description:"optional,readonly command"`
	Script  string `json:"script,omitempty" jsonschema_description:"optional,script command"`
}

// HostInventoryInput 主机清单查询输入。
type HostInventoryInput struct {
	Status  string `json:"status,omitempty" jsonschema_description:"optional host status filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on name/ip/hostname"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max hosts,default=50"`
}

// OSCPUMemInput CPU/内存诊断输入。
type OSCPUMemInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
}

// OSDiskInput 磁盘诊断输入。
type OSDiskInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
}

// OSNetInput 网络诊断输入。
type OSNetInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
}

// OSProcessTopInput 进程排行输入。
type OSProcessTopInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
	Limit  int    `json:"limit,omitempty" jsonschema_description:"top process count,default=10"`
}

// OSJournalInput 日志查询输入。
type OSJournalInput struct {
	Target  string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
	Service string `json:"service" jsonschema_description:"required,systemd service unit"`
	Lines   int    `json:"lines,omitempty" jsonschema_description:"log lines,default=200"`
}

// OSContainerRuntimeInput 容器运行时查询输入。
type OSContainerRuntimeInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
}

// serviceUnitRegexp 服务单元名称正则表达式，用于验证输入安全性。
var serviceUnitRegexp = regexp.MustCompile(`^[a-zA-Z0-9_.@-]+$`)

// =============================================================================
// 工具入口
// =============================================================================

// NewHostTools 创建主机只读工具子集。
//
// 返回只读工具列表，包括：
//   - 主机命令执行（host_exec）
//   - 主机清单查询（host_list_inventory）
//   - 批量执行预览（host_batch_exec_preview）
//   - 系统诊断：CPU/内存、磁盘、网络、进程、日志、容器运行时
//
// 这些工具不修改任何状态，可安全用于诊断场景。
func NewHostTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		HostExec(ctx),
		HostListInventory(ctx),
		OSGetCPUMem(ctx),
		OSGetDiskFS(ctx),
		OSGetNetStat(ctx),
		OSGetProcessTop(ctx),
		OSGetJournalTail(ctx),
		OSGetContainerRuntime(ctx),
	}
}

type HostExecOutput struct {
	HostID   int    `json:"host_id"`
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Status   string `json:"status,omitempty"`

	PolicyDecision string            `json:"policy_decision,omitempty"`
	PolicyReasons  []string          `json:"policy_reasons,omitempty"`
	Violations     []PolicyViolation `json:"violations,omitempty"`
}

func HostExec(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"host_exec",
		"Execute a command or script on a single host only when policy allows. Provide exactly one of command or script together with host_id. Approval is enforced by the middleware interrupt flow.",
		func(ctx context.Context, input *HostExecInput, opts ...tool.Option) (*HostExecOutput, error) {
			hostID := input.HostID
			cmd := strings.TrimSpace(input.Command)
			script := strings.TrimSpace(input.Script)
			if hostID <= 0 {
				return nil, fmt.Errorf("host_id is required")
			}
			if (cmd == "" && script == "") || (cmd != "" && script != "") {
				return nil, fmt.Errorf("provide exactly one of command or script")
			}
			execText := cmd
			if execText == "" {
				execText = script
			}
			return runPolicyAwareExecByTarget(ctx, svcCtx, "host_exec", strconv.Itoa(hostID), execText)
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func runPolicyAwareExecByTarget(ctx context.Context, svcCtx *svc.ServiceContext, toolName, target, cmd string) (*HostExecOutput, error) {
	engine := NewHostCommandPolicyEngine(DefaultReadonlyAllowlist())
	decision := engine.Evaluate(PolicyInput{
		ToolName:   toolName,
		CommandRaw: cmd,
		Target:     strings.TrimSpace(target),
	})

	if decision.DecisionType != DecisionAllowReadonlyExecute && !approvedHostResume(ctx) {
		return nil, fmt.Errorf(
			"approval required: decision=%s reasons=%v violations=%v",
			decision.DecisionType,
			decision.ReasonCodes,
			decision.Violations,
		)
	}

	node, err := resolveNodeByTarget(svcCtx, target)
	if err != nil {
		return &HostExecOutput{
			HostID:         0,
			Command:        cmd,
			Stdout:         "",
			Stderr:         err.Error(),
			ExitCode:       1,
			Status:         "completed",
			PolicyDecision: string(decision.DecisionType),
			PolicyReasons:  decision.ReasonCodes,
			Violations:     decision.Violations,
		}, nil
	}
	if node == nil {
		out, runErr := runLocalCommand(ctx, 6*time.Second, "sh", []string{"-c", cmd}...)
		if runErr != nil {
			return &HostExecOutput{
				HostID:         0,
				Command:        cmd,
				Stdout:         out,
				Stderr:         runErr.Error(),
				ExitCode:       1,
				Status:         "completed",
				PolicyDecision: string(decision.DecisionType),
				PolicyReasons:  decision.ReasonCodes,
				Violations:     decision.Violations,
			}, nil
		}
		return &HostExecOutput{
			HostID:         0,
			Command:        cmd,
			Stdout:         out,
			Stderr:         "",
			ExitCode:       0,
			Status:         "completed",
			PolicyDecision: string(decision.DecisionType),
			PolicyReasons:  decision.ReasonCodes,
			Violations:     decision.Violations,
		}, nil
	}

	out, runErr := executeHostCommand(svcCtx, node, cmd)
	if runErr != nil {
		return &HostExecOutput{
			HostID:         int(node.ID),
			Command:        cmd,
			Stdout:         out,
			Stderr:         runErr.Error(),
			ExitCode:       1,
			Status:         "completed",
			PolicyDecision: string(decision.DecisionType),
			PolicyReasons:  decision.ReasonCodes,
			Violations:     decision.Violations,
		}, nil
	}
	return &HostExecOutput{
		HostID:         int(node.ID),
		Command:        cmd,
		Stdout:         out,
		Stderr:         "",
		ExitCode:       0,
		Status:         "completed",
		PolicyDecision: string(decision.DecisionType),
		PolicyReasons:  decision.ReasonCodes,
		Violations:     decision.Violations,
	}, nil
}

func approvedHostResume(ctx context.Context) bool {
	isTarget, hasData, result := tool.GetResumeContext[*common.ApprovalResult](ctx)
	return isTarget && hasData && result != nil && result.Approved
}

type HostListInventoryOutput struct {
	Total int              `json:"total"`
	List  []map[string]any `json:"list"`
}

func HostListInventory(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"host_list_inventory",
		"Query host inventory list with detailed information including CPU, memory, disk, SSH configuration, and status. Optional parameters: status filters by host status (online/offline/maintenance), keyword searches by name/IP/hostname, limit controls max results (default 50, max 200). Example: {\"status\":\"online\",\"keyword\":\"web\",\"limit\":20}.",
		func(ctx context.Context, input *HostInventoryInput, opts ...tool.Option) (*HostListInventoryOutput, error) {
			if svcCtx.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := svcCtx.DB.Model(&model.Node{})
			if status := strings.TrimSpace(input.Status); status != "" {
				query = query.Where("status = ?", status)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR ip LIKE ? OR hostname LIKE ?", pattern, pattern, pattern)
			}
			var nodes []model.Node
			if err := query.Order("id desc").Limit(limit).Find(&nodes).Error; err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(nodes))
			for _, node := range nodes {
				items = append(items, map[string]any{
					"id":         uint64(node.ID),
					"name":       node.Name,
					"ip":         node.IP,
					"hostname":   node.Hostname,
					"status":     node.Status,
					"auth_type":  detectNodeAuthType(&node),
					"ssh_user":   node.SSHUser,
					"port":       node.Port,
					"cpu_cores":  node.CpuCores,
					"memory_mb":  node.MemoryMB,
					"disk_gb":    node.DiskGB,
					"labels":     parseHostLabels(node.Labels),
					"updated_at": node.UpdatedAt,
				})
			}
			return &HostListInventoryOutput{
				Total: len(items),
				List:  items,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type OSGetCPUMemOutput struct {
	Loadavg string `json:"loadavg"`
	Meminfo string `json:"meminfo"`
	Uptime  string `json:"uptime"`
}

func OSGetCPUMem(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"os_get_cpu_mem",
		"Get CPU, memory and load average information from a target host. Returns loadavg from /proc/loadavg, meminfo from /proc/meminfo, and uptime output. Target can be host ID, IP address, hostname, or 'localhost' (default) for local execution. Example: {\"target\":\"10.0.0.5\"}.",
		func(ctx context.Context, input *OSCPUMemInput, opts ...tool.Option) (*OSGetCPUMemOutput, error) {
			target := strings.TrimSpace(input.Target)
			loadavg, _, _ := runOnTarget(ctx, svcCtx, target, "cat", []string{"/proc/loadavg"}, "cat /proc/loadavg")
			mem, _, err := runOnTarget(ctx, svcCtx, target, "cat", []string{"/proc/meminfo"}, "cat /proc/meminfo")
			if err != nil {
				return nil, err
			}
			uptime, _, _ := runOnTarget(ctx, svcCtx, target, "uptime", nil, "uptime")
			return &OSGetCPUMemOutput{
				Loadavg: loadavg,
				Meminfo: mem,
				Uptime:  uptime,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type OSGetDiskFSOutput struct {
	Filesystem string `json:"filesystem"`
}

func OSGetDiskFS(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"os_get_disk_fs",
		"Get disk and filesystem usage information using 'df -h' command. Shows mounted filesystems, total size, used space, available space, and mount points. Target can be host ID, IP address, hostname, or 'localhost' (default). Example: {\"target\":\"web-server-01\"}.",
		func(ctx context.Context, input *OSDiskInput, opts ...tool.Option) (*OSGetDiskFSOutput, error) {
			target := strings.TrimSpace(input.Target)
			out, _, err := runOnTarget(ctx, svcCtx, target, "df", []string{"-h"}, "df -h")
			if err != nil {
				return nil, err
			}
			return &OSGetDiskFSOutput{Filesystem: out}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type OSGetNetStatOutput struct {
	NetDev         string `json:"net_dev"`
	ListeningPorts string `json:"listening_ports"`
}

func OSGetNetStat(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"os_get_net_stat",
		"Get network statistics including network device traffic from /proc/net/dev and listening TCP ports using 'ss -ltn'. Shows bytes sent/received per interface and all listening ports. Target can be host ID, IP address, hostname, or 'localhost' (default). Example: {\"target\":\"192.168.1.10\"}.",
		func(ctx context.Context, input *OSNetInput, opts ...tool.Option) (*OSGetNetStatOutput, error) {
			target := strings.TrimSpace(input.Target)
			dev, _, err := runOnTarget(ctx, svcCtx, target, "cat", []string{"/proc/net/dev"}, "cat /proc/net/dev")
			if err != nil {
				return nil, err
			}
			listen, _, _ := runOnTarget(ctx, svcCtx, target, "ss", []string{"-ltn"}, "ss -ltn")
			return &OSGetNetStatOutput{
				NetDev:         dev,
				ListeningPorts: listen,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type OSGetProcessTopOutput struct {
	TopProcesses string `json:"top_processes"`
	Limit        int    `json:"limit"`
}

func OSGetProcessTop(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"os_get_process_top",
		"Get top processes sorted by CPU usage using 'ps aux --sort=-%cpu'. Returns the most CPU-intensive processes. Limit parameter controls how many processes to show (default 10, max 50). Target can be host ID, IP address, hostname, or 'localhost' (default). Example: {\"target\":\"localhost\",\"limit\":20}.",
		func(ctx context.Context, input *OSProcessTopInput, opts ...tool.Option) (*OSGetProcessTopOutput, error) {
			target := strings.TrimSpace(input.Target)
			limit := input.Limit
			if limit <= 0 {
				limit = 10
			}
			if limit > 50 {
				limit = 50
			}
			cmd := fmt.Sprintf("ps aux --sort=-%%cpu | head -n %d", limit+1)
			out, _, err := runOnTarget(ctx, svcCtx, target, "ps", []string{"aux", "--sort=-%cpu"}, cmd)
			if err != nil {
				return nil, err
			}
			return &OSGetProcessTopOutput{
				TopProcesses: out,
				Limit:        limit,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type OSGetJournalTailOutput struct {
	Service string `json:"service"`
	Lines   int    `json:"lines"`
	Logs    string `json:"logs"`
}

func OSGetJournalTail(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"os_get_journal_tail",
		"Get systemd journal logs for a specific service using 'journalctl -u <service> -n <lines>'. Service name is required. Lines parameter controls how many log lines to retrieve (default 200, max 500). Target can be host ID, IP address, hostname, or 'localhost' (default). Example: {\"target\":\"10.0.0.1\",\"service\":\"nginx\",\"lines\":100}.",
		func(ctx context.Context, input *OSJournalInput, opts ...tool.Option) (*OSGetJournalTailOutput, error) {
			target := strings.TrimSpace(input.Target)
			service := strings.TrimSpace(input.Service)
			if service == "" {
				return nil, fmt.Errorf("service is required")
			}
			if !serviceUnitRegexp.MatchString(service) {
				return nil, fmt.Errorf("invalid service name")
			}
			lines := input.Lines
			if lines <= 0 {
				lines = 200
			}
			if lines > 500 {
				lines = 500
			}
			localArgs := []string{"-u", service, "-n", strconv.Itoa(lines), "--no-pager"}
			remoteCmd := fmt.Sprintf("journalctl -u %s -n %d --no-pager", service, lines)
			out, _, err := runOnTarget(ctx, svcCtx, target, "journalctl", localArgs, remoteCmd)
			if err != nil {
				return nil, err
			}
			return &OSGetJournalTailOutput{
				Service: service,
				Lines:   lines,
				Logs:    out,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type OSGetContainerRuntimeOutput struct {
	Runtime    string `json:"runtime"`
	Containers string `json:"containers"`
}

func OSGetContainerRuntime(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"os_get_container_runtime",
		"Get container runtime information and running containers. Detects Docker or containerd. For Docker, runs 'docker ps' to show container ID, image, and status. For containerd, runs 'ctr -n k8s.io containers list'. Target can be host ID, IP address, hostname, or 'localhost' (default). Example: {\"target\":\"node-01\"}.",
		func(ctx context.Context, input *OSContainerRuntimeInput, opts ...tool.Option) (*OSGetContainerRuntimeOutput, error) {
			target := strings.TrimSpace(input.Target)
			out, _, err := runOnTarget(ctx, svcCtx, target, "docker", []string{"ps", "--format", "{{.ID}} {{.Image}} {{.Status}}"}, "docker ps --format '{{.ID}} {{.Image}} {{.Status}}'")
			if err == nil {
				return &OSGetContainerRuntimeOutput{
					Runtime:    "docker",
					Containers: out,
				}, nil
			}
			out2, _, err2 := runOnTarget(ctx, svcCtx, target, "ctr", []string{"-n", "k8s.io", "containers", "list"}, "ctr -n k8s.io containers list")
			if err2 == nil {
				return &OSGetContainerRuntimeOutput{
					Runtime:    "containerd",
					Containers: out2,
				}, nil
			}
			return nil, fmt.Errorf("docker/containerd unavailable: %v / %v", err, err2)
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// 辅助函数
// =============================================================================

// executeHostCommand 在指定主机上执行命令。
//
// 通过 SSH 连接到目标主机并执行命令，支持密钥和密码认证。
func executeHostCommand(svcCtx *svc.ServiceContext, node *model.Node, command string) (string, error) {
	privateKey, passphrase, err := loadNodePrivateKey(svcCtx, node)
	if err != nil {
		return "", err
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	return sshclient.RunCommand(cli, command)
}

// loadNodePrivateKey 加载节点的 SSH 私钥。
//
// 从数据库加载私钥，如果加密则先解密。
func loadNodePrivateKey(svcCtx *svc.ServiceContext, node *model.Node) (string, string, error) {
	if svcCtx.DB == nil || node == nil || node.SSHKeyID == nil {
		return "", "", nil
	}
	var key model.SSHKey
	if err := svcCtx.DB.Select("id", "private_key", "passphrase", "encrypted").Where("id = ?", uint64(*node.SSHKeyID)).First(&key).Error; err != nil {
		return "", "", err
	}
	pk := strings.TrimSpace(key.PrivateKey)
	pp := strings.TrimSpace(key.Passphrase)
	if !key.Encrypted {
		return pk, pp, nil
	}
	decrypted, err := utils.DecryptText(pk, config.CFG.Security.EncryptionKey)
	if err != nil {
		return "", "", err
	}
	return decrypted, pp, nil
}

// loadHostNodesMap 批量加载主机节点并构建映射。
//
// 返回节点映射和缺失的 ID 列表。
func loadHostNodesMap(svcCtx *svc.ServiceContext, hostIDs []uint64) (map[uint64]*model.Node, []uint64, error) {
	if svcCtx.DB == nil {
		return nil, nil, fmt.Errorf("db unavailable")
	}
	var nodes []model.Node
	if err := svcCtx.DB.Where("id IN ?", hostIDs).Find(&nodes).Error; err != nil {
		return nil, nil, err
	}
	byID := make(map[uint64]*model.Node, len(nodes))
	for i := range nodes {
		byID[uint64(nodes[i].ID)] = &nodes[i]
	}
	missing := make([]uint64, 0)
	for _, id := range hostIDs {
		if _, ok := byID[id]; !ok {
			missing = append(missing, id)
		}
	}
	return byID, missing, nil
}

// parseHostLabels 解析主机标签字符串。
//
// 支持 JSON 数组和逗号分隔两种格式。
func parseHostLabels(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
			out := make([]string, 0, len(arr))
			for _, item := range arr {
				if s := strings.TrimSpace(item); s != "" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// isReadonlyHostCommand 检查命令是否为安全的只读命令。
func isReadonlyHostCommand(cmd string) bool {
	switch strings.TrimSpace(cmd) {
	case "hostname", "uptime", "df -h", "free -m", "ps aux --sort=-%cpu":
		return true
	default:
		return false
	}
}

// detectNodeAuthType 检测节点的认证类型。
//
// 返回 "key"（密钥认证）、"password"（密码认证）或 "unknown"。
func detectNodeAuthType(node *model.Node) string {
	if node == nil {
		return "unknown"
	}
	if node.SSHKeyID != nil && uint64(*node.SSHKeyID) > 0 {
		return "key"
	}
	if strings.TrimSpace(node.SSHPassword) != "" {
		return "password"
	}
	return "unknown"
}
