package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	sshclient "github.com/cy77cc/k8s-manage/internal/client/ssh"
	"github.com/cy77cc/k8s-manage/internal/model"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// runWithPolicyAndEvent 执行工具操作并处理相关的策略检查、事件发送和错误处理
//
// 参数:
//   - ctx: 上下文对象，用于传递请求范围的值和控制取消
//   - meta: 工具元数据，包含工具名称、提供者等信息
//   - input: 工具执行的输入数据
//   - do: 实际执行工具逻辑的函数
//
// 返回值:
//   - ToolResult: 工具执行结果
//   - error: 执行过程中的错误
func runWithPolicyAndEvent[T any](ctx context.Context, meta ToolMeta, input T, do func(T) (any, string, error)) (ToolResult, error) {
	// 记录开始时间，用于计算执行延迟
	start := time.Now()

	// 将输入结构体转换为映射
	originalParams := structToMap(input)

	// 解析工具参数
	resolvedParams, resolution := resolveToolParams(ctx, meta, originalParams, "")

	// 在执行前先做通用参数校验，提前返回更友好的提示
	if err := validateResolvedParams(meta, resolvedParams); err != nil {
		res := ToolResult{
			OK:        false,
			ErrorCode: "invalid_param",
			Error:     err.Error(),
			Source:    meta.Provider,
			LatencyMS: time.Since(start).Milliseconds(),
		}
		EmitToolEvent(ctx, "tool_result", map[string]any{
			"tool":             meta.Name,
			"call_id":          nextToolCallID(),
			"result":           res,
			"param_resolution": resolution,
			"retry":            false,
		})
		return res, nil
	}

	// 将解析后的参数转换回输入类型
	runtimeInput, convErr := mapToInput[T](resolvedParams)

	// 检查参数转换是否成功
	if convErr != nil {
		// 构建参数错误结果
		res := ToolResult{
			OK:        false,
			ErrorCode: "invalid_param",
			Error:     convErr.Error(),
			Source:    meta.Provider,
			LatencyMS: time.Since(start).Milliseconds(),
		}

		// 发送工具结果事件
		EmitToolEvent(ctx, "tool_result", map[string]any{
			"tool":             meta.Name,
			"call_id":          nextToolCallID(),
			"result":           res,
			"param_resolution": resolution,
			"retry":            false,
		})

		return res, nil
	}

	// 生成调用ID
	callID := nextToolCallID()

	// 发送工具调用事件
	EmitToolEvent(ctx, "tool_call", map[string]any{
		"tool":             meta.Name,
		"call_id":          callID,
		"params":           resolvedParams,
		"param_resolution": resolution,
		"retry":            false,
	})

	// 检查工具执行是否符合策略要求
	if err := CheckToolPolicy(ctx, meta, resolvedParams); err != nil {
		errCode := "policy_denied"
		if _, ok := IsApprovalRequired(err); ok {
			errCode = "approval_required"
		} else if _, ok := IsConfirmationRequired(err); ok {
			errCode = "confirmation_required"
		}
		emitPolicyRequiredEvent(ctx, meta, err)

		// 构建策略拒绝结果
		res := ToolResult{
			OK:        false,
			ErrorCode: errCode,
			Error:     err.Error(),
			Source:    meta.Provider,
			LatencyMS: time.Since(start).Milliseconds(),
		}

		// 发送工具结果事件
		EmitToolEvent(ctx, "tool_result", map[string]any{
			"tool":             meta.Name,
			"call_id":          callID,
			"result":           res,
			"param_resolution": resolution,
			"retry":            false,
		})

		if _, ok := IsApprovalRequired(err); ok {
			return res, nil
		}
		if _, ok := IsConfirmationRequired(err); ok {
			return res, nil
		}
		return res, err
	}

	// 执行工具逻辑
	data, source, err := runToolAttempt(ctx, meta, runtimeInput, do)

	// 标记是否进行了重试
	retried := false

	// 处理执行错误
	if err != nil {
		// 检查是否为参数缺失错误
		if ie, ok := AsToolInputError(err); ok && ie.Code == "missing_param" {
			// 构建首次尝试的结果（在重试前）
			firstRes := ToolResult{
				OK:        false,
				ErrorCode: ie.Code,
				Error:     err.Error(),
				Source:    firstNonEmpty(source, meta.Provider),
				LatencyMS: time.Since(start).Milliseconds(),
			}

			// 发送首次尝试的结果事件
			EmitToolEvent(ctx, "tool_result", map[string]any{
				"tool":             meta.Name,
				"call_id":          callID,
				"result":           firstRes,
				"param_resolution": resolution,
				"retry":            true,
			})

			// 标记进行了重试
			retried = true

			// 重新解析参数（针对缺失的参数）
			resolvedRetry, resolutionRetry := resolveToolParams(ctx, meta, resolvedParams, ie.Field)

			// 检查参数是否有变化
			if !equalJSONMap(resolvedRetry, resolvedParams) {
				// 更新参数和解析结果
				resolvedParams = resolvedRetry
				runtimeInput, convErr = mapToInput[T](resolvedParams)

				// 检查参数转换是否成功
				if convErr == nil {
					// 发送重试的工具调用事件
					EmitToolEvent(ctx, "tool_call", map[string]any{
						"tool":             meta.Name,
						"call_id":          callID,
						"params":           resolvedParams,
						"param_resolution": resolutionRetry,
						"retry":            true,
					})

					// 再次检查策略
					if err := CheckToolPolicy(ctx, meta, resolvedParams); err != nil {
						errCode := "policy_denied"
						if _, ok := IsApprovalRequired(err); ok {
							errCode = "approval_required"
						} else if _, ok := IsConfirmationRequired(err); ok {
							errCode = "confirmation_required"
						}
						emitPolicyRequiredEvent(ctx, meta, err)

						// 构建策略拒绝结果
						res := ToolResult{
							OK:        false,
							ErrorCode: errCode,
							Error:     err.Error(),
							Source:    meta.Provider,
							LatencyMS: time.Since(start).Milliseconds(),
						}

						// 发送工具结果事件
						EmitToolEvent(ctx, "tool_result", map[string]any{
							"tool":             meta.Name,
							"call_id":          callID,
							"result":           res,
							"param_resolution": resolutionRetry,
							"retry":            true,
						})

						if _, ok := IsApprovalRequired(err); ok {
							return res, nil
						}
						if _, ok := IsConfirmationRequired(err); ok {
							return res, nil
						}
						return res, err
					}

					// 再次执行工具逻辑
					data, source, err = runToolAttempt(ctx, meta, runtimeInput, do)

					// 更新解析结果
					resolution = resolutionRetry
				}
			}
		}
	}

	// 对可恢复错误执行一次智能重试
	if err != nil && shouldRetryTool(meta, err) {
		retried = true
		EmitToolEvent(ctx, "tool_call", map[string]any{
			"tool":             meta.Name,
			"call_id":          callID,
			"params":           resolvedParams,
			"param_resolution": resolution,
			"retry":            true,
		})
		time.Sleep(120 * time.Millisecond)
		data, source, err = runToolAttempt(ctx, meta, runtimeInput, do)
	}

	// 如果没有指定源，则使用工具提供者作为源
	if source == "" {
		source = meta.Provider
	}

	// 处理最终错误
	if err != nil {
		// 确定错误代码
		errCode := "tool_error"
		if ie, ok := AsToolInputError(err); ok {
			errCode = ie.Code
		}
		execErr := buildToolExecutionError(meta, errCode, err.Error())

		// 构建错误结果
		res := ToolResult{OK: false, ErrorCode: errCode, Error: err.Error(), Source: source, LatencyMS: time.Since(start).Milliseconds()}

		// 发送工具结果事件
		EmitToolEvent(ctx, "tool_result", map[string]any{
			"tool":             meta.Name,
			"call_id":          callID,
			"result":           res,
			"execution_error":  execErr,
			"param_resolution": resolution,
			"retry":            retried,
		})

		return res, nil
	}

	// 构建成功结果
	res := ToolResult{OK: true, Data: data, Source: source, LatencyMS: time.Since(start).Milliseconds()}

	// 发送工具结果事件
	EmitToolEvent(ctx, "tool_result", map[string]any{
		"tool":             meta.Name,
		"call_id":          callID,
		"result":           res,
		"params":           resolvedParams,
		"param_resolution": resolution,
		"retry":            retried,
	})

	// 如果执行成功，存储最后使用的工具参数到工具内存中
	if mem := ToolMemoryAccessorFromContext(ctx); mem != nil && res.OK {
		mem.SetLastToolParams(meta.Name, resolvedParams)
	}

	return res, nil
}

