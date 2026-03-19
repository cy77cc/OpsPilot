// Package volcengine 提供火山云 ECS 实例查询适配器实现。
//
// 本包封装火山云 Go SDK，实现 cloud.CloudProvider 接口，
// 支持火山云 ECS 实例的查询和导入功能。
package volcengine

import (
	"context"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// Client 火山云 ECS 客户端封装。
//
// 封装火山云 SDK 的 ECS 服务客户端，提供简化的实例查询接口。
type Client struct {
	// ecs ECS 服务客户端实例。
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
	config := volcengine.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(ak, sk, "")).
		WithRegion(region)

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
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
