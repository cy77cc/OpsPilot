// Package events provides SSE event handling for AI service
package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// WriteSSE writes a SSE event to the response
func WriteSSE(c *gin.Context, flusher http.Flusher, event string, payload any) bool {
	raw, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	if _, err = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, raw); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

type sseEvent struct {
	event   string
	payload gin.H
}

// SSEWriter handles SSE event streaming
type SSEWriter struct {
	c       *gin.Context
	flusher http.Flusher
	turnID  string
	ch      chan sseEvent
	done    chan struct{}
	mu      sync.RWMutex
	err     error
}

// NewSSEWriter creates a new SSE writer
func NewSSEWriter(c *gin.Context, flusher http.Flusher, turnID string) *SSEWriter {
	w := &SSEWriter{
		c:       c,
		flusher: flusher,
		turnID:  turnID,
		ch:      make(chan sseEvent, 64),
		done:    make(chan struct{}),
	}
	go w.loop()
	return w
}

func (w *SSEWriter) loop() {
	defer close(w.done)
	for evt := range w.ch {
		payload := evt.payload
		if payload == nil {
			payload = gin.H{}
		}
		payload["turn_id"] = w.turnID
		if !WriteSSE(w.c, w.flusher, evt.event, payload) {
			w.setErr(fmt.Errorf("stream write failed"))
			return
		}
	}
}

// Emit sends an event to the stream
func (w *SSEWriter) Emit(event string, payload gin.H) bool {
	if w.hasErr() {
		return false
	}
	select {
	case <-w.done:
		return false
	case w.ch <- sseEvent{event: event, payload: payload}:
		return true
	}
}

// Close closes the writer and waits for completion
func (w *SSEWriter) Close() error {
	close(w.ch)
	<-w.done
	return w.Err()
}

// Err returns any error that occurred during streaming
func (w *SSEWriter) Err() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.err
}

func (w *SSEWriter) hasErr() bool {
	return w.Err() != nil
}

func (w *SSEWriter) setErr(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err == nil {
		w.err = err
	}
}

// HeartbeatLoop runs a heartbeat loop until stopped
func HeartbeatLoop(stop <-chan struct{}, emit func(event string, payload gin.H) bool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if !emit("heartbeat", gin.H{"status": "alive"}) {
				return
			}
		case <-stop:
			return
		}
	}
}
