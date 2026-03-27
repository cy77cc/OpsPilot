package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
)

// CreateWithProbe 通过探测令牌创建主机。
//
// 如果提供了探测令牌，从探测结果创建主机；否则从请求参数直接创建。
// 探测失败时，只有管理员可以强制创建。
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户 ID
//   - isAdmin: 是否为管理员
//   - req: 创建请求参数
//
// 返回: 创建的主机对象
func (s *HostService) CreateWithProbe(ctx context.Context, userID uint64, isAdmin bool, req CreateReq) (*model.Node, error) {
	if req.Force && !isAdmin {
		return nil, errors.New("force create requires admin")
	}

	if strings.TrimSpace(req.ProbeToken) == "" {
		return s.createFromLegacyReq(ctx, req)
	}

	probe, err := s.consumeProbe(ctx, userID, req.ProbeToken)
	if err != nil {
		return nil, err
	}
	if !probe.Reachable && !req.Force {
		return nil, errors.New("probe failed and force is disabled")
	}

	facts := ProbeFacts{}
	_ = json.Unmarshal([]byte(probe.FactsJSON), &facts)
	node := &model.Node{
		Name:        firstNonEmpty(req.Name, probe.Name),
		Hostname:    facts.Hostname,
		Description: req.Description,
		IP:          probe.IP,
		Port:        probe.Port,
		SSHUser:     probe.Username,
		SSHPassword: probe.PasswordCipher,
		Labels:      EncodeLabels(req.Labels),
		Status:      buildStatus(probe.Reachable),
		OS:          facts.OS,
		Arch:        facts.Arch,
		Kernel:      facts.Kernel,
		CpuCores:    facts.CPUCores,
		MemoryMB:    facts.MemoryMB,
		DiskGB:      facts.DiskGB,
		Role:        req.Role,
		ClusterID:   req.ClusterID,
		Source:      firstNonEmpty(req.Source, "manual_ssh"),
		Provider:    req.Provider,
		ProviderID:  req.ProviderID,
		LastCheckAt: time.Now(),
	}
	if probe.SSHKeyID != nil {
		node.SSHKeyID = nodeIDPtr(*probe.SSHKeyID)
	}
	if req.ParentHostID != nil {
		node.ParentHostID = nodeIDPtr(*req.ParentHostID)
	}

	if err := s.svcCtx.DB.WithContext(ctx).Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// UpdateCredentials 更新主机 SSH 凭证。
//
// 验证新凭证是否可用，更新成功后返回主机信息和探测结果。
//
// 参数:
//   - ctx: 上下文
//   - id: 主机 ID
//   - req: 凭证更新请求
//
// 返回:
//   - *model.Node: 更新后的主机对象
//   - *ProbeResp: 探测结果
//   - error: 错误信息
func (s *HostService) UpdateCredentials(ctx context.Context, id uint64, req UpdateCredentialsReq) (*model.Node, *ProbeResp, error) {
	normalizeCredentialReq(&req)
	if req.Username == "" {
		return nil, nil, errors.New("username is required")
	}

	node, err := s.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	backup := *node

	probeReq := ProbeReq{
		Name:     node.Name,
		IP:       node.IP,
		Port:     req.Port,
		AuthType: req.AuthType,
		Username: req.Username,
		Password: req.Password,
		SSHKeyID: req.SSHKeyID,
	}
	resp, err := s.Probe(ctx, 0, probeReq)
	if err != nil {
		return nil, nil, err
	}
	if !resp.Reachable {
		return &backup, resp, errors.New("credential probe failed")
	}

	node.Port = req.Port
	node.SSHUser = req.Username
	node.SSHPassword = req.Password
	node.LastCheckAt = time.Now()
	if req.SSHKeyID != nil {
		node.SSHKeyID = nodeIDPtr(*req.SSHKeyID)
	} else {
		node.SSHKeyID = nil
	}
	if err := s.svcCtx.DB.WithContext(ctx).Save(node).Error; err != nil {
		return nil, nil, err
	}
	return node, resp, nil
}

// createFromLegacyReq 从传统请求参数创建主机（无探测令牌）。
//
// 参数:
//   - ctx: 上下文
//   - req: 创建请求参数
//
// 返回: 创建的主机对象
func (s *HostService) createFromLegacyReq(ctx context.Context, req CreateReq) (*model.Node, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.IP) == "" {
		return nil, errors.New("name and ip are required")
	}
	if req.Port <= 0 {
		req.Port = DefaultSSHPort
	}
	status := req.Status
	if status == "" {
		status = "offline"
	}
	node := &model.Node{
		Name:        req.Name,
		IP:          req.IP,
		Port:        req.Port,
		SSHUser:     firstNonEmpty(req.Username, "root"),
		SSHPassword: req.Password,
		Description: req.Description,
		Labels:      EncodeLabels(req.Labels),
		Status:      status,
		Role:        req.Role,
		ClusterID:   req.ClusterID,
		Source:      firstNonEmpty(req.Source, "manual_ssh"),
		Provider:    req.Provider,
		ProviderID:  req.ProviderID,
	}
	if req.SSHKeyID != nil {
		node.SSHKeyID = nodeIDPtr(*req.SSHKeyID)
	}
	if req.ParentHostID != nil {
		node.ParentHostID = nodeIDPtr(*req.ParentHostID)
	}
	if err := s.svcCtx.DB.WithContext(ctx).Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// firstNonEmpty 返回第一个非空字符串。
//
// 参数:
//   - v: 字符串列表
//
// 返回: 第一个非空字符串，全空则返回空字符串
func firstNonEmpty(v ...string) string {
	for _, item := range v {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// buildStatus 根据可达性构建状态字符串。
//
// 参数:
//   - reachable: 是否可达
//
// 返回: 状态字符串 (online/offline)
func buildStatus(reachable bool) string {
	if reachable {
		return "online"
	}
	return "offline"
}

// nodeIDPtr 将 uint64 转换为 *model.NodeID。
//
// 参数:
//   - v: uint64 值
//
// 返回: NodeID 指针
func nodeIDPtr(v uint64) *model.NodeID {
	n := model.NodeID(v)
	return &n
}
