// Package host 提供主机运维相关的工具实现。
//
// 本文件实现主机操作工具集，包括：
//   - SSH 命令执行（host_exec，需审批）
//   - 主机清单查询（host_list_inventory）
//
// 注意：os_* 诊断工具已移除，agent 应通过 host-diagnostic skill + host_exec 进行诊断。
package host

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	common "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
	hostpolicy "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/hostpolicy"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/toolutil"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginlogic "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/logic"
	opsagentlogic "github.com/cy77cc/OpsPilot/internal/modules/opsagent/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

func serviceContextFromRuntime(ctx context.Context) *svc.ServiceContext {
	return toolutil.ServiceContextFromRuntime(ctx)
}

// =============================================================================
// 输入类型定义
// =============================================================================

// HostExecInput 主机命令执行输入。
type HostExecInput struct {
	HostID  int    `json:"host_id" jsonschema_description:"optional host id"`
	Target  string `json:"target,omitempty" jsonschema_description:"optional target host id/ip/hostname"`
	Command string `json:"command,omitempty" jsonschema_description:"optional,readonly command"`
	Script  string `json:"script,omitempty" jsonschema_description:"optional,script command"`
}

// HostInventoryInput 主机清单查询输入。
type HostInventoryInput struct {
	Status  string `json:"status,omitempty" jsonschema_description:"optional host status filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on name/ip/hostname"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max hosts,default=50"`
}

// =============================================================================
// 工具入口
// =============================================================================

// NewHostTools 创建主机工具集。
func NewHostTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		HostListInventory(ctx),
		HostExec(ctx),
	}
}

// NewHostReadonlyTools 创建主机只读工具子集。
//
// 注意：delegated specialist 必须保持只读，仅包含清单查询。
func NewHostReadonlyTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		HostListInventory(ctx),
	}
}

type CatalogMetadata struct {
	ToolName         string
	Domain           string
	Capability       string
	RiskLevel        string
	OutputMode       string
	Description      string
	DirectlyCallable bool
	AccessPath       string
}

// CatalogMetadataList returns host tool metadata for search-first indexing.
func CatalogMetadataList() []CatalogMetadata {
	return []CatalogMetadata{
		{ToolName: "host_exec", Domain: "host", Capability: "command_execution", RiskLevel: "high", OutputMode: "inline", Description: "execute a command on a host", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
		{ToolName: "host_list_inventory", Domain: "host", Capability: "listing", RiskLevel: "low", OutputMode: "inline", Description: "list hosts in inventory", DirectlyCallable: false, AccessPath: "specialist_or_runtime_dispatch"},
	}
}

type HostExecOutput struct {
	HostID   int    `json:"host_id"`
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Status   string `json:"status,omitempty"`

	PolicyDecision string                       `json:"policy_decision,omitempty"`
	PolicyReasons  []string                     `json:"policy_reasons,omitempty"`
	Violations     []hostpolicy.PolicyViolation `json:"violations,omitempty"`
}

func HostExec(ctx context.Context) tool.InvokableTool {
	svcCtx := serviceContextFromRuntime(ctx)
	t, err := einoutils.InferOptionableTool(
		"host_exec",
		"Execute a command or script on a single host only when policy allows. Provide exactly one of command or script together with host_id or target (id/ip/hostname). Approval is enforced by the middleware interrupt flow.",
		func(ctx context.Context, input *HostExecInput, opts ...tool.Option) (*HostExecOutput, error) {
			hostID := input.HostID
			target := strings.TrimSpace(input.Target)
			cmd := strings.TrimSpace(input.Command)
			script := strings.TrimSpace(input.Script)
			if hostID <= 0 && target == "" {
				return nil, fmt.Errorf("host_id or target is required")
			}
			if (cmd == "" && script == "") || (cmd != "" && script != "") {
				return nil, fmt.Errorf("provide exactly one of command or script")
			}
			execText := cmd
			if execText == "" {
				execText = script
			}
			if target == "" {
				target = strconv.Itoa(hostID)
			}
			if script != "" {
				return runPolicyAwareExecScriptByTarget(ctx, svcCtx, "host_exec", target, script)
			}
			return runPolicyAwareExecByTarget(ctx, svcCtx, "host_exec", target, cmd)
		},
	)
	if err != nil {
		return toolutil.UnavailableInvokableTool("host_exec", err)
	}
	return t
}

func runPolicyAwareExecByTarget(ctx context.Context, svcCtx *svc.ServiceContext, toolName, target, cmd string) (*HostExecOutput, error) {
	engine := hostpolicy.NewHostCommandPolicyEngine(hostpolicy.DefaultReadonlyAllowlist())
	decision := engine.Evaluate(hostpolicy.PolicyInput{
		ToolName:   toolName,
		CommandRaw: cmd,
		Target:     strings.TrimSpace(target),
	})

	if decision.DecisionType != hostpolicy.DecisionAllowReadonlyExecute && !approvedHostResume(ctx) {
		return nil, fmt.Errorf(
			"approval required: decision=%s reasons=%v violations=%v",
			decision.DecisionType,
			decision.ReasonCodes,
			decision.Violations,
		)
	}

	node, err := resolveNodeByTarget(svcCtx, target)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("plugin required: localhost execution is not supported for host_exec")
	}

	instance, err := hostpluginlogic.NewService(svcCtx).RequireOnlineCapability(ctx, uint64(node.ID), "exec.shell")
	if err != nil {
		return nil, fmt.Errorf("plugin required: %w", err)
	}
	result, err := opsagentlogic.NewDispatcher(svcCtx).ExecuteCommand(ctx, instance, cmd)
	if err != nil {
		return nil, err
	}
	return &HostExecOutput{
		HostID:         int(node.ID),
		Command:        cmd,
		Stdout:         result.Stdout,
		Stderr:         result.Stderr,
		ExitCode:       result.ExitCode,
		Status:         "completed",
		PolicyDecision: string(decision.DecisionType),
		PolicyReasons:  decision.ReasonCodes,
		Violations:     decision.Violations,
	}, nil
}

