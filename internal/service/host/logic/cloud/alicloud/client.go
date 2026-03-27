// Package alicloud 提供阿里云 ECS 实例查询适配器实现。
package alicloud

import (
	"context"
	"fmt"

	ecs "github.com/alibabacloud-go/ecs-20140526/client"
	rpc "github.com/alibabacloud-go/tea-rpc/client"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Client 阿里云 ECS 客户端封装。
type Client struct {
	ecs *ecs.Client
	ak  string // 保存用于日志脱敏检查
}

// NewClient 创建阿里云 ECS 客户端。
//
// 参数:
//   - ak: AccessKey ID
//   - sk: AccessKey Secret
//   - region: 地域标识（如 "cn-hangzhou"、"cn-shanghai"）
//
// 注意：查询地域列表（DescribeRegions）时，需使用默认地域如 cn-hangzhou。
func NewClient(ak, sk, region string) (*Client, error) {
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("阿里云 AccessKey ID 和 Secret 不能为空")
	}
	if region == "" {
		return nil, fmt.Errorf("阿里云地域不能为空，如 cn-hangzhou、cn-shanghai")
	}

	endpoint := fmt.Sprintf("ecs.%s.aliyuncs.com", region)
	config := &rpc.Config{
		AccessKeyId:     tea.String(ak),
		AccessKeySecret: tea.String(sk),
		RegionId:        tea.String(region),
		Endpoint:        tea.String(endpoint),
	}
	client, err := ecs.NewClient(config)
	if err != nil {
		// 注意：错误信息中不包含 sk，避免泄露
		return nil, fmt.Errorf("创建阿里云客户端失败: %w", err)
	}
	return &Client{ecs: client, ak: ak}, nil
}

// DescribeInstances 查询 ECS 实例列表（带重试）。
func (c *Client) DescribeInstances(ctx context.Context, req *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
	return cloud.DoWithRetry(ctx, "alicloud", cloud.DefaultRetryConfig, "DescribeInstances", func() (*ecs.DescribeInstancesResponse, error) {
		return c.ecs.DescribeInstances(req)
	})
}

// DescribeRegions 查询地域列表（带重试）。
//
// 返回结果包含 LocalName 字段（中文名称），无需硬编码映射。
func (c *Client) DescribeRegions(ctx context.Context, req *ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error) {
	return cloud.DoWithRetry(ctx, "alicloud", cloud.DefaultRetryConfig, "DescribeRegions", func() (*ecs.DescribeRegionsResponse, error) {
		return c.ecs.DescribeRegions(req)
	})
}

// DescribeZones 查询可用区列表（带重试）。
func (c *Client) DescribeZones(ctx context.Context, req *ecs.DescribeZonesRequest) (*ecs.DescribeZonesResponse, error) {
	return cloud.DoWithRetry(ctx, "alicloud", cloud.DefaultRetryConfig, "DescribeZones", func() (*ecs.DescribeZonesResponse, error) {
		return c.ecs.DescribeZones(req)
	})
}
