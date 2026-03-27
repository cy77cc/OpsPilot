// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件实现外部集群导入相关的业务逻辑，包括连接测试、凭证加密和节点同步。
package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// clusterCredentialMetadata 集群凭证元数据结构。
type clusterCredentialMetadata struct {
	SkipTLSVerify bool `json:"skip_tls_verify,omitempty"` // 是否跳过 TLS 验证
}

// ImportCluster 导入外部 Kubernetes 集群。
//
// 参数:
//   - ctx: 上下文
//   - uid: 用户 ID
//   - req: 集群创建请求
//
// 返回: 集群详情，失败返回错误
//
// 流程:
//  1. 验证认证方式和参数
//  2. 测试集群连接
//  3. 创建集群和凭证记录
//  4. 同步节点信息
func (h *Handler) ImportCluster(ctx context.Context, uid uint64, req ClusterCreateReq) (*ClusterDetail, error) {
	authMethod := strings.TrimSpace(req.AuthMethod)
	if authMethod == "cert" {
		authMethod = "certificate"
	}
	if authMethod == "" {
		if strings.TrimSpace(req.Kubeconfig) != "" {
			authMethod = "kubeconfig"
		} else if strings.TrimSpace(req.CACert) != "" && strings.TrimSpace(req.Cert) != "" && strings.TrimSpace(req.Key) != "" {
			authMethod = "certificate"
		} else if strings.TrimSpace(req.Token) != "" {
			authMethod = "token"
		}
	}

	// Validate based on auth method
	switch authMethod {
	case "kubeconfig":
		if strings.TrimSpace(req.Kubeconfig) == "" {
			return nil, fmt.Errorf("kubeconfig is required for kubeconfig auth method")
		}
		if _, err := clientcmd.Load([]byte(req.Kubeconfig)); err != nil {
			return nil, fmt.Errorf("invalid kubeconfig: %w", err)
		}
	case "certificate":
		if strings.TrimSpace(req.Endpoint) == "" {
			return nil, fmt.Errorf("endpoint is required for certificate auth method")
		}
		if strings.TrimSpace(req.CACert) == "" || strings.TrimSpace(req.Cert) == "" || strings.TrimSpace(req.Key) == "" {
			return nil, fmt.Errorf("ca_cert, cert and key are required for certificate auth method")
		}
	case "token":
		if strings.TrimSpace(req.Endpoint) == "" {
			return nil, fmt.Errorf("endpoint is required for token auth method")
		}
		if strings.TrimSpace(req.Token) == "" {
			return nil, fmt.Errorf("token is required for token auth method")
		}
	}

	// Test connection and get cluster info
	endpoint, version, err := h.testConnection(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Create cluster record
	cluster := &model.Cluster{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Source:      "external_managed",
		Type:        "kubernetes",
		Endpoint:    endpoint,
		Version:     version,
		K8sVersion:  version,
		Status:      "active",
		AuthMethod:  authMethod,
		Nodes:       "[]", // 空JSON数组，避免MySQL JSON类型字段报错
		CreatedBy:   fmt.Sprintf("%d", uid),
	}

	if err := h.repo.CreateCluster(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster record: %w", err)
	}

	// Create credential record
	cred := &model.ClusterCredential{
		Name:        fmt.Sprintf("%s-credential", cluster.Name),
		RuntimeType: "k8s",
		Source:      "external_managed",
		ClusterID:   cluster.ID,
		Endpoint:    endpoint,
		AuthMethod:  authMethod,
		Status:      "active",
		CreatedBy:   uid,
	}
	if req.SkipTLSVerify {
		meta, _ := json.Marshal(clusterCredentialMetadata{SkipTLSVerify: true})
		cred.MetadataJSON = string(meta)
	}

	if err := h.encryptCredentialMaterials(cred, req); err != nil {
		h.svcCtx.DB.Delete(cluster)
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	if err := h.repo.CreateClusterCredential(ctx, cred); err != nil {
		_ = h.repo.DeleteClusterWithRelations(ctx, cluster.ID)
		return nil, fmt.Errorf("failed to create credential record: %w", err)
	}

	// Update cluster with credential ID
	_ = h.repo.UpdateClusterCredentialID(ctx, cluster.ID, cred.ID)

	// Sync nodes using the credential
	h.syncClusterNodesWithCred(ctx, cluster.ID, cred)
	h.invalidateClusterCache(ctx, cluster.ID)

	return &ClusterDetail{
		ID:          cluster.ID,
		Name:        cluster.Name,
		Description: cluster.Description,
		Version:     cluster.Version,
		K8sVersion:  cluster.K8sVersion,
		Status:      cluster.Status,
		Source:      cluster.Source,
		Type:        cluster.Type,
		Endpoint:    cluster.Endpoint,
		CreatedAt:   cluster.CreatedAt,
		UpdatedAt:   cluster.UpdatedAt,
	}, nil
}

// testKubeconfigConnection 使用 kubeconfig 测试集群连接。
//
// 参数:
//   - kubeconfig: kubeconfig 内容
//
// 返回: API Server 地址、版本信息、错误
func (h *Handler) testKubeconfigConnection(kubeconfig string) (string, string, error) {
	if strings.TrimSpace(kubeconfig) == "" {
		return "", "", fmt.Errorf("kubeconfig is empty")
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return "", "", fmt.Errorf("failed to build rest config: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create k8s client: %w", err)
	}

	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return "", "", fmt.Errorf("failed to get server version: %w", err)
	}

	return restConfig.Host, version.GitVersion, nil
}

// testConnection 测试集群连接。
//
// 参数:
//   - req: 集群创建请求
//
// 返回: API Server 地址、版本信息、错误
func (h *Handler) testConnection(req ClusterCreateReq) (string, string, error) {
	restConfig, err := h.buildRestConfigFromRequest(req)
	if err != nil {
		return "", "", err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create k8s client: %w", err)
	}

	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return "", "", fmt.Errorf("failed to get server version: %w", err)
	}

	return restConfig.Host, version.GitVersion, nil
}

