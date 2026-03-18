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

const (
	DefaultSSHPort    = 22
	ProbeTokenTTL     = 10 * time.Minute
	ProbeTimeout      = 8 * time.Second
	NodeSunsetDateRFC = "Mon, 30 Jun 2026 00:00:00 GMT"
)

type HostService struct {
	svcCtx *svc.ServiceContext
}

var hostHealthCollectorOnce sync.Once

func NewHostService(svcCtx *svc.ServiceContext) *HostService {
	return &HostService{svcCtx: svcCtx}
}

type ProbeReq struct {
	Name     string  `json:"name"`
	IP       string  `json:"ip"`
	Port     int     `json:"port"`
	AuthType string  `json:"auth_type"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	SSHKeyID *uint64 `json:"ssh_key_id"`
}

type ProbeFacts struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Kernel   string `json:"kernel"`
	CPUCores int    `json:"cpu_cores"`
	MemoryMB int    `json:"memory_mb"`
	DiskGB   int    `json:"disk_gb"`
}

type ProbeResp struct {
	ProbeToken string     `json:"probe_token"`
	Reachable  bool       `json:"reachable"`
	LatencyMS  int64      `json:"latency_ms"`
	Facts      ProbeFacts `json:"facts"`
	Warnings   []string   `json:"warnings"`
	ErrorCode  string     `json:"error_code,omitempty"`
	Message    string     `json:"message,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

type CreateReq struct {
	ProbeToken   string   `json:"probe_token"`
	Name         string   `json:"name"`
	IP           string   `json:"ip"`
	Port         int      `json:"port"`
	AuthType     string   `json:"auth_type"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	SSHKeyID     *uint64  `json:"ssh_key_id"`
	Description  string   `json:"description"`
	Labels       []string `json:"labels"`
	Role         string   `json:"role"`
	ClusterID    uint     `json:"cluster_id"`
	Source       string   `json:"source"`
	Provider     string   `json:"provider"`
	ProviderID   string   `json:"provider_instance_id"`
	ParentHostID *uint64  `json:"parent_host_id"`
	Force        bool     `json:"force"`
	Status       string   `json:"status"`
}

type UpdateCredentialsReq struct {
	AuthType string  `json:"auth_type"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	SSHKeyID *uint64 `json:"ssh_key_id"`
	Port     int     `json:"port"`
}

func (s *HostService) List(ctx context.Context) ([]model.Node, error) {
	var list []model.Node
	return list, s.svcCtx.DB.WithContext(ctx).Find(&list).Error
}

func (s *HostService) Get(ctx context.Context, id uint64) (*model.Node, error) {
	var node model.Node
	if err := s.svcCtx.DB.WithContext(ctx).First(&node, id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *HostService) Update(ctx context.Context, id uint64, patch map[string]any) (*model.Node, error) {
	if err := s.svcCtx.DB.WithContext(ctx).Model(&model.Node{}).Where("id = ?", id).Updates(patch).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *HostService) Delete(ctx context.Context, id uint64) error {
	return s.svcCtx.DB.WithContext(ctx).Delete(&model.Node{}, id).Error
}

func (s *HostService) UpdateStatus(ctx context.Context, id uint64, status string) error {
	return s.UpdateStatusWithMeta(ctx, id, status, "", nil, 0)
}

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
