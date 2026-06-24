package collector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/glrs/observer/agent/model"
)

// dockerSocketPath is the default Docker daemon socket path.
const dockerSocketPath = "/var/run/docker.sock"

// dockerContainerJSON matches the relevant fields of Docker's /containers/json response.
type dockerContainerJSON struct {
	ID      string `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	State   string   `json:"State"`
	Status  string   `json:"Status"`
	Ports   []dockerPortJSON `json:"Ports"`
	Created int64    `json:"Created"`
	HostConfig *dockerHostConfigJSON `json:"HostConfig,omitempty"`
	NetworkSettings *dockerNetworkSettingsJSON `json:"NetworkSettings,omitempty"`
}

type dockerHostConfigJSON struct {
	NetworkMode string `json:"NetworkMode"`
}

type dockerNetworkSettingsJSON struct {
	Networks map[string]dockerNetworkJSON `json:"Networks,omitempty"`
}

type dockerNetworkJSON struct {
	NetworkID string `json:"NetworkID"`
}

type dockerPortJSON struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

// CollectDockerContainers attempts to list all containers via the Docker socket.
// Returns nil if Docker is not available (no socket, permission denied, etc.),
// so the agent continues to work even without Docker.
func CollectDockerContainers() ([]model.DockerContainer, *model.DockerStats) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext:     dockerDialContext,
			MaxIdleConns:    1,
			IdleConnTimeout: 30 * time.Second,
		},
	}

	resp, err := client.Get("http://docker/v1.41/containers/json?all=true")
	if err != nil {
		// Docker not available — not an error
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var containers []dockerContainerJSON
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, nil
	}

	result := make([]model.DockerContainer, 0, len(containers))
	stats := &model.DockerStats{}

	for _, c := range containers {
		// Clean container name (remove leading /)
		name := c.Names[0]
		name = strings.TrimPrefix(name, "/")

		// Format ports
		ports := formatDockerPorts(c.Ports)

		// Extract network names
		var networks []string
		if c.NetworkSettings != nil && c.NetworkSettings.Networks != nil {
			for netName := range c.NetworkSettings.Networks {
				networks = append(networks, netName)
			}
		}
		if len(networks) == 0 && c.HostConfig != nil && c.HostConfig.NetworkMode != "" && c.HostConfig.NetworkMode != "default" {
			networks = []string{c.HostConfig.NetworkMode}
		}
		if len(networks) == 0 {
			networks = []string{"bridge"}
		}

		dc := model.DockerContainer{
			ID:       c.ID[:12], // short ID
			Name:     name,
			Image:    c.Image,
			State:    c.State,
			Status:   c.Status,
			Ports:    ports,
			Networks: networks,
			Created:  c.Created,
		}
		result = append(result, dc)

		// Update stats
		stats.Total++
		switch c.State {
		case "running":
			stats.Running++
		default:
			stats.Stopped++
		}
	}

	return result, stats
}

// formatDockerPorts formats port mappings into a concise string.
func formatDockerPorts(ports []dockerPortJSON) string {
	if len(ports) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		if p.PublicPort > 0 {
			parts = append(parts, fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type))
		} else {
			parts = append(parts, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
		}
	}
	return strings.Join(parts, ", ")
}
