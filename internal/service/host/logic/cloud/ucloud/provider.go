// Package ucloud 提供 UCLOUD UHost 实例查询适配器实现。
//
// 本包封装 UCLOUD Go SDK，实现 cloud.CloudProvider 接口，
// 支持 UCLOUD UHost 实例的查询和导入功能。
package ucloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/ucloud/ucloud-sdk-go/services/uaccount"
	"github.com/ucloud/ucloud-sdk-go/services/uhost"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Provider UCLOUD 适配器。
//
// 实现 cloud.CloudProvider 接口，提供 UCLOUD UHost 实例查询能力。
type Provider struct{}

// New 创建 UCLOUD 适配器实例。
func New() *Provider {
	return &Provider{}
}

// Name 返回云厂商标识。
func (p *Provider) Name() string {
	return "ucloud"
}

// DisplayName 返回云厂商显示名称。
func (p *Provider) DisplayName() string {
	return "UCLOUD"
}

// Capabilities 返回 UCLOUD 能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
	return cloud.ProviderCapabilities{
		DynamicRegions: true,
	}
}

// ValidateCredential 验证 UCLOUD 凭证是否有效。
//
// 通过调用 DescribeUHostInstance API（限制返回 1 条）验证凭证有效性。
func (p *Provider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	client, err := NewClient(ak, sk, region)
	if err != nil {
		return err
	}

	// 通过查询实例验证凭证，限制返回 1 条
	limit := 1
	req := &uhost.DescribeUHostInstanceRequest{}
	req.Region = &region
	req.Limit = &limit

	_, err = client.DescribeUHostInstance(ctx, req)
	if err != nil {
		return fmt.Errorf("UCLOUD 凭证验证失败: %w", p.wrapError(err))
	}
	return nil
}

// ListInstances 查询 UCLOUD UHost 实例列表。
func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
	client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
	if err != nil {
		return nil, err
	}

	// 构建查询参数
	input := &uhost.DescribeUHostInstanceRequest{}
	input.Region = &req.Region

	// 可用区过滤（仅当 Zone 有效时才设置）
	if req.Zone != "" && req.Zone != "undefined" {
		input.Zone = &req.Zone
	}

	// 分页参数（UCLOUD 使用 Offset/Limit）
	limit := 100
	if req.PageSize > 0 {
		limit = req.PageSize
	}
	input.Limit = &limit

	if req.PageNumber > 1 {
		offset := (req.PageNumber - 1) * req.PageSize
		input.Offset = &offset
	}

	// 调用 API
	output, err := client.DescribeUHostInstance(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("查询 UCLOUD 实例失败: %w", p.wrapError(err))
	}

	// 转换实例数据
	instances := make([]cloud.CloudInstance, 0, len(output.UHostSet))
	for i := range output.UHostSet {
		converted := ConvertInstance(&output.UHostSet[i], req.Region)

		// 如果有关键词，进行过滤
		if req.Keyword != "" {
			kw := strings.ToLower(req.Keyword)
			if !strings.Contains(strings.ToLower(converted.Name), kw) &&
				!strings.Contains(strings.ToLower(converted.InstanceID), kw) &&
				!strings.Contains(converted.IP, kw) &&
				!strings.Contains(converted.PrivateIP, kw) {
				continue
			}
		}

		instances = append(instances, *converted)
	}

	return instances, nil
}

// ListRegions 查询 UCLOUD 支持的地域列表。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
	// UCLOUD GetRegion 不需要指定地域，使用默认值
	client, err := NewClient(ak, sk, "cn-bj2")
	if err != nil {
		return nil, err
	}

	req := &uaccount.GetRegionRequest{}
	output, err := client.GetRegion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("查询 UCLOUD 地域失败: %w", p.wrapError(err))
	}

	// UCLOUD 返回的 Regions 包含重复地域（每个可用区一条记录），需要去重
	regionSet := make(map[string]bool)
	for _, r := range output.Regions {
		regionSet[r.Region] = true
	}

	regions := make([]cloud.Region, 0, len(regionSet))
	for regionId := range regionSet {
		regions = append(regions, cloud.Region{
			RegionId:  regionId,
			LocalName: getRegionLocalName(regionId),
		})
	}
	return regions, nil
}

// ListZones 查询 UCLOUD 指定地域的可用区列表。
func (p *Provider) ListZones(ctx context.Context, ak, sk, region string) ([]cloud.Zone, error) {
	if region == "" {
		return nil, fmt.Errorf("地域不能为空")
	}

	client, err := NewClient(ak, sk, region)
	if err != nil {
		return nil, err
	}

	req := &uaccount.GetRegionRequest{}
	output, err := client.GetRegion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("查询 UCLOUD 可用区失败: %w", p.wrapError(err))
	}

	// UCLOUD 返回每个可用区一条记录，筛选指定地域的可用区
	zoneSet := make(map[string]bool)
	for _, r := range output.Regions {
		if r.Region == region {
			zoneSet[r.Zone] = true
		}
	}

	zones := make([]cloud.Zone, 0, len(zoneSet))
	for zoneId := range zoneSet {
		zones = append(zones, cloud.Zone{
			ZoneId:    zoneId,
			LocalName: getZoneLocalName(zoneId),
		})
	}
	return zones, nil
}

// wrapError 包装 UCLOUD 错误，提供更友好的错误信息。
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	// UCLOUD SDK 错误格式检查
	errStr := err.Error()
	if strings.Contains(errStr, "160") {
		return fmt.Errorf("地域无效，请使用正确的地域标识如 cn-bj2、cn-sh2")
	}
	if strings.Contains(errStr, "170") {
		return fmt.Errorf("认证失败，请检查 AccessKey ID 和 Secret")
	}
	if strings.Contains(errStr, "171") {
		return fmt.Errorf("AccessKey 无效")
	}
	if strings.Contains(errStr, "172") {
		return fmt.Errorf("请求频率限制，已重试多次仍失败，请稍后重试")
	}

	return fmt.Errorf("[UCLOUD] %w", err)
}
