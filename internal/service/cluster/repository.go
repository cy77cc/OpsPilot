// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件实现集群服务的数据访问层 (Repository)，提供数据库 CRUD 操作。
package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// Repository 集群服务数据访问层。
//
// 职责:
//   - 封装数据库操作
//   - 提供集群、节点、凭证等数据的 CRUD 方法
type Repository struct {
	db *gorm.DB // GORM 数据库实例
}

// NewRepository 创建集群服务数据访问层实例。
//
// 参数:
//   - db: GORM 数据库实例
//
// 返回: Repository 实例
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ListClusters 查询集群列表。
//
// 参数:
//   - ctx: 上下文
//   - status: 状态筛选条件 (可选)
//   - source: 来源筛选条件 (可选)
//
// 返回: 集群列表，失败返回错误
func (r *Repository) ListClusters(ctx context.Context, status, source string) ([]ClusterListItem, error) {
	var clusters []model.Cluster
	q := r.db.WithContext(ctx).Model(&model.Cluster{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if source != "" {
		q = q.Where("source = ?", source)
	}
	if err := q.Order("id DESC").Find(&clusters).Error; err != nil {
		return nil, err
	}

	type nodeCountRow struct {
		ClusterID uint
		Count     int64
	}
	counts := []nodeCountRow{}
	if len(clusters) > 0 {
		ids := make([]uint, 0, len(clusters))
		for _, c := range clusters {
			ids = append(ids, c.ID)
		}
		if err := r.db.WithContext(ctx).Model(&model.ClusterNode{}).
			Select("cluster_id, COUNT(1) as count").
			Where("cluster_id IN ?", ids).
			Group("cluster_id").
			Find(&counts).Error; err != nil {
			return nil, err
		}
	}
	countMap := map[uint]int64{}
	for _, row := range counts {
		countMap[row.ClusterID] = row.Count
	}

	items := make([]ClusterListItem, 0, len(clusters))
	for _, cl := range clusters {
		items = append(items, ClusterListItem{
			ID:          cl.ID,
			Name:        cl.Name,
			Version:     cl.Version,
			K8sVersion:  cl.K8sVersion,
			Status:      cl.Status,
			Source:      cl.Source,
			NodeCount:   int(countMap[cl.ID]),
			Endpoint:    cl.Endpoint,
			Description: cl.Description,
			LastSyncAt:  cl.LastSyncAt,
			CreatedAt:   cl.CreatedAt,
		})
	}
	return items, nil
}

// GetClusterModel 根据 ID 获取集群模型。
//
// 参数:
//   - ctx: 上下文
//   - id: 集群 ID
//
// 返回: 集群模型，不存在返回 ErrRecordNotFound
func (r *Repository) GetClusterModel(ctx context.Context, id uint) (*model.Cluster, error) {
	var row model.Cluster
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetClusterDetail 根据 ID 获取集群详情。
//
// 参数:
//   - ctx: 上下文
//   - id: 集群 ID
//
// 返回: 集群详情结构，不存在返回错误
func (r *Repository) GetClusterDetail(ctx context.Context, id uint) (ClusterDetail, error) {
	var cluster model.Cluster
	if err := r.db.WithContext(ctx).First(&cluster, id).Error; err != nil {
		return ClusterDetail{}, err
	}

	var nodeCount int64
	if err := r.db.WithContext(ctx).Model(&model.ClusterNode{}).Where("cluster_id = ?", cluster.ID).Count(&nodeCount).Error; err != nil {
		return ClusterDetail{}, err
	}

	return ClusterDetail{
		ID:             cluster.ID,
		Name:           cluster.Name,
		Description:    cluster.Description,
		Version:        cluster.Version,
		K8sVersion:     cluster.K8sVersion,
		Status:         cluster.Status,
		Source:         cluster.Source,
		Type:           cluster.Type,
		NodeCount:      int(nodeCount),
		Endpoint:       cluster.Endpoint,
		PodCIDR:        cluster.PodCIDR,
		ServiceCIDR:    cluster.ServiceCIDR,
		ManagementMode: cluster.ManagementMode,
		CredentialID:   cluster.CredentialID,
		LastSyncAt:     cluster.LastSyncAt,
		CreatedAt:      cluster.CreatedAt,
		UpdatedAt:      cluster.UpdatedAt,
	}, nil
}

// ListClusterNodes 查询集群节点列表。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//
// 返回: 节点列表，失败返回错误
func (r *Repository) ListClusterNodes(ctx context.Context, clusterID uint) ([]ClusterNode, error) {
	var nodes []model.ClusterNode
	if err := r.db.WithContext(ctx).
		Where("cluster_id = ?", clusterID).
		Order("role DESC, name ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	items := make([]ClusterNode, 0, len(nodes))
	for _, n := range nodes {
		items = append(items, ClusterNode{
			ID:               n.ID,
			ClusterID:        n.ClusterID,
			HostID:           n.HostID,
			Name:             n.Name,
			IP:               n.IP,
			Role:             n.Role,
			Status:           n.Status,
			KubeletVersion:   n.KubeletVersion,
			ContainerRuntime: n.ContainerRuntime,
			OSImage:          n.OSImage,
			KernelVersion:    n.KernelVersion,
			AllocatableCPU:   n.AllocatableCPU,
			AllocatableMem:   n.AllocatableMem,
			Labels:           n.Labels,
			CreatedAt:        n.CreatedAt,
			UpdatedAt:        n.UpdatedAt,
		})
	}
	return items, nil
}

// ListBootstrapProfiles 查询引导配置列表。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 引导配置列表，失败返回错误
func (r *Repository) ListBootstrapProfiles(ctx context.Context) ([]BootstrapProfileItem, error) {
	var rows []model.ClusterBootstrapProfile
	if err := r.db.WithContext(ctx).Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]BootstrapProfileItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toBootstrapProfileItem(row))
	}
	return items, nil
}

// CreateCluster 创建集群记录。
//
// 参数:
//   - ctx: 上下文
//   - in: 集群模型
//
// 返回: 失败返回错误
func (r *Repository) CreateCluster(ctx context.Context, in *model.Cluster) error {
	return r.db.WithContext(ctx).Create(in).Error
}

// CreateClusterCredential 创建集群凭证记录。
//
// 参数:
//   - ctx: 上下文
//   - in: 凭证模型
//
// 返回: 失败返回错误
func (r *Repository) CreateClusterCredential(ctx context.Context, in *model.ClusterCredential) error {
	return r.db.WithContext(ctx).Create(in).Error
}

// UpdateClusterCredentialID 更新集群的凭证 ID。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//   - credentialID: 凭证 ID
//
// 返回: 失败返回错误
func (r *Repository) UpdateClusterCredentialID(ctx context.Context, clusterID, credentialID uint) error {
	return r.db.WithContext(ctx).Model(&model.Cluster{}).Where("id = ?", clusterID).Update("credential_id", credentialID).Error
}

// UpdateCluster 更新集群信息。
//
// 参数:
//   - ctx: 上下文
//   - id: 集群 ID
//   - updates: 更新字段映射
//
// 返回: 不存在返回 ErrRecordNotFound
func (r *Repository) UpdateCluster(ctx context.Context, id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.Cluster{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteClusterWithRelations 删除集群及其关联数据。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//
// 返回: 在事务中删除节点、凭证和集群记录
func (r *Repository) DeleteClusterWithRelations(ctx context.Context, clusterID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("cluster_id = ?", clusterID).Delete(&model.ClusterNode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("cluster_id = ?", clusterID).Delete(&model.ClusterCredential{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", clusterID).Delete(&model.Cluster{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// FindClusterCredentialByClusterID 根据集群 ID 查找凭证。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//
// 返回: 凭证模型，不存在返回错误
func (r *Repository) FindClusterCredentialByClusterID(ctx context.Context, clusterID uint) (*model.ClusterCredential, error) {
	var cred model.ClusterCredential
	if err := r.db.WithContext(ctx).Where("cluster_id = ?", clusterID).First(&cred).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

// UpsertClusterNode 插入或更新集群节点。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//   - nodeName: 节点名称
//   - row: 节点模型 (用于创建)
//   - updates: 更新字段映射 (用于更新)
//
// 返回: 失败返回错误
func (r *Repository) UpsertClusterNode(ctx context.Context, clusterID uint, nodeName string, row model.ClusterNode, updates map[string]interface{}) error {
	var existing model.ClusterNode
	res := r.db.WithContext(ctx).Where("cluster_id = ? AND name = ?", clusterID, nodeName).First(&existing)
	if res.Error == nil {
		return r.db.WithContext(ctx).Model(&existing).Updates(updates).Error
	}
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		return res.Error
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

// UpdateClusterLastSync 更新集群最后同步时间。
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群 ID
//   - ts: 同步时间
//
// 返回: 失败返回错误
func (r *Repository) UpdateClusterLastSync(ctx context.Context, clusterID uint, ts *time.Time) error {
	return r.db.WithContext(ctx).Model(&model.Cluster{}).Where("id = ?", clusterID).Update("last_sync_at", ts).Error
}

// MustNotBeNil 检查 Repository 是否已初始化。
//
// 返回: 未初始化返回错误
func (r *Repository) MustNotBeNil() error {
	if r == nil || r.db == nil {
		return fmt.Errorf("cluster repository is not initialized")
	}
	return nil
}