func emitPolicyRequiredEvent(ctx context.Context, meta ToolMeta, err error) {
	if apErr, ok := IsApprovalRequired(err); ok {
		EmitToolEvent(ctx, "approval_required", map[string]any{
			"tool":           meta.Name,
			"approval_token": apErr.Token,
			"expiresAt":      apErr.ExpiresAt,
			"message":        apErr.Error(),
		})
		return
	}
	if cfErr, ok := IsConfirmationRequired(err); ok {
		EmitToolEvent(ctx, "confirmation_required", map[string]any{
			"tool":               meta.Name,
			"confirmation_token": cfErr.Token,
			"expiresAt":          cfErr.ExpiresAt,
			"preview":            cfErr.Preview,
			"message":            cfErr.Error(),
		})
	}
}

func shouldRetryTool(meta ToolMeta, err error) bool {
	if err == nil {
		return false
	}
	ie, ok := AsToolInputError(err)
	if !ok {
		return false
	}
	// 仅对只读工具的临时性错误重试
	if meta.Mode != ToolModeReadonly {
		return false
	}
	return ie.Code == "tool_timeout" || ie.Code == "tool_error" || ie.Code == "tool_canceled"
}

func buildToolExecutionError(meta ToolMeta, code, message string) ToolExecutionError {
	out := ToolExecutionError{
		Code:        strings.TrimSpace(code),
		Message:     strings.TrimSpace(message),
		Recoverable: false,
	}
	switch out.Code {
	case "missing_param":
		out.Recoverable = true
		out.HintAction = "补充参数后重试"
		out.Suggestions = []string{
			"确认必填参数是否已提供",
			"优先调用对应 inventory/list 工具查询可用 ID",
		}
	case "invalid_param":
		out.Recoverable = true
		out.HintAction = "修正参数格式后重试"
		out.Suggestions = []string{
			"检查参数类型与枚举范围",
			"使用参数提示接口查看可选值",
		}
	case "policy_denied":
		out.HintAction = "申请权限或切换只读操作"
		out.Suggestions = []string{"当前工具受策略限制，请使用低风险替代工具"}
	case "tool_timeout", "tool_canceled":
		out.Recoverable = true
		out.HintAction = "缩小查询范围后重试"
		out.Suggestions = []string{"减少 limit 或时间范围", "确认目标系统连接状态"}
	default:
		out.Recoverable = true
		out.HintAction = "稍后重试或切换替代工具"
		out.Suggestions = []string{"可先执行只读诊断工具确认资源状态"}
	}
	if meta.Mode == ToolModeMutating {
		out.Suggestions = append(out.Suggestions, "变更类工具请确认审批令牌有效")
	}
	return out
}

