# ScaleObs

**Server observation and coding agent management dashboard.**

ScaleObs is a self-hosted operations portal that auto-discovers servers via [Headscale](https://headscale.net/), monitors them via a lightweight agent, and shows CPU/memory/disk/network/Docker/coding agents in a single dashboard.

![Dashboard](https://img.shields.io/badge/status-active-brightgreen)
![Go](https://img.shields.io/badge/Go-1.22%2B-blue)
![Tauri](https://img.shields.io/badge/Tauri-2.x-purple)

---

## Features

- **Auto-discovery** — Pulls server nodes from one or more Headscale networks
- **Lightweight agent** — Installs on each host; reports CPU, memory, disk, network, Docker containers
- **Coding agent detection** — Auto-detects `opencode`, `codex`, `claude code` running on each host; shows badges on server cards
- **Remote Docker monitoring** — Polls remote Docker daemons via TCP API, merges container info into server status
- **AI Agent Server panel** — Shows detected coding agents; manually add entries for unreachable hosts
- **Dashboard** — Tauri + Vue 3 desktop app with sections for servers, networks, agent servers, and service tiles
- **YAML configuration** — Edit `services.yml` via Settings page or add entries through the UI
- **Agent binary distribution** — Download pre-built agent binaries for Linux, macOS, and Windows

## Positioning

ScaleObs is **not** a replacement for Grafana, Portainer, or Prometheus — it stands **in front of them.**

```
                    ┌──────────────────┐
                    │    ScaleObs      │  ← First glance: every server at a glance
                    │  "At a glance"   │
                    └──────┬───────────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
     ┌──────▼─────┐ ┌─────▼──────┐ ┌─────▼──────┐
     │  Grafana   │ │  Portainer │ │ Headscale  │  ← Drill down when needed
     │  trends    │ │ containers │ │ nodes      │
     └────────────┘ └────────────┘ └────────────┘
```

### Problems other tools don't solve

| Problem | Existing tools | ScaleObs |
|---------|---------------|----------|
| "What's the overall state of my 10 servers?" | Grafana needs Prometheus per host + manual dashboards | Install one agent and it appears |
| "Which nodes are running AI coding agents?" | No tool can answer this | Purple badges at a glance |
| "I added a Mac Mini to Tailscale, I want to see it" | Manually add targets in Grafana, endpoints in Portainer | Auto-syncs from Headscale, appears instantly |
| "I want to see all services on one page" | 5 browser tabs open | One page, all covered |
| "My cheap 2C4G VPS can barely run, what monitoring fits?" | Prometheus + Grafana eat 1GB RAM | Agent is < 20MB |

### What ScaleObs really does

- **Auto-discovery** — Server joins Tailscale → appears on dashboard. This is the core differentiator.
- **Ultra-lightweight** — One binary, zero dependencies. Perfect for resource-constrained edge nodes.
- **AI Agent awareness** — Unique feature with no equivalent in the ecosystem.
- **Single entry point** — No more switching between 5 panels. ScaleObs is your first stop.
- **Simple config** — One YAML file handles everything.

### When ScaleObs truly shines

- You have **3+ machines** scattered across home, cloud, and office
- You use **Tailscale** for networking
- You run **AI coding agents** (OpenCode / Codex / Claude Code)
- You want to know **"is everything OK right now"** — not query historical trends

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Agent(s)    │────▶│  Gateway     │◀────│  Dashboard   │
│  (collects   │WS   │  (Go server) │HTTP │  (Tauri+Vue) │
│   metrics)   │     │  :8080       │     │  :4173       │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
                    ┌───────┴───────┐
                    │  Headscale    │
                    │  API          │
                    │  :8444        │
                    └───────────────┘
```

### Components

| Component | Tech | Description |
|-----------|------|-------------|
| **Gateway** | Go | Central server: API, WebSocket for agents, Headscale sync, config management |
| **Agent** | Go | Per-host metrics collector: CPU, memory, disk, Docker, coding agent detection |
| **Dashboard** | Tauri 2 + Vue 3 + Vite | Desktop GUI with server cards, network overview, AI agent panel |
| **Headscale** | External | Tailscale-compatible coordination server for node discovery |

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+ / Bun
- Rust (for Tauri desktop build)
- A Headscale server (optional — for auto-discovery)

### 1. Start the Gateway

```bash
cd gateway
export CONFIG_PATH=../config/services.yml
export JWT_SECRET=your-secret-key
export ADMIN_USERNAME=admin
export ADMIN_PASSWORD=your-password
export AGENT_TOKEN=agent-secret-token
go run .
```

Gateway runs on `http://localhost:8080`.

### 2. Start the Dashboard (development)

```bash
cd dashboard
bun install
bun run dev          # Vite dev server on :5173
# or
bun run tauri dev    # Tauri desktop window
```

### 3. Install Agents on Hosts

Download the agent from the Settings page or directly:

```bash
# Linux
wget http://your-gateway:8080/api/agent/download/linux/amd64 -O /usr/local/bin/scaleobs-agent
chmod +x /usr/local/bin/scaleobs-agent

export GATEWAY_URL=ws://your-gateway:8080
export SERVER_ID=my-server
export AGENT_TOKEN=agent-secret-token
scaleobs-agent &
```

## Configuration

Edit `config/services.yml` or use the Dashboard Settings page.

```yaml
# Headscale networks for auto-discovery
headscale_networks:
  - name: "primary"
    url: https://headscale.example.com:8444
    api_key: "your-api-key"

# Remote Docker daemons to monitor
docker_hosts:
  - name: "server-1"
    host: "100.64.0.4"
    port: 2375

# Coding agent annotations by host IP
host_agents:
  "100.64.0.1": [opencode]
  "100.64.0.4": [codex]
```

## Development

```bash
# Build agent
cd agent && go build -o scaleobs-agent .

# Build gateway
cd gateway && go build -o scaleobs-gateway .

# Build dashboard for production
cd dashboard && bun run build
```

## Community

Join the **ScaleObs WeChat group** (Chinese) for discussions about server ops, monitoring, and coding agent management:

> Group: ScaleObs3  
> Scan the QR code:  
> ![ScaleObs WeChat](docs/images/wechat-scaleobs3.png)

If the QR code expired, open an Issue or contact the maintainer.

**Pull Requests are welcome!** Bug fixes, improvements, and new features — all contributions are appreciated.

## License

MIT
