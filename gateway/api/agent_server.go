package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/glrs/observer/gateway/model"
	"gopkg.in/yaml.v3"
)

// AgentServerHandler manages coding agent server status checking.
type AgentServerHandler struct {
	mu         sync.RWMutex
	servers    []model.AgentServerStatus
	configPath string
}

// NewAgentServerHandler creates a handler and starts periodic health pings.
func NewAgentServerHandler(cfgServers []model.AgentServerConfig) *AgentServerHandler {
	h := &AgentServerHandler{}
	for _, s := range cfgServers {
		h.servers = append(h.servers, model.AgentServerStatus{
			AgentServerConfig: s,
		})
	}
	go h.pingLoop()
	return h
}

// SetConfigPath sets the config file path for saving new entries.
func (h *AgentServerHandler) SetConfigPath(path string) {
	h.configPath = path
}

func (h *AgentServerHandler) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	h.pingAll()
	for range ticker.C {
		h.pingAll()
	}
}

func (h *AgentServerHandler) pingAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.servers {
		h.pingOne(i)
	}
}

func (h *AgentServerHandler) pingOne(i int) {
	s := h.servers[i]
	status := "ok"
	errMsg := ""

	reqURL := s.URL
	if s.User != "" && s.Password != "" {
		u, err := url.Parse(s.URL)
		if err == nil {
			u.User = url.UserPassword(s.User, s.Password)
			reqURL = u.String()
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		status = "error"
		errMsg = err.Error()
	} else {
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			status = "ok"
		} else {
			status = "error"
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	h.servers[i].Online = (status == "ok")
	h.servers[i].LastPing = time.Now().Unix()
	if errMsg != "" {
		h.servers[i].Error = errMsg
	} else {
		h.servers[i].Error = ""
	}
}

// handleAgentServers routes GET and POST requests.
func (h *AgentServerHandler) ListAgentServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getAgentServers(w, r)
	case http.MethodPost:
		h.addAgentServer(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *AgentServerHandler) getAgentServers(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]model.AgentServerStatus, len(h.servers))
	for i, s := range h.servers {
		s.Password = "" // never expose password to frontend
		result[i] = s
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AgentServerHandler) addAgentServer(w http.ResponseWriter, r *http.Request) {
	var req model.AgentServerConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.URL == "" {
		http.Error(w, `{"error":"name and url are required"}`, http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	idx := len(h.servers)
	h.servers = append(h.servers, model.AgentServerStatus{
		AgentServerConfig: req,
		LastPing:          time.Now().Unix(),
	})
	h.mu.Unlock()

	// Quick ping in background
	go func() {
		h.mu.Lock()
		h.pingOne(idx)
		h.mu.Unlock()
	}()

	// Save to config
	if h.configPath != "" {
		if err := h.saveToConfig(req); err != nil {
			log.Printf("[agent-server] save to config failed: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AgentServerHandler) saveToConfig(entry model.AgentServerConfig) error {
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if len(doc.Content) == 0 {
		return fmt.Errorf("empty config")
	}
	mapping := doc.Content[0]

	var agentSrvNode *yaml.Node
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == "agent_servers" {
			agentSrvNode = mapping.Content[i+1]
			break
		}
	}
	if agentSrvNode == nil {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "agent_servers"}
		valNode := &yaml.Node{Kind: yaml.SequenceNode}
		mapping.Content = append(mapping.Content, keyNode, valNode)
		agentSrvNode = valNode
	}

	entryNode := &yaml.Node{Kind: yaml.MappingNode}
	entryNode.Content = append(entryNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "name"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: entry.Name},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "url"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: entry.URL},
	)
	if entry.User != "" {
		entryNode.Content = append(entryNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "user"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: entry.User},
		)
	}
	if entry.Password != "" {
		entryNode.Content = append(entryNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "password"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: entry.Password},
		)
	}

	agentSrvNode.Content = append(agentSrvNode.Content, entryNode)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(h.configPath, out, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	log.Printf("[agent-server] saved %q to %s", entry.Name, h.configPath)
	return nil
}
