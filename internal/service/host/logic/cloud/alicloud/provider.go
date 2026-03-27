// Package alicloud 提供阿里云 ECS 实例查询适配器实现。
//
// 本包封装阿里云 Go SDK，实现 cloud.CloudProvider 接口，
// 支持阿里云 ECS 实例的查询和导入功能。
package alicloud

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/client"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Provider 阿里云适配器。
//
// 实现 cloud.CloudProvider 接口，提供阿里云 ECS 实例查询能力。
type Provider struct{}

// New 创建阿里云适配器实例。
func New() *Provider {
	return &Provider{}
}

// Name 返回云厂商标识。
func (p *Provider) Name() string {
	return "alicloud"
}

// DisplayName 返回云厂商显示名称。
func (p *Provider) DisplayName() string {
	return "阿里云"
}

// Capabilities 返回阿里云能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
	return cloud.ProviderCapabilities{
		DynamicRegions: true,
	}
}

// ValidateCredential 验证阿里云凭证是否有效。
//
// 通过调用 DescribeRegions API 验证凭证有效性。
func (p *Provider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	client, err := NewClient(ak, sk, region)
	if err != nil {
		return err
	}

	// 通过查询地域验证凭证
	_, err = client.DescribeRegions(ctx, &ecs.DescribeRegionsRequest{})
	if err != nil {
		return fmt.Errorf("阿里云凭证验证失败: %w", p.wrapError(err))
	}
	return nil
}

// ListInstances 查询阿里云 ECS 实例列表。
func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
	client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
	if err != nil {
		return nil, err
	}

	// 构建查询参数
	input := &ecs.DescribeInstancesRequest{
		RegionId: tea.String(req.Region),
	}

	// 可用区过滤
	if req.Zone != "" {
		input.ZoneId = tea.String(req.Zone)
	}

	// 分页参数
	if req.PageSize > 0 {
		input.PageSize = tea.Int(req.PageSize)
	} else {
		input.PageSize = tea.Int(100)
	}

	if req.PageNumber > 0 {
		input.PageNumber = tea.Int(req.PageNumber)
	} else {
		input.PageNumber = tea.Int(1)
	}

	// 调用 API
	output, err := client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("查询阿里云实例失败: %w", p.wrapError(err))
	}

	// 转换实例数据
	instances := make([]cloud.CloudInstance, 0, len(output.Instances.Instance))
	for _, inst := range output.Instances.Instance {
		converted := ConvertInstance(inst)

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

// ListRegions 查询阿里云支持的地域列表。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
	// 使用默认地域创建客户端（DescribeRegions 不需要指定地域）
	client, err := NewClient(ak, sk, "cn-hangzhou")
	if err != nil {
		return nil, err
	}

	output, err := client.DescribeRegions(ctx, &ecs.DescribeRegionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("查询阿里云地域失败: %w", p.wrapError(err))
	}

	regions := make([]cloud.Region, 0, len(output.Regions.Region))
	for _, r := range output.Regions.Region {
		regions = append(regions, cloud.Region{
			RegionId:  *r.RegionId,
			LocalName: *r.LocalName, // 直接使用 API 返回的中文名称
		})
	}
	return regions, nil
}

// ListZones 查询阿里云指定地域的可用区列表。
func (p *Provider) ListZones(ctx context.Context, ak, sk, region string) ([]cloud.Zone, error) {
	if region == "" {
		return nil, fmt.Errorf("地域不能为空")
	}

	client, err := NewClient(ak, sk, region)
	if err != nil {
		return nil, err
	}

	output, err := client.DescribeZones(ctx, &ecs.DescribeZonesRequest{
		RegionId: tea.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("查询阿里云可用区失败: %w", p.wrapError(err))
	}

	zones := make([]cloud.Zone, 0, len(output.Zones.Zone))
	for _, z := range output.Zones.Zone {
		zones = append(zones, cloud.Zone{
			ZoneId:    *z.ZoneId,
			LocalName: *z.LocalName,
		})
	}
	return zones, nil
}

// wrapError 包装阿里云错误，提供更友好的错误信息。
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	// 尝试解析 Tea 错误
	var teaErr interface{ GetErrorCode() string }
	if errors.As(err, &teaErr) {
		code := teaErr.GetErrorCode()
		switch code {
		case "InvalidAccessKeyId.NotFound":
			return fmt.Errorf("AccessKey ID 不存在")
		case "SignatureDoesNotMatch":
			return fmt.Errorf("签名验证失败，请检查 AccessKey Secret")
		case "InvalidRegionId":
			return fmt.Errorf("地域无效，请使用正确的地域标识如 cn-hangzhou、cn-shanghai")
		case "UnauthorizedOperation":
			return fmt.Errorf("无权限执行此操作，请检查 AccessKey 是否有 ECS 权限")
		case "MissingParameter":
			return fmt.Errorf("缺少必要参数")
		case "Throttling":
			return fmt.Errorf("请求过于频繁，已重试多次仍失败，请稍后重试")
		case "ServiceUnavailable":
			return fmt.Errorf("服务暂时不可用，请稍后重试")
		}
		return fmt.Errorf("[阿里云][%s] %s", code, err.Error())
	}

	return fmt.Errorf("[阿里云] %w", err)
}
