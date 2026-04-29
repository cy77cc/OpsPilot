package model

import (
	"sync"
	"time"
)

// Session 表示一个在线 OpsAgent 连接会话。
type Session struct {
	HostID      uint64
	AgentID     string
	Stream      any
	ConnectedAt time.Time
}

type SessionRegistry struct {
	mu       sync.RWMutex
	byAgent  map[string]*Session
	byHostID map[uint64]*Session
}

func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{
		byAgent:  map[string]*Session{},
		byHostID: map[uint64]*Session{},
	}
}

func (r *SessionRegistry) Put(session *Session) *Session {
	if r == nil || session == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if prev, ok := r.byAgent[session.AgentID]; ok {
		delete(r.byHostID, prev.HostID)
	}
	if prev, ok := r.byHostID[session.HostID]; ok {
		delete(r.byAgent, prev.AgentID)
	}

	r.byAgent[session.AgentID] = session
	r.byHostID[session.HostID] = session
	return session
}

func (r *SessionRegistry) DeleteByAgent(agentID string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.byAgent[agentID]
	if !ok {
		return
	}
	delete(r.byAgent, agentID)
	delete(r.byHostID, session.HostID)
}

func (r *SessionRegistry) GetByAgent(agentID string) (*Session, bool) {
	if r == nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.byAgent[agentID]
	return session, ok
}

func (r *SessionRegistry) GetByHostID(hostID uint64) (*Session, bool) {
	if r == nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.byHostID[hostID]
	return session, ok
}
