// Package ulighthost 提供 UCloud 轻量应用云主机查询适配器实现。
package ulighthost

import (
	"context"
	"fmt"

	"github.com/ucloud/ucloud-sdk-go/services/ulighthost"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"

	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
)

// Client ULightHost 客户端封装。
type Client struct {
	client *ulighthost.ULightHostClient
}

// NewClient 创建 ULightHost 客户端。
func NewClient(ak, sk, region string) (*Client, error) {
	return NewClientWithConfig(ak, sk, region, false)
}

// NewClientWithConfig 创建 ULightHost 客户端（支持额外配置）。
func NewClientWithConfig(ak, sk, region string, isIntl bool) (*Client, error) {
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("UCLOUD AccessKey ID 和 Secret 不能为空")
	}
	if region == "" {
		return nil, fmt.Errorf("UCLOUD 地域不能为空，如 cn-bj2、hk")
	}

	config := ucloud.NewConfig()
	config.Region = region

	// 国际版使用不同的 API endpoint
	if isIntl {
		config.BaseUrl = "https://api.intl.ucloud.cn"
	}

	credential := auth.NewCredential()
	credential.PublicKey = ak
	credential.PrivateKey = sk

	return &Client{
		client: ulighthost.NewClient(&config, &credential),
	}, nil
}

// DescribeULHostInstance 查询轻量应用云主机实例列表（带重试）。
func (c *Client) DescribeULHostInstance(ctx context.Context, req *ulighthost.DescribeULHostInstanceRequest) (*ulighthost.DescribeULHostInstanceResponse, error) {
	return cloud.DoWithRetry(ctx, "ucloud-ulighthost", cloud.DefaultRetryConfig, "DescribeULHostInstance", func() (*ulighthost.DescribeULHostInstanceResponse, error) {
		return c.client.DescribeULHostInstance(req)
	})
}