// buildRestConfigFromRequest 从请求构建 REST 配置。
//
// 参数:
//   - req: 集群创建请求
//
// 返回: REST 配置，失败返回错误
func (h *Handler) buildRestConfigFromRequest(req ClusterCreateReq) (*rest.Config, error) {
	// Try kubeconfig first
	if strings.TrimSpace(req.Kubeconfig) != "" {
		return clientcmd.RESTConfigFromKubeConfig([]byte(req.Kubeconfig))
	}

	// Build from endpoint + cert/token
	endpoint := strings.TrimSpace(req.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for cert/token auth")
	}

	config := &rest.Config{
		Host: endpoint,
	}
	if req.SkipTLSVerify {
		config.TLSClientConfig.Insecure = true
	}

	// Set CA cert if provided
	if strings.TrimSpace(req.CACert) != "" {
		config.TLSClientConfig = rest.TLSClientConfig{
			CAData:   []byte(strings.TrimSpace(req.CACert)),
			Insecure: req.SkipTLSVerify,
		}
	}

	// Set client cert if provided
	if strings.TrimSpace(req.Cert) != "" && strings.TrimSpace(req.Key) != "" {
		config.TLSClientConfig.CertData = []byte(strings.TrimSpace(req.Cert))
		config.TLSClientConfig.KeyData = []byte(strings.TrimSpace(req.Key))
	}

	// Set token if provided
	if strings.TrimSpace(req.Token) != "" {
		config.BearerToken = strings.TrimSpace(req.Token)
	}

	return config, nil
}

