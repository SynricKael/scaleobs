package monitor

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/glrs/observer/gateway/config"
	"github.com/glrs/observer/gateway/model"
)

// PerServerDockerConfig holds per-server Docker API connection settings.
type PerServerDockerConfig struct {
	ServerID  string
	Host      string
	Port      int
	TLS       bool
	TLSCACert string
	TLSCert   string
	TLSKey    string
}

// DockerHub polls remote Docker daemons for container lists.
type DockerHub struct {
	mu           sync.RWMutex
	hosts        []config.DockerHostConfig
	byHost       map[string]*DockerHostInfo
	dynamicHosts map[string]PerServerDockerConfig // key: serverID
}

// DockerHostInfo holds the last-known containers for a Docker host.
type DockerHostInfo struct {
	Name       string
	Host       string
	Containers []model.DockerContainer
	Stats      *model.DockerStats
	LastPoll   time.Time
	PollError  string
}

// NewDockerHub creates a DockerHub from configured Docker hosts.
func NewDockerHub(hosts []config.DockerHostConfig) *DockerHub {
	return &DockerHub{
		hosts:        hosts,
		byHost:       make(map[string]*DockerHostInfo),
		dynamicHosts: make(map[string]PerServerDockerConfig),
	}
}

// SetDynamicHost adds or updates a per-server Docker host configuration.
func (dh *DockerHub) SetDynamicHost(serverID string, cfg PerServerDockerConfig) {
	dh.mu.Lock()
	defer dh.mu.Unlock()
	dh.dynamicHosts[serverID] = cfg
}

// RemoveDynamicHost removes a per-server Docker host configuration.
func (dh *DockerHub) RemoveDynamicHost(serverID string) {
	dh.mu.Lock()
	defer dh.mu.Unlock()
	delete(dh.dynamicHosts, serverID)
	// Also clear cached result so stale data disappears
	delete(dh.byHost, serverID)
}

// SyncDynamicHosts synchronizes the dynamic host list from the given overrides.
// Only entries with Docker.Mode == "api" and a non-empty host are added.
func (dh *DockerHub) SyncDynamicHosts(overrides []model.ServerOverride) {
	dh.mu.Lock()
	defer dh.mu.Unlock()

	// Build new set from overrides
	active := make(map[string]bool)
	for _, ov := range overrides {
		if ov.Docker == nil || ov.Docker.Mode != "api" || ov.Docker.Host == "" {
			continue
		}
		dh.dynamicHosts[ov.ID] = PerServerDockerConfig{
			ServerID:  ov.ID,
			Host:      ov.Docker.Host,
			Port:      ov.Docker.Port,
			TLS:       ov.Docker.TLS,
			TLSCACert: ov.Docker.TLSCACert,
			TLSCert:   ov.Docker.TLSCert,
			TLSKey:    ov.Docker.TLSKey,
		}
		active[ov.ID] = true
	}

	// Remove stale entries and their cached results
	for id := range dh.dynamicHosts {
		if !active[id] {
			delete(dh.dynamicHosts, id)
			delete(dh.byHost, id)
		}
	}
}

// Start begins periodic polling of all configured Docker hosts.
func (dh *DockerHub) Start(interval time.Duration, done <-chan struct{}) {
	log.Printf("[docker] monitoring %d Docker host(s), polling every %s", len(dh.hosts), interval)
	dh.pollAll()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			dh.pollAll()
		case <-done:
			log.Println("[docker] stopping")
			return
		}
	}
}

// pollAll queries all configured Docker hosts for container lists.
func (dh *DockerHub) pollAll() {
	// Poll static hosts
	for _, h := range dh.hosts {
		info := dh.pollHost(h.Host, h.Port, false, "", "", "")
		dh.mu.Lock()
		dh.byHost[h.Host] = info
		dh.mu.Unlock()
	}

	// Poll dynamic (per-server) hosts
	dh.mu.RLock()
	dyn := make([]PerServerDockerConfig, 0, len(dh.dynamicHosts))
	for _, dc := range dh.dynamicHosts {
		dyn = append(dyn, dc)
	}
	dh.mu.RUnlock()

	for _, dc := range dyn {
		info := dh.pollHost(dc.Host, dc.Port, dc.TLS, dc.TLSCACert, dc.TLSCert, dc.TLSKey)
		dh.mu.Lock()
		// Store by host IP for lookup in GetAllStatuses
		dh.byHost[dc.Host] = info
		// Also store by serverID for direct lookup
		dh.byHost[dc.ServerID] = info
		dh.mu.Unlock()
	}
}

