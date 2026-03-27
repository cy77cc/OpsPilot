// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件实现集群节点管理相关的 HTTP Handler 和业务逻辑，
// 包括节点同步、添加、删除和详情查询等功能。
package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// AddNodeReq 添加节点请求结构。
type AddNodeReq struct {
	HostIDs []uint `json:"host_ids" binding:"required"` // 主机 ID 列表 (必填)
	Role    string `json:"role"`                        // 节点角色: worker/control-plane
}

// NodeDetail 节点详情响应结构。
type NodeDetail struct {
	ID               uint        `json:"id"`                // 节点 ID
	ClusterID        uint        `json:"cluster_id"`        // 所属集群 ID
	HostID           *uint       `json:"host_id"`           // 关联主机 ID
	HostName         string      `json:"host_name,omitempty"` // 主机名称
	Name             string      `json:"name"`              // 节点名称
	IP               string      `json:"ip"`                // 节点 IP
	Role             string      `json:"role"`              // 节点角色
	Status           string      `json:"status"`            // 节点状态
	KubeletVersion   string      `json:"kubelet_version"`   // Kubelet 版本
	KubeProxyVersion string      `json:"kube_proxy_version"` // Kube-proxy 版本
	ContainerRuntime string      `json:"container_runtime"` // 容器运行时
	OSImage          string      `json:"os_image"`          // 操作系统镜像
	KernelVersion    string      `json:"kernel_version"`    // 内核版本
	AllocatableCPU   string      `json:"allocatable_cpu"`   // 可分配 CPU
	AllocatableMem   string      `json:"allocatable_mem"`   // 可分配内存
	AllocatablePods  int         `json:"allocatable_pods"`  // 可分配 Pod 数量
	Labels           MapString   `json:"labels"`            // 节点标签
	Taints           []Taint     `json:"taints"`            // 节点污点
	Conditions       []Condition `json:"conditions"`        // 节点条件
	JoinedAt         *time.Time  `json:"joined_at"`         // 加入时间
	LastSeenAt       *time.Time  `json:"last_seen_at"`      // 最后心跳时间
	CreatedAt        time.Time   `json:"created_at"`        // 创建时间
	UpdatedAt        time.Time   `json:"updated_at"`        // 更新时间
}

// MapString 字符串映射类型。
type MapString map[string]string

// Taint 节点污点结构。
type Taint struct {
	Key    string `json:"key"`    // 污点键
	Value  string `json:"value"`  // 污点值
	Effect string `json:"effect"` // 污点效果: NoSchedule/NoExecute/PreferNoSchedule
}

// Condition 节点条件结构。
type Condition struct {
	Type               string     `json:"type"`                       // 条件类型
	Status             string     `json:"status"`                     // 条件状态
	Reason             string     `json:"reason"`                     // 条件原因
	Message            string     `json:"message"`                    // 条件消息
	LastTransitionTime *time.Time `json:"last_transition_time,omitempty"` // 最后转换时间
}

