package host

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/core"
	sshclient "github.com/cy77cc/k8s-manage/internal/client/ssh"
	"github.com/cy77cc/k8s-manage/internal/config"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/utils"
)

// Input types

type HostSSHReadonlyInput struct {
	HostID  int    `json:"host_id" jsonschema_description:"required,host id"`
	Command string `json:"command" jsonschema_description:"required,readonly command"`
}

type HostExecInput struct {
	HostID  int    `json:"host_id" jsonschema_description:"required,host id"`
	Command string `json:"command" jsonschema_description:"required,readonly command"`
}

type HostInventoryInput struct {
	Status  string `json:"status,omitempty" jsonschema_description:"optional host status filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on name/ip/hostname"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max hosts,default=50"`
}

type HostBatchExecPreviewInput struct {
	HostIDs []int  `json:"host_ids" jsonschema_description:"required,target host ids"`
	Command string `json:"command" jsonschema_description:"required,shell command to run"`
	Reason  string `json:"reason,omitempty" jsonschema_description:"execution reason for audit context"`
}

type HostBatchExecApplyInput struct {
	HostIDs []int  `json:"host_ids" jsonschema_description:"required,target host ids"`
	Command string `json:"command" jsonschema_description:"required,shell command to run"`
	Reason  string `json:"reason,omitempty" jsonschema_description:"execution reason for audit context"`
}

type HostBatchInput struct {
	HostIDs []int  `json:"host_ids" jsonschema_description:"required,target host ids"`
	Command string `json:"command" jsonschema_description:"required,shell command to run"`
	Reason  string `json:"reason,omitempty" jsonschema_description:"execution reason for audit context"`
}

type HostBatchStatusInput struct {
	HostIDs []int  `json:"host_ids" jsonschema_description:"required,target host ids"`
	Action  string `json:"action" jsonschema_description:"required,status action: online/offline/maintenance"`
	Reason  string `json:"reason,omitempty" jsonschema_description:"change reason for audit context"`
}

type OSCPUMemInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
}

type OSDiskInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
}

type OSNetInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
}

type OSProcessTopInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
	Limit  int    `json:"limit,omitempty" jsonschema_description:"top process count,default=10"`
}

type OSJournalInput struct {
	Target  string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
	Service string `json:"service" jsonschema_description:"required,systemd service unit"`
	Lines   int    `json:"lines,omitempty" jsonschema_description:"log lines,default=200"`
}

type OSContainerRuntimeInput struct {
	Target string `json:"target,omitempty" jsonschema_description:"target host id/ip/hostname,default=localhost"`
}

var serviceUnitRegexp = regexp.MustCompile(`^[a-zA-Z0-9_.@-]+$`)

// NewHostTools returns all host tools.
func NewHostTools(ctx context.Context, deps core.PlatformDeps) []tool.InvokableTool {
	return []tool.InvokableTool{
		HostSSHReadonly(ctx, deps),
		HostExec(ctx, deps),
		HostListInventory(ctx, deps),
		HostBatch(ctx, deps),
		HostBatchExecPreview(ctx, deps),
		HostBatchExecApply(ctx, deps),
		HostBatchStatusUpdate(ctx, deps),
		OSGetCPUMem(ctx, deps),
		OSGetDiskFS(ctx, deps),
		OSGetNetStat(ctx, deps),
		OSGetProcessTop(ctx, deps),
		OSGetJournalTail(ctx, deps),
		OSGetContainerRuntime(ctx, deps),
	}
}

type HostSSHReadonlyOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func HostSSHReadonly(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"host_ssh_exec_readonly",
		"Execute a readonly SSH command on a host. host_id and command are required. Only predefined safe readonly commands are allowed such as: hostname, uptime, df -h, free -m, ps aux --sort=-%cpu. Example: {\"host_id\":1,\"command\":\"uptime\"}.",
		func(ctx context.Context, input *HostSSHReadonlyInput, opts ...tool.Option) (*HostSSHReadonlyOutput, error) {
			hostID := input.HostID
			cmd := strings.TrimSpace(input.Command)
			if hostID <= 0 {
				return nil, fmt.Errorf("host_id is required")
			}
			if cmd == "" {
				return nil, fmt.Errorf("command is required")
			}
			if !isReadonlyHostCommand(cmd) {
				return nil, fmt.Errorf("command not allowed: only readonly commands are permitted")
			}
			var node model.Node
			if err := deps.DB.First(&node, hostID).Error; err != nil {
				return nil, err
			}
			out, err := executeHostCommand(deps, &node, cmd)
			if err != nil {
				return &HostSSHReadonlyOutput{Stdout: out, Stderr: err.Error(), ExitCode: 1}, nil
			}
			return &HostSSHReadonlyOutput{Stdout: out, Stderr: "", ExitCode: 0}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type HostExecOutput struct {
	HostID   int    `json:"host_id"`
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func HostExec(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"host_exec",
		"Execute a readonly command on a single host via SSH. host_id and command are required. Only safe readonly commands are allowed. Returns stdout, stderr and exit code. Example: {\"host_id\":1,\"command\":\"df -h\"}.",
		func(ctx context.Context, input *HostExecInput, opts ...tool.Option) (*HostExecOutput, error) {
			hostID := input.HostID
			cmd := strings.TrimSpace(input.Command)
			if hostID <= 0 {
				return nil, fmt.Errorf("host_id is required")
			}
			if cmd == "" {
				return nil, fmt.Errorf("command is required")
			}
			if !isReadonlyHostCommand(cmd) {
				return nil, fmt.Errorf("command not allowed: only readonly commands are permitted")
			}
			var node model.Node
			if err := deps.DB.First(&node, hostID).Error; err != nil {
				return nil, err
			}
			out, err := executeHostCommand(deps, &node, cmd)
			if err != nil {
				return &HostExecOutput{
					HostID:   hostID,
					Command:  cmd,
					Stdout:   out,
					Stderr:   err.Error(),
					ExitCode: 1,
				}, nil
			}
			return &HostExecOutput{
				HostID:   hostID,
				Command:  cmd,
				Stdout:   out,
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type HostListInventoryOutput struct {
	Total int              `json:"total"`
	List  []map[string]any `json:"list"`
}

func HostListInventory(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"host_list_inventory",
		"Query host inventory list with detailed information including CPU, memory, disk, SSH configuration, and status. Optional parameters: status filters by host status (online/offline/maintenance), keyword searches by name/IP/hostname, limit controls max results (default 50, max 200). Example: {\"status\":\"online\",\"keyword\":\"web\",\"limit\":20}.",
		func(ctx context.Context, input *HostInventoryInput, opts ...tool.Option) (*HostListInventoryOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := deps.DB.Model(&model.Node{})
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

type HostBatchOutput struct {
	Reason         string         `json:"reason"`
	Command        string         `json:"command"`
	CommandClass   string         `json:"command_class"`
	Risk           string         `json:"risk"`
	TargetCount    int            `json:"target_count"`
	MissingHostIDs []uint64       `json:"missing_host_ids"`
	Succeeded      int            `json:"succeeded"`
	Failed         int            `json:"failed"`
	Results        map[string]any `json:"results"`
}

func HostBatch(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"host_batch",
		"Execute a command on multiple hosts in batch. host_ids (array of integers) and command are required. Dangerous commands like 'rm -rf /', 'mkfs', 'shutdown' are blocked. Returns execution results for each host including stdout, stderr, and exit code. Example: {\"host_ids\":[1,2,3],\"command\":\"uptime\",\"reason\":\"health check\"}.",
		func(ctx context.Context, input *HostBatchInput, opts ...tool.Option) (*HostBatchOutput, error) {
			hostIDs, err := normalizeHostIDs(input.HostIDs)
			if err != nil {
				return nil, err
			}
			cmd := strings.TrimSpace(input.Command)
			if cmd == "" {
				return nil, fmt.Errorf("command is required")
			}
			class, risk, blocked := classifyHostCommand(cmd)
			if blocked {
				return nil, fmt.Errorf("dangerous command is blocked")
			}
			nodesByID, missing, err := loadHostNodesMap(deps, hostIDs)
			if err != nil {
				return nil, err
			}
			results := map[string]any{}
			succeeded := 0
			failed := 0
			for _, id := range hostIDs {
				key := strconv.FormatUint(id, 10)
				node, ok := nodesByID[id]
				if !ok {
					results[key] = map[string]any{"stdout": "", "stderr": "host not found", "exit_code": 1}
					failed++
					continue
				}
				out, execErr := executeHostCommand(deps, node, cmd)
				if execErr != nil {
					results[key] = map[string]any{"stdout": out, "stderr": execErr.Error(), "exit_code": 1}
					failed++
					continue
				}
				results[key] = map[string]any{"stdout": out, "stderr": "", "exit_code": 0}
				succeeded++
			}
			return &HostBatchOutput{
				Reason:         strings.TrimSpace(input.Reason),
				Command:        cmd,
				CommandClass:   class,
				Risk:           risk,
				TargetCount:    len(hostIDs),
				MissingHostIDs: missing,
				Succeeded:      succeeded,
				Failed:         failed,
				Results:        results,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type HostBatchExecPreviewOutput struct {
	Command        string           `json:"command"`
	Reason         string           `json:"reason"`
	CommandClass   string           `json:"command_class"`
	Risk           string           `json:"risk"`
	Blocked        bool             `json:"blocked"`
	TargetCount    int              `json:"target_count"`
	ResolvedCount  int              `json:"resolved_count"`
	MissingHostIDs []uint64         `json:"missing_host_ids"`
	Targets        []map[string]any `json:"targets"`
}

func HostBatchExecPreview(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"host_batch_exec_preview",
		"Preview batch command execution before actually running it. host_ids (array) and command are required. Returns resolved target hosts, command classification (readonly/mutating/dangerous), risk level, and whether the command is blocked. Use this to verify the impact before executing with host_batch_exec_apply. Example: {\"host_ids\":[1,2],\"command\":\"systemctl status nginx\"}.",
		func(ctx context.Context, input *HostBatchExecPreviewInput, opts ...tool.Option) (*HostBatchExecPreviewOutput, error) {
			hostIDs, err := normalizeHostIDs(input.HostIDs)
			if err != nil {
				return nil, err
			}
			cmd := strings.TrimSpace(input.Command)
			if cmd == "" {
				return nil, fmt.Errorf("command is required")
			}
			class, risk, blocked := classifyHostCommand(cmd)
			targets, missing, err := loadHostBatchTargets(deps, hostIDs)
			if err != nil {
				return nil, err
			}
			missingIDs := make([]uint64, 0, len(missing))
			for _, id := range missing {
				missingIDs = append(missingIDs, uint64(id))
			}
			return &HostBatchExecPreviewOutput{
				Command:        cmd,
				Reason:         strings.TrimSpace(input.Reason),
				CommandClass:   class,
				Risk:           risk,
				Blocked:        blocked,
				TargetCount:    len(hostIDs),
				ResolvedCount:  len(targets),
				MissingHostIDs: missingIDs,
				Targets:        targets,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type HostBatchExecApplyOutput struct {
	Reason         string         `json:"reason"`
	Command        string         `json:"command"`
	CommandClass   string         `json:"command_class"`
	Risk           string         `json:"risk"`
	TargetCount    int            `json:"target_count"`
	MissingHostIDs []uint64       `json:"missing_host_ids"`
	Succeeded      int            `json:"succeeded"`
	Failed         int            `json:"failed"`
	Results        map[string]any `json:"results"`
}

func HostBatchExecApply(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"host_batch_exec_apply",
		"Execute a command on multiple hosts after preview confirmation. host_ids (array) and command are required. Dangerous commands are blocked. Returns execution results for each host. This is a mutating operation - ensure you have previewed with host_batch_exec_preview first. Example: {\"host_ids\":[1,2],\"command\":\"systemctl restart nginx\",\"reason\":\"restart nginx service\"}.",
		func(ctx context.Context, input *HostBatchExecApplyInput, opts ...tool.Option) (*HostBatchExecApplyOutput, error) {
			hostIDs, err := normalizeHostIDs(input.HostIDs)
			if err != nil {
				return nil, err
			}
			cmd := strings.TrimSpace(input.Command)
			if cmd == "" {
				return nil, fmt.Errorf("command is required")
			}
			class, risk, blocked := classifyHostCommand(cmd)
			if blocked {
				return nil, fmt.Errorf("dangerous command is blocked")
			}
			nodesByID, missing, err := loadHostNodesMap(deps, hostIDs)
			if err != nil {
				return nil, err
			}
			results := map[string]any{}
			succeeded := 0
			failed := 0
			for _, id := range hostIDs {
				key := strconv.FormatUint(id, 10)
				node, ok := nodesByID[id]
				if !ok {
					results[key] = map[string]any{"stdout": "", "stderr": "host not found", "exit_code": 1}
					failed++
					continue
				}
				out, execErr := executeHostCommand(deps, node, cmd)
				if execErr != nil {
					results[key] = map[string]any{"stdout": out, "stderr": execErr.Error(), "exit_code": 1}
					failed++
					continue
				}
				results[key] = map[string]any{"stdout": out, "stderr": "", "exit_code": 0}
				succeeded++
			}
			return &HostBatchExecApplyOutput{
				Reason:         strings.TrimSpace(input.Reason),
				Command:        cmd,
				CommandClass:   class,
				Risk:           risk,
				TargetCount:    len(hostIDs),
				MissingHostIDs: missing,
				Succeeded:      succeeded,
				Failed:         failed,
				Results:        results,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type HostBatchStatusUpdateOutput struct {
	Action       string `json:"action"`
	Reason       string `json:"reason"`
	TargetCount  int    `json:"target_count"`
	UpdatedCount int64  `json:"updated_count"`
}

func HostBatchStatusUpdate(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"host_batch_status_update",
		"Batch update host status to online, offline, or maintenance. host_ids (array) and action are required. Action must be one of: online, offline, maintenance. Use this to change the operational status of multiple hosts at once. Example: {\"host_ids\":[1,2,3],\"action\":\"maintenance\",\"reason\":\"scheduled maintenance\"}.",
		func(ctx context.Context, input *HostBatchStatusInput, opts ...tool.Option) (*HostBatchStatusUpdateOutput, error) {
			hostIDs, err := normalizeHostIDs(input.HostIDs)
			if err != nil {
				return nil, err
			}
			action := strings.ToLower(strings.TrimSpace(input.Action))
			if action == "" {
				return nil, fmt.Errorf("action is required")
			}
			if action != "online" && action != "offline" && action != "maintenance" {
				return nil, fmt.Errorf("action must be online/offline/maintenance")
			}
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			res := deps.DB.Model(&model.Node{}).Where("id IN ?", hostIDs).Update("status", action)
			if res.Error != nil {
				return nil, res.Error
			}
			return &HostBatchStatusUpdateOutput{
				Action:       action,
				Reason:       strings.TrimSpace(input.Reason),
				TargetCount:  len(hostIDs),
				UpdatedCount: res.RowsAffected,
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

func OSGetCPUMem(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"os_get_cpu_mem",
		"Get CPU, memory and load average information from a target host. Returns loadavg from /proc/loadavg, meminfo from /proc/meminfo, and uptime output. Target can be host ID, IP address, hostname, or 'localhost' (default) for local execution. Example: {\"target\":\"10.0.0.5\"}.",
		func(ctx context.Context, input *OSCPUMemInput, opts ...tool.Option) (*OSGetCPUMemOutput, error) {
			target := strings.TrimSpace(input.Target)
			loadavg, _, _ := runOnTarget(ctx, deps, target, "cat", []string{"/proc/loadavg"}, "cat /proc/loadavg")
			mem, _, err := runOnTarget(ctx, deps, target, "cat", []string{"/proc/meminfo"}, "cat /proc/meminfo")
			if err != nil {
				return nil, err
			}
			uptime, _, _ := runOnTarget(ctx, deps, target, "uptime", nil, "uptime")
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

func OSGetDiskFS(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"os_get_disk_fs",
		"Get disk and filesystem usage information using 'df -h' command. Shows mounted filesystems, total size, used space, available space, and mount points. Target can be host ID, IP address, hostname, or 'localhost' (default). Example: {\"target\":\"web-server-01\"}.",
		func(ctx context.Context, input *OSDiskInput, opts ...tool.Option) (*OSGetDiskFSOutput, error) {
			target := strings.TrimSpace(input.Target)
			out, _, err := runOnTarget(ctx, deps, target, "df", []string{"-h"}, "df -h")
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

func OSGetNetStat(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"os_get_net_stat",
		"Get network statistics including network device traffic from /proc/net/dev and listening TCP ports using 'ss -ltn'. Shows bytes sent/received per interface and all listening ports. Target can be host ID, IP address, hostname, or 'localhost' (default). Example: {\"target\":\"192.168.1.10\"}.",
		func(ctx context.Context, input *OSNetInput, opts ...tool.Option) (*OSGetNetStatOutput, error) {
			target := strings.TrimSpace(input.Target)
			dev, _, err := runOnTarget(ctx, deps, target, "cat", []string{"/proc/net/dev"}, "cat /proc/net/dev")
			if err != nil {
				return nil, err
			}
			listen, _, _ := runOnTarget(ctx, deps, target, "ss", []string{"-ltn"}, "ss -ltn")
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

func OSGetProcessTop(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
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
			out, _, err := runOnTarget(ctx, deps, target, "ps", []string{"aux", "--sort=-%cpu"}, cmd)
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

func OSGetJournalTail(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
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
			out, _, err := runOnTarget(ctx, deps, target, "journalctl", localArgs, remoteCmd)
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

func OSGetContainerRuntime(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"os_get_container_runtime",
		"Get container runtime information and running containers. Detects Docker or containerd. For Docker, runs 'docker ps' to show container ID, image, and status. For containerd, runs 'ctr -n k8s.io containers list'. Target can be host ID, IP address, hostname, or 'localhost' (default). Example: {\"target\":\"node-01\"}.",
		func(ctx context.Context, input *OSContainerRuntimeInput, opts ...tool.Option) (*OSGetContainerRuntimeOutput, error) {
			target := strings.TrimSpace(input.Target)
			out, _, err := runOnTarget(ctx, deps, target, "docker", []string{"ps", "--format", "{{.ID}} {{.Image}} {{.Status}}"}, "docker ps --format '{{.ID}} {{.Image}} {{.Status}}'")
			if err == nil {
				return &OSGetContainerRuntimeOutput{
					Runtime:    "docker",
					Containers: out,
				}, nil
			}
			out2, _, err2 := runOnTarget(ctx, deps, target, "ctr", []string{"-n", "k8s.io", "containers", "list"}, "ctr -n k8s.io containers list")
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

func executeHostCommand(deps core.PlatformDeps, node *model.Node, command string) (string, error) {
	privateKey, passphrase, err := loadNodePrivateKey(deps, node)
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

func loadNodePrivateKey(deps core.PlatformDeps, node *model.Node) (string, string, error) {
	if deps.DB == nil || node == nil || node.SSHKeyID == nil {
		return "", "", nil
	}
	var key model.SSHKey
	if err := deps.DB.Select("id", "private_key", "passphrase", "encrypted").Where("id = ?", uint64(*node.SSHKeyID)).First(&key).Error; err != nil {
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

func normalizeHostIDs(raw []int) ([]uint64, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("host_ids is required")
	}
	uniq := map[uint64]struct{}{}
	out := make([]uint64, 0, len(raw))
	for _, id := range raw {
		if id <= 0 {
			continue
		}
		uid := uint64(id)
		if _, ok := uniq[uid]; ok {
			continue
		}
		uniq[uid] = struct{}{}
		out = append(out, uid)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("host_ids must contain at least one positive id")
	}
	return out, nil
}

func loadHostNodesMap(deps core.PlatformDeps, hostIDs []uint64) (map[uint64]*model.Node, []uint64, error) {
	if deps.DB == nil {
		return nil, nil, fmt.Errorf("db unavailable")
	}
	var nodes []model.Node
	if err := deps.DB.Where("id IN ?", hostIDs).Find(&nodes).Error; err != nil {
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

func loadHostBatchTargets(deps core.PlatformDeps, hostIDs []uint64) ([]map[string]any, []int, error) {
	byID, _, err := loadHostNodesMap(deps, hostIDs)
	if err != nil {
		return nil, nil, err
	}
	targets := make([]map[string]any, 0, len(byID))
	missing := make([]int, 0)
	for _, id := range hostIDs {
		node, ok := byID[id]
		if !ok {
			missing = append(missing, int(id))
			continue
		}
		targets = append(targets, map[string]any{
			"id":       id,
			"name":     node.Name,
			"ip":       node.IP,
			"status":   node.Status,
			"hostname": node.Hostname,
		})
	}
	return targets, missing, nil
}

func classifyHostCommand(cmd string) (class string, risk string, blocked bool) {
	trimmed := strings.ToLower(strings.TrimSpace(cmd))
	if isReadonlyHostCommand(cmd) {
		return "readonly", "low", false
	}
	dangerous := []string{
		"rm -rf /", "mkfs", "shutdown", "poweroff", "reboot", "init 0",
		"dd if=", "iptables -f", "userdel", "chown -r /", "chmod -r 777 /",
	}
	for _, keyword := range dangerous {
		if strings.Contains(trimmed, keyword) {
			return "dangerous", "high", true
		}
	}
	return "mutating", "medium", false
}

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

func isReadonlyHostCommand(cmd string) bool {
	switch strings.TrimSpace(cmd) {
	case "hostname", "uptime", "df -h", "free -m", "ps aux --sort=-%cpu":
		return true
	default:
		return false
	}
}

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
