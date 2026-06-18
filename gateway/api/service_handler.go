package api

import (
	"net/http"

	"github.com/glrs/observer/gateway/monitor"
	"github.com/glrs/observer/gateway/plugin"
)

// ServiceHandler handles service-related API endpoints.
type ServiceHandler struct {
	registry     *plugin.Registry
	healthChecker *monitor.HealthChecker
}

// NewServiceHandler creates a ServiceHandler.
func NewServiceHandler(registry *plugin.Registry, healthChecker *monitor.HealthChecker) *ServiceHandler {
	return &ServiceHandler{
		registry:      registry,
		healthChecker: healthChecker,
	}
}

// ListServices handles GET /api/services
func (h *ServiceHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	statuses := h.healthChecker.GetStatus()
	if statuses == nil {
		// Fall back to registry listing if no health data yet
		services := h.registry.List()
		type svcResponse struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Icon       string `json:"icon"`
			URL        string `json:"url"`
			Category   string `json:"category"`
			OpenMethod string `json:"open_method"`
			PanelURL   string `json:"panel_url,omitempty"`
			Status     string `json:"status"`
		}
		resp := make([]svcResponse, len(services))
		for i, s := range services {
			resp[i] = svcResponse{
				ID:         s.ID,
				Name:       s.Name,
				Icon:       s.Icon,
				URL:        s.URL,
				Category:   s.Category,
				OpenMethod: s.OpenMethod,
				PanelURL:   s.PanelURL,
				Status:     "unknown",
			}
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	type serviceStatusResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Icon       string `json:"icon"`
		URL        string `json:"url"`
		Category   string `json:"category"`
		OpenMethod string `json:"open_method"`
		PanelURL   string `json:"panel_url,omitempty"`
		Status     string `json:"status"`
		LastCheck  int64  `json:"last_check,omitempty"`
		Error      string `json:"error,omitempty"`
	}
	resp := make([]serviceStatusResponse, len(statuses))
	for i, s := range statuses {
		resp[i] = serviceStatusResponse{
			ID:         s.ID,
			Name:       s.Name,
			Icon:       s.Icon,
			URL:        s.URL,
			Category:   s.Category,
			OpenMethod: s.OpenMethod,
			PanelURL:   s.PanelURL,
			Status:     s.Status,
			LastCheck:  s.LastCheck,
			Error:      s.Error,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}
