package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// ConfigHandler handles reading/writing the configuration file
type ConfigHandler struct {
	configPath string
}

func NewConfigHandler(configPath string) *ConfigHandler {
	return &ConfigHandler{configPath: configPath}
}

func (h *ConfigHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetConfig(w, r)
	case http.MethodPost:
		h.PostConfig(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// GetConfig returns the raw config file content as JSON { "content": "..." }
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		log.Printf("[config] read error: %v", err)
		http.Error(w, `{"error":"cannot read config file"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"content": string(data)})
}

// PostConfig writes raw content to the config file
// Accepts: { "content": "..." }
func (h *ConfigHandler) PostConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := os.WriteFile(h.configPath, []byte(req.Content), 0644); err != nil {
		log.Printf("[config] write error: %v", err)
		http.Error(w, `{"error":"cannot write config file"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[config] config saved to %s (%d bytes)", h.configPath, len(req.Content))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
