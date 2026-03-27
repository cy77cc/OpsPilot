// Package deployment 提供集群引导的业务逻辑实现。
//
// 本文件包含 Kubernetes 集群引导预览和执行逻辑。
package deployment

import (
	"context"
	"fmt"
	"strings"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/model"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
)

// PreviewClusterBootstrap 预览集群引导步骤。
//
// 参数:
//   - ctx: 上下文
//   - req: 引导预览请求
//
// 返回: 引导预览响应，包含执行步骤
func (l *Logic) PreviewClusterBootstrap(ctx context.Context, req ClusterBootstrapPreviewReq) (ClusterBootstrapPreviewResp, error) {
	control, workers, err := l.loadBootstrapHosts(ctx, req.ControlPlaneID, req.WorkerIDs)
	if err != nil {
		return ClusterBootstrapPreviewResp{}, err
	}
	name := strings.TrimSpace(req.Name)
	cni := defaultIfEmpty(req.CNI, "flannel")
	steps := []string{
		fmt.Sprintf("控制平面节点: %s(%s), 执行 kubeadm init --pod-network-cidr=10.244.0.0/16", control.Name, control.IP),
		fmt.Sprintf("安装 CNI: %s", cni),
		fmt.Sprintf("工作节点数量: %d, 执行 kubeadm join", len(workers)),
		"采集 kubeconfig 并注册集群",
		"自动创建 k8s 部署目标并绑定当前项目/团队",
	}
	return ClusterBootstrapPreviewResp{
		Name:             name,
		ControlPlaneID:   req.ControlPlaneID,
		WorkerHostIDs:    req.WorkerIDs,
		CNI:              cni,
		Steps:            steps,
		ExpectedEndpoint: fmt.Sprintf("https://%s:6443", control.IP),
	}, nil
}

