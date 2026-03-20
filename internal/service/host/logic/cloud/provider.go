// Package cloud 提供云厂商主机导入的统一接口和适配器管理。
//
// 本包采用接口适配器模式，支持多云厂商的实例查询和导入：
//   - 定义统一的 CloudProvider 接口
//   - 通过 Registry 管理所有云厂商适配器
//   - 每个云厂商实现独立的适配器（如 volcengine、alicloud、tencent）
//
// 使用方式:
//
//	cloud.Register(volcengine.New())
//	provider, err := cloud.GetProvider("volcengine")
//	instances, err := provider.ListInstances(ctx, req)
package cloud

import "context"

// CloudProvider 定义云厂商适配器接口。
//
// 每个云厂商需要实现此接口，提供实例查询和凭证验证能力。
// 实现者应处理各自的 SDK 初始化、API 调用和数据转换。
type CloudProvider interface {
	// Name 返回云厂商标识。
	//
	// 返回值:
	//   - 云厂商唯一标识，如 "volcengine"、"alicloud"、"tencent"
	//   - 用于注册表查找和数据库存储
	Name() string

	// DisplayName 返回云厂商显示名称。
	//
	// 返回值:
	//   - 用户友好的显示名称，如 "火山云"、"阿里云"、"腾讯云"
	//   - 用于前端下拉选项展示
	DisplayName() string

	// ValidateCredential 验证云账号凭证是否有效。
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
	ValidateCredential(ctx context.Context, ak, sk, region string) error

	// ListInstances 查询云厂商实例列表。
	//
	// 参数:
	//   - ctx: 请求上下文
	//   - req: 查询请求参数
	//
	// 返回:
	//   - 实例列表（统一格式的 CloudInstance）
	//   - 查询失败返回具体错误
	ListInstances(ctx context.Context, req ListInstancesRequest) ([]CloudInstance, error)

	// ListRegions 查询云厂商支持的地域列表。
	//
	// 参数:
	//   - ctx: 请求上下文
	//   - ak: AccessKey ID
	//   - sk: AccessKey Secret
	//
	// 返回:
	//   - 地域列表
	//   - 查询失败返回具体错误
	ListRegions(ctx context.Context, ak, sk string) ([]Region, error)

	// ListZones 查询云厂商指定地域的可用区列表。
	//
	// 参数:
	//   - ctx: 请求上下文
	//   - ak: AccessKey ID
	//   - sk: AccessKey Secret
	//   - region: 地域标识
	//
	// 返回:
	//   - 可用区列表
	//   - 查询失败返回具体错误
	ListZones(ctx context.Context, ak, sk, region string) ([]Zone, error)
}
