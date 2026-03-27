// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/websocket"
)

// terminalSession 终端会话结构。
//
// 存储单个 SSH 终端会话的状态和连接信息。
type terminalSession struct {
	// ID 会话唯一标识
	ID string

	// HostID 关联的主机 ID
	HostID uint64

	// UserID 创建会话的用户 ID
	UserID uint64

	// CreatedAt 会话创建时间
	CreatedAt time.Time

	// UpdatedAt 会话最后更新时间
	UpdatedAt time.Time

	// Status 会话状态 (active/closed)
	Status string

	// client SSH 客户端连接
	client *ssh.Client

	// session SSH 会话
	session *ssh.Session

	// stdin 标准输入管道
	stdin io.WriteCloser

	// stdout 标准输出管道
	stdout io.Reader

	// stderr 标准错误管道
	stderr io.Reader

	// mu 并发锁
	mu sync.Mutex
}

// close 关闭终端会话。
//
// 安全关闭 SSH 会话和客户端连接，更新会话状态为 closed。
func (s *terminalSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		_ = s.session.Close()
	}
	if s.client != nil {
		_ = s.client.Close()
	}
	s.Status = "closed"
	s.UpdatedAt = time.Now()
}

// terminalSessionManager 终端会话管理器。
//
// 管理所有活跃的终端会话，支持并发安全访问。
type terminalSessionManager struct {
	// mu 读写锁
	mu sync.RWMutex

	// sessions 会话映射表
	sessions map[string]*terminalSession
}

// hostTerminalSessions 全局终端会话管理器实例。
var hostTerminalSessions = &terminalSessionManager{sessions: map[string]*terminalSession{}}

// set 添加会话到管理器。
func (m *terminalSessionManager) set(s *terminalSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
}

// get 从管理器获取会话。
func (m *terminalSessionManager) get(id string) (*terminalSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// remove 从管理器移除会话。
func (m *terminalSessionManager) remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

// CreateTerminalSession 创建终端会话。
//
// @Summary 创建终端会话
// @Description 创建一个新的 SSH 终端会话，返回会话 ID 和 WebSocket 连接地址
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/terminal/sessions [post]
func (h *Handler) CreateTerminalSession(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.hostService.Get(c.Request.Context(), hostID)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	privateKey, passphrase, err := h.loadNodePrivateKey(c, node)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
		return
	}
	sess, err := cli.NewSession()
	if err != nil {
		_ = cli.Close()
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
		return
	}

	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		_ = cli.Close()
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
		return
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		_ = sess.Close()
		_ = cli.Close()
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
		return
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		_ = sess.Close()
		_ = cli.Close()
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
		return
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sess.RequestPty("xterm-256color", 40, 120, modes); err != nil {
		_ = sess.Close()
		_ = cli.Close()
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
		return
	}
	if err := sess.Shell(); err != nil {
		_ = sess.Close()
		_ = cli.Close()
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
		return
	}

	now := time.Now()
	sessionID := fmt.Sprintf("hts-%d", now.UnixNano())
	ts := &terminalSession{
		ID:        sessionID,
		HostID:    hostID,
		UserID:    getUID(c),
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "active",
		client:    cli,
		session:   sess,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
	}
	hostTerminalSessions.set(ts)

	httpx.OK(c, gin.H{
		"session_id": sessionID,
		"status":     ts.Status,
		"ws_path":    fmt.Sprintf("/api/v1/hosts/%d/terminal/sessions/%s/ws", hostID, sessionID),
		"created_at": ts.CreatedAt,
		"expires_at": ts.CreatedAt.Add(30 * time.Minute),
	})
}

// GetTerminalSession 获取终端会话状态。
//
// @Summary 获取终端会话
// @Description 获取指定终端会话的状态信息
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param session_id path string true "会话 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/{id}/terminal/sessions/{session_id} [get]
func (h *Handler) GetTerminalSession(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		httpx.Fail(c, xcode.ParamError, "session_id is required")
		return
	}
	ts, found := hostTerminalSessions.get(sessionID)
	if !found || ts.HostID != hostID {
		httpx.Fail(c, xcode.NotFound, "terminal session not found")
		return
	}
	httpx.OK(c, gin.H{
		"session_id": ts.ID,
		"host_id":    ts.HostID,
		"user_id":    ts.UserID,
		"status":     ts.Status,
		"created_at": ts.CreatedAt,
		"updated_at": ts.UpdatedAt,
	})
}

// DeleteTerminalSession 删除终端会话。
//
// @Summary 删除终端会话
// @Description 关闭并删除指定的终端会话
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param session_id path string true "会话 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/{id}/terminal/sessions/{session_id} [delete]
func (h *Handler) DeleteTerminalSession(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(c.Param("session_id"))
	ts, found := hostTerminalSessions.get(sessionID)
	if !found || ts.HostID != hostID {
		httpx.Fail(c, xcode.NotFound, "terminal session not found")
		return
	}
	ts.close()
	hostTerminalSessions.remove(sessionID)
	httpx.OK(c, gin.H{"session_id": sessionID, "status": "closed"})
}

// TerminalWebsocket WebSocket 终端连接。
//
// @Summary WebSocket 终端
// @Description 通过 WebSocket 提供交互式终端会话，支持输入、输出和终端大小调整
// @Tags 主机管理
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param session_id path string true "会话 ID"
// @Router /hosts/{id}/terminal/sessions/{session_id}/ws [get]
func (h *Handler) TerminalWebsocket(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(c.Param("session_id"))
	ts, found := hostTerminalSessions.get(sessionID)
	if !found || ts.HostID != hostID {
		httpx.Fail(c, xcode.NotFound, "terminal session not found")
		return
	}
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		writeMu := sync.Mutex{}
		send := func(msgType string, payload any) {
			writeMu.Lock()
			defer writeMu.Unlock()
			_ = websocket.JSON.Send(ws, gin.H{"type": msgType, "payload": payload})
		}

		send("ready", gin.H{"session_id": sessionID})
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := ts.stdout.Read(buf)
				if n > 0 {
					send("output", gin.H{"data": string(buf[:n])})
				}
				if err != nil {
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := ts.stderr.Read(buf)
				if n > 0 {
					send("output", gin.H{"data": string(buf[:n])})
				}
				if err != nil {
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for {
				var raw []byte
				if err := websocket.Message.Receive(ws, &raw); err != nil {
					return
				}
				var ctrl struct {
					Type  string `json:"type"`
					Input string `json:"input"`
					Cols  int    `json:"cols"`
					Rows  int    `json:"rows"`
				}
				if err := json.Unmarshal(raw, &ctrl); err != nil {
					continue
				}
				switch ctrl.Type {
				case "input":
					_, _ = io.WriteString(ts.stdin, ctrl.Input)
				case "resize":
					rows := ctrl.Rows
					cols := ctrl.Cols
					if rows <= 0 {
						rows = 40
					}
					if cols <= 0 {
						cols = 120
					}
					_ = ts.session.WindowChange(rows, cols)
				case "ping":
					send("pong", gin.H{"ts": time.Now().UTC().Format(time.RFC3339Nano)})
				}
			}
		}()

		wg.Wait()
		ts.close()
		hostTerminalSessions.remove(sessionID)
	}).ServeHTTP(c.Writer, c.Request)
}
