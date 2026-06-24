package api

import (
	"net/http"
	"strings"

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

// DeleteServer handles DELETE /api/servers/{id}
func (h *ServerHandler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	// Path is /api/servers/{id}, extract {id}
	id := r.URL.Path[len("/api/servers/"):]
	if id == "" || id == "/" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "server ID is required"})
		return
	}
	// Strip trailing slash if any
	id = strings.TrimSuffix(id, "/")
	// Strip /settings or other subpath suffix
	if idx := strings.Index(id, "/"); idx >= 0 {
		id = id[:idx]
	}
	if h.agentHub.RemoveServer(id) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "id": id})
	} else {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "server not found", "id": id})
	}
}
