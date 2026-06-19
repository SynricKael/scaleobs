package monitor

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/glrs/observer/gateway/model"
	"github.com/gorilla/websocket"
)

// AgentHub manages WebSocket connections from agents and Headscale node discovery.
type AgentHub struct {
	servers     map[string]*agentConnection
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
	agentTokens map[string]string
	dockerHub   *DockerHub
	hostAgents  map[string][]string // host IP -> coding agents
	overrides   map[string]model.ServerOverride // server ID -> override settings
}

type agentConnection struct {
	ServerID  string
	Name      string
	Host      string            // IP or hostname of the server
	Conn      *websocket.Conn
	LastSeen  time.Time
	Metrics   *model.ServerMetrics
	Connected bool
	Source    string // "config", "agent", "headscale"
	Network   string // headscale network name (if from headscale)
	Agents    []string // coding agents installed (codex, claude code, opencode, etc.)
}

// SetDockerHub attaches a DockerHub for remote Docker container polling.
func (hub *AgentHub) SetDockerHub(dh *DockerHub) {
	hub.dockerHub = dh
}

// SetHostAgents stores coding agent annotations and applies them to matching hosts.
func (hub *AgentHub) SetHostAgents(hostAgents map[string][]string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.hostAgents = hostAgents
	for _, ac := range hub.servers {
		if agents, ok := hostAgents[ac.Host]; ok && len(agents) > 0 {
			ac.Agents = agents
		}
	}
}

// SetOverrides stores per-server override settings (group, SSH).
func (hub *AgentHub) SetOverrides(overrides []model.ServerOverride) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.overrides = make(map[string]model.ServerOverride, len(overrides))
	for _, o := range overrides {
		hub.overrides[o.ID] = o
	}
}

// applyHostAgents checks if a host IP has agent annotations and assigns them.
// Must be called with hub.mu held.
func (hub *AgentHub) applyHostAgents(ac *agentConnection) {
	if hub.hostAgents == nil {
		return
	}
	if agents, ok := hub.hostAgents[ac.Host]; ok && len(agents) > 0 {
		ac.Agents = agents
	}
}

// NewAgentHub creates an AgentHub from the configured server list.
func NewAgentHub(servers []model.ServerConfig, agentTokens map[string]string) *AgentHub {
	hub := &AgentHub{
		servers:     make(map[string]*agentConnection),
		agentTokens: agentTokens,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	// Pre-populate from config
	for _, s := range servers {
		hub.servers[s.ID] = &agentConnection{
			ServerID:  s.ID,
			Name:      s.Name,
			Host:      s.Host,
			LastSeen:  time.Now(),
			Connected: false,
			Source:    "config",
			Agents:    s.Agents,
		}
	}

	return hub
}

// SyncHeadscaleNodes upserts server entries from Headscale node discovery.
// Called periodically by the HeadscaleSyncer.
func (hub *AgentHub) SyncHeadscaleNodes(nodes []HeadscaleNode) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	for _, n := range nodes {
		// Use Tailscale IP as the ID for headscale-discovered nodes
		primaryIP := ""
		for _, ip := range n.IPs {
			if len(ip) > 0 && ip[0] != 'f' { // Prefer IPv4 (not starting with 'f' = fd7a...)
				primaryIP = ip
				break
			}
		}
		if primaryIP == "" && len(n.IPs) > 0 {
			primaryIP = n.IPs[0]
		}

		id := "hs-" + primaryIP

		// If already exists and connected via agent, just update name/host if missing
		if existing, ok := hub.servers[id]; ok && existing.Connected && existing.Source == "agent" {
			if existing.Name == "" && n.Name != "" {
				existing.Name = n.Name
			}
			if existing.Host == "" && primaryIP != "" {
				existing.Host = primaryIP
			}
			if n.NetworkName != "" {
				existing.Network = n.NetworkName
			}
			continue
		}

		ac := &agentConnection{
			ServerID:  id,
			Name:      n.Name,
			Host:      primaryIP,
			Connected: n.Online,
			LastSeen:  n.LastSeen,
			Source:    "headscale",
			Network:   n.NetworkName,
		}
		hub.applyHostAgents(ac)
		hub.servers[id] = ac
	}
}

