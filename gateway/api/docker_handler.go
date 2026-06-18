package api

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

// DockerHandler handles container management actions.
type DockerHandler struct{}

// NewDockerHandler creates a DockerHandler.
func NewDockerHandler() *DockerHandler {
	return &DockerHandler{}
}

// HandleContainerAction handles POST /api/docker/{serverId}/containers/{containerId}/{action}
func (h *DockerHandler) HandleContainerAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Parse path: /api/docker/{serverId}/containers/{containerId}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/docker/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[1] != "containers" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid path: expected /api/docker/{serverId}/containers/{containerId}/{action}",
		})
		return
	}

	containerID := parts[2]
	action := parts[3]

	if action != "start" && action != "stop" && action != "restart" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action: use start, stop, or restart"})
		return
	}

	log.Printf("[docker] executing: docker %s %s", action, containerID[:12])

	out, err := exec.Command("docker", action, containerID).CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(out))
		if errMsg == "" {
			errMsg = err.Error()
		}
		log.Printf("[docker] %s %s failed: %s", action, containerID[:12], errMsg)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	log.Printf("[docker] %s %s OK: %s", action, containerID[:12], strings.TrimSpace(string(out)))
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"action":    action,
		"container": containerID,
	})
}

func init() {
	// Check docker availability at startup
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Println("[docker] WARNING: docker CLI not found in PATH — container actions disabled")
	}
}