// encryptCredentialMaterials 加密并存储凭证材料。
//
// 参数:
//   - cred: 凭证模型
//   - req: 集群创建请求
//
// 返回: 失败返回错误
func (h *Handler) encryptCredentialMaterials(cred *model.ClusterCredential, req ClusterCreateReq) error {
	enc := strings.TrimSpace(config.CFG.Security.EncryptionKey)
	if enc == "" {
		return fmt.Errorf("security.encryption_key is required")
	}

	var err error
	if strings.TrimSpace(req.Kubeconfig) != "" {
		cred.KubeconfigEnc, err = utils.EncryptText(strings.TrimSpace(req.Kubeconfig), enc)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(req.CACert) != "" {
		cred.CACertEnc, err = utils.EncryptText(strings.TrimSpace(req.CACert), enc)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(req.Cert) != "" {
		cred.CertEnc, err = utils.EncryptText(strings.TrimSpace(req.Cert), enc)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(req.Key) != "" {
		cred.KeyEnc, err = utils.EncryptText(strings.TrimSpace(req.Key), enc)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(req.Token) != "" {
		cred.TokenEnc, err = utils.EncryptText(strings.TrimSpace(req.Token), enc)
		if err != nil {
			return err
		}
	}

	return nil
}

// syncClusterNodes 从 Kubernetes API 同步集群节点。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//   - kubeconfig: kubeconfig 内容
//
// 返回: 失败返回错误
func (h *Handler) syncClusterNodes(ctx context.Context, clusterID uint, kubeconfig string) error {
	if strings.TrimSpace(kubeconfig) == "" {
		return nil
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
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
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" {
				if condition.Status == "True" {
					status = "ready"
				} else {
					status = "notready"
				}
				break
			}
		}

		// Get IP addresses
		var ip string
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				ip = addr.Address
				break
			}
		}

		// Get labels
		labelsJSON, _ := json.Marshal(node.Labels)

		// Get taints
		taintsJSON, _ := json.Marshal(node.Spec.Taints)

		// Get allocatable resources
		allocatableCPU := node.Status.Allocatable.Cpu().String()
		allocatableMem := node.Status.Allocatable.Memory().String()

		// Get kubelet version
		kubeletVersion := node.Status.NodeInfo.KubeletVersion
		containerRuntime := node.Status.NodeInfo.ContainerRuntimeVersion
		osImage := node.Status.NodeInfo.OSImage
		kernelVersion := node.Status.NodeInfo.KernelVersion

		clusterNode := model.ClusterNode{
			ClusterID:        clusterID,
			Name:             node.Name,
			IP:               ip,
			Role:             role,
			Status:           status,
			KubeletVersion:   kubeletVersion,
			ContainerRuntime: containerRuntime,
			OSImage:          osImage,
			KernelVersion:    kernelVersion,
			AllocatableCPU:   allocatableCPU,
			AllocatableMem:   allocatableMem,
			Labels:           string(labelsJSON),
			Taints:           string(taintsJSON),
			LastSeenAt:       &now,
		}

		// Upsert node
		var existing model.ClusterNode
		result := h.svcCtx.DB.WithContext(ctx).
			Where("cluster_id = ? AND name = ?", clusterID, node.Name).
			First(&existing)

		if result.Error == nil {
			// Update existing
			h.svcCtx.DB.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
				"ip":                ip,
				"role":              role,
				"status":            status,
				"kubelet_version":   kubeletVersion,
				"container_runtime": containerRuntime,
				"os_image":          osImage,
				"kernel_version":    kernelVersion,
				"allocatable_cpu":   allocatableCPU,
				"allocatable_mem":   allocatableMem,
				"labels":            string(labelsJSON),
				"taints":            string(taintsJSON),
				"last_seen_at":      &now,
			})
		} else {
			// Create new
			h.svcCtx.DB.WithContext(ctx).Create(&clusterNode)
		}
	}

	// Update cluster last_sync_at
	h.svcCtx.DB.WithContext(ctx).Model(&model.Cluster{}).
		Where("id = ?", clusterID).
		Update("last_sync_at", &now)

	return nil
}

