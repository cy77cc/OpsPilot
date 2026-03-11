// Package utils 提供通用工具函数。
//
// 本文件实现通用辅助函数，包括类型转换、时间戳生成和 Redis TTL 扩展。
package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/redis/go-redis/v9"
)

// SlicesToString 将泛型切片转换为字符串。
//
// 使用指定的分隔符连接元素，元素之间额外添加空格。
func SlicesToString[T any](s []T, sep string) string {
	if len(s) == 0 {
		return ""
	}
	var b strings.Builder
	for i, v := range s {
		if i > 0 {
			b.WriteString(sep)
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%v", v)
	}
	return b.String()
}

// MapToString 将泛型映射转换为字符串。
//
// 格式为 "key:value sep key:value ..."。
func MapToString[K comparable, V any](m map[K]V, sep string) string {
	if len(m) == 0 {
		return ""
	}

	var b strings.Builder
	for k, v := range m {
		fmt.Fprintf(&b, "%v:%v%s", k, v, sep)
	}
	return b.String()
}

// GetTimestamp 返回当前 Unix 时间戳（秒）。
func GetTimestamp() int64 {
	return time.Now().Unix()
}

// ExtendTTL 扩展 Redis 键的过期时间。
//
// 在当前 TTL 基础上增加 constants.RdbAddTTL 时长。
func ExtendTTL(ctx context.Context, rdb redis.UniversalClient, key string) error {
	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	if ttl < 0 {
		// key 不存在或无过期时间，可以直接设置 add 作为 TTL
		ttl = 0
	}
	return rdb.Expire(ctx, key, ttl+constants.RdbAddTTL).Err()
}
