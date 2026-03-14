// Package state 提供 AI 会话状态的 Redis 持久化。
//
// SessionState 管理多轮对话的上下文（历史消息等），
// 与 runtime.ExecutionStore（执行状态）相互独立，分别存储。
package state

import "github.com/redis/go-redis/v9"

// SessionState 管理单个会话的持久化状态，以 Redis 为后端存储。
type SessionState struct {
	rdb    redis.UniversalClient
	prefix string
}

// NewSessionState 创建 SessionState 实例。
// prefix 为 Redis key 前缀，传空字符串时使用默认前缀。
func NewSessionState(rdb redis.UniversalClient, prefix string) *SessionState {
	return &SessionState{rdb: rdb, prefix: prefix}
}
