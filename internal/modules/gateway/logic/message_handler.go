// Package logic 提供 gateway 模块的消息处理逻辑。
//
// 本文件实现 gateway 消息处理函数，处理来自 gateway agent (B) 的消息。
package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	gatewaymodel "github.com/cy77cc/OpsPilot/internal/modules/gateway/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	pb "github.com/cy77cc/OpsPilot/proto"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// ExecResultHandler 是处理执行结果的回调函数类型。
// 用于避免 gateway 与 opsagent 之间的循环依赖。
type ExecResultHandler func(taskID string, result *pb.ExecResult)

// ExecOutputHandler 是处理执行输出的回调函数类型。
type ExecOutputHandler func(output *pb.ExecOutput)

// MessageHandler 处理 gateway 模块的消息。
type MessageHandler struct {
	svcCtx         *svc.ServiceContext
	tunnelManager  *TunnelManager
	routeTable     *RouteTable
	onExecResult   ExecResultHandler
	onExecOutput   ExecOutputHandler
}

// NewMessageHandler 创建消息处理器实例。
func NewMessageHandler(
	svcCtx *svc.ServiceContext,
	tunnelManager *TunnelManager,
	routeTable *RouteTable,
	onExecResult ExecResultHandler,
	onExecOutput ExecOutputHandler,
) *MessageHandler {
	return &MessageHandler{
		svcCtx:       svcCtx,
		tunnelManager: tunnelManager,
		routeTable:    routeTable,
		onExecResult:  onExecResult,
		onExecOutput:  onExecOutput,
	}
}

func (h *MessageHandler) db() *gorm.DB {
	if h == nil || h.svcCtx == nil {
		return nil
	}
	return h.svcCtx.DB
}

