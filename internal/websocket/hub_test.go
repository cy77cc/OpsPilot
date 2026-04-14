package websocket

import (
	"encoding/json"
	"testing"
)

func TestPushUpdate_UsesNumericStringID(t *testing.T) {
	h := &Hub{
		broadcast: make(chan *BroadcastMessage, 1),
	}

	h.PushUpdate(42, 65, nil, nil, nil)

	var pushed *BroadcastMessage
	select {
	case pushed = <-h.broadcast:
	default:
		t.Fatal("expected broadcast message, got none")
	}

	var msg WSMessage
	if err := json.Unmarshal(pushed.Message, &msg); err != nil {
		t.Fatalf("failed to unmarshal websocket message: %v", err)
	}

	if msg.Type != "update" {
		t.Fatalf("expected type %q, got %q", "update", msg.Type)
	}

	if msg.ID != "65" {
		t.Fatalf("expected numeric string ID %q, got %q", "65", msg.ID)
	}
}
