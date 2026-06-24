package model

// Metrics represents the full system metrics snapshot.
type Metrics struct {
	CPUPercent       float64            `json:"cpu_percent"`
	Memory           MemoryInfo         `json:"memory"`
	Disks            []DiskInfo         `json:"disks,omitempty"`
	Network          *NetworkInfo       `json:"network,omitempty"`
	UptimeSec        int64              `json:"uptime_seconds"`
	DockerStats      *DockerStats       `json:"docker_stats,omitempty"`
	DockerContainers []DockerContainer  `json:"docker_containers,omitempty"`
	Agents           []string           `json:"agents,omitempty"`
	Timestamp        int64              `json:"timestamp"`
}

// NetworkInfo holds network I/O rates.
type NetworkInfo struct {
	BytesSentPerSec float64 `json:"bytes_sent_per_sec"` // upload rate (bytes/sec)
	BytesRecvPerSec float64 `json:"bytes_recv_per_sec"` // download rate (bytes/sec)
}

// MemoryInfo holds RAM information.
type MemoryInfo struct {
	TotalMB  int64   `json:"total_mb"`
	UsedMB   int64   `json:"used_mb"`
	Percent  float64 `json:"percent"`
}

// DiskInfo holds a single mount point.
type DiskInfo struct {
	Mount   string  `json:"mount"`
	TotalGB int64   `json:"total_gb"`
	UsedGB  int64   `json:"used_gb"`
	Percent float64 `json:"percent"`
}

// DockerStats summarizes container states (optional).
type DockerStats struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
}

// DockerContainer holds details about a single container.
type DockerContainer struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	State     string   `json:"state"`     // running, exited, paused, etc.
	Status    string   `json:"status"`    // human-readable status
	Ports     string   `json:"ports,omitempty"`
	Networks  []string `json:"networks,omitempty"` // Docker network names
	Created   int64    `json:"created"`  // Unix timestamp
}

// AgentMessage is the WebSocket message format (matching Gateway).
type AgentMessage struct {
	Type      string   `json:"type"`
	ServerID  string   `json:"server_id,omitempty"`
	Token     string   `json:"token,omitempty"`
	Timestamp string   `json:"timestamp,omitempty"`
	Data      *Metrics `json:"data,omitempty"`
}