// SyncClusterNodes 从 Kubernetes API 同步节点到数据库。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//
// 返回: 失败返回错误
func (h *Handler) SyncClusterNodes(ctx context.Context, clusterID uint) error {
	// Get credential
	cred, err := h.repo.FindClusterCredentialByClusterID(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("credential not found: %w", err)
	}

	// Build client
	restConfig, err := h.buildRestConfigFromCredential(cred)
	if err != nil {
		return fmt.Errorf("failed to build rest config: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Get nodes from API
	nodes, err := client.CoreV1().Nodes().List(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	now := time.Now().UTC()

	for _, node := range nodes.Items {
		// Determine node role
		role := "worker"
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			role = "control-plane"
		} else if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			role = "control-plane"
		}

		// Determine node status
		status := "unknown"
		var conditions []Condition
		for _, c := range node.Status.Conditions {
			cond := Condition{
				Type:    string(c.Type),
				Status:  string(c.Status),
				Reason:  c.Reason,
				Message: c.Message,
			}
			if c.LastTransitionTime.Time.Unix() > 0 {
				t := c.LastTransitionTime.Time
				cond.LastTransitionTime = &t
			}
			conditions = append(conditions, cond)

			if c.Type == "Ready" {
				if c.Status == "True" {
					status = "ready"
				} else {
					status = "notready"
				}
			}
		}

		// Get IP
		var ip string
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				ip = addr.Address
				break
			}
		}

		// Parse labels and taints
		labelsJSON, _ := json.Marshal(node.Labels)
		var taints []Taint
		for _, t := range node.Spec.Taints {
			taints = append(taints, Taint{
				Key:    t.Key,
				Value:  t.Value,
				Effect: string(t.Effect),
			})
		}
		taintsJSON, _ := json.Marshal(taints)
		conditionsJSON, _ := json.Marshal(conditions)

		// Get allocatable resources
		allocatableCPU := node.Status.Allocatable.Cpu().String()
		allocatableMem := node.Status.Allocatable.Memory().String()
		allocatablePods := node.Status.Allocatable.Pods().Value()

		// Build node record
		clusterNode := model.ClusterNode{
			ClusterID:        clusterID,
			Name:             node.Name,
			IP:               ip,
			Role:             role,
			Status:           status,
			KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
			KubeProxyVersion: node.Status.NodeInfo.KubeProxyVersion,
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			OSImage:          node.Status.NodeInfo.OSImage,
			KernelVersion:    node.Status.NodeInfo.KernelVersion,
			AllocatableCPU:   allocatableCPU,
			AllocatableMem:   allocatableMem,
			AllocatablePods:  int(allocatablePods),
			Labels:           string(labelsJSON),
			Taints:           string(taintsJSON),
			Conditions:       string(conditionsJSON),
			LastSeenAt:       &now,
		}

		// Try to find matching host by IP
		var host model.Node
		if err := h.svcCtx.DB.WithContext(ctx).Where("ip = ?", ip).First(&host).Error; err == nil {
			hostID := uint(host.ID)
			clusterNode.HostID = &hostID
		}

		updates := map[string]interface{}{
			"ip":                 ip,
			"role":               role,
			"status":             status,
			"kubelet_version":    node.Status.NodeInfo.KubeletVersion,
			"kube_proxy_version": node.Status.NodeInfo.KubeProxyVersion,
			"container_runtime":  node.Status.NodeInfo.ContainerRuntimeVersion,
			"os_image":           node.Status.NodeInfo.OSImage,
			"kernel_version":     node.Status.NodeInfo.KernelVersion,
			"allocatable_cpu":    allocatableCPU,
			"allocatable_mem":    allocatableMem,
			"allocatable_pods":   int(allocatablePods),
			"labels":             string(labelsJSON),
			"taints":             string(taintsJSON),
			"conditions":         string(conditionsJSON),
			"host_id":            clusterNode.HostID,
			"last_seen_at":       &now,
		}
		if err := h.repo.UpsertClusterNode(ctx, clusterID, node.Name, clusterNode, updates); err != nil {
			return err
		}
	}

	// Update cluster last_sync_at
	if err := h.repo.UpdateClusterLastSync(ctx, clusterID, &now); err != nil {
		return err
	}
	h.invalidateClusterCache(ctx, clusterID)

	return nil
}

// SyncClusterNodesHandler 处理节点同步 HTTP 请求。
//
// @Summary 同步集群节点
// @Description 从 Kubernetes API 同步节点信息到数据库
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/nodes/sync [post]
func (h *Handler) SyncClusterNodesHandler(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	if err := h.SyncClusterNodes(c.Request.Context(), id); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	// Return updated node list
	h.GetClusterNodes(c)
}

// AddClusterNodes 向集群添加节点。
//
// @Summary 添加集群节点
// @Description 将主机添加到平台管理的集群作为工作节点或控制面节点
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Param request body AddNodeReq true "添加节点请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/nodes [post]
func (h *Handler) AddClusterNodes(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	var req AddNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	// Get cluster
	var cluster model.Cluster
	if err := h.svcCtx.DB.First(&cluster, id).Error; err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	// Check if platform managed
	if cluster.Source != "platform_managed" {
		httpx.BadRequest(c, "cannot add nodes to externally managed cluster")
		return
	}

	// Get credential to retrieve join command
	var cred model.ClusterCredential
	if err := h.svcCtx.DB.Where("cluster_id = ?", cluster.ID).First(&cred).Error; err != nil {
		httpx.ServerErr(c, fmt.Errorf("credential not found: %w", err))
		return
	}

	// Get join command from control plane
	joinCommand, err := h.getJoinCommand(c.Request.Context(), cluster.ID)
	if err != nil {
		httpx.ServerErr(c, fmt.Errorf("failed to get join command: %w", err))
		return
	}

	// Execute join on each host
	role := defaultIfEmpty(req.Role, "worker")
	results := make([]map[string]interface{}, 0, len(req.HostIDs))

	for _, hostID := range req.HostIDs {
		var host model.Node
		if err := h.svcCtx.DB.First(&host, hostID).Error; err != nil {
			results = append(results, map[string]interface{}{
				"host_id": hostID,
				"success": false,
				"message": "host not found",
			})
			continue
		}

		// Execute join via SSH
		err := h.executeJoinOnHost(c.Request.Context(), &host, joinCommand, role)
		if err != nil {
			results = append(results, map[string]interface{}{
				"host_id":   hostID,
				"host_name": host.Name,
				"success":   false,
				"message":   err.Error(),
			})
			continue
		}

		results = append(results, map[string]interface{}{
			"host_id":   hostID,
			"host_name": host.Name,
			"success":   true,
			"message":   "node joined successfully",
		})
	}

	// Sync nodes
	go h.SyncClusterNodes(runtimectx.Detach(c.Request.Context()), cluster.ID)
	h.invalidateClusterCache(c.Request.Context(), cluster.ID)

	httpx.OK(c, gin.H{
		"results": results,
		"message": fmt.Sprintf("Processed %d hosts", len(req.HostIDs)),
	})
}

