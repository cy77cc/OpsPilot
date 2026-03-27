// Package logic 提供主机管理的业务逻辑实现。
//
// 本包包含所有主机相关的业务逻辑，包括：
//   - 主机 CRUD 操作 (host_service.go)
//   - SSH 连接探测 (probe.go)
//   - SSH 密钥管理 (credentials.go)
//   - KVM 虚拟化 (virtualization.go)
//   - 主机准入判断 (eligibility.go)
//   - 云主机导入 (cloud.go, cloud/)
//   - 主机上线流程 (onboarding.go)
package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/config"
	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/service/notification"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"gorm.io/gorm"
)

// 默认配置常量。
const (
	// DefaultSSHPort 默认 SSH 端口
	DefaultSSHPort = 22

	// ProbeTokenTTL 探测令牌有效期
	ProbeTokenTTL = 10 * time.Minute

	// ProbeTimeout 探测超时时间
	ProbeTimeout = 8 * time.Second

	// NodeSunsetDateRFC Node 模型废弃日期
	NodeSunsetDateRFC = "Mon, 30 Jun 2026 00:00:00 GMT"
)

// HostService 主机管理业务服务。
//
// 提供主机的 CRUD、健康检查、状态管理、探测等核心业务逻辑。
type HostService struct {
	// svcCtx 服务上下文
	svcCtx *svc.ServiceContext
}

// hostHealthCollectorOnce 健康检查收集器单例控制。
var hostHealthCollectorOnce sync.Once

// NewHostService 创建主机管理服务实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: HostService 实例
func NewHostService(svcCtx *svc.ServiceContext) *HostService {
	return &HostService{svcCtx: svcCtx}
}

// ProbeReq SSH 探测请求参数。
type ProbeReq struct {
	Name     string  `json:"name"`      // 主机名称
	IP       string  `json:"ip"`        // 主机 IP 地址
	Port     int     `json:"port"`      // SSH 端口
	AuthType string  `json:"auth_type"` // 认证类型 (password/key)
	Username string  `json:"username"`  // SSH 用户名
	Password string  `json:"password"`  // SSH 密码
	SSHKeyID *uint64 `json:"ssh_key_id"` // SSH 密钥 ID
}

// ProbeFacts 主机系统信息。
type ProbeFacts struct {
	Hostname string `json:"hostname"`  // 主机名
	OS       string `json:"os"`        // 操作系统
	Arch     string `json:"arch"`      // 架构
	Kernel   string `json:"kernel"`    // 内核版本
	CPUCores int    `json:"cpu_cores"` // CPU 核心数
	MemoryMB int    `json:"memory_mb"` // 内存大小（MB）
	DiskGB   int    `json:"disk_gb"`   // 磁盘大小（GB）
}

// ProbeResp SSH 探测响应。
type ProbeResp struct {
	ProbeToken string     `json:"probe_token"` // 探测令牌，用于后续创建主机
	Reachable  bool       `json:"reachable"`   // 是否可达
	LatencyMS  int64      `json:"latency_ms"`  // 延迟（毫秒）
	Facts      ProbeFacts `json:"facts"`       // 系统信息
	Warnings   []string   `json:"warnings"`    // 警告信息
	ErrorCode  string     `json:"error_code,omitempty"` // 错误码
	Message    string     `json:"message,omitempty"`    // 错误消息
	ExpiresAt  time.Time  `json:"expires_at"`           // 令牌过期时间
}

// CreateReq 创建主机请求参数。
type CreateReq struct {
	ProbeToken   string   `json:"probe_token"`        // 探测令牌
	Name         string   `json:"name"`               // 主机名称
	IP           string   `json:"ip"`                 // IP 地址
	Port         int      `json:"port"`               // SSH 端口
	AuthType     string   `json:"auth_type"`          // 认证类型
	Username     string   `json:"username"`           // SSH 用户名
	Password     string   `json:"password"`           // SSH 密码
	SSHKeyID     *uint64  `json:"ssh_key_id"`         // SSH 密钥 ID
	Description  string   `json:"description"`        // 描述
	Labels       []string `json:"labels"`             // 标签列表
	Role         string   `json:"role"`               // 角色
	ClusterID    uint     `json:"cluster_id"`         // 集群 ID
	Source       string   `json:"source"`             // 来源
	Provider     string   `json:"provider"`           // 云厂商
	ProviderID   string   `json:"provider_instance_id"` // 云实例 ID
	ParentHostID *uint64  `json:"parent_host_id"`     // 父主机 ID（虚拟化场景）
	Force        bool     `json:"force"`              // 强制创建（忽略探测失败）
	Status       string   `json:"status"`             // 初始状态
}