// HandleTunnelOpen 处理隧道建立消息 (B -> platform)。
//
// 1. 通过 agent_id 查找 host_plugin_instance 获取 hostID
// 2. 在 TunnelManager 中注册隧道会话
// 3. 更新路由表绑定 tunnel_id 到主机
// 4. 标记主机为在线
func (h *MessageHandler) HandleTunnelOpen(ctx context.Context, open *pb.TunnelOpen) error {
	if open == nil {
		return fmt.Errorf("tunnel open message is nil")
	}

	db := h.db()
	if db == nil {
		return fmt.Errorf("gateway service context requires db")
	}

	agentID := strings.TrimSpace(open.GetAgentId())
	if agentID == "" {
		return fmt.Errorf("tunnel open agent_id is required")
	}

	// 1. 查找 host_plugin_instance
	var instance hostpluginmodel.HostPluginInstance
	err := db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		First(&instance).Error
	if err != nil {
		return fmt.Errorf("lookup host_plugin_instance by agent_id %q: %w", agentID, err)
	}

	hostID := instance.HostID
	tunnelID := strings.TrimSpace(open.GetTunnelId())
	if tunnelID == "" {
		return fmt.Errorf("tunnel open tunnel_id is required")
	}

	// 2. 注册隧道会话
	h.tunnelManager.Open(tunnelID, agentID, 0, hostID)

	// 3. 更新路由表
	route := gatewaymodel.HostRoute{
		HostID:   hostID,
		TunnelID: tunnelID,
		Mode:     "tunnel",
		Direct:   false,
	}
	if err := h.routeTable.Set(route); err != nil {
		h.tunnelManager.Close(tunnelID, "route_update_failed")
		return fmt.Errorf("update route table: %w", err)
	}

	// 4. 持久化隧道会话到 DB
	now := time.Now()
	dbSession := gatewaymodel.TunnelSession{
		TunnelID:  tunnelID,
		HostID:    hostID,
		AgentID:   agentID,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.WithContext(ctx).Create(&dbSession).Error; err != nil {
		h.tunnelManager.Close(tunnelID, "db_persist_failed")
		return fmt.Errorf("persist tunnel session: %w", err)
	}

	// 5. 标记主机实例为在线
	if err := db.WithContext(ctx).
		Model(&hostpluginmodel.HostPluginInstance{}).
		Where("id = ?", instance.ID).
		Updates(map[string]any{
			"runtime_status": "online",
			"last_seen_at":   &now,
		}).Error; err != nil {
		return fmt.Errorf("update instance runtime_status: %w", err)
	}

	return nil
}

// HandleTunnelData 处理隧道数据转发 (B -> platform -> 解析为 AgentMessage)。
//
// 1. 通过 tunnel_id 查找隧道会话
// 2. 将 payload 反序列化为 AgentMessage
// 3. 路由内部消息到 handleProxiedAgentMessage
func (h *MessageHandler) HandleTunnelData(ctx context.Context, data *pb.TunnelData) error {
	if data == nil {
		return fmt.Errorf("tunnel data message is nil")
	}

	tunnelID := strings.TrimSpace(data.GetTunnelId())
	if tunnelID == "" {
		return fmt.Errorf("tunnel data tunnel_id is required")
	}

	// 1. 查找隧道会话
	session, ok := h.tunnelManager.Get(tunnelID)
	if !ok {
		return fmt.Errorf("tunnel session %q not found", tunnelID)
	}

	// 2. 反序列化 payload 为 AgentMessage
	payload := data.GetPayload()
	if len(payload) == 0 {
		return nil // 空 payload，忽略
	}

	var innerMsg pb.AgentMessage
	if err := proto.Unmarshal(payload, &innerMsg); err != nil {
		return fmt.Errorf("unmarshal tunneled agent message: %w", err)
	}

	// 3. 路由到内部消息处理
	return h.handleProxiedAgentMessage(ctx, session.HostID, &innerMsg)
}

// HandleTunnelClose 处理隧道关闭消息 (B -> platform)。
//
// 1. 查找隧道会话
// 2. 在 TunnelManager 中关闭
// 3. 标记主机为离线
// 4. 清理 DB 隧道会话记录
func (h *MessageHandler) HandleTunnelClose(ctx context.Context, close *pb.TunnelClose) error {
	if close == nil {
		return fmt.Errorf("tunnel close message is nil")
	}

	tunnelID := strings.TrimSpace(close.GetTunnelId())
	if tunnelID == "" {
		return fmt.Errorf("tunnel close tunnel_id is required")
	}

	reason := close.GetReason()

	// 1. 查找隧道会话获取 hostID
	session, ok := h.tunnelManager.Get(tunnelID)
	var hostID uint64
	if ok {
		hostID = session.HostID
	}

	// 2. 关闭隧道
	h.tunnelManager.Close(tunnelID, reason)

	// 3. 标记主机为离线（如果已知 hostID）
	db := h.db()
	if db != nil && hostID > 0 {
		now := time.Now()
		_ = db.WithContext(ctx).
			Model(&hostpluginmodel.HostPluginInstance{}).
			Where("host_id = ?", hostID).
			Updates(map[string]any{
				"runtime_status": "offline",
				"last_seen_at":   &now,
			}).Error
	}

	// 4. 清理 DB 隧道会话记录
	if db != nil {
		_ = db.WithContext(ctx).
			Model(&gatewaymodel.TunnelSession{}).
			Where("tunnel_id = ?", tunnelID).
			Updates(map[string]any{
				"status":     "closed",
				"updated_at": time.Now(),
			}).Error
	}

	return nil
}

// HandleProxyRegister 处理代理主机注册消息 (B -> platform)。
//
// 1. 解析 host_id
// 2. 标记主机为在线
func (h *MessageHandler) HandleProxyRegister(ctx context.Context, reg *pb.ProxyHostRegister) error {
	if reg == nil {
		return fmt.Errorf("proxy register message is nil")
	}

	hostIDStr := strings.TrimSpace(reg.GetHostId())
	if hostIDStr == "" {
		return fmt.Errorf("proxy register host_id is required")
	}

	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("parse proxy host_id %q: %w", hostIDStr, err)
	}

	db := h.db()
	if db == nil {
		return fmt.Errorf("gateway service context requires db")
	}

	// 标记主机实例为在线
	now := time.Now()
	if err := db.WithContext(ctx).
		Model(&hostpluginmodel.HostPluginInstance{}).
		Where("host_id = ?", hostID).
		Updates(map[string]any{
			"runtime_status": "online",
			"last_seen_at":   &now,
		}).Error; err != nil {
		return fmt.Errorf("update proxy host runtime_status: %w", err)
	}

	return nil
}

// HandleProxyResponse 处理代理命令执行结果 (B -> platform)。
//
// 1. 使用 proxy-{host_id}-{command} 格式构造 waiterKey
// 2. 转换为 ExecResult 并通过回调路由到 exec waiter
func (h *MessageHandler) HandleProxyResponse(resp *pb.ProxyCommandResponse) error {
	if resp == nil {
		return fmt.Errorf("proxy response message is nil")
	}

	hostID := strings.TrimSpace(resp.GetHostId())
	command := strings.TrimSpace(resp.GetCommand())
	if hostID == "" || command == "" {
		return fmt.Errorf("proxy response host_id and command are required")
	}

	// 构造 waiterKey，与 exec_dispatch.go 中的格式一致
	waiterKey := fmt.Sprintf("proxy-%s-%s", hostID, command)

	// 转换为 ExecResult
	execResult := &pb.ExecResult{
		TaskId:     waiterKey,
		ExitCode:   resp.GetExitCode(),
		DurationMs: resp.GetDurationMs(),
		TimedOut:   resp.GetTimedOut(),
	}

	// 通过回调路由到 exec waiter
	if h.onExecResult != nil {
		h.onExecResult(waiterKey, execResult)
	}

	return nil
}

