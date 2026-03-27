// Package volcengine 提供火山云 ECS 实例查询适配器实现。
package volcengine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/volcengineerr"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Provider 火山云适配器。
//
// 实现 cloud.CloudProvider 接口，提供火山云 ECS 实例查询能力。
type Provider struct{}

// New 创建火山云适配器实例。
func New() *Provider {
	return &Provider{}
}

// Name 返回云厂商标识。
func (p *Provider) Name() string {
	return "volcengine"
}

// DisplayName 返回云厂商显示名称。
func (p *Provider) DisplayName() string {
	return "火山云"
}

// Capabilities 返回火山云能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
	return cloud.ProviderCapabilities{
		DynamicRegions: true,
	}
}

// ValidateCredential 验证火山云凭证是否有效。
//
// 通过调用 DescribeInstances API（限制返回 1 条）验证凭证有效性。
func (p *Provider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	client, err := NewClient(ak, sk, region)
	if err != nil {
		return err
	}

	// 通过查询实例验证凭证，限制返回 1 条
	_, err = client.DescribeInstances(ctx, &ecs.DescribeInstancesInput{
		MaxResults: volcengine.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("火山云凭证验证失败: %w", p.wrapError(err))
	}
	return nil
}

// ListInstances 查询火山云 ECS 实例列表。
func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
	client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
	if err != nil {
		return nil, err
	}

	// 构建查询参数
	input := &ecs.DescribeInstancesInput{}

	// 可用区过滤
	if req.Zone != "" {
		input.SetZoneId(req.Zone)
	}

	// 关键词过滤（支持实例名称）
	if req.Keyword != "" {
		input.SetInstanceName(req.Keyword)
	}

	// 分页参数（默认返回 100 条）
	if req.PageSize > 0 {
		input.SetMaxResults(int32(req.PageSize))
	} else {
		input.SetMaxResults(100)
	}

	// 调用 API
	output, err := client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("查询火山云实例失败: %w", p.wrapError(err))
	}

	// 转换实例数据
	instances := make([]cloud.CloudInstance, 0, len(output.Instances))
	for _, inst := range output.Instances {
		// 转换实例，传入 region（实例数据中无 Region 字段）
		converted := ConvertInstance(inst, req.Region)

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

// regionNames 地域中文名称映射。
var regionNames = map[string]string{
	"cn-beijing":  "华北2（北京）",
	"cn-shanghai": "华东2（上海）",
	"cn-guangzhou": "华南1（广州）",
	"cn-nanjing":  "华东1（南京）",
}

// zoneNames 可用区中文名称映射（根据地域动态生成）。
func getZoneName(zoneId string) string {
	// 从可用区 ID 提取后缀（如 "cn-beijing-a" -> "可用区A"）
	parts := strings.Split(zoneId, "-")
	if len(parts) >= 3 {
		suffix := parts[len(parts)-1]
		switch suffix {
		case "a":
			return "可用区A"
		case "b":
			return "可用区B"
		case "c":
			return "可用区C"
		case "d":
			return "可用区D"
		}
	}
	return zoneId
}

// ListRegions 查询火山云支持的地域列表。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
	// 火山云 DescribeRegions 不需要指定 region，使用默认值
	client, err := NewClient(ak, sk, "cn-beijing")
	if err != nil {
		return nil, err
	}

	output, err := client.DescribeRegions(ctx, &ecs.DescribeRegionsInput{})
	if err != nil {
		return nil, fmt.Errorf("查询火山云地域失败: %w", p.wrapError(err))
	}

	regions := make([]cloud.Region, 0, len(output.Regions))
	for _, r := range output.Regions {
		regionId := volcengine.StringValue(r.RegionId)
		localName := regionNames[regionId]
		if localName == "" {
			localName = regionId
		}
		regions = append(regions, cloud.Region{
			RegionId:  regionId,
			LocalName: localName,
		})
	}
	return regions, nil
}

// ListZones 查询火山云指定地域的可用区列表。
func (p *Provider) ListZones(ctx context.Context, ak, sk, region string) ([]cloud.Zone, error) {
	if region == "" {
		return nil, fmt.Errorf("地域不能为空")
	}

	client, err := NewClient(ak, sk, region)
	if err != nil {
		return nil, err
	}

	output, err := client.DescribeZones(ctx, &ecs.DescribeZonesInput{})
	if err != nil {
		return nil, fmt.Errorf("查询火山云可用区失败: %w", p.wrapError(err))
	}

	zones := make([]cloud.Zone, 0, len(output.Zones))
	for _, z := range output.Zones {
		zoneId := volcengine.StringValue(z.ZoneId)
		zones = append(zones, cloud.Zone{
			ZoneId:    zoneId,
			LocalName: getZoneName(zoneId),
		})
	}
	return zones, nil
}

// wrapError 包装火山云错误，提供更友好的错误信息。
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	// 尝试解析火山云错误
	var be volcengineerr.Error
	if errors.As(err, &be) {
		switch be.Code() {
		case "InvalidAccessKey":
			return fmt.Errorf("AccessKey 无效，请检查 AccessKey ID 和 Secret 是否正确")
		case "SignatureDoesNotMatch":
			return fmt.Errorf("签名验证失败，请检查 AccessKey Secret 是否正确")
		case "InvalidRegion":
			return fmt.Errorf("地域无效，请使用正确的地域标识如 cn-beijing、cn-shanghai")
		case "UnauthorizedOperation":
			return fmt.Errorf("无权限执行此操作，请检查 AccessKey 是否有 ECS 权限")
		case "MissingParameter":
			return fmt.Errorf("缺少必要参数: %s", be.Message())
		}
		return fmt.Errorf("[%s] %s", be.Code(), be.Message())
	}

	return err
}
