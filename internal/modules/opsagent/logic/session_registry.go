package logic

import (
	"errors"
	"time"

	opsagentmodel "github.com/cy77cc/OpsPilot/internal/modules/opsagent/model"
	pb "github.com/cy77cc/OpsPilot/proto"
)

type SessionHandle = opsagentmodel.Session

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

func (r *SessionRegistry) Put(hostID uint64, agentID string, stream pb.AgentService_ConnectServer) *opsagentmodel.Session {
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

func (r *SessionRegistry) DeleteByAgentStream(agentID string, stream pb.AgentService_ConnectServer) {
	if r == nil || r.inner == nil {
		return
	}
	r.inner.DeleteByAgentStream(agentID, stream)
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

func (r *SessionRegistry) AgentIDByStream(stream pb.AgentService_ConnectServer) (string, error) {
	if r == nil || r.inner == nil {
		return "", errors.New("opsagent session registry is unavailable")
	}
	agentID, ok := r.inner.AgentIDByStream(stream)
	if !ok {
		return "", errors.New("opsagent session not found for stream")
	}
	return agentID, nil
}