// HandleProxyMetrics 处理代理主机指标 (B -> platform)。
//
// 1. 解析 host_id
// 2. 使用 NormalizeMetricBatch 处理指标
// 3. 持久化到 DB
func (h *MessageHandler) HandleProxyMetrics(ctx context.Context, batch *pb.ProxyMetricBatch) error {
	if batch == nil {
		return fmt.Errorf("proxy metrics message is nil")
	}

	hostIDStr := strings.TrimSpace(batch.GetHostId())
	if hostIDStr == "" {
		return fmt.Errorf("proxy metrics host_id is required")
	}

	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("parse proxy metrics host_id %q: %w", hostIDStr, err)
	}

	// 构造 MetricBatch 用于标准化处理
	metricBatch := &pb.MetricBatch{
		Metrics: batch.GetMetrics(),
	}

	// 标准化指标
	snapshot := NormalizeMetricBatch(hostID, metricBatch)

	// 持久化到 DB
	db := h.db()
	if db == nil {
		return fmt.Errorf("gateway service context requires db")
	}

	if err := db.WithContext(ctx).Create(snapshot).Error; err != nil {
		return fmt.Errorf("persist proxy metrics: %w", err)
	}

	// 更新主机健康状态
	if err := db.WithContext(ctx).Model(&hostmodel.Node{}).
		Where("id = ?", hostID).
		Updates(map[string]any{
			"health_state":  snapshot.State,
			"last_check_at": snapshot.CheckedAt,
		}).Error; err != nil {
		return fmt.Errorf("update host health state: %w", err)
	}

	return nil
}

// handleProxiedAgentMessage 处理通过隧道转发的 AgentMessage。
//
// - Registration: 更新实例 runtime_status 为 online
// - Heartbeat: 更新实例状态
// - Metrics: 通过 NormalizeMetricBatch 持久化
// - ExecOutput: 路由到 exec waiter
// - ExecResult: 路由到 exec waiter
// - Ack: 忽略
func (h *MessageHandler) handleProxiedAgentMessage(ctx context.Context, hostID uint64, msg *pb.AgentMessage) error {
	if msg == nil {
		return nil
	}

	db := h.db()

	switch payload := msg.GetPayload().(type) {
	case *pb.AgentMessage_Registration:
		// 更新实例 runtime_status 为 online
		if db == nil {
			return nil
		}
		now := time.Now()
		return db.WithContext(ctx).
			Model(&hostpluginmodel.HostPluginInstance{}).
			Where("host_id = ?", hostID).
			Updates(map[string]any{
				"runtime_status": "online",
				"last_seen_at":   &now,
			}).Error

	case *pb.AgentMessage_Heartbeat:
		// 更新实例状态
		if db == nil || payload.Heartbeat == nil {
			return nil
		}
		now := time.Now()
		if ts := payload.Heartbeat.GetTimestampMs(); ts > 0 {
			now = time.UnixMilli(ts)
		}
		return db.WithContext(ctx).
			Model(&hostpluginmodel.HostPluginInstance{}).
			Where("host_id = ?", hostID).
			Updates(map[string]any{
				"runtime_status": "online",
				"health_status":  "healthy",
				"last_seen_at":   &now,
			}).Error

	case *pb.AgentMessage_Metrics:
		// 持久化指标
		if db == nil || payload.Metrics == nil {
			return nil
		}
		snapshot := NormalizeMetricBatch(hostID, payload.Metrics)
		if err := db.WithContext(ctx).Create(snapshot).Error; err != nil {
			return err
		}
		return db.WithContext(ctx).Model(&hostmodel.Node{}).
			Where("id = ?", hostID).
			Updates(map[string]any{
				"health_state":  snapshot.State,
				"last_check_at": snapshot.CheckedAt,
			}).Error

	case *pb.AgentMessage_ExecOutput:
		// 路由到 exec waiter
		if h.onExecOutput != nil && payload.ExecOutput != nil {
			h.onExecOutput(payload.ExecOutput)
		}
		return nil

	case *pb.AgentMessage_ExecResult:
		// 路由到 exec waiter
		if h.onExecResult != nil && payload.ExecResult != nil {
			h.onExecResult(payload.ExecResult.GetTaskId(), payload.ExecResult)
		}
		return nil

	case *pb.AgentMessage_Ack:
		// 忽略
		return nil

	default:
		return nil
	}
}

