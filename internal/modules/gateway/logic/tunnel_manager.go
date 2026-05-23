package logic

import (
	"fmt"
	"sync"

	pb "github.com/cy77cc/OpsPilot/proto"
)

// TunnelSession 是内存中的隧道会话。
type TunnelSession struct {
	TunnelID  string
	GatewayID uint64
	HostID    uint64
	AgentID   string
	Stream    pb.AgentService_ConnectServer
}

// TunnelManager 管理活跃隧道会话。
type TunnelManager struct {
	sessions map[string]*TunnelSession // tunnelID -> session
	mu       sync.RWMutex
}

// NewTunnelManager 创建隧道管理器。
func NewTunnelManager() *TunnelManager {
	return &TunnelManager{
		sessions: make(map[string]*TunnelSession),
	}
}

// Open 注册新的隧道会话。
func (tm *TunnelManager) Open(tunnelID, agentID string, gatewayID, hostID uint64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.sessions[tunnelID] = &TunnelSession{
		TunnelID:  tunnelID,
		GatewayID: gatewayID,
		HostID:    hostID,
		AgentID:   agentID,
	}
}

// Close 关闭并移除隧道会话。
func (tm *TunnelManager) Close(tunnelID, reason string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.sessions, tunnelID)
}

// Get 获取隧道会话。
func (tm *TunnelManager) Get(tunnelID string) (*TunnelSession, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	s, ok := tm.sessions[tunnelID]
	return s, ok
}

// SetStream 设置隧道的 gateway 流。
func (tm *TunnelManager) SetStream(tunnelID string, stream pb.AgentService_ConnectServer) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	s, ok := tm.sessions[tunnelID]
	if !ok {
		return fmt.Errorf("tunnel %s not found", tunnelID)
	}
	s.Stream = stream
	return nil
}

// GetStream 获取隧道的 gateway 流。
func (tm *TunnelManager) GetStream(tunnelID string) (pb.AgentService_ConnectServer, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	s, ok := tm.sessions[tunnelID]
	if !ok || s.Stream == nil {
		return nil, false
	}
	return s.Stream, true
}

// Count 返回活跃隧道数。
func (tm *TunnelManager) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.sessions)
}