// ApplyClusterBootstrap 执行集群引导，创建集群和部署目标。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID
//   - req: 引导请求
//
// 返回: 引导响应，包含任务 ID 和集群 ID
func (l *Logic) ApplyClusterBootstrap(ctx context.Context, uid uint64, req ClusterBootstrapPreviewReq) (ClusterBootstrapApplyResp, error) {
	preview, err := l.PreviewClusterBootstrap(ctx, req)
	if err != nil {
		return ClusterBootstrapApplyResp{}, err
	}
	task := &model.ClusterBootstrapTask{
		ID:             fmt.Sprintf("boot-%d", time.Now().UnixNano()),
		Name:           preview.Name,
		ControlPlaneID: req.ControlPlaneID,
		WorkerIDsJSON:  toJSON(req.WorkerIDs),
		CNI:            preview.CNI,
		Status:         "running",
		CreatedBy:      uid,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(task).Error; err != nil {
		return ClusterBootstrapApplyResp{}, err
	}
	control, _, err := l.loadBootstrapHosts(ctx, req.ControlPlaneID, req.WorkerIDs)
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		_ = l.svcCtx.DB.WithContext(ctx).Save(task).Error
		return ClusterBootstrapApplyResp{TaskID: task.ID, Status: task.Status}, err
	}
	privateKey, passphrase, err := l.loadNodePrivateKey(ctx, control)
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		_ = l.svcCtx.DB.WithContext(ctx).Save(task).Error
		return ClusterBootstrapApplyResp{TaskID: task.ID, Status: task.Status}, err
	}
	password := strings.TrimSpace(control.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(control.SSHUser, password, control.IP, control.Port, privateKey, passphrase)
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		_ = l.svcCtx.DB.WithContext(ctx).Save(task).Error
		return ClusterBootstrapApplyResp{TaskID: task.ID, Status: task.Status}, err
	}
	defer cli.Close()
	preflightOut, preflightErr := sshclient.RunCommand(cli, "command -v kubeadm >/dev/null 2>&1 && command -v kubectl >/dev/null 2>&1 && echo ok")
	if preflightErr != nil {
		task.Status = "failed"
		task.ErrorMessage = fmt.Sprintf("preflight failed: %s", truncateText(preflightErr.Error(), 240))
		task.ResultJSON = toJSON(map[string]any{"preflight_output": truncateText(preflightOut, 1000)})
		_ = l.svcCtx.DB.WithContext(ctx).Save(task).Error
		return ClusterBootstrapApplyResp{TaskID: task.ID, Status: task.Status}, fmt.Errorf("%s", task.ErrorMessage)
	}

	cluster := model.Cluster{
		Name:       preview.Name,
		Endpoint:   preview.ExpectedEndpoint,
		Status:     "provisioning",
		Type:       "kubernetes",
		AuthMethod: "kubeconfig",
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&cluster).Error; err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		_ = l.svcCtx.DB.WithContext(ctx).Save(task).Error
		return ClusterBootstrapApplyResp{TaskID: task.ID, Status: task.Status}, err
	}
	target := model.DeploymentTarget{
		Name:       fmt.Sprintf("%s-target", preview.Name),
		TargetType: "k8s",
		ClusterID:  cluster.ID,
		ProjectID:  1,
		TeamID:     1,
		Env:        "staging",
		Status:     "active",
		CreatedBy:  uint(uid),
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&target).Error; err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		task.ResultJSON = toJSON(map[string]any{"cluster_id": cluster.ID})
		_ = l.svcCtx.DB.WithContext(ctx).Save(task).Error
		return ClusterBootstrapApplyResp{TaskID: task.ID, Status: task.Status}, err
	}

	task.Status = "succeeded"
	task.ResultJSON = toJSON(map[string]any{
		"cluster_id":           cluster.ID,
		"target_id":            target.ID,
		"next_manual_action":   "登录控制平面节点执行 kubeadm init 并回填 kubeconfig 到集群配置",
		"preflight_checked_at": time.Now().Format(time.RFC3339),
	})
	_ = l.svcCtx.DB.WithContext(ctx).Save(task).Error
	return ClusterBootstrapApplyResp{TaskID: task.ID, Status: task.Status, ClusterID: cluster.ID, TargetID: target.ID}, nil
}

// GetClusterBootstrapTask 获取集群引导任务状态。
//
// 参数:
//   - ctx: 上下文
//   - taskID: 任务 ID
//
// 返回: 引导任务对象
func (l *Logic) GetClusterBootstrapTask(ctx context.Context, taskID string) (*model.ClusterBootstrapTask, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	var task model.ClusterBootstrapTask
	if err := l.svcCtx.DB.WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// loadBootstrapHosts 加载引导所需的主机列表。
//
// 参数:
//   - ctx: 上下文
//   - controlID: 控制平面主机 ID
//   - workerIDs: 工作节点 ID 列表
//
// 返回: 控制平面节点和工作节点列表
func (l *Logic) loadBootstrapHosts(ctx context.Context, controlID uint, workerIDs []uint) (*model.Node, []model.Node, error) {
	var control model.Node
	if err := l.svcCtx.DB.WithContext(ctx).First(&control, controlID).Error; err != nil {
		return nil, nil, fmt.Errorf("control plane host not found: %w", err)
	}
	if ok, reason := hostlogic.EvaluateOperationalEligibility(&control); !ok {
		return nil, nil, fmt.Errorf("control plane host unavailable: %s", reason)
	}
	workers := make([]model.Node, 0, len(workerIDs))
	for _, id := range workerIDs {
		if id == 0 || id == controlID {
			continue
		}
		var row model.Node
		if err := l.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
			return nil, nil, fmt.Errorf("worker host %d not found", id)
		}
		if ok, reason := hostlogic.EvaluateOperationalEligibility(&row); !ok {
			return nil, nil, fmt.Errorf("worker host %d unavailable: %s", id, reason)
		}
		workers = append(workers, row)
	}
	return &control, workers, nil
}