// runToolAttempt 执行工具操作的尝试，包含超时控制和异常捕获
//
// 参数:
//   - ctx: 上下文对象，用于传递请求范围的值和控制取消
//   - meta: 工具元数据，包含工具名称、提供者、模式等信息
//   - input: 工具执行的输入数据
//   - do: 实际执行工具逻辑的函数
//
// 返回值:
//   - data: 工具执行的结果数据
//   - source: 结果的来源
//   - err: 执行过程中的错误
func runToolAttempt[T any](ctx context.Context, meta ToolMeta, input T, do func(T) (any, string, error)) (data any, source string, err error) {
	// 设置超时时间，根据工具模式不同设置不同的超时
	// 普通工具超时时间为8秒，修改型工具超时时间为20秒
	timeout := 8 * time.Second
	if meta.Mode == ToolModeMutating {
		timeout = 20 * time.Second
	}

	// 创建带超时的上下文
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 定义尝试执行的输出结构
	type attemptOut struct {
		data   any    // 执行结果数据
		source string // 结果来源
		err    error  // 执行错误
	}

	// 创建通道用于接收执行结果
	ch := make(chan attemptOut, 1)

	// 在goroutine中执行工具操作
	go func() {
		// 捕获执行过程中的panic
		defer func() {
			if r := recover(); r != nil {
				// 将panic转换为工具输入错误
				ch <- attemptOut{err: &ToolInputError{Code: "tool_panic", Message: fmt.Sprintf("tool panic: %v", r)}}
			}
		}()

		// 执行工具逻辑
		d, s, e := do(input)

		// 将执行结果发送到通道
		ch <- attemptOut{data: d, source: s, err: e}
	}()

	// 等待执行结果或超时
	select {
	case <-attemptCtx.Done():
		// 处理超时或取消情况
		if errors.Is(attemptCtx.Err(), context.DeadlineExceeded) {
			// 超时错误
			return nil, meta.Provider, &ToolInputError{Code: "tool_timeout", Message: "tool execution timeout"}
		}
		// 取消错误
		return nil, meta.Provider, &ToolInputError{Code: "tool_canceled", Message: "tool execution canceled"}
	case out := <-ch:
		// 处理执行结果
		if out.source == "" {
			// 如果没有指定源，则使用工具提供者作为源
			out.source = meta.Provider
		}

		if out.err == nil {
			// 执行成功
			return out.data, out.source, nil
		}

		// 执行失败，标准化错误
		return out.data, out.source, normalizeToolErr(out.err)
	}
}