// UpdateCredentialsReq 更新凭证请求参数。
type UpdateCredentialsReq struct {
	AuthType string  `json:"auth_type"`  // 认证类型
	Username string  `json:"username"`   // SSH 用户名
	Password string  `json:"password"`   // SSH 密码
	SSHKeyID *uint64 `json:"ssh_key_id"` // SSH 密钥 ID
	Port     int     `json:"port"`       // SSH 端口
}

// List 获取所有主机列表。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 主机列表
func (s *HostService) List(ctx context.Context) ([]model.Node, error) {
	var list []model.Node
	return list, s.svcCtx.DB.WithContext(ctx).Find(&list).Error
}

// Get 根据 ID 获取主机信息。
//
// 参数:
//   - ctx: 上下文
//   - id: 主机 ID
//
// 返回: 主机对象，不存在返回 ErrRecordNotFound
func (s *HostService) Get(ctx context.Context, id uint64) (*model.Node, error) {
	var node model.Node
	if err := s.svcCtx.DB.WithContext(ctx).First(&node, id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

// Update 更新主机信息。
//
// 参数:
//   - ctx: 上下文
//   - id: 主机 ID
//   - patch: 更新字段映射
//
// 返回: 更新后的主机对象
func (s *HostService) Update(ctx context.Context, id uint64, patch map[string]any) (*model.Node, error) {
	if err := s.svcCtx.DB.WithContext(ctx).Model(&model.Node{}).Where("id = ?", id).Updates(patch).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Delete 删除主机。
//
// 参数:
//   - ctx: 上下文
//   - id: 主机 ID
//
// 返回: 删除错误
func (s *HostService) Delete(ctx context.Context, id uint64) error {
	return s.svcCtx.DB.WithContext(ctx).Delete(&model.Node{}, id).Error
}

// UpdateStatus 更新主机状态（简化版本）。
//
// 参数:
//   - ctx: 上下文
//   - id: 主机 ID
//   - status: 新状态
//
// 返回: 更新错误
func (s *HostService) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return s.UpdateStatusWithMeta(ctx, id, status, "", nil, 0)
}

// UpdateStatusWithMeta 更新主机状态（带元数据）。
//
// 当状态为 maintenance 时，记录维护原因、操作者和预期结束时间。
// 当状态从 maintenance 变为其他状态时，清理维护相关字段。
//
// 参数:
//   - ctx: 上下文
//   - id: 主机 ID
//   - status: 新状态
//   - reason: 维护原因
//   - until: 维护结束时间
//   - operator: 操作者 ID
//
// 返回: 更新错误
func (s *HostService) UpdateStatusWithMeta(ctx context.Context, id uint64, status, reason string, until *time.Time, operator uint64) error {
	normalized := strings.ToLower(strings.TrimSpace(status))
	updates := map[string]any{"status": normalized}
	if normalized == "maintenance" {
		now := time.Now()
		updates["maintenance_reason"] = strings.TrimSpace(reason)
		updates["maintenance_by"] = operator
		updates["maintenance_started_at"] = &now
		updates["maintenance_until"] = until
	} else {
		updates["maintenance_reason"] = ""
		updates["maintenance_by"] = 0
		updates["maintenance_started_at"] = nil
		updates["maintenance_until"] = nil
	}
	if err := s.svcCtx.DB.WithContext(ctx).Model(&model.Node{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	node, err := s.Get(ctx, id)
	if err != nil {
		return nil
	}
	s.emitMaintenanceLifecycle(ctx, node, normalized, strings.TrimSpace(reason), operator, until)
	return nil
}

// BatchUpdateStatus 批量更新主机状态。
//
// 参数:
//   - ctx: 上下文
//   - ids: 主机 ID 列表
//   - status: 新状态
//
// 返回: 更新错误
func (s *HostService) BatchUpdateStatus(ctx context.Context, ids []uint64, status string) error {
	if len(ids) == 0 || status == "" {
		return nil
	}
	updates := map[string]any{"status": strings.ToLower(strings.TrimSpace(status))}
	if strings.EqualFold(status, "maintenance") {
		now := time.Now()
		updates["maintenance_started_at"] = &now
	} else {
		updates["maintenance_reason"] = ""
		updates["maintenance_by"] = 0
		updates["maintenance_started_at"] = nil
		updates["maintenance_until"] = nil
	}
	return s.svcCtx.DB.WithContext(ctx).Model(&model.Node{}).Where("id IN ?", ids).Updates(updates).Error
}

// ListHealthSnapshots 获取主机健康检查快照列表。
//
// 参数:
//   - ctx: 上下文
//   - hostID: 主机 ID
//   - limit: 返回数量限制
//
// 返回: 健康检查快照列表
func (s *HostService) ListHealthSnapshots(ctx context.Context, hostID uint64, limit int) ([]model.HostHealthSnapshot, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []model.HostHealthSnapshot
	err := s.svcCtx.DB.WithContext(ctx).
		Where("host_id = ?", hostID).
		Order("checked_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// RunHealthCheck 执行主机健康检查。
//
// 通过 SSH 连接主机，收集 CPU 负载、内存使用、磁盘使用、inode 使用等指标，
// 并根据阈值判断健康状态（healthy/degraded/critical）。
//
// 参数:
//   - ctx: 上下文
//   - hostID: 主机 ID
//   - operator: 操作者 ID
//
// 返回: 健康检查快照
func (s *HostService) RunHealthCheck(ctx context.Context, hostID uint64, operator uint64) (*model.HostHealthSnapshot, error) {
	node, err := s.Get(ctx, hostID)
	if err != nil {
		return nil, err
	}

	snapshot := &model.HostHealthSnapshot{
		HostID:             hostID,
		State:              "unknown",
		ConnectivityStatus: "unknown",
		ResourceStatus:     "unknown",
		SystemStatus:       "unknown",
		CheckedAt:          time.Now(),
	}
	start := time.Now()

	privateKey, passphrase, err := s.loadNodePrivateKey(ctx, node)
	if err != nil {
		snapshot.ErrorMessage = err.Error()
		snapshot.State = "critical"
		snapshot.ConnectivityStatus = "critical"
		_ = s.persistHealthSnapshot(ctx, snapshot, node)
		return snapshot, nil
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		snapshot.ErrorMessage = err.Error()
		snapshot.State = "critical"
		snapshot.ConnectivityStatus = "critical"
		_ = s.persistHealthSnapshot(ctx, snapshot, node)
		return snapshot, nil
	}
	defer cli.Close()
	snapshot.LatencyMS = time.Since(start).Milliseconds()
	snapshot.ConnectivityStatus = "healthy"

	loadRaw, loadErr := sshclient.RunCommand(cli, `awk '{print $1}' /proc/loadavg`)
	memRaw, memErr := sshclient.RunCommand(cli, `free -m | awk '/Mem:/{print $3":"$2}'`)
	diskRaw, diskErr := sshclient.RunCommand(cli, `df -P / | awk 'NR==2{gsub("%","",$5); print $5}'`)
	inodeRaw, inodeErr := sshclient.RunCommand(cli, `df -Pi / | awk 'NR==2{gsub("%","",$5); print $5}'`)

	if loadErr != nil || memErr != nil || diskErr != nil || inodeErr != nil {
		snapshot.State = "degraded"
		snapshot.ResourceStatus = "degraded"
		snapshot.SystemStatus = "degraded"
		snapshot.ErrorMessage = "partial check failed"
	} else {
		snapshot.ResourceStatus = "healthy"
		snapshot.SystemStatus = "healthy"
		snapshot.State = "healthy"
	}

	if v, err := strconv.ParseFloat(strings.TrimSpace(loadRaw), 64); err == nil {
		snapshot.CpuLoad = v
		if v >= 4 {
			snapshot.State = "degraded"
			snapshot.SystemStatus = "degraded"
		}
	}
	if parts := strings.Split(strings.TrimSpace(memRaw), ":"); len(parts) == 2 {
		snapshot.MemoryUsedMB, _ = strconv.Atoi(parts[0])
		snapshot.MemoryTotalMB, _ = strconv.Atoi(parts[1])
		if snapshot.MemoryTotalMB > 0 {
			usedPct := float64(snapshot.MemoryUsedMB) / float64(snapshot.MemoryTotalMB)
			if usedPct >= 0.9 {
				snapshot.State = "critical"
				snapshot.ResourceStatus = "critical"
			} else if usedPct >= 0.8 && snapshot.State == "healthy" {
				snapshot.State = "degraded"
				snapshot.ResourceStatus = "degraded"
			}
		}
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(diskRaw), 64); err == nil {
		snapshot.DiskUsedPct = v
		if v >= 95 {
			snapshot.State = "critical"
			snapshot.ResourceStatus = "critical"
		} else if v >= 85 && snapshot.State == "healthy" {
			snapshot.State = "degraded"
			snapshot.ResourceStatus = "degraded"
		}
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(inodeRaw), 64); err == nil {
		snapshot.InodeUsedPct = v
		if v >= 95 {
			snapshot.State = "critical"
			snapshot.ResourceStatus = "critical"
		} else if v >= 85 && snapshot.State == "healthy" {
			snapshot.State = "degraded"
			snapshot.ResourceStatus = "degraded"
		}
	}

	summary := map[string]any{
		"operator":   operator,
		"checked_at": snapshot.CheckedAt,
		"load_raw":   loadRaw,
		"mem_raw":    memRaw,
		"disk_raw":   diskRaw,
		"inode_raw":  inodeRaw,
	}
	raw, _ := json.Marshal(summary)
	snapshot.SummaryJSON = string(raw)

	_ = s.persistHealthSnapshot(ctx, snapshot, node)
	return snapshot, nil
}

// StartHealthSnapshotCollector 启动主机健康检查收集器。
//
// 启动后台定时任务，每 2 分钟对所有主机执行一次健康检查。
// 使用 sync.Once 确保只启动一次。
func (s *HostService) StartHealthSnapshotCollector() {
	hostHealthCollectorOnce.Do(func() {
		rootCtx := runtimectx.WithServices(context.Background(), s.svcCtx)
		go func() {
			ticker := time.NewTicker(2 * time.Minute)
			defer ticker.Stop()
			for {
				roundCtx, cancel := context.WithTimeout(rootCtx, 90*time.Second)
				s.CollectHealthSnapshots(roundCtx)
				cancel()
				<-ticker.C
			}
		}()
		roundCtx, cancel := context.WithTimeout(rootCtx, 90*time.Second)
		s.CollectHealthSnapshots(roundCtx)
		cancel()
	})
}

// CollectHealthSnapshots 收集所有主机的健康快照。
//
// 并发执行健康检查，使用信号量控制最大并发数为 6，
// 使用确定性抖动避免同步重连风暴。
//
// 参数:
//   - ctx: 上下文
func (s *HostService) CollectHealthSnapshots(ctx context.Context) {
	var hosts []model.Node
	if err := s.svcCtx.DB.WithContext(ctx).
		Select("id", "status", "ip").
		Where("ip <> ''").
		Order("id ASC").
		Limit(500).
		Find(&hosts).Error; err != nil {
		return
	}
	if len(hosts) == 0 {
		return
	}

	const (
		maxConcurrency = 6
		maxRetries     = 3
	)
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	for i := range hosts {
		host := hosts[i]
		wg.Add(1)
		sem <- struct{}{}
		go func(node model.Node) {
			defer wg.Done()
			defer func() { <-sem }()

			// Retry with deterministic jitter to avoid synchronized reconnect storms.
			for attempt := 1; attempt <= maxRetries; attempt++ {
				runCtx, cancel := context.WithTimeout(ctx, 18*time.Second)
				_, err := s.RunHealthCheck(runCtx, uint64(node.ID), 0)
				cancel()
				if err == nil {
					return
				}
				if attempt >= maxRetries {
					return
				}
				jitter := time.Duration((int(node.ID)%7)*90+attempt*150) * time.Millisecond
				select {
				case <-ctx.Done():
					return
				case <-time.After(jitter):
				}
			}
		}(host)
	}
	wg.Wait()
}

// persistHealthSnapshot 持久化健康检查快照。
//
// 保存快照到数据库，更新主机的健康状态和最后检查时间，
// 并推送指标到 Prometheus（如果配置）。
//
// 参数:
//   - ctx: 上下文
//   - snapshot: 健康检查快照
//   - node: 主机对象
//
// 返回: 保存错误
func (s *HostService) persistHealthSnapshot(ctx context.Context, snapshot *model.HostHealthSnapshot, node *model.Node) error {
	if snapshot == nil || node == nil {
		return nil
	}
	if err := s.svcCtx.DB.WithContext(ctx).Create(snapshot).Error; err != nil {
		return err
	}
	now := snapshot.CheckedAt
	updates := map[string]any{
		"health_state":  snapshot.State,
		"last_check_at": now,
	}
	if err := s.svcCtx.DB.WithContext(ctx).Model(&model.Node{}).Where("id = ?", node.ID).Updates(updates).Error; err != nil {
		return err
	}

	// 推送指标到 Prometheus
	if s.svcCtx.MetricsPusher != nil {
		metricSnapshot := prominfra.HostMetricSnapshot{
			HostID:             uint64(node.ID),
			HostName:           node.Name,
			HostIP:             node.IP,
			CPULoad:            snapshot.CpuLoad,
			MemoryUsedMB:       snapshot.MemoryUsedMB,
			MemoryTotalMB:      snapshot.MemoryTotalMB,
			DiskUsagePercent:   snapshot.DiskUsedPct,
			InodeUsagePercent:  snapshot.InodeUsedPct,
			HealthState:        snapshot.State,
			ConnectivityStatus: snapshot.ConnectivityStatus,
		}
		if err := s.svcCtx.MetricsPusher.PushHostMetrics(ctx, metricSnapshot); err != nil {
			// 推送失败不影响主流程，记录日志
			logger.L().Warn("failed to push host metrics to prometheus",
				logger.Error(err),
				logger.Int("host_id", int(node.ID)),
				logger.String("host_name", node.Name),
			)
		}
	}

	return nil
}

// loadNodePrivateKey 加载主机关联的 SSH 私钥。
//
// 从数据库加载 SSH 密钥，处理加密存储的私钥解密。
//
// 参数:
//   - ctx: 上下文
//   - node: 主机对象
//
// 返回:
//   - privateKey: 私钥内容
//   - passphrase: 私钥密码
//   - error: 错误信息
func (s *HostService) loadNodePrivateKey(ctx context.Context, node *model.Node) (string, string, error) {
	if node == nil || node.SSHKeyID == nil {
		return "", "", nil
	}
	var key model.SSHKey
	if err := s.svcCtx.DB.WithContext(ctx).
		Select("id", "private_key", "passphrase", "encrypted").
		Where("id = ?", uint64(*node.SSHKeyID)).
		First(&key).Error; err != nil {
		return "", "", err
	}
	passphrase := strings.TrimSpace(key.Passphrase)
	if !key.Encrypted {
		return strings.TrimSpace(key.PrivateKey), passphrase, nil
	}
	privateKey, err := utils.DecryptText(strings.TrimSpace(key.PrivateKey), config.CFG.Security.EncryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("decrypt private key: %w", err)
	}
	return privateKey, passphrase, nil
}

// emitMaintenanceLifecycle 发送维护生命周期事件。
//
// 当主机进入或退出维护模式时，创建审计日志并发送通知。
//
// 参数:
//   - ctx: 上下文
//   - node: 主机对象
//   - status: 新状态
//   - reason: 维护原因
//   - operator: 操作者 ID
//   - until: 维护结束时间
func (s *HostService) emitMaintenanceLifecycle(ctx context.Context, node *model.Node, status, reason string, operator uint64, until *time.Time) {
	if strings.TrimSpace(status) != "maintenance" && node.MaintenanceStartedAt != nil {
		// Keep emitting exit event; no-op branch kept for readability.
	}
	detail := map[string]any{
		"host_id":    node.ID,
		"host_name":  node.Name,
		"host_ip":    node.IP,
		"status":     status,
		"reason":     strings.TrimSpace(reason),
		"operator":   operator,
		"until":      until,
		"changed_at": time.Now(),
	}
	action := "host_maintenance_exited"
	title := fmt.Sprintf("主机维护结束: %s", node.Name)
	content := fmt.Sprintf("主机 %s(%s) 已退出维护模式", node.Name, node.IP)
	if status == "maintenance" {
		action = "host_maintenance_entered"
		title = fmt.Sprintf("主机进入维护: %s", node.Name)
		if strings.TrimSpace(reason) == "" {
			content = fmt.Sprintf("主机 %s(%s) 已进入维护模式", node.Name, node.IP)
		} else {
			content = fmt.Sprintf("主机 %s(%s) 进入维护模式，原因：%s", node.Name, node.IP, strings.TrimSpace(reason))
		}
	}
	_ = s.svcCtx.DB.WithContext(ctx).Create(&model.AuditLog{
		ActionType:   action,
		ResourceType: "host",
		ResourceID:   uint(node.ID),
		ActorID:      uint(operator),
		ActorName:    "",
		Detail:       detail,
	}).Error

	if operator == 0 {
		return
	}
	integrator := notification.NewNotificationIntegrator(s.svcCtx.DB)
	_ = integrator.CreateSystemNotification(ctx, title, content, []uint64{operator})
}

// consumeProbe 消费探测令牌。
//
// 验证探测令牌的有效性（存在、未过期、未消费、所有者匹配），
// 并标记为已消费。
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户 ID
//   - token: 探测令牌
//
// 返回: 探测会话对象
func (s *HostService) consumeProbe(ctx context.Context, userID uint64, token string) (*model.HostProbeSession, error) {
	hash := hashToken(token)
	var probe model.HostProbeSession
	if err := s.svcCtx.DB.WithContext(ctx).Where("token_hash = ?", hash).First(&probe).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("probe_not_found")
		}
		return nil, err
	}
	if probe.CreatedBy != 0 && userID != 0 && probe.CreatedBy != userID {
		return nil, errors.New("probe_not_found")
	}
	if probe.ConsumedAt != nil {
		return nil, errors.New("probe_not_found")
	}
	if time.Now().After(probe.ExpiresAt) {
		return nil, errors.New("probe_expired")
	}

	now := time.Now()
	if err := s.svcCtx.DB.WithContext(ctx).Model(&model.HostProbeSession{}).
		Where("id = ? AND consumed_at IS NULL", probe.ID).
		Update("consumed_at", &now).Error; err != nil {
		return nil, err
	}
	probe.ConsumedAt = &now
	return &probe, nil
}

// ParseLabels 解析标签字符串为标签数组。
//
// 支持两种格式：
//   - JSON 数组格式（推荐）：`["tag1", "tag2"]`
//   - 逗号分隔格式（兼容）：`tag1,tag2`
//
// 参数:
//   - labels: 标签字符串
//
// 返回: 标签数组
func ParseLabels(labels string) []string {
	trimmed := strings.TrimSpace(labels)
	if trimmed == "" {
		return nil
	}

	// Preferred format: JSON array string persisted in `nodes.labels`.
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

	// Backward compatibility: legacy comma-separated storage.
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		if s := strings.TrimSpace(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// EncodeLabels 将标签数组编码为 JSON 字符串。
//
// 参数:
//   - labels: 标签数组
//
// 返回: JSON 格式的标签字符串
func EncodeLabels(labels []string) string {
	out := make([]string, 0, len(labels))
	for _, item := range labels {
		if s := strings.TrimSpace(item); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "[]"
	}
	return string(raw)
}
