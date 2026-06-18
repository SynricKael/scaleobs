package model

// ServerConfig is the configuration for a monitored server (YAML-persisted).
type ServerConfig struct {
	ID         string   `yaml:"id" json:"id"`
	Name       string   `yaml:"name" json:"name"`
	Host       string   `yaml:"host" json:"host"`
	AgentPort  int      `yaml:"agent_port" json:"agent_port"`
	AgentToken string   `yaml:"agent_token" json:"-"`
	Tags       []string `yaml:"tags" json:"tags,omitempty"`
	Location   string   `yaml:"location" json:"location,omitempty"`
	Agents     []string `yaml:"agents" json:"agents,omitempty"`
	Group      string   `yaml:"group,omitempty" json:"group,omitempty"`
	SSH        *SSHConfig `yaml:"ssh,omitempty" json:"ssh,omitempty"`
}

// SSHConfig holds SSH connection settings for remote management.
type SSHConfig struct {
	Host     string `yaml:"host,omitempty" json:"host,omitempty"`
	Port     int    `yaml:"port,omitempty" json:"port,omitempty"`
	User     string `yaml:"user,omitempty" json:"user,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	KeyPath  string `yaml:"key_path,omitempty" json:"key_path,omitempty"`
}

// ServerOverride stores per-server overrides (SSH, group) for any server regardless of source.
type ServerOverride struct {
	ID    string     `yaml:"id" json:"id"`
	Group string     `yaml:"group,omitempty" json:"group,omitempty"`
	SSH   *SSHConfig `yaml:"ssh,omitempty" json:"ssh,omitempty"`
}

// ServerStatus is the live runtime status of a server.
type ServerStatus struct {
	ServerConfig
	Online    bool             `json:"online"`
	Metrics   *ServerMetrics   `json:"metrics,omitempty"`
	LastSeen  int64            `json:"last_seen"` // Unix timestamp
}

// ServerMetrics holds the latest metrics snapshot from an Agent.
type ServerMetrics struct {
	CPUPercent       float64            `json:"cpu_percent"`
	Memory           MemoryMetrics      `json:"memory"`
	Disks            []DiskMetrics      `json:"disks,omitempty"`
	Network          *NetworkMetrics    `json:"network,omitempty"`
	UptimeSec        int64              `json:"uptime_seconds"`
	DockerStats      *DockerStats       `json:"docker_stats,omitempty"`
	DockerContainers []DockerContainer  `json:"docker_containers,omitempty"`
	Agents           []string           `json:"agents,omitempty"`
}

// NetworkMetrics holds network I/O rates.
type NetworkMetrics struct {
	BytesSentPerSec float64 `json:"bytes_sent_per_sec"`
	BytesRecvPerSec float64 `json:"bytes_recv_per_sec"`
}

// MemoryMetrics holds RAM usage.
type MemoryMetrics struct {
	TotalMB int64   `json:"total_mb"`
	UsedMB  int64   `json:"used_mb"`
	Percent float64 `json:"percent"`
}

// DiskMetrics holds a single mount point's usage.
type DiskMetrics struct {
	Mount   string  `json:"mount"`
	TotalGB int64   `json:"total_gb"`
	UsedGB  int64   `json:"used_gb"`
	Percent float64 `json:"percent"`
}

// DockerStats summarizes container states.
type DockerStats struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
}

// AgentServerConfig defines a coding agent server (Codex, Claude Code, OpenCode, etc.)
type AgentServerConfig struct {
	Name     string `yaml:"name" json:"name"`
	URL      string `yaml:"url" json:"url"`
	User     string `yaml:"user,omitempty" json:"user,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// AgentServerStatus is the live status of an agent server.
type AgentServerStatus struct {
	AgentServerConfig
	Online   bool   `json:"online"`
	LastPing int64  `json:"last_ping"`
	Error    string `json:"error,omitempty"`
}

// DockerContainer holds details about a single container.
type DockerContainer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	State   string `json:"state"`
	Status  string `json:"status"`
	Ports   string `json:"ports,omitempty"`
	Created int64  `json:"created"`
}
