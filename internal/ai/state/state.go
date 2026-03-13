package state

import "github.com/redis/go-redis/v9"

type SessionState struct {
	rdb    redis.UniversalClient
	prefix string
}

func NewSessionState(rdb redis.UniversalClient, prefix string) *SessionState {
	return &SessionState{rdb: rdb, prefix: prefix}
}