// pollHost queries a single Docker daemon.
func (dh *DockerHub) pollHost(host string, port int, tls bool, tlsCACert, tlsCert, tlsKey string) *DockerHostInfo {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	var transport *http.Transport
	if tls {
		tlsConfig, err := buildTLSConfig(tlsCACert, tlsCert, tlsKey)
		if err != nil {
			log.Printf("[docker] TLS config error for %s: %v", host, err)
			return &DockerHostInfo{Name: host, Host: host, LastPoll: time.Now(), PollError: err.Error()}
		}
		transport = &http.Transport{
			TLSClientConfig: tlsConfig,
			DialContext:     (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		}
	} else {
		transport = &http.Transport{
			DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		}
	}

	client := &http.Client{
		Timeout:   8 * time.Second,
		Transport: transport,
	}

	scheme := "http"
	if tls {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://%s/v1.41/containers/json?all=true", scheme, addr)
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[docker] poll %s: %v", host, err)
		return &DockerHostInfo{Name: host, Host: host, LastPoll: time.Now(), PollError: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		log.Printf("[docker] poll %s: %s", host, errMsg)
		return &DockerHostInfo{Name: host, Host: host, LastPoll: time.Now(), PollError: errMsg}
	}

	type dockerContainerJSON struct {
		ID      string   `json:"Id"`
		Names   []string `json:"Names"`
		Image   string   `json:"Image"`
		State   string   `json:"State"`
		Status  string   `json:"Status"`
		Ports   []struct {
			IP          string `json:"IP"`
			PrivatePort int    `json:"PrivatePort"`
			PublicPort  int    `json:"PublicPort"`
			Type        string `json:"Type"`
		} `json:"Ports"`
		Created int64              `json:"Created"`
		Labels  map[string]string  `json:"Labels"`
		NetworkSettings *struct {
			Networks map[string]struct {
				NetworkID string `json:"NetworkID"`
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}

	var containers []dockerContainerJSON
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		log.Printf("[docker] decode %s: %v", host, err)
		return &DockerHostInfo{Name: host, Host: host, LastPoll: time.Now(), PollError: err.Error()}
	}

	result := make([]model.DockerContainer, 0, len(containers))
	stats := &model.DockerStats{}

	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		ports := ""
		if len(c.Ports) > 0 {
			parts := make([]string, 0, len(c.Ports))
			for _, p := range c.Ports {
				if p.PublicPort > 0 {
					parts = append(parts, fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type))
				} else {
					parts = append(parts, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
				}
			}
			for i, s := range parts {
				if i > 0 {
					ports += ", "
				}
				ports += s
			}
		}

		// Extract labels
		var labels map[string]string
		if len(c.Labels) > 0 {
			labels = make(map[string]string, len(c.Labels))
			for k, v := range c.Labels {
				// Skip noisy Docker internal labels
				if k == "com.docker.compose.config-hash" || k == "com.docker.compose.container-number" || k == "com.docker.compose.depends_on" || k == "com.docker.compose.version" {
					continue
				}
				labels[k] = v
			}
			if len(labels) == 0 {
				labels = nil
			}
		}

		// Extract Docker network names
		var networks []string
		if c.NetworkSettings != nil && c.NetworkSettings.Networks != nil {
			networks = make([]string, 0, len(c.NetworkSettings.Networks))
			for netName := range c.NetworkSettings.Networks {
				networks = append(networks, netName)
			}
		}

		shortID := c.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		dc := model.DockerContainer{
			ID:       shortID,
			Name:     name,
			Image:    c.Image,
			State:    c.State,
			Status:   c.Status,
			Ports:    ports,
			Created:  c.Created,
			Labels:   labels,
			Networks: networks,
		}
		result = append(result, dc)

		stats.Total++
		if c.State == "running" {
			stats.Running++
		}
	}
	stats.Stopped = stats.Total - stats.Running

	return &DockerHostInfo{
		Name:       host,
		Host:       host,
		Containers: result,
		Stats:      stats,
		LastPoll:   time.Now(),
	}
}

// GetByHost returns Docker info for a given host IP or server ID, or nil.
func (dh *DockerHub) GetByHost(hostIP string) *DockerHostInfo {
	dh.mu.RLock()
	defer dh.mu.RUnlock()
	info, ok := dh.byHost[hostIP]
	if !ok {
		return nil
	}
	return info
}

// buildTLSConfig creates a tls.Config from PEM-encoded cert files or inline content.
func buildTLSConfig(caCert, cert, key string) (*tls.Config, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

	// CA certificate
	if caCert != "" {
		caPool := x509.NewCertPool()
		var caPEM []byte
		// Try as file path first
		if _, err := os.Stat(caCert); err == nil {
			caPEM, err = os.ReadFile(caCert)
			if err != nil {
				return nil, fmt.Errorf("read CA cert: %w", err)
			}
		} else {
			caPEM = []byte(caCert)
		}
		if !caPool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsCfg.RootCAs = caPool
	}

	// Client certificate
	if cert != "" && key != "" {
		var certPEM, keyPEM []byte

		if _, err := os.Stat(cert); err == nil {
			certPEM, err = os.ReadFile(cert)
			if err != nil {
				return nil, fmt.Errorf("read cert: %w", err)
			}
		} else {
			certPEM = []byte(cert)
		}
		if _, err := os.Stat(key); err == nil {
			keyPEM, err = os.ReadFile(key)
			if err != nil {
				return nil, fmt.Errorf("read key: %w", err)
			}
		} else {
			keyPEM = []byte(key)
		}

		certPair, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("load client cert pair: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{certPair}
	}

	return tlsCfg, nil
}
