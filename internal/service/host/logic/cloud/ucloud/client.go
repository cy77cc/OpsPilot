// Package ucloud 提供 UCLOUD UHost 实例查询适配器实现。
//
// 本包封装 UCLOUD Go SDK，实现 cloud.CloudProvider 接口，
// 支持 UCLOUD UHost 实例的查询和导入功能。
package ucloud

import (
	"context"
	"fmt"

	"github.com/ucloud/ucloud-sdk-go/services/uaccount"
	"github.com/ucloud/ucloud-sdk-go/services/uhost"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Client UCLOUD UHost 客户端封装。
//
// 封装 UCLOUD SDK 的 UHost 服务客户端，提供简化的实例查询接口。
type Client struct {
	uhost   *uhost.UHostClient
	uaccount *uaccount.UAccountClient
	ak      string
}

// NewClient 创建 UCLOUD UHost 客户端。
//
// 参数:
//   - ak: AccessKey ID (PublicKey)
//   - sk: AccessKey Secret (PrivateKey)
//   - region: 地域标识（如 "cn-bj2"、"cn-sh2"）
//
// 返回:
//   - 成功返回客户端实例
//   - 失败返回错误（如凭证格式错误）
func NewClient(ak, sk, region string) (*Client, error) {
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("UCLOUD AccessKey ID 和 Secret 不能为空")
	}
	if region == "" {
		return nil, fmt.Errorf("UCLOUD 地域不能为空，如 cn-bj2、cn-sh2")
	}

	config := ucloud.NewConfig()
	config.Region = region

	credential := auth.NewCredential()
	credential.PublicKey = ak
	credential.PrivateKey = sk

	return &Client{
		uhost:    uhost.NewClient(&config, &credential),
		uaccount: uaccount.NewClient(&config, &credential),
		ak:       ak,
	}, nil
}

// DescribeUHostInstance 查询 UHost 实例列表（带重试）。
//
// 参数:
//   - ctx: 请求上下文
//   - req: 查询参数
//
// 返回:
//   - 成功返回实例列表
//   - 失败返回错误（如凭证无效、网络超时）
func (c *Client) DescribeUHostInstance(ctx context.Context, req *uhost.DescribeUHostInstanceRequest) (*uhost.DescribeUHostInstanceResponse, error) {
	return cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "DescribeUHostInstance", func() (*uhost.DescribeUHostInstanceResponse, error) {
		return c.uhost.DescribeUHostInstance(req)
	})
}

// GetRegion 查询地域和可用区列表（带重试）。
//
// UCLOUD 使用 uaccount 服务的 GetRegion API 查询地域和可用区信息。
//
// 参数:
//   - ctx: 请求上下文
//   - req: 查询参数
//
// 返回:
//   - 成功返回地域列表（包含可用区信息）
//   - 失败返回错误
func (c *Client) GetRegion(ctx context.Context, req *uaccount.GetRegionRequest) (*uaccount.GetRegionResponse, error) {
	return cloud.DoWithRetry(ctx, "ucloud", cloud.DefaultRetryConfig, "GetRegion", func() (*uaccount.GetRegionResponse, error) {
		return c.uaccount.GetRegion(req)
	})
}