// syncClusterNodesWithCred 使用存储的凭证同步集群节点。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//   - cred: 凭证模型
//
// 返回: 失败返回错误
func (h *Handler) syncClusterNodesWithCred(ctx context.Context, clusterID uint, cred *model.ClusterCredential) error {
	restConfig, err := h.buildRestConfigFromCredential(cred)
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
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
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" {
				if condition.Status == "True" {
					status = "ready"
				} else {
					status = "notready"
				}
				break
			}
		}

		// Get IP addresses
		var ip string
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				ip = addr.Address
				break
			}
		}

		// Get labels
		labelsJSON, _ := json.Marshal(node.Labels)

		// Get taints
		taintsJSON, _ := json.Marshal(node.Spec.Taints)

		// Get allocatable resources
		allocatableCPU := node.Status.Allocatable.Cpu().String()
		allocatableMem := node.Status.Allocatable.Memory().String()

		// Get kubelet version
		kubeletVersion := node.Status.NodeInfo.KubeletVersion
		containerRuntime := node.Status.NodeInfo.ContainerRuntimeVersion
		osImage := node.Status.NodeInfo.OSImage
		kernelVersion := node.Status.NodeInfo.KernelVersion

		clusterNode := model.ClusterNode{
			ClusterID:        clusterID,
			Name:             node.Name,
			IP:               ip,
			Role:             role,
			Status:           status,
			KubeletVersion:   kubeletVersion,
			ContainerRuntime: containerRuntime,
			OSImage:          osImage,
			KernelVersion:    kernelVersion,
			AllocatableCPU:   allocatableCPU,
			AllocatableMem:   allocatableMem,
			Labels:           string(labelsJSON),
			Taints:           string(taintsJSON),
			LastSeenAt:       &now,
		}

		// Upsert node
		var existing model.ClusterNode
		result := h.svcCtx.DB.WithContext(ctx).
			Where("cluster_id = ? AND name = ?", clusterID, node.Name).
			First(&existing)

		if result.Error == nil {
			// Update existing
			h.svcCtx.DB.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
				"ip":                ip,
				"role":              role,
				"status":            status,
				"kubelet_version":   kubeletVersion,
				"container_runtime": containerRuntime,
				"os_image":          osImage,
				"kernel_version":    kernelVersion,
				"allocatable_cpu":   allocatableCPU,
				"allocatable_mem":   allocatableMem,
				"labels":            string(labelsJSON),
				"taints":            string(taintsJSON),
				"last_seen_at":      &now,
			})
		} else {
			// Create new
			h.svcCtx.DB.WithContext(ctx).Create(&clusterNode)
		}
	}

	// Update cluster last_sync_at
	h.svcCtx.DB.WithContext(ctx).Model(&model.Cluster{}).
		Where("id = ?", clusterID).
		Update("last_sync_at", &now)

	return nil
}

// TestConnectivity 测试集群连通性。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//
// 返回: 连通性测试结果
func (h *Handler) TestConnectivity(ctx context.Context, clusterID uint) (*ClusterTestResp, error) {
	var cred model.ClusterCredential
	if err := h.svcCtx.DB.WithContext(ctx).
		Where("cluster_id = ?", clusterID).
		First(&cred).Error; err != nil {
		return nil, fmt.Errorf("credential not found: %w", err)
	}

	restConfig, err := h.buildRestConfigFromCredential(&cred)
	if err != nil {
		return &ClusterTestResp{
			ClusterID: clusterID,
			Connected: false,
			Message:   err.Error(),
		}, nil
	}

	start := time.Now()
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return &ClusterTestResp{
			ClusterID: clusterID,
			Connected: false,
			Message:   err.Error(),
		}, nil
	}

	version, err := client.Discovery().ServerVersion()
	latency := time.Since(start).Milliseconds()

	result := &ClusterTestResp{
		ClusterID: clusterID,
		LatencyMS: latency,
	}

	if err != nil {
		result.Connected = false
		result.Message = err.Error()
	} else {
		result.Connected = true
		result.Message = "OK"
		result.Version = version.GitVersion
	}

	// Update credential test status
	now := time.Now().UTC()
	status := "failed"
	if result.Connected {
		status = "ok"
	}
	h.svcCtx.DB.WithContext(ctx).Model(&cred).Updates(map[string]interface{}{
		"last_test_at":      &now,
		"last_test_status":  status,
		"last_test_message": result.Message,
	})

	return result, nil
}

