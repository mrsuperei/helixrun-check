package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"helixrun/internal/agents"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
)

// ChatServer handles /chat SSE requests.
type ChatServer struct {
	registry *agents.Registry
}

// NewChatServer creates a ChatServer.
func NewChatServer(reg *agents.Registry) *ChatServer {
	return &ChatServer{registry: reg}
}

// ChatRequest is the JSON payload accepted by /chat.
type ChatRequest struct {
	AgentID   string `json:"agent_id"`
	Message   string `json:"message"`
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// sseEnvelope is what we encode into each SSE data: line.
type sseEnvelope struct {
	Type  string       `json:"type"`  // event.Response.Object when available
	Event *event.Event `json:"event"` // raw tRPC-Agent-Go event
}

// ChatHandler is an HTTP handler for POST /chat that streams SSE.
func (s *ChatServer) ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.AgentID == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		req.UserID = "anonymous"
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := r.Context()

	agt, err := s.registry.BuildAgent(ctx, req.AgentID)
	if err != nil {
		log.Printf("build agent %q failed: %v", req.AgentID, err)
		writeSSEError(w, flusher, err)
		return
	}

	sessionService := inmemory.NewSessionService()
	appRunner := runner.NewRunner(
		"helixrun-starter",
		agt,
		runner.WithSessionService(sessionService),
	)

	msg := model.NewUserMessage(req.Message)
	events, err := appRunner.Run(ctx, req.UserID, req.SessionID, msg)
	if err != nil {
		log.Printf("runner.Run error: %v", err)
		writeSSEError(w, flusher, err)
		return
	}

	enc := json.NewEncoder(w)

	for ev := range events {
		typ := ""
		if ev.Response != nil {
			typ = ev.Response.Object
		}
		env := sseEnvelope{
			Type:  typ,
			Event: ev,
		}

		if _, err := fmt.Fprint(w, "data: "); err != nil {
			return
		}
		if err := enc.Encode(env); err != nil {
			log.Printf("encode event failed: %v", err)
			return
		}
		// Encoder writes newline; SSE requires an extra blank line.
		if _, err := fmt.Fprint(w, "\n"); err != nil {
			return
		}
		flusher.Flush()
	}
}

func writeSSEError(w http.ResponseWriter, flusher http.Flusher, err error) {
	env := map[string]any{
		"type": "error",
		"error": map[string]any{
			"message": err.Error(),
		},
	}

	data, marshalErr := json.Marshal(env)
	if marshalErr != nil {
		log.Printf("marshal error env failed: %v", marshalErr)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", string(data))
	flusher.Flush()
}
