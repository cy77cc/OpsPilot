package logic

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/google/uuid"
)

// KVMPreviewReq KVM 虚拟机预览请求参数。
type KVMPreviewReq struct {
	Name          string `json:"name"`           // 虚拟机名称
	CPU           int    `json:"cpu"`            // CPU 核心数
	MemoryMB      int    `json:"memory_mb"`      // 内存大小（MB）
	DiskGB        int    `json:"disk_gb"`        // 磁盘大小（GB）
	NetworkBridge string `json:"network_bridge"` // 网络桥接名称
	Template      string `json:"template"`       // 模板名称
}

// KVMProvisionReq KVM 虚拟机创建请求参数。
type KVMProvisionReq struct {
	Name          string  `json:"name"`           // 虚拟机名称
	CPU           int     `json:"cpu"`            // CPU 核心数
	MemoryMB      int     `json:"memory_mb"`      // 内存大小（MB）
	DiskGB        int     `json:"disk_gb"`        // 磁盘大小（GB）
	NetworkBridge string  `json:"network_bridge"` // 网络桥接名称
	Template      string  `json:"template"`       // 模板名称
	IP            string  `json:"ip"`             // IP 地址
	SSHUser       string  `json:"ssh_user"`       // SSH 用户名
	Password      string  `json:"password"`       // SSH 密码
	SSHKeyID      *uint64 `json:"ssh_key_id"`     // SSH 密钥 ID
}

// KVMPreview 预览 KVM 虚拟机配置。
//
// 验证宿主机状态和请求参数，返回预览结果。
// 当前为 MVP 阶段，仅返回预览信息，不实际创建虚拟机。
//
// 参数:
//   - ctx: 上下文
//   - hostID: 宿主机 ID
//   - req: 预览请求参数
//
// 返回: 预览结果
func (s *HostService) KVMPreview(ctx context.Context, hostID uint64, req KVMPreviewReq) (map[string]any, error) {
	host, err := s.Get(ctx, hostID)
	if err != nil {
		return nil, err
	}
	if host.Status == "offline" {
		return nil, errors.New("host is offline")
	}
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.CPU <= 0 {
		req.CPU = 2
	}
	if req.MemoryMB <= 0 {
		req.MemoryMB = 4096
	}
	if req.DiskGB <= 0 {
		req.DiskGB = 50
	}
	return map[string]any{
		"host_id":    hostID,
		"hypervisor": "kvm",
		"ready":      true,
		"preview":    req,
		"message":    "mvp preview only, libvirt execution can be attached later",
	}, nil
}

// KVMProvision 创建 KVM 虚拟机。
//
// 在指定宿主机上创建 KVM 虚拟机，并注册为新主机。
// 当前为 MVP 阶段，仅创建任务记录和主机记录，不实际调用 libvirt。
//
// 参数:
//   - ctx: 上下文
//   - uid: 操作用户 ID
//   - hostID: 宿主机 ID
//   - req: 创建请求参数
//
// 返回:
//   - *model.HostVirtualizationTask: 虚拟化任务
//   - *model.Node: 创建的主机对象
//   - error: 错误信息
func (s *HostService) KVMProvision(ctx context.Context, uid uint64, hostID uint64, req KVMProvisionReq) (*model.HostVirtualizationTask, *model.Node, error) {
	if req.Name == "" {
		return nil, nil, errors.New("name is required")
	}
	host, err := s.Get(ctx, hostID)
	if err != nil {
		return nil, nil, err
	}
	task := &model.HostVirtualizationTask{
		ID:         uuid.NewString(),
		HostID:     hostID,
		Hypervisor: "kvm",
		VMName:     req.Name,
		VMIP:       req.IP,
		Status:     "running",
		CreatedBy:  uid,
	}
	rawReq, _ := json.Marshal(req)
	task.RequestJSON = string(rawReq)
	if err := s.svcCtx.DB.WithContext(ctx).Create(task).Error; err != nil {
		return nil, nil, err
	}

	node := &model.Node{
		Name:         req.Name,
		IP:           req.IP,
		Port:         DefaultSSHPort,
		SSHUser:      firstNonEmpty(req.SSHUser, "root"),
		SSHPassword:  req.Password,
		Status:       "online",
		Source:       "kvm_provision",
		Provider:     "kvm",
		ProviderID:   task.ID,
		ParentHostID: &host.ID,
		OS:           host.OS,
		CpuCores:     req.CPU,
		MemoryMB:     req.MemoryMB,
		DiskGB:       req.DiskGB,
		LastCheckAt:  time.Now(),
	}
	if req.SSHKeyID != nil {
		node.SSHKeyID = nodeIDPtr(*req.SSHKeyID)
	}
	if err := s.svcCtx.DB.WithContext(ctx).Create(node).Error; err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		_ = s.svcCtx.DB.WithContext(ctx).Save(task).Error
		return task, nil, err
	}
	task.Status = "success"
	if err := s.svcCtx.DB.WithContext(ctx).Save(task).Error; err != nil {
		return nil, nil, err
	}
	return task, node, nil
}

// GetVirtualizationTask 获取虚拟化任务详情。
//
// 参数:
//   - ctx: 上下文
//   - taskID: 任务 ID
//
// 返回: 虚拟化任务对象
func (s *HostService) GetVirtualizationTask(ctx context.Context, taskID string) (*model.HostVirtualizationTask, error) {
	var task model.HostVirtualizationTask
	if err := s.svcCtx.DB.WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}