func normalizeToolErr(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := AsToolInputError(err); ok {
		return err
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "timeout"):
		return &ToolInputError{Code: "tool_timeout", Message: err.Error()}
	case strings.Contains(msg, "canceled"):
		return &ToolInputError{Code: "tool_canceled", Message: err.Error()}
	default:
		return err
	}
}

func firstNonEmpty(v ...string) string {
	for _, item := range v {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return ""
}

// resolveK8sClient 解析 Kubernetes 客户端，根据参数和依赖项选择合适的客户端
//
// 参数:
//   - deps: 平台依赖项，包含数据库连接和默认客户端等
//   - params: 参数字典，可能包含 cluster_id 等信息
//
// 返回值:
//   - *kubernetes.Clientset: Kubernetes 客户端实例
//   - string: 客户端来源标识
//   - error: 解析过程中的错误
func resolveK8sClient(deps PlatformDeps, params map[string]any) (*kubernetes.Clientset, string, error) {
	// 从参数中获取 cluster_id 并转换为整数
	clusterID := toInt(params["cluster_id"])

	// 首先尝试从数据库中获取指定集群的客户端
	if clusterID > 0 && deps.DB != nil {
		var cluster model.Cluster
		// 从数据库中查询集群信息
		if err := deps.DB.First(&cluster, clusterID).Error; err == nil && strings.TrimSpace(cluster.KubeConfig) != "" {
			// 使用集群的 KubeConfig 创建 REST 配置
			cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
			if err != nil {
				return nil, "cluster_kubeconfig", err
			}

			// 使用 REST 配置创建 Kubernetes 客户端
			cli, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return nil, "cluster_kubeconfig", err
			}

			// 返回从集群配置创建的客户端
			return cli, "cluster_kubeconfig", nil
		}
	}

	// 如果没有指定集群或获取失败，尝试使用默认客户端
	if deps.Clientset != nil {
		return deps.Clientset, "default_clientset", nil
	}

	// 如果所有尝试都失败，返回错误
	return nil, "fallback", errors.New("k8s client unavailable")
}

func runLocalCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	out, err := cmd.CombinedOutput()
	if cctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), errors.New("command timeout")
	}
	return strings.TrimSpace(string(out)), err
}

// runOnTarget 在指定目标上执行命令，支持本地和远程执行
//
// 参数:
//   - ctx: 上下文，用于取消操作
//   - deps: 平台依赖项，包含数据库连接和默认客户端等
//   - target: 目标节点标识，支持本地执行（localhost 或空字符串）或节点 ID/IP/名称/主机名
//   - localName: 本地执行命令名称（如 "ssh"）
//   - localArgs: 本地执行命令参数（如 "-p 2222 user@host"）
//   - remoteCmd: 远程执行的 shell 命令（如 "kubectl get nodes"）
//
// 返回值:
//   - string: 命令执行输出
//   - string: 执行来源标识（"local" 或 "remote_ssh"）
//   - error: 执行过程中的错误
func runOnTarget(ctx context.Context, deps PlatformDeps, target, localName string, localArgs []string, remoteCmd string) (string, string, error) {
	node, err := resolveNodeByTarget(deps, target)
	if err != nil {
		return "", "target_check", err
	}
	if node == nil {
		out, err := runLocalCommand(ctx, 6*time.Second, localName, localArgs...)
		return out, "local", err
	}
	privateKey, passphrase, err := loadNodePrivateKey(deps, node)
	if err != nil {
		return "", "remote_ssh_credential", err
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		return "", "remote_ssh", err
	}
	defer cli.Close()
	out, err := sshclient.RunCommand(cli, remoteCmd)
	return out, "remote_ssh", err
}

func resolveNodeByTarget(deps PlatformDeps, target string) (*model.Node, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || trimmed == "localhost" {
		return nil, nil
	}
	if deps.DB == nil {
		return nil, errors.New("db unavailable")
	}
	var node model.Node
	if id, err := strconv.ParseUint(trimmed, 10, 64); err == nil {
		if err := deps.DB.First(&node, id).Error; err == nil {
			return &node, nil
		}
	}
	if err := deps.DB.Where("ip = ? OR name = ? OR hostname = ?", trimmed, trimmed, trimmed).First(&node).Error; err != nil {
		return nil, errors.New("target not in host whitelist")
	}
	return &node, nil
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case uint64:
		return int(x)
	case json.Number:
		n, _ := strconv.Atoi(x.String())
		return n
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	default:
		return 0
	}
}

func structToMap(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func mapToInput[T any](input map[string]any) (T, error) {
	var out T
	raw, err := json.Marshal(input)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

func equalJSONMap(a, b map[string]any) bool {
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	return string(ra) == string(rb)
}