func runPolicyAwareExecScriptByTarget(ctx context.Context, svcCtx *svc.ServiceContext, toolName, target, script string) (*HostExecOutput, error) {
	engine := hostpolicy.NewHostCommandPolicyEngine(hostpolicy.DefaultReadonlyAllowlist())
	decision := engine.Evaluate(hostpolicy.PolicyInput{
		ToolName:   toolName,
		CommandRaw: script,
		Target:     strings.TrimSpace(target),
	})
	if decision.DecisionType != hostpolicy.DecisionAllowReadonlyExecute && !approvedHostResume(ctx) {
		return nil, fmt.Errorf(
			"approval required: decision=%s reasons=%v violations=%v",
			decision.DecisionType,
			decision.ReasonCodes,
			decision.Violations,
		)
	}

	node, err := resolveNodeByTarget(svcCtx, target)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("plugin required: localhost execution is not supported for host_exec")
	}

	instance, err := hostpluginlogic.NewService(svcCtx).RequireOnlineCapability(ctx, uint64(node.ID), "exec.script.shell")
	if err != nil {
		return nil, fmt.Errorf("plugin required: %w", err)
	}
	result, err := opsagentlogic.NewDispatcher(svcCtx).ExecuteScript(ctx, instance, "sh", script)
	if err != nil {
		return nil, err
	}
	return &HostExecOutput{
		HostID:         int(node.ID),
		Command:        script,
		Stdout:         result.Stdout,
		Stderr:         result.Stderr,
		ExitCode:       result.ExitCode,
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
			query := svcCtx.DB.Model(&hostmodel.Node{})
			if status := strings.TrimSpace(input.Status); status != "" {
				query = query.Where("status = ?", status)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + toolutil.EscapeLikePattern(kw) + "%"
				query = query.Where("name LIKE ? ESCAPE '\\' OR ip LIKE ? ESCAPE '\\' OR hostname LIKE ? ESCAPE '\\'", pattern, pattern, pattern)
			}
			var nodes []hostmodel.Node
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
		return toolutil.UnavailableInvokableTool("host_list_inventory", err)
	}
	return t
}

// =============================================================================
// 辅助函数
// =============================================================================

// executeHostCommand 在指定主机上执行命令。
func executeHostCommand(svcCtx *svc.ServiceContext, node *hostmodel.Node, command string) (string, error) {
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
func loadNodePrivateKey(svcCtx *svc.ServiceContext, node *hostmodel.Node) (string, string, error) {
	if svcCtx.DB == nil || node == nil || node.SSHKeyID == nil {
		return "", "", nil
	}
	var key hostmodel.SSHKey
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

// parseHostLabels 解析主机标签字符串。
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

// detectNodeAuthType 检测节点的认证类型。
func detectNodeAuthType(node *hostmodel.Node) string {
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
