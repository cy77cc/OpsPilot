// Package constants 提供应用程序全局常量定义。
//
// 本文件定义 Redis 键前缀、TTL 时长等常量。
package constants

import "time"

// Redis 键前缀和 TTL 常量。
const (
	JwtWhiteListKey = "jwt:blacklist:"  // JWT 黑名单键前缀
	UserIdKey       = "user:id:"         // 用户 ID 键前缀
	UserNameKey     = "user:name:"       // 用户名键前缀
	RdbTTL          = time.Hour * 24 * 2 // Redis 默认 TTL（2 天）
	RdbAddTTL       = time.Minute * 10   // TTL 增量（10 分钟）
	NodeKey         = "node:id:"         // 节点 ID 键前缀
	SSHKey          = "node:ssh:key:id:" // SSH 密钥键前缀
)