package command

// ChatRequest is the command input accepted by the AI chat use case.
type ChatRequest struct{}

// ChatHandler handles AI chat commands.
type ChatHandler interface {
	Handle(*ChatRequest) error
}

type chatCommandHandler struct{}

// NewChatCommandHandler creates the default chat command handler.
func NewChatCommandHandler() ChatHandler {
	return &chatCommandHandler{}
}

func (h *chatCommandHandler) Handle(*ChatRequest) error {
	return nil
}
