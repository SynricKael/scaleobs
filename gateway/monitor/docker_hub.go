package monitor

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/glrs/observer/gateway/config"
	"github.com/glrs/observer/gateway/model"
)

// DockerHub polls remote Docker daemons for container lists.
type DockerHub struct {
	mu     sync.RWMutex
	hosts  []config.DockerHostConfig
	byHost map[string]*DockerHostInfo // key: host IP
}

// DockerHostInfo holds the last-known containers for a Docker host.
type DockerHostInfo struct {
	Name             string
	Host             string
	Containers       []model.DockerContainer
	Stats            *model.DockerStats
	LastPoll         time.Time
	PollError        string
}

// NewDockerHub creates a DockerHub from configured Docker hosts.
func NewDockerHub(hosts []config.DockerHostConfig) *DockerHub {
	return &DockerHub{
		hosts:  hosts,
		byHost: make(map[string]*DockerHostInfo),
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
	for _, h := range dh.hosts {
		info := dh.pollHost(h)
		dh.mu.Lock()
		dh.byHost[h.Host] = info
		dh.mu.Unlock()
	}
}

// pollHost queries a single Docker daemon.
func (dh *DockerHub) pollHost(h config.DockerHostConfig) *DockerHostInfo {
	addr := net.JoinHostPort(h.Host, fmt.Sprintf("%d", h.Port))
	client := &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		},
	}

	url := fmt.Sprintf("http://%s/v1.41/containers/json?all=true", addr)
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[docker] poll %s (%s): %v", h.Name, h.Host, err)
		return &DockerHostInfo{Name: h.Name, Host: h.Host, LastPoll: time.Now(), PollError: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		log.Printf("[docker] poll %s: %s", h.Host, errMsg)
		return &DockerHostInfo{Name: h.Name, Host: h.Host, LastPoll: time.Now(), PollError: errMsg}
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
		Created int64 `json:"Created"`
	}

	var containers []dockerContainerJSON
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		log.Printf("[docker] decode %s: %v", h.Host, err)
		return &DockerHostInfo{Name: h.Name, Host: h.Host, LastPoll: time.Now(), PollError: err.Error()}
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

		shortID := c.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		dc := model.DockerContainer{
			ID:      shortID,
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Ports:   ports,
			Created: c.Created,
		}
		result = append(result, dc)

		stats.Total++
		if c.State == "running" {
			stats.Running++
		}
	}
	stats.Stopped = stats.Total - stats.Running

	return &DockerHostInfo{
		Name:       h.Name,
		Host:       h.Host,
		Containers: result,
		Stats:      stats,
		LastPoll:   time.Now(),
	}
}

// GetByHost returns Docker info for a given host IP, or nil.
func (dh *DockerHub) GetByHost(hostIP string) *DockerHostInfo {
	dh.mu.RLock()
	defer dh.mu.RUnlock()
	info, ok := dh.byHost[hostIP]
	if !ok {
		return nil
	}
	return info
}