// NormalizeMetricBatch 标准化指标批次，生成主机健康快照。
// 从 opsagent/logic/metrics_ingest.go 复制，避免循环依赖。
func NormalizeMetricBatch(hostID uint64, batch *pb.MetricBatch) *hostmodel.HostHealthSnapshot {
	snapshot := &hostmodel.HostHealthSnapshot{
		HostID:             hostID,
		State:              "healthy",
		ConnectivityStatus: "healthy",
		ResourceStatus:     "healthy",
		SystemStatus:       "healthy",
		CheckedAt:          time.Now(),
	}
	if batch == nil || len(batch.GetMetrics()) == 0 {
		return snapshot
	}

	summary := map[string]any{
		"metric_count": len(batch.GetMetrics()),
	}

	for _, metric := range batch.GetMetrics() {
		if metric == nil {
			continue
		}
		if ts := metric.GetTimestampMs(); ts > 0 {
			checkedAt := time.UnixMilli(ts)
			if checkedAt.After(snapshot.CheckedAt) {
				snapshot.CheckedAt = checkedAt
			}
		}
		fields := fieldMap(metric.GetFields())
		switch metric.GetName() {
		case "cpu":
			if usagePercent, ok := floatField(fields, "usage_percent"); ok {
				summary["cpu_usage_percent"] = usagePercent
				snapshot.CpuLoad = usagePercent / 20.0
			}
		case "memory":
			if totalBytes, ok := intField(fields, "total_bytes"); ok {
				snapshot.MemoryTotalMB = int(totalBytes / (1024 * 1024))
			}
			if usedBytes, ok := intField(fields, "used_bytes"); ok {
				snapshot.MemoryUsedMB = int(usedBytes / (1024 * 1024))
			}
			if usedPercent, ok := floatField(fields, "used_percent"); ok {
				summary["memory_used_percent"] = usedPercent
				if snapshot.MemoryTotalMB > 0 && snapshot.MemoryUsedMB == 0 {
					snapshot.MemoryUsedMB = int(math.Round(float64(snapshot.MemoryTotalMB) * usedPercent / 100.0))
				}
			}
		case "disk":
			if usedPercent, ok := floatField(fields, "used_percent"); ok {
				snapshot.DiskUsedPct = usedPercent
				summary["disk_used_percent"] = usedPercent
			}
			if inodeUsedPercent, ok := floatField(fields, "inode_used_percent"); ok {
				snapshot.InodeUsedPct = inodeUsedPercent
				summary["inode_used_percent"] = inodeUsedPercent
			}
			if diskIO, ok := floatField(fields, "io_iops"); ok {
				summary["disk_io_iops"] = diskIO
			} else if diskIO, ok := floatField(fields, "iops"); ok {
				summary["disk_io_iops"] = diskIO
			}
		case "net":
			if rxBytes, ok := intField(fields, "rx_bytes"); ok {
				summary["net_rx_bytes"] = rxBytes
			}
			if txBytes, ok := intField(fields, "tx_bytes"); ok {
				summary["net_tx_bytes"] = txBytes
			}
		case "process":
			if count, ok := intField(fields, "process_count"); ok {
				summary["process_count"] = count
			}
		}
	}

	raw, _ := json.Marshal(summary)
	snapshot.SummaryJSON = string(raw)
	return snapshot
}

// fieldMap 将 Field 切片转换为 map。
func fieldMap(fields []*pb.Field) map[string]*pb.Field {
	out := make(map[string]*pb.Field, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		out[field.GetKey()] = field
	}
	return out
}

// floatField 从字段 map 中获取 float64 值。
func floatField(fields map[string]*pb.Field, key string) (float64, bool) {
	field, ok := fields[key]
	if !ok || field == nil {
		return 0, false
	}
	switch value := field.GetValue().(type) {
	case *pb.Field_DoubleValue:
		return value.DoubleValue, true
	case *pb.Field_IntValue:
		return float64(value.IntValue), true
	default:
		return 0, false
	}
}

// intField 从字段 map 中获取 int64 值。
func intField(fields map[string]*pb.Field, key string) (int64, bool) {
	field, ok := fields[key]
	if !ok || field == nil {
		return 0, false
	}
	switch value := field.GetValue().(type) {
	case *pb.Field_IntValue:
		return value.IntValue, true
	case *pb.Field_DoubleValue:
		return int64(value.DoubleValue), true
	default:
		return 0, false
	}
}
