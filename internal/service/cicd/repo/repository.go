// Package repo 提供 CI/CD 服务的数据访问层。
//
// 本文件包含以下数据访问方法:
//   - CI 配置: GetServiceCIConfig, UpsertServiceCIConfig, DeleteServiceCIConfig
//   - CI 运行: CreateCIRun, ListCIRuns
//   - CD 配置: GetDeploymentCDConfig, UpsertDeploymentCDConfig
//   - 发布: CreateRelease, GetRelease, SaveRelease, ListReleases
//   - 审批: CreateApproval, ListApprovals
//   - 审计: CreateAuditEvent, ListAuditEventsByService, ListAuditEvents
package repo

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// Repository 是 CI/CD 服务的数据访问层。
//
// 封装数据库操作，提供 CRUD 接口。
type Repository struct {
	db *gorm.DB // 数据库连接
}

// New 创建 Repository 实例。
//
// 参数:
//   - db: GORM 数据库连接
//
// 返回: 初始化的 Repository 实例
func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetServiceCIConfig 获取服务 CI 配置。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//
// 返回: CI 配置模型和错误信息
func (r *Repository) GetServiceCIConfig(ctx context.Context, serviceID uint) (*model.CICDServiceCIConfig, error) {
	var row model.CICDServiceCIConfig
	if err := r.db.WithContext(ctx).Where("service_id = ?", serviceID).Order("id DESC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// UpsertServiceCIConfig 创建或更新服务 CI 配置。
//
// 如果服务已存在 CI 配置则更新，否则创建新配置。
//
// 参数:
//   - ctx: 上下文
//   - in: CI 配置模型
//
// 返回: 创建或更新后的配置和错误信息
func (r *Repository) UpsertServiceCIConfig(ctx context.Context, in model.CICDServiceCIConfig) (*model.CICDServiceCIConfig, error) {
	var existing model.CICDServiceCIConfig
	err := r.db.WithContext(ctx).Where("service_id = ?", in.ServiceID).First(&existing).Error
	if err == nil {
		existing.RepoURL = in.RepoURL
		existing.Branch = in.Branch
		existing.BuildStepsJSON = in.BuildStepsJSON
		existing.ArtifactTarget = in.ArtifactTarget
		existing.TriggerMode = in.TriggerMode
		existing.Status = in.Status
		existing.UpdatedBy = in.UpdatedBy
		if uerr := r.db.WithContext(ctx).Save(&existing).Error; uerr != nil {
			return nil, uerr
		}
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if cerr := r.db.WithContext(ctx).Create(&in).Error; cerr != nil {
		return nil, cerr
	}
	return &in, nil
}

// DeleteServiceCIConfig 删除服务 CI 配置。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//
// 返回: 错误信息
func (r *Repository) DeleteServiceCIConfig(ctx context.Context, serviceID uint) error {
	return r.db.WithContext(ctx).Where("service_id = ?", serviceID).Delete(&model.CICDServiceCIConfig{}).Error
}

// CreateCIRun 创建 CI 运行记录。
//
// 参数:
//   - ctx: 上下文
//   - in: CI 运行模型
//
// 返回: 创建后的运行记录和错误信息
func (r *Repository) CreateCIRun(ctx context.Context, in model.CICDServiceCIRun) (*model.CICDServiceCIRun, error) {
	if err := r.db.WithContext(ctx).Create(&in).Error; err != nil {
		return nil, err
	}
	return &in, nil
}

// ListCIRuns 获取服务 CI 运行列表。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//
// 返回: CI 运行列表和错误信息
func (r *Repository) ListCIRuns(ctx context.Context, serviceID uint) ([]model.CICDServiceCIRun, error) {
	rows := make([]model.CICDServiceCIRun, 0)
	if err := r.db.WithContext(ctx).Where("service_id = ?", serviceID).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetDeploymentCDConfig 获取部署 CD 配置。
//
// 参数:
//   - ctx: 上下文
//   - deploymentID: 部署目标 ID
//   - env: 环境名称 (可选)
//   - runtimeType: 运行时类型 (可选)
//
// 返回: CD 配置模型和错误信息
func (r *Repository) GetDeploymentCDConfig(ctx context.Context, deploymentID uint, env, runtimeType string) (*model.CICDDeploymentCDConfig, error) {
	var row model.CICDDeploymentCDConfig
	q := r.db.WithContext(ctx).Where("deployment_id = ?", deploymentID)
	if env != "" {
		q = q.Where("env = ?", env)
	}
	if runtimeType != "" {
		q = q.Where("runtime_type = ?", runtimeType)
	}
	if err := q.Order("id DESC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// UpsertDeploymentCDConfig 创建或更新部署 CD 配置。
//
// 如果部署目标已存在 CD 配置则更新，否则创建新配置。
//
// 参数:
//   - ctx: 上下文
//   - in: CD 配置模型
//
// 返回: 创建或更新后的配置和错误信息
func (r *Repository) UpsertDeploymentCDConfig(ctx context.Context, in model.CICDDeploymentCDConfig) (*model.CICDDeploymentCDConfig, error) {
	var existing model.CICDDeploymentCDConfig
	err := r.db.WithContext(ctx).Where("deployment_id = ? AND env = ? AND runtime_type = ?", in.DeploymentID, in.Env, in.RuntimeType).First(&existing).Error
	if err == nil {
		existing.Strategy = in.Strategy
		existing.RuntimeType = in.RuntimeType
		existing.StrategyConfigJSON = in.StrategyConfigJSON
		existing.ApprovalRequired = in.ApprovalRequired
		existing.UpdatedBy = in.UpdatedBy
		if uerr := r.db.WithContext(ctx).Save(&existing).Error; uerr != nil {
			return nil, uerr
		}
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if cerr := r.db.WithContext(ctx).Create(&in).Error; cerr != nil {
		return nil, cerr
	}
	return &in, nil
}

// CreateRelease 创建发布记录。
//
// 参数:
//   - ctx: 上下文
//   - in: 发布模型
//
// 返回: 创建后的发布记录和错误信息
func (r *Repository) CreateRelease(ctx context.Context, in model.CICDRelease) (*model.CICDRelease, error) {
	if err := r.db.WithContext(ctx).Create(&in).Error; err != nil {
		return nil, err
	}
	return &in, nil
}

// GetRelease 获取发布记录。
//
// 参数:
//   - ctx: 上下文
//   - id: 发布 ID
//
// 返回: 发布模型和错误信息
func (r *Repository) GetRelease(ctx context.Context, id uint) (*model.CICDRelease, error) {
	var row model.CICDRelease
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// SaveRelease 保存发布记录。
//
// 参数:
//   - ctx: 上下文
//   - row: 发布模型指针
//
// 返回: 错误信息
func (r *Repository) SaveRelease(ctx context.Context, row *model.CICDRelease) error {
	return r.db.WithContext(ctx).Save(row).Error
}

// ListReleases 获取发布列表。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID (可选)
//   - deploymentID: 部署目标 ID (可选)
//   - runtimeType: 运行时类型 (可选)
//
// 返回: 发布列表和错误信息
func (r *Repository) ListReleases(ctx context.Context, serviceID, deploymentID uint, runtimeType string) ([]model.CICDRelease, error) {
	q := r.db.WithContext(ctx).Model(&model.CICDRelease{})
	if serviceID > 0 {
		q = q.Where("service_id = ?", serviceID)
	}
	if deploymentID > 0 {
		q = q.Where("deployment_id = ?", deploymentID)
	}
	if runtimeType != "" {
		q = q.Where("runtime_type = ?", runtimeType)
	}
	rows := make([]model.CICDRelease, 0)
	if err := q.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateApproval 创建审批记录。
//
// 参数:
//   - ctx: 上下文
//   - in: 审批模型
//
// 返回: 创建后的审批记录和错误信息
func (r *Repository) CreateApproval(ctx context.Context, in model.CICDReleaseApproval) (*model.CICDReleaseApproval, error) {
	if err := r.db.WithContext(ctx).Create(&in).Error; err != nil {
		return nil, err
	}
	return &in, nil
}

// ListApprovals 获取发布审批列表。
//
// 参数:
//   - ctx: 上下文
//   - releaseID: 发布 ID
//
// 返回: 审批列表和错误信息
func (r *Repository) ListApprovals(ctx context.Context, releaseID uint) ([]model.CICDReleaseApproval, error) {
	rows := make([]model.CICDReleaseApproval, 0)
	if err := r.db.WithContext(ctx).Where("release_id = ?", releaseID).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateAuditEvent 创建审计事件。
//
// 参数:
//   - ctx: 上下文
//   - in: 审计事件模型
//
// 返回: 创建后的审计事件和错误信息
func (r *Repository) CreateAuditEvent(ctx context.Context, in model.CICDAuditEvent) (*model.CICDAuditEvent, error) {
	if err := r.db.WithContext(ctx).Create(&in).Error; err != nil {
		return nil, err
	}
	return &in, nil
}

// ListAuditEventsByService 获取服务的审计事件列表。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID
//   - limit: 返回数量限制，默认 100
//
// 返回: 审计事件列表和错误信息
func (r *Repository) ListAuditEventsByService(ctx context.Context, serviceID uint, limit int) ([]model.CICDAuditEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows := make([]model.CICDAuditEvent, 0)
	if err := r.db.WithContext(ctx).Where("service_id = ?", serviceID).Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListAuditEvents 获取审计事件列表。
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务 ID (可选)
//   - traceID: 追踪 ID (可选)
//   - commandID: 命令 ID (可选)
//   - limit: 返回数量限制，默认 100
//
// 返回: 审计事件列表和错误信息
func (r *Repository) ListAuditEvents(ctx context.Context, serviceID uint, traceID, commandID string, limit int) ([]model.CICDAuditEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	q := r.db.WithContext(ctx).Model(&model.CICDAuditEvent{})
	if serviceID > 0 {
		q = q.Where("service_id = ?", serviceID)
	}
	if traceID != "" {
		q = q.Where("trace_id = ?", traceID)
	}
	if commandID != "" {
		q = q.Where("command_id = ?", commandID)
	}
	rows := make([]model.CICDAuditEvent, 0)
	if err := q.Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