// RemoveClusterNode 从集群移除节点。
//
// @Summary 移除集群节点
// @Description 从平台管理的集群中移除指定节点
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param name path string true "节点名称"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/nodes/{name} [delete]
func (h *Handler) RemoveClusterNode(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	clusterID := httpx.UintFromParam(c, "id")
	nodeName := strings.TrimSpace(c.Param("name"))

	if clusterID == 0 || nodeName == "" {
		httpx.BindErr(c, nil)
		return
	}

	// Get cluster
	var cluster model.Cluster
	if err := h.svcCtx.DB.First(&cluster, clusterID).Error; err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	// Check if platform managed
	if cluster.Source != "platform_managed" {
		httpx.BadRequest(c, "cannot remove nodes from externally managed cluster")
		return
	}

	// Get node record
	var node model.ClusterNode
	if err := h.svcCtx.DB.Where("cluster_id = ? AND name = ?", clusterID, nodeName).First(&node).Error; err != nil {
		httpx.NotFound(c, "node not found")
		return
	}

	// Check if it's the last control plane node
	if node.Role == "control-plane" {
		var cpCount int64
		h.svcCtx.DB.Model(&model.ClusterNode{}).
			Where("cluster_id = ? AND role = ?", clusterID, "control-plane").
			Count(&cpCount)
		if cpCount <= 1 {
			httpx.BadRequest(c, "cannot remove the last control plane node")
			return
		}
	}

	// Drain and delete node via kubectl
	err := h.drainAndDeleteNode(c.Request.Context(), cluster.ID, nodeName)
	if err != nil {
		// Log error but continue with reset
		fmt.Printf("Warning: failed to drain node: %v\n", err)
	}

	// Execute kubeadm reset on the host
	if node.HostID != nil {
		var host model.Node
		if err := h.svcCtx.DB.First(&host, *node.HostID).Error; err == nil {
			h.executeResetOnHost(c.Request.Context(), &host)
		}
	}

	// Delete node record from database
	h.svcCtx.DB.Delete(&node)

	// Update host's cluster_id
	if node.HostID != nil {
		h.svcCtx.DB.Model(&model.Node{}).Where("id = ?", *node.HostID).Update("cluster_id", nil)
	}
	h.invalidateClusterCache(c.Request.Context(), clusterID)

	httpx.OK(c, gin.H{
		"message": fmt.Sprintf("Node %s removed from cluster", nodeName),
	})
}

// GetNodeDetail 获取节点详情。
//
// @Summary 获取节点详情
// @Description 获取指定集群节点的详细信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param name path string true "节点名称"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=NodeDetail}
// @Failure 400 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /clusters/{id}/nodes/{name} [get]
func (h *Handler) GetNodeDetail(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	nodeName := strings.TrimSpace(c.Param("name"))

	if clusterID == 0 || nodeName == "" {
		httpx.BindErr(c, nil)
		return
	}

	var node model.ClusterNode
	if err := h.svcCtx.DB.Where("cluster_id = ? AND name = ?", clusterID, nodeName).First(&node).Error; err != nil {
		httpx.NotFound(c, "node not found")
		return
	}

	// Parse JSON fields
	var labels map[string]string
	var taints []Taint
	var conditions []Condition

	if node.Labels != "" {
		json.Unmarshal([]byte(node.Labels), &labels)
	}
	if node.Taints != "" {
		json.Unmarshal([]byte(node.Taints), &taints)
	}
	if node.Conditions != "" {
		json.Unmarshal([]byte(node.Conditions), &conditions)
	}

	// Get host name if linked
	var hostName string
	if node.HostID != nil {
		var host model.Node
		if err := h.svcCtx.DB.First(&host, *node.HostID).Error; err == nil {
			hostName = host.Name
		}
	}

	detail := NodeDetail{
		ID:               node.ID,
		ClusterID:        node.ClusterID,
		HostID:           node.HostID,
		HostName:         hostName,
		Name:             node.Name,
		IP:               node.IP,
		Role:             node.Role,
		Status:           node.Status,
		KubeletVersion:   node.KubeletVersion,
		KubeProxyVersion: node.KubeProxyVersion,
		ContainerRuntime: node.ContainerRuntime,
		OSImage:          node.OSImage,
		KernelVersion:    node.KernelVersion,
		AllocatableCPU:   node.AllocatableCPU,
		AllocatableMem:   node.AllocatableMem,
		AllocatablePods:  node.AllocatablePods,
		Labels:           labels,
		Taints:           taints,
		Conditions:       conditions,
		JoinedAt:         node.JoinedAt,
		LastSeenAt:       node.LastSeenAt,
		CreatedAt:        node.CreatedAt,
		UpdatedAt:        node.UpdatedAt,
	}

	httpx.OK(c, detail)
}

