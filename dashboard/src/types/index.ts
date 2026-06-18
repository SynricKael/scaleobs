// Service represents a third-party web management service
export interface Service {
  id: string
  name: string
  icon: string
  url: string
  category: string
  open_method: 'iframe' | 'browser'
  panel_url?: string
  status: 'online' | 'degraded' | 'offline' | 'unknown'
  last_check?: number
  error?: string
}

// AgentServerStatus is the live status of a coding agent server (Codex, Claude Code, etc.)
export interface AgentServerStatus {
  name: string
  url: string
  user?: string
  online: boolean
  last_ping: number
  error?: string
}

// ServerStatus represents the live status of a monitored server
export interface ServerStatus {
  id: string
  name?: string
  host?: string
  online: boolean
  last_seen: number
  source?: string   // "config" | "agent" | "headscale"
  network_name?: string
  agents?: string[]  // coding agents installed (codex, claude code, opencode, etc.)
  metrics?: ServerMetrics
  group?: string     // group assignment for organizing servers
  ssh?: SSHConfig    // SSH connection settings
}

// SSHConfig holds SSH connection settings for remote management
export interface SSHConfig {
  host?: string
  port?: number
  user?: string
  password?: string
  key_path?: string
}

// NetworkMetrics holds network I/O rates
export interface NetworkMetrics {
  bytes_sent_per_sec: number
  bytes_recv_per_sec: number
}

// ServerMetrics holds the latest metrics snapshot
export interface ServerMetrics {
  cpu_percent: number
  memory: {
    total_mb: number
    used_mb: number
    percent: number
  }
  disks?: Array<{
    mount: string
    total_gb: number
    used_gb: number
    percent: number
  }>
  network?: NetworkMetrics
  uptime_seconds: number
  docker_stats?: {
    total: number
    running: number
    stopped: number
  }
  docker_containers?: DockerContainer[]
  agents?: string[]  // auto-detected coding agents from agent process scan
}

export interface DockerContainer {
  id: string
  name: string
  image: string
  state: string   // running, exited, paused, etc.
  status: string  // human-readable status
  ports?: string
  created: number // Unix timestamp
}

// Login request/response
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_at: number
  username: string
}
