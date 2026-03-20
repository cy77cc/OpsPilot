// Package volcengine 提供火山云 ECS 实例查询适配器实现。
//
// 本包封装火山云 Go SDK，实现 cloud.CloudProvider 接口，
// 支持火山云 ECS 实例的查询和导入功能。
package volcengine

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// Client 火山云 ECS 客户端封装。
//
// 封装火山云 SDK 的 ECS 服务客户端，提供简化的实例查询接口。
type Client struct {
	ecs *ecs.ECS
}

// NewClient 创建火山云 ECS 客户端。
//
// 参数:
//   - ak: AccessKey ID
//   - sk: AccessKey Secret
//   - region: 地域标识（如 "cn-beijing"、"cn-shanghai"）
//
// 返回:
//   - 成功返回客户端实例
//   - 失败返回错误（如凭证格式错误）
func NewClient(ak, sk, region string) (*Client, error) {
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("火山云 AccessKey ID 和 Secret 不能为空")
	}
	if region == "" {
		return nil, fmt.Errorf("火山云地域不能为空，如 cn-beijing、cn-shanghai")
	}

	// 构建配置
	// 火山云 ECS 是区域服务，Endpoint 格式: ecs.<region>.volcengineapi.com
	endpoint := fmt.Sprintf("ecs.%s.volcengineapi.com", region)

	config := volcengine.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(ak, sk, "")).
		WithRegion(region).
		WithEndpoint(endpoint)

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("创建火山云会话失败: %w", err)
	}

	return &Client{
		ecs: ecs.New(sess),
	}, nil
}

// DescribeInstances 查询 ECS 实例列表。
//
// 参数:
//   - ctx: 请求上下文
//   - input: 查询参数
//
// 返回:
//   - 成功返回实例列表
//   - 失败返回错误（如凭证无效、网络超时）
func (c *Client) DescribeInstances(ctx context.Context, input *ecs.DescribeInstancesInput) (*ecs.DescribeInstancesOutput, error) {
	return c.ecs.DescribeInstancesWithContext(ctx, input)
}
