package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/glrs/observer/gateway/config"
	"github.com/glrs/observer/gateway/monitor"
	"gopkg.in/yaml.v3"
)

// ServerSettingsHandler handles PATCH /api/servers/{id}/settings
// to update per-server SSH config and group assignment.
type ServerSettingsHandler struct {
	configPath string
	hub        *monitor.AgentHub
}

// NewServerSettingsHandler creates a handler.
func NewServerSettingsHandler(configPath string, hub *monitor.AgentHub) *ServerSettingsHandler {
	return &ServerSettingsHandler{configPath: configPath, hub: hub}
}

// HandleSettingsUpdate handles PATCH /api/servers/{id}/settings and DELETE /api/servers/{id}
func (h *ServerSettingsHandler) HandleSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/servers/{id}[/settings]
	path := strings.TrimPrefix(r.URL.Path, "/api/servers/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path: expected /api/servers/{id}"})
		return
	}
	serverID := parts[0]

	// DELETE /api/servers/{id}
	if r.Method == http.MethodDelete {
		if h.hub.RemoveServer(serverID) {
			writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "id": serverID})
		} else {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "server not found", "id": serverID})
		}
		return
	}

	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if len(parts) < 2 || parts[1] != "settings" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path: expected /api/servers/{id}/settings"})
		return
	}

	var req ServerSettingsUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if err := h.updateServerConfig(serverID, req); err != nil {
		log.Printf("[settings] update %s failed: %v", serverID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("[settings] updated server %q: group=%q ssh_host=%q docker_mode=%q", serverID, req.Group, req.SSHHost, req.DockerMode)
	
	// Reload config and update hub's in-memory overrides
	if err := h.reloadOverrides(); err != nil {
		log.Printf("[settings] warning: failed to reload overrides: %v", err)
	}
	
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// reloadOverrides reloads the config and pushes overrides to the hub.
func (h *ServerSettingsHandler) reloadOverrides() error {
	cfg, err := config.Load(h.configPath)
	if err != nil {
		return err
	}
	if cfg != nil {
		h.hub.SetOverrides(cfg.ServerOverrides)
	}
	return nil
}

type ServerSettingsUpdate struct {
	Group   string `json:"group,omitempty"`
	SSHHost string `json:"ssh_host,omitempty"`
	SSHPort int    `json:"ssh_port,omitempty"`
	SSHUser string `json:"ssh_user,omitempty"`
	SSHPass string `json:"ssh_pass,omitempty"`
	SSHKey  string `json:"ssh_key_path,omitempty"`

	DockerMode      string `json:"docker_mode,omitempty"`
	DockerHost      string `json:"docker_host,omitempty"`
	DockerPort      int    `json:"docker_port,omitempty"`
	DockerTLS       *bool  `json:"docker_tls,omitempty"`
	DockerTLSCACert string `json:"docker_tls_ca,omitempty"`
	DockerTLSCert   string `json:"docker_tls_cert,omitempty"`
	DockerTLSKey    string `json:"docker_tls_key,omitempty"`
}

func (h *ServerSettingsHandler) updateServerConfig(serverID string, req ServerSettingsUpdate) error {
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

	// Find or create server_overrides section in the YAML
	overridesNode := findOrCreateKey(mapping, "server_overrides", yaml.SequenceNode)

	// Find or create an entry for this server ID
	var entry *yaml.Node
	for _, e := range overridesNode.Content {
		if e.Kind != yaml.MappingNode {
			continue
		}
		id := ""
		for j := 0; j < len(e.Content)-1; j += 2 {
			if e.Content[j].Value == "id" {
				id = e.Content[j+1].Value
				break
			}
		}
		if id == serverID {
			entry = e
			break
		}
	}

	if entry == nil {
		// Create new override entry
		entry = &yaml.Node{Kind: yaml.MappingNode}
		entry.Content = append(entry.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "id"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: serverID},
		)
		overridesNode.Content = append(overridesNode.Content, entry)
	}

	// Set fields
	if req.Group != "" {
		setField(entry, "group", req.Group)
	} else {
		removeField(entry, "group")
	}

	if req.SSHHost != "" || req.SSHPort > 0 || req.SSHUser != "" || req.SSHPass != "" || req.SSHKey != "" {
		sshNode := ensureMappingField(entry, "ssh")
		setField(sshNode, "host", req.SSHHost)
		if req.SSHPort > 0 {
			setIntField(sshNode, "port", req.SSHPort)
		} else {
			removeField(sshNode, "port")
		}
		setField(sshNode, "user", req.SSHUser)
		setField(sshNode, "password", req.SSHPass)
		setField(sshNode, "key_path", req.SSHKey)
	} else {
		removeField(entry, "ssh")
	}

	// Docker config: write or remove based on mode
	if req.DockerMode == "api" && req.DockerHost != "" {
		dockerNode := ensureMappingField(entry, "docker")
		setField(dockerNode, "mode", "api")
		setField(dockerNode, "host", req.DockerHost)
		if req.DockerPort > 0 {
			setIntField(dockerNode, "port", req.DockerPort)
		} else {
			setIntField(dockerNode, "port", 2375)
		}
		tlsVal := req.DockerTLS != nil && *req.DockerTLS
		setBoolField(dockerNode, "tls", tlsVal)
		// Only write TLS cert fields if TLS is enabled and values provided
		if req.DockerTLS != nil && *req.DockerTLS {
			setField(dockerNode, "tls_ca_cert", req.DockerTLSCACert)
			setField(dockerNode, "tls_cert", req.DockerTLSCert)
			setField(dockerNode, "tls_key", req.DockerTLSKey)
		} else {
			removeField(dockerNode, "tls_ca_cert")
			removeField(dockerNode, "tls_cert")
			removeField(dockerNode, "tls_key")
		}
	} else {
		removeField(entry, "docker")
	}

	// Write back
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(h.configPath, out, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// findOrCreateKey finds a key in a mapping node, creating it if missing.
// Returns the value node for that key.
func findOrCreateKey(mapping *yaml.Node, key string, kind yaml.Kind) *yaml.Node {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	node := &yaml.Node{Kind: kind}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		node,
	)
	return node
}

// setField sets a scalar field in a YAML mapping node.
func setField(mapping *yaml.Node, key, value string) {
	if value == "" {
		removeField(mapping, key)
		return
	}
	for j := 0; j < len(mapping.Content)-1; j += 2 {
		if mapping.Content[j].Value == key {
			mapping.Content[j+1].Value = value
			mapping.Content[j+1].Tag = "!!str"
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

// removeField removes a key from a YAML mapping node.
func removeField(mapping *yaml.Node, key string) {
	for j := 0; j < len(mapping.Content)-1; j += 2 {
		if mapping.Content[j].Value == key {
			mapping.Content = append(mapping.Content[:j], mapping.Content[j+2:]...)
			return
		}
	}
}

// ensureMappingField returns or creates a sub-mapping field.
func ensureMappingField(parent *yaml.Node, key string) *yaml.Node {
	for j := 0; j < len(parent.Content)-1; j += 2 {
		if parent.Content[j].Value == key {
			if parent.Content[j+1].Kind == yaml.MappingNode {
				return parent.Content[j+1]
			}
			node := &yaml.Node{Kind: yaml.MappingNode}
			parent.Content[j+1] = node
			return node
		}
	}
	node := &yaml.Node{Kind: yaml.MappingNode}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		node,
	)
	return node
}

// setIntField sets an integer field in a YAML mapping node.
func setIntField(mapping *yaml.Node, key string, value int) {
	for j := 0; j < len(mapping.Content)-1; j += 2 {
		if mapping.Content[j].Value == key {
			mapping.Content[j+1].Value = fmt.Sprintf("%d", value)
			mapping.Content[j+1].Tag = "!!int"
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", value)},
	)
}

// setBoolField sets a boolean field in a YAML mapping node.
func setBoolField(mapping *yaml.Node, key string, value bool) {
	strVal := "false"
	if value {
		strVal = "true"
	}
	tag := "!!bool"
	if !value {
		tag = "!!bool"
	}
	for j := 0; j < len(mapping.Content)-1; j += 2 {
		if mapping.Content[j].Value == key {
			mapping.Content[j+1].Value = strVal
			mapping.Content[j+1].Tag = tag
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: strVal},
	)
}
