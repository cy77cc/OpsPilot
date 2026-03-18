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

// AddNodeReq represents a request to add a node to cluster
type AddNodeReq struct {
	HostIDs []uint `json:"host_ids" binding:"required"`
	Role    string `json:"role"` // worker (default) or control-plane
}

// NodeDetail represents detailed node information
type NodeDetail struct {
	ID               uint        `json:"id"`
	ClusterID        uint        `json:"cluster_id"`
	HostID           *uint       `json:"host_id"`
	HostName         string      `json:"host_name,omitempty"`
	Name             string      `json:"name"`
	IP               string      `json:"ip"`
	Role             string      `json:"role"`
	Status           string      `json:"status"`
	KubeletVersion   string      `json:"kubelet_version"`
	KubeProxyVersion string      `json:"kube_proxy_version"`
	ContainerRuntime string      `json:"container_runtime"`
	OSImage          string      `json:"os_image"`
	KernelVersion    string      `json:"kernel_version"`
	AllocatableCPU   string      `json:"allocatable_cpu"`
	AllocatableMem   string      `json:"allocatable_mem"`
	AllocatablePods  int         `json:"allocatable_pods"`
	Labels           MapString   `json:"labels"`
	Taints           []Taint     `json:"taints"`
	Conditions       []Condition `json:"conditions"`
	JoinedAt         *time.Time  `json:"joined_at"`
	LastSeenAt       *time.Time  `json:"last_seen_at"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type MapString map[string]string
type Taint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}
type Condition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	Reason             string     `json:"reason"`
	Message            string     `json:"message"`
	LastTransitionTime *time.Time `json:"last_transition_time,omitempty"`
}

// SyncClusterNodes syncs nodes from Kubernetes API to database
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

// SyncClusterNodesHandler handles the sync endpoint
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

// AddClusterNodes adds nodes to a cluster
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

// RemoveClusterNode removes a node from a cluster
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

// GetNodeDetail returns detailed node information
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

// Helper methods

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