// Helper methods - 辅助方法

// getJoinCommand 获取集群加入命令。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//
// 返回: 加入命令，失败返回错误
func (h *Handler) getJoinCommand(ctx context.Context, clusterID uint) (string, error) {
	// Get credential and build client
	var cred model.ClusterCredential
	if err := h.svcCtx.DB.Where("cluster_id = ?", clusterID).First(&cred).Error; err != nil {
		return "", err
	}

	restConfig, err := h.buildRestConfigFromCredential(&cred)
	if err != nil {
		return "", err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", err
	}

	// Generate token if needed - using kubeadm token create via SSH is more reliable
	// For simplicity, we'll use kubeadm token create via SSH
	_ = client // client is available if needed for future operations

	// Return join command template
	return "kubeadm token create --print-join-command", nil
}

// executeJoinOnHost 在主机上执行加入集群命令。
//
// 参数:
//   - ctx: 上下文
//   - host: 主机模型
//   - joinCommand: 加入命令
//   - role: 节点角色
//
// 返回: 失败返回错误
func (h *Handler) executeJoinOnHost(ctx context.Context, host *model.Node, joinCommand, role string) error {
	privateKey, passphrase, err := h.loadNodePrivateKey(ctx, host)
	if err != nil {
		return err
	}

	password := strings.TrimSpace(host.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}

	cli, err := sshclient.NewSSHClient(host.SSHUser, password, host.IP, host.Port, privateKey, passphrase)
	if err != nil {
		return err
	}
	defer cli.Close()

	// First get the actual join command from control plane
	// For now, we'll use a placeholder
	cmd := fmt.Sprintf("JOIN_COMMAND=$(bash -c '%s') && bash $JOIN_COMMAND", joinCommand)
	if role == "control-plane" {
		cmd += " --control-plane"
	}

	_, err = sshclient.RunCommand(cli, cmd)
	return err
}

// executeResetOnHost 在主机上执行 kubeadm reset。
//
// 参数:
//   - ctx: 上下文
//   - host: 主机模型
//
// 返回: 失败返回错误
func (h *Handler) executeResetOnHost(ctx context.Context, host *model.Node) error {
	privateKey, passphrase, err := h.loadNodePrivateKey(ctx, host)
	if err != nil {
		return err
	}

	password := strings.TrimSpace(host.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}

	cli, err := sshclient.NewSSHClient(host.SSHUser, password, host.IP, host.Port, privateKey, passphrase)
	if err != nil {
		return err
	}
	defer cli.Close()

	_, err = sshclient.RunCommand(cli, "kubeadm reset -f")
	return err
}

// drainAndDeleteNode 驱逐并删除节点。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//   - nodeName: 节点名称
//
// 返回: 失败返回错误
func (h *Handler) drainAndDeleteNode(ctx context.Context, clusterID uint, nodeName string) error {
	// Get credential
	var cred model.ClusterCredential
	if err := h.svcCtx.DB.Where("cluster_id = ?", clusterID).First(&cred).Error; err != nil {
		return err
	}

	restConfig, err := h.buildRestConfigFromCredential(&cred)
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// Cordon node
	_, err = client.CoreV1().Nodes().Patch(ctx, nodeName, types.StrategicMergePatchType,
		[]byte(`{"spec":{"unschedulable":true}}}`), v1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	// Delete pods on the node
	pods, err := client.CoreV1().Pods("").List(ctx, v1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		// Skip daemonset pods
		if pod.ObjectMeta.OwnerReferences != nil {
			for _, owner := range pod.ObjectMeta.OwnerReferences {
				if owner.Kind == "DaemonSet" {
					continue
				}
			}
		}

		// Delete pod
		gracePeriod := int64(0)
		client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, v1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})
	}

	// Delete node from API
	return client.CoreV1().Nodes().Delete(ctx, nodeName, v1.DeleteOptions{})
}
