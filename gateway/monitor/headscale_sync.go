package monitor

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/glrs/observer/gateway/config"
)

// HeadscaleSyncer periodically polls one or more Headscale networks
// for node discovery and syncs them into the AgentHub.
type HeadscaleSyncer struct {
	hub  *AgentHub
	networks []config.HeadscaleNetworkConfig
	client   *http.Client
}

// NewHeadscaleSyncer creates a syncer for the given networks.
func NewHeadscaleSyncer(hub *AgentHub, networks []config.HeadscaleNetworkConfig) *HeadscaleSyncer {
	return &HeadscaleSyncer{
		hub:      hub,
		networks: networks,
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// Poll fetches nodes from all configured Headscale networks and syncs them.
func (s *HeadscaleSyncer) Poll() {
	if len(s.networks) == 0 {
		return
	}

	for _, nw := range s.networks {
		nodes, err := s.fetchNetwork(nw)
		if err != nil {
			log.Printf("[headscale] poll %s (%s) error: %v", nw.Name, nw.URL, err)
			continue
		}
		log.Printf("[headscale] %s: %d nodes", nw.Name, len(nodes))
		s.hub.SyncHeadscaleNodes(nodes)
	}
}

// hsNodeJSON matches the relevant fields of Headscale's ListNodes response.
type hsListResponse struct {
	Nodes []hsNodeJSON `json:"nodes"`
}
type hsNodeJSON struct {
	ID         string   `json:"id"`
	GivenName  string   `json:"givenName"`
	IPs        []string `json:"ipAddresses"`
	Online     bool     `json:"online"`
	LastSeen   string   `json:"lastSeen"`
}

func (s *HeadscaleSyncer) fetchNetwork(nw config.HeadscaleNetworkConfig) ([]HeadscaleNode, error) {
	req, err := http.NewRequest("GET", nw.URL+"/api/v1/node", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+nw.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var list hsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	nodes := make([]HeadscaleNode, 0, len(list.Nodes))
	for _, n := range list.Nodes {
		var lastSeen time.Time
		if n.LastSeen != "" && n.LastSeen != "0001-01-01T00:00:00Z" {
			if t, err := time.Parse(time.RFC3339Nano, n.LastSeen); err == nil {
				lastSeen = t
			}
		}

		nodes = append(nodes, HeadscaleNode{
			ID:          n.ID,
			Name:        n.GivenName,
			IPs:         n.IPs,
			Online:      n.Online,
			LastSeen:    lastSeen,
			NetworkName: nw.Name,
		})
	}

	return nodes, nil
}

// Start begins periodic polling in a background goroutine.
func (s *HeadscaleSyncer) Start(interval time.Duration, stop <-chan struct{}) {
	if len(s.networks) == 0 {
		log.Println("[headscale] no networks configured, sync disabled")
		return
	}

	log.Printf("[headscale] starting sync (interval=%v, networks=%d)", interval, len(s.networks))

	// Run once immediately
	s.Poll()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.Poll()
		case <-stop:
			log.Println("[headscale] sync stopped")
			return
		}
	}
}
