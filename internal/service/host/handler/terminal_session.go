package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	sshclient "github.com/cy77cc/k8s-manage/internal/client/ssh"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/websocket"
)

type terminalSession struct {
	ID        string
	HostID    uint64
	UserID    uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    string

	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader

	mu sync.Mutex
}

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

type terminalSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*terminalSession
}

var hostTerminalSessions = &terminalSessionManager{sessions: map[string]*terminalSession{}}

func (m *terminalSessionManager) set(s *terminalSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
}

func (m *terminalSessionManager) get(id string) (*terminalSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

func (m *terminalSessionManager) remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

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
