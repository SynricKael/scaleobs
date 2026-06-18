package api

import (
	"net/http"

	"github.com/glrs/observer/gateway/monitor"
)

// ServerHandler handles server-related API endpoints.
type ServerHandler struct {
	agentHub *monitor.AgentHub
}

// NewServerHandler creates a ServerHandler.
func NewServerHandler(agentHub *monitor.AgentHub) *ServerHandler {
	return &ServerHandler{agentHub: agentHub}
}

// ListServers handles GET /api/servers
func (h *ServerHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	servers := h.agentHub.GetAllStatuses()
	if servers == nil {
		servers = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, servers)
}