// buildRestConfigFromCredential 从存储的凭证构建 REST 配置。
//
// 参数:
//   - cred: 凭证模型
//
// 返回: REST 配置，失败返回错误
func (h *Handler) buildRestConfigFromCredential(cred *model.ClusterCredential) (*rest.Config, error) {
	enc := strings.TrimSpace(config.CFG.Security.EncryptionKey)
	if enc == "" {
		return nil, fmt.Errorf("security.encryption_key is required")
	}

	if strings.TrimSpace(cred.KubeconfigEnc) != "" {
		kubeconfig, err := utils.DecryptText(cred.KubeconfigEnc, enc)
		if err != nil {
			return nil, err
		}
		return clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	}

	meta := clusterCredentialMetadata{}
	if strings.TrimSpace(cred.MetadataJSON) != "" {
		_ = json.Unmarshal([]byte(cred.MetadataJSON), &meta)
	}

	// Build from cert/token
	result := &rest.Config{
		Host: strings.TrimSpace(cred.Endpoint),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: meta.SkipTLSVerify,
		},
	}

	if strings.TrimSpace(cred.CACertEnc) != "" {
		ca, err := utils.DecryptText(cred.CACertEnc, enc)
		if err != nil {
			return nil, err
		}
		result.TLSClientConfig.CAData = []byte(ca)
	}

	if strings.TrimSpace(cred.CertEnc) != "" {
		cert, err := utils.DecryptText(cred.CertEnc, enc)
		if err != nil {
			return nil, err
		}
		result.TLSClientConfig.CertData = []byte(cert)

		key, err := utils.DecryptText(cred.KeyEnc, enc)
		if err != nil {
			return nil, err
		}
		result.TLSClientConfig.KeyData = []byte(key)
	}

	if strings.TrimSpace(cred.TokenEnc) != "" {
		token, err := utils.DecryptText(cred.TokenEnc, enc)
		if err != nil {
			return nil, err
		}
		result.BearerToken = token
	}

	return result, nil
}

// ImportExternalCluster 处理外部集群导入 HTTP 请求。
//
// @Summary 导入外部集群
// @Description 导入外部 Kubernetes 集群到平台管理
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body ClusterCreateReq true "集群导入请求"
// @Success 200 {object} httpx.Response{data=ClusterDetail}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/import [post]
func (h *Handler) ImportExternalCluster(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cluster:write") {
		return
	}

	var req ClusterCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	uid := httpx.UIDFromCtx(c)
	cluster, err := h.ImportCluster(c.Request.Context(), uid, req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, cluster)
}

// ValidateImport 验证导入参数但不实际导入。
//
// @Summary 验证集群导入
// @Description 验证集群导入参数和连接性
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param request body ClusterCreateReq true "集群导入请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Router /clusters/import/validate [post]
func (h *Handler) ValidateImport(c *gin.Context) {
	var req ClusterCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	// Determine auth method
	authMethod := strings.TrimSpace(req.AuthMethod)
	if authMethod == "cert" {
		authMethod = "certificate"
	}
	if authMethod == "" {
		if strings.TrimSpace(req.Kubeconfig) != "" {
			authMethod = "kubeconfig"
		} else if strings.TrimSpace(req.Endpoint) != "" && strings.TrimSpace(req.CACert) != "" && strings.TrimSpace(req.Cert) != "" && strings.TrimSpace(req.Key) != "" {
			authMethod = "certificate"
		} else if strings.TrimSpace(req.Endpoint) != "" && strings.TrimSpace(req.Token) != "" {
			authMethod = "token"
		}
	}

	// Validate required fields based on auth method
	switch authMethod {
	case "kubeconfig":
		if strings.TrimSpace(req.Kubeconfig) == "" {
			httpx.BadRequest(c, "kubeconfig is required")
			return
		}
		// Validate kubeconfig format
		if _, err := clientcmd.Load([]byte(req.Kubeconfig)); err != nil {
			httpx.BadRequest(c, fmt.Sprintf("invalid kubeconfig: %v", err))
			return
		}
	case "certificate":
		if strings.TrimSpace(req.Endpoint) == "" {
			httpx.BadRequest(c, "endpoint is required")
			return
		}
		if strings.TrimSpace(req.CACert) == "" || strings.TrimSpace(req.Cert) == "" || strings.TrimSpace(req.Key) == "" {
			httpx.BadRequest(c, "ca_cert, cert and key are required for certificate auth")
			return
		}
	case "token":
		if strings.TrimSpace(req.Endpoint) == "" {
			httpx.BadRequest(c, "endpoint is required")
			return
		}
		if strings.TrimSpace(req.Token) == "" {
			httpx.BadRequest(c, "token is required")
			return
		}
	default:
		// No valid auth method detected, try to test anyway
	}

	// Test connection
	endpoint, version, err := h.testConnection(req)
	if err != nil {
		httpx.OK(c, gin.H{
			"valid":    false,
			"message":  err.Error(),
			"endpoint": endpoint,
		})
		return
	}

	httpx.OK(c, gin.H{
		"valid":       true,
		"message":     "Connection successful",
		"endpoint":    endpoint,
		"version":     version,
		"auth_method": authMethod,
	})
}
