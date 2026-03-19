// Package logic 提供主机管理的业务逻辑实现。
package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud/volcengine"
	"github.com/cy77cc/OpsPilot/internal/utils"
)

// CloudAccountReq 创建云账号请求参数。
type CloudAccountReq struct {
	Provider        string `json:"provider"`
	AccountName     string `json:"account_name"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	RegionDefault   string `json:"region_default"`
}

// CloudQueryReq 查询云实例请求参数。
type CloudQueryReq struct {
	Provider  string `json:"provider"`
	AccountID uint64 `json:"account_id"`
	Region    string `json:"region"`
	Keyword   string `json:"keyword"`
}

// CloudImportReq 导入云实例请求参数。
type CloudImportReq struct {
	Provider  string               `json:"provider"`
	AccountID uint64               `json:"account_id"`
	Instances []CloudInstanceInfo  `json:"instances"`
	Role      string               `json:"role"`
	Labels    []string             `json:"labels"`
}

// CloudInstanceInfo 云实例信息（用于导入）。
type CloudInstanceInfo struct {
	InstanceID string `json:"instance_id"`
	Name       string `json:"name"`
	IP         string `json:"ip"`
	Region     string `json:"region"`
	Status     string `json:"status"`
	OS         string `json:"os"`
	CPU        int    `json:"cpu"`
	MemoryMB   int    `json:"memory_mb"`
	DiskGB     int    `json:"disk_gb"`
}

// init 初始化云厂商适配器注册表。
func init() {
	// 注册火山云适配器
	cloud.Register(volcengine.New())

	// 注册 Mock 适配器（阿里云、腾讯云）
	cloud.Register(cloud.NewMockProvider("alicloud", "阿里云"))
	cloud.Register(cloud.NewMockProvider("tencent", "腾讯云"))
}

// CreateCloudAccount 创建云账号。
//
// 参数:
//   - ctx: 请求上下文
//   - uid: 操作用户 ID
//   - req: 创建请求参数
//
// 返回:
//   - 成功返回创建的云账号
//   - 失败返回错误
func (s *HostService) CreateCloudAccount(ctx context.Context, uid uint64, req CloudAccountReq) (*model.HostCloudAccount, error) {
	if strings.TrimSpace(config.CFG.Security.EncryptionKey) == "" {
		return nil, errors.New("security.encryption_key is required")
	}
	if req.Provider == "" || req.AccountName == "" || req.AccessKeyID == "" || req.AccessKeySecret == "" {
		return nil, errors.New("provider/account_name/access_key_id/access_key_secret are required")
	}

	secretEnc, err := utils.EncryptText(req.AccessKeySecret, config.CFG.Security.EncryptionKey)
	if err != nil {
		return nil, err
	}

	acc := &model.HostCloudAccount{
		Provider:           req.Provider,
		AccountName:        req.AccountName,
		AccessKeyID:        req.AccessKeyID,
		AccessKeySecretEnc: secretEnc,
		RegionDefault:      req.RegionDefault,
		Status:             "active",
		CreatedBy:          uid,
	}

	if err := s.svcCtx.DB.WithContext(ctx).Create(acc).Error; err != nil {
		return nil, err
	}
	return acc, nil
}

// ListCloudAccounts 列出云账号。
//
// 参数:
//   - ctx: 请求上下文
//   - provider: 云厂商过滤（可选）
//
// 返回:
//   - 成功返回云账号列表
//   - 失败返回错误
func (s *HostService) ListCloudAccounts(ctx context.Context, provider string) ([]model.HostCloudAccount, error) {
	query := s.svcCtx.DB.WithContext(ctx).Model(&model.HostCloudAccount{}).
		Select("id", "provider", "account_name", "access_key_id", "region_default", "status", "created_by", "created_at", "updated_at")
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}

	var list []model.HostCloudAccount
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// TestCloudAccount 测试云账号凭证。
//
// 参数:
//   - ctx: 请求上下文
//   - req: 测试请求参数
//
// 返回:
//   - 成功返回测试结果
//   - 失败返回错误
func (s *HostService) TestCloudAccount(ctx context.Context, req CloudAccountReq) (map[string]any, error) {
	if req.Provider == "" || req.AccessKeyID == "" || req.AccessKeySecret == "" {
		return nil, errors.New("provider/access_key_id/access_key_secret are required")
	}

	// 获取云厂商适配器
	provider, err := cloud.GetProvider(req.Provider)
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}, nil
	}

	// 验证凭证
	region := req.RegionDefault
	if region == "" {
		region = "cn-beijing" // 默认地域
	}

	err = provider.ValidateCredential(ctx, req.AccessKeyID, req.AccessKeySecret, region)
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}, nil
	}

	return map[string]any{"ok": true, "provider": req.Provider, "message": "凭证验证成功"}, nil
}

// QueryCloudInstances 查询云实例列表。
//
// 参数:
//   - ctx: 请求上下文
//   - req: 查询请求参数
//
// 返回:
//   - 成功返回实例列表
//   - 失败返回错误
func (s *HostService) QueryCloudInstances(ctx context.Context, req CloudQueryReq) ([]CloudInstanceInfo, error) {
	if req.AccountID == 0 {
		return nil, errors.New("account_id is required")
	}

	// 获取云账号信息
	var account model.HostCloudAccount
	if err := s.svcCtx.DB.WithContext(ctx).First(&account, req.AccountID).Error; err != nil {
		return nil, err
	}

	// 获取云厂商适配器
	provider, err := cloud.GetProvider(account.Provider)
	if err != nil {
		return nil, err
	}

	// 解密 Secret
	secret, err := utils.DecryptText(account.AccessKeySecretEnc, config.CFG.Security.EncryptionKey)
	if err != nil {
		return nil, err
	}

	// 确定地域
	region := firstNonEmpty(req.Region, account.RegionDefault)

	// 调用适配器查询实例
	instances, err := provider.ListInstances(ctx, cloud.ListInstancesRequest{
		AccessKeyID:     account.AccessKeyID,
		AccessKeySecret: secret,
		Region:          region,
		Keyword:         req.Keyword,
	})
	if err != nil {
		return nil, err
	}

	// 转换为 CloudInstanceInfo
	result := make([]CloudInstanceInfo, 0, len(instances))
	for _, inst := range instances {
		result = append(result, CloudInstanceInfo{
			InstanceID: inst.InstanceID,
			Name:       inst.Name,
			IP:         inst.IP,
			Region:     inst.Region,
			Status:     inst.Status,
			OS:         inst.OS,
			CPU:        inst.CPU,
			MemoryMB:   inst.MemoryMB,
			DiskGB:     inst.DiskGB,
		})
	}

	return result, nil
}

// ImportCloudInstances 导入云实例。
//
// 参数:
//   - ctx: 请求上下文
//   - uid: 操作用户 ID
//   - req: 导入请求参数
//
// 返回:
//   - 成功返回导入任务和创建的节点列表
//   - 失败返回错误
func (s *HostService) ImportCloudInstances(ctx context.Context, uid uint64, req CloudImportReq) (*model.HostImportTask, []model.Node, error) {
	if len(req.Instances) == 0 {
		return nil, nil, errors.New("instances is empty")
	}

	task := &model.HostImportTask{
		ID:        uuid.NewString(),
		Provider:  req.Provider,
		AccountID: req.AccountID,
		Status:    "running",
		CreatedBy: uid,
	}

	requestJSON, _ := json.Marshal(req)
	task.RequestJSON = string(requestJSON)

	if err := s.svcCtx.DB.WithContext(ctx).Create(task).Error; err != nil {
		return nil, nil, err
	}

	created := make([]model.Node, 0, len(req.Instances))
	for _, ins := range req.Instances {
		node := model.Node{
			Name:        ins.Name,
			IP:          ins.IP,
			Port:        DefaultSSHPort,
			SSHUser:     "root",
			Status:      "online",
			Role:        req.Role,
			Labels:      strings.Join(req.Labels, ","),
			OS:          ins.OS,
			CpuCores:    ins.CPU,
			MemoryMB:    ins.MemoryMB,
			DiskGB:      ins.DiskGB,
			Source:      "cloud_import",
			Provider:    req.Provider,
			ProviderID:  ins.InstanceID,
			LastCheckAt: time.Now(),
		}

		if err := s.svcCtx.DB.WithContext(ctx).Create(&node).Error; err != nil {
			task.Status = "failed"
			task.ErrorMessage = err.Error()
			_ = s.svcCtx.DB.WithContext(ctx).Save(task).Error
			return task, nil, err
		}
		created = append(created, node)
	}

	resultJSON, _ := json.Marshal(created)
	task.Status = "success"
	task.ResultJSON = string(resultJSON)

	if err := s.svcCtx.DB.WithContext(ctx).Save(task).Error; err != nil {
		return nil, nil, err
	}

	return task, created, nil
}

// GetImportTask 获取导入任务详情。
//
// 参数:
//   - ctx: 请求上下文
//   - taskID: 任务 ID
//
// 返回:
//   - 成功返回任务详情
//   - 失败返回错误
func (s *HostService) GetImportTask(ctx context.Context, taskID string) (*model.HostImportTask, error) {
	var task model.HostImportTask
	if err := s.svcCtx.DB.WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

