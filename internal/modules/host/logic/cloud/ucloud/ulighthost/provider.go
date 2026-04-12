// Package ulighthost 提供 UCloud 轻量应用云主机查询适配器实现。
package ulighthost

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ucloud/ucloud-sdk-go/services/uaccount"
	"github.com/ucloud/ucloud-sdk-go/services/ulighthost"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"

	"github.com/cy77cc/OpsPilot/internal/modules/host/logic/cloud"
)

// UCloudExtraConfig UCLOUD 额外配置。
type UCloudExtraConfig struct {
	ProjectId string `json:"project_id"` // 项目 ID（子账户必须填写）
	IsIntl    bool   `json:"is_intl"`    // 是否为国际版账户
}

// parseExtraConfig 解析额外配置。
func parseExtraConfig(extraJSON string) *UCloudExtraConfig {
	if extraJSON == "" {
		return nil
	}
	var cfg UCloudExtraConfig
	if err := json.Unmarshal([]byte(extraJSON), &cfg); err != nil {
		return nil
	}
	return &cfg
}

// Provider ULightHost 适配器。
type Provider struct{}

// New 创建 ULightHost 适配器实例。
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

// ProductType 返回产品类型标识。
func (p *Provider) ProductType() string {
	return "ulighthost"
}

// ProductTypeName 返回产品类型显示名称。
func (p *Provider) ProductTypeName() string {
	return "轻量应用云主机"
}

// Capabilities 返回能力标识。
func (p *Provider) Capabilities() cloud.ProviderCapabilities {
	return cloud.ProviderCapabilities{
		DynamicRegions: true,
	}
}

// ValidateCredential 验证凭证是否有效。
func (p *Provider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	client, err := NewClient(ak, sk, region)
	if err != nil {
		return err
	}

	limit := 1
	req := &ulighthost.DescribeULHostInstanceRequest{}
	req.Region = &region
	req.Limit = &limit

	_, err = client.DescribeULHostInstance(ctx, req)
	if err != nil {
		return fmt.Errorf("ULightHost 凭证验证失败: %w", p.wrapError(err))
	}
	return nil
}

// ListInstances 查询实例列表。
func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
	// 解析额外配置
	extraConfig := parseExtraConfig(req.Extra)
	isIntl := false
	if extraConfig != nil {
		isIntl = extraConfig.IsIntl
	}

	client, err := NewClientWithConfig(req.AccessKeyID, req.AccessKeySecret, req.Region, isIntl)
	if err != nil {
		return nil, err
	}

	input := &ulighthost.DescribeULHostInstanceRequest{}
	input.Region = &req.Region

	// 可用区过滤
	if req.Zone != "" && req.Zone != "undefined" && req.Zone != "all" {
		input.Zone = &req.Zone
	}

	// 分页参数
	limit := 100
	if req.PageSize > 0 {
		limit = req.PageSize
	}
	input.Limit = &limit

	if req.PageNumber > 1 {
		offset := (req.PageNumber - 1) * req.PageSize
		input.Offset = &offset
	}

	output, err := client.DescribeULHostInstance(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("查询 ULightHost 实例失败: %w", p.wrapError(err))
	}

	instances := make([]cloud.CloudInstance, 0, len(output.ULHostInstanceSets))
	for i := range output.ULHostInstanceSets {
		converted := ConvertInstance(output.ULHostInstanceSets[i], req.Region)

		// 关键词过滤
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

// ListRegions 查询地域列表（复用 UHost 的地域查询）。
func (p *Provider) ListRegions(ctx context.Context, ak, sk string) ([]cloud.Region, error) {
	// 解析额外配置中的 IsIntl
	config := ucloud.NewConfig()
	config.Region = "cn-bj2"

	credential := auth.NewCredential()
	credential.PublicKey = ak
	credential.PrivateKey = sk

	uaccountClient := uaccount.NewClient(&config, &credential)

	req := &uaccount.GetRegionRequest{}
	output, err := cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "GetRegion", func() (*uaccount.GetRegionResponse, error) {
		return uaccountClient.GetRegion(req)
	})
	if err != nil {
		return nil, fmt.Errorf("查询地域失败: %w", p.wrapError(err))
	}

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

// ListZones 查询可用区列表（复用 UHost 的可用区查询）。
func (p *Provider) ListZones(ctx context.Context, ak, sk, region string) ([]cloud.Zone, error) {
	if region == "" {
		return nil, fmt.Errorf("地域不能为空")
	}

	config := ucloud.NewConfig()
	config.Region = region

	credential := auth.NewCredential()
	credential.PublicKey = ak
	credential.PrivateKey = sk

	uaccountClient := uaccount.NewClient(&config, &credential)

	req := &uaccount.GetRegionRequest{}
	output, err := cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "GetRegion", func() (*uaccount.GetRegionResponse, error) {
		return uaccountClient.GetRegion(req)
	})
	if err != nil {
		return nil, fmt.Errorf("查询可用区失败: %w", p.wrapError(err))
	}

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

// wrapError 包装错误信息。
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	if strings.Contains(errStr, "160") {
		return fmt.Errorf("地域无效，请使用正确的地域标识如 cn-bj2、hk")
	}
	if strings.Contains(errStr, "170") {
		return fmt.Errorf("认证失败，请检查 AccessKey ID 和 Secret")
	}
	if strings.Contains(errStr, "171") {
		return fmt.Errorf("AccessKey 无效")
	}

	return fmt.Errorf("[ULightHost] %w", err)
}
