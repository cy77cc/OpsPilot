// Package volcengine 提供火山云 ECS 实例查询适配器实现。
package volcengine

import (
	"context"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"

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

// ValidateCredential 验证火山云凭证是否有效。
//
// 通过调用 DescribeInstances API（限制返回 1 条）验证凭证有效性。
//
// 参数:
//   - ctx: 请求上下文
//   - ak: AccessKey ID
//   - sk: AccessKey Secret
//   - region: 地域标识
//
// 返回:
//   - 验证成功返回 nil
//   - 验证失败返回具体错误
func (p *Provider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	client, err := NewClient(ak, sk, region)
	if err != nil {
		return err
	}

	// 通过查询实例验证凭证，限制返回 1 条
	_, err = client.DescribeInstances(ctx, &ecs.DescribeInstancesInput{
		MaxResults: volcengine.Int32(1),
	})
	return err
}

// ListInstances 查询火山云 ECS 实例列表。
//
// 参数:
//   - ctx: 请求上下文
//   - req: 查询请求参数
//
// 返回:
//   - 成功返回实例列表
//   - 失败返回具体错误
func (p *Provider) ListInstances(ctx context.Context, req cloud.ListInstancesRequest) ([]cloud.CloudInstance, error) {
	client, err := NewClient(req.AccessKeyID, req.AccessKeySecret, req.Region)
	if err != nil {
		return nil, err
	}

	// 构建查询参数
	input := &ecs.DescribeInstancesInput{}

	// 关键词过滤（支持实例名称）
	if req.Keyword != "" {
		input.SetInstanceName(req.Keyword)
	}

	// 分页参数
	if req.PageSize > 0 {
		input.SetMaxResults(int32(req.PageSize))
	}

	// 调用 API
	output, err := client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, err
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