// HandleWS handles WebSocket upgrade requests at /api/ws/agent
func (hub *AgentHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := hub.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[agent] websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Println("[agent] new agent connection")

	var authMsg model.AgentMessage
	if err := conn.ReadJSON(&authMsg); err != nil {
		log.Printf("[agent] auth read error: %v", err)
		return
	}

	if authMsg.Type != "auth" {
		log.Printf("[agent] expected auth message, got %s", authMsg.Type)
		conn.WriteJSON(model.AgentMessage{Type: "auth_error", ServerID: authMsg.ServerID})
		return
	}

	expectedToken, exists := hub.agentTokens[authMsg.ServerID]
	sharedToken := hub.agentTokens["*"]
	if (!exists || expectedToken != authMsg.Token) && sharedToken != authMsg.Token {
		log.Printf("[agent] invalid token for server %s", authMsg.ServerID)
		conn.WriteJSON(model.AgentMessage{Type: "auth_error", ServerID: authMsg.ServerID})
		return
	}

	hub.mu.Lock()
	existing, hasExisting := hub.servers[authMsg.ServerID]
	ac := &agentConnection{
		ServerID:  authMsg.ServerID,
		Conn:      conn,
		LastSeen:  time.Now(),
		Connected: true,
		Source:    "agent",
	}
	if hasExisting {
		ac.Name = existing.Name
		ac.Host = existing.Host
	}
	hub.servers[authMsg.ServerID] = ac
	hub.mu.Unlock()

	log.Printf("[agent] server %s authenticated", authMsg.ServerID)
	conn.WriteJSON(model.AgentMessage{Type: "auth_ok", ServerID: authMsg.ServerID})

	for {
		var msg model.AgentMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("[agent] read error from %s: %v", authMsg.ServerID, err)
			break
		}

		switch msg.Type {
		case "metrics":
			hub.mu.Lock()
			if ac, ok := hub.servers[authMsg.ServerID]; ok {
				ac.LastSeen = time.Now()
				ac.Metrics = msg.Data
				// Copy auto-detected agents to the connection record
				if msg.Data != nil && len(msg.Data.Agents) > 0 {
					ac.Agents = msg.Data.Agents
				}
			}
			hub.mu.Unlock()
		default:
			log.Printf("[agent] unknown message type from %s: %s", authMsg.ServerID, msg.Type)
		}
	}

	hub.mu.Lock()
	oldEntry := hub.servers[authMsg.ServerID]
	name := ""
	host := ""
	if oldEntry != nil {
		name = oldEntry.Name
		host = oldEntry.Host
	}
	hub.servers[authMsg.ServerID] = &agentConnection{
		ServerID:  authMsg.ServerID,
		Name:      name,
		Host:      host,
		Connected: false,
		LastSeen:  time.Now(),
		Source:    "config",
	}
	hub.mu.Unlock()

	log.Printf("[agent] server %s disconnected", authMsg.ServerID)
}

// GetAllStatuses returns combined status of all known servers:
// configured servers + agent-connected servers + headscale-discovered nodes.
// Merges Docker container info from remote Docker hosts where applicable.
func (hub *AgentHub) GetAllStatuses() []map[string]interface{} {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(hub.servers))
	for id, ac := range hub.servers {
		status := map[string]interface{}{
			"id":        id,
			"name":      ac.Name,
			"host":      ac.Host,
			"online":    ac.Connected,
			"last_seen": ac.LastSeen.Unix(),
			"source":    ac.Source,
		}
		if ac.Network != "" {
			status["network_name"] = ac.Network
		}
		if len(ac.Agents) > 0 {
			status["agents"] = ac.Agents
		}

		// Apply overrides (group, SSH config)
		if ov, ok := hub.overrides[id]; ok {
			if ov.Group != "" {
				status["group"] = ov.Group
			}
			if ov.SSH != nil {
				status["ssh"] = map[string]interface{}{
					"host":     ov.SSH.Host,
					"port":     ov.SSH.Port,
					"user":     ov.SSH.User,
					"password": ov.SSH.Password,
					"key_path": ov.SSH.KeyPath,
				}
			}
		}

		// Start with agent metrics (if any)
		metrics := ac.Metrics

		// Merge Docker container info from remote Docker hub (if host IP matches)
		if hub.dockerHub != nil && ac.Host != "" {
			if dockerInfo := hub.dockerHub.GetByHost(ac.Host); dockerInfo != nil && dockerInfo.Containers != nil {
				if metrics == nil {
					metrics = &model.ServerMetrics{}
				}
				metrics.DockerContainers = dockerInfo.Containers
				metrics.DockerStats = dockerInfo.Stats
			}
		}

		if metrics != nil {
			status["metrics"] = metrics
		}
		result = append(result, status)
	}
	return result
}

// HeadscaleNode represents a node from Headscale API discovery.
type HeadscaleNode struct {
	ID          string
	Name        string
	IPs         []string
	Online      bool
	LastSeen    time.Time
	NetworkName string
}
