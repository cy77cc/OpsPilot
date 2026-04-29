package logic

import (
	"time"

	opsagentmodel "github.com/cy77cc/OpsPilot/internal/modules/opsagent/model"
)

type SessionRegistry struct {
	inner *opsagentmodel.SessionRegistry
}

func NewSessionRegistry() *SessionRegistry {
	return WrapSessionRegistry(opsagentmodel.NewSessionRegistry())
}

func WrapSessionRegistry(inner *opsagentmodel.SessionRegistry) *SessionRegistry {
	if inner == nil {
		inner = opsagentmodel.NewSessionRegistry()
	}
	return &SessionRegistry{inner: inner}
}

func (r *SessionRegistry) Put(hostID uint64, agentID string, stream AgentServiceConnectServer) *opsagentmodel.Session {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.Put(&opsagentmodel.Session{
		HostID:      hostID,
		AgentID:     agentID,
		Stream:      stream,
		ConnectedAt: time.Now(),
	})
}

func (r *SessionRegistry) DeleteByAgent(agentID string) {
	if r == nil || r.inner == nil {
		return
	}
	r.inner.DeleteByAgent(agentID)
}

func (r *SessionRegistry) GetByAgent(agentID string) (*opsagentmodel.Session, bool) {
	if r == nil || r.inner == nil {
		return nil, false
	}
	return r.inner.GetByAgent(agentID)
}

func (r *SessionRegistry) GetByHostID(hostID uint64) (*opsagentmodel.Session, bool) {
	if r == nil || r.inner == nil {
		return nil, false
	}
	return r.inner.GetByHostID(hostID)
}
