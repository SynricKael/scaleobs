# ScaleObs

**Lightweight distributed node & container overview dashboard.**

Auto-discover Headscale / Tailscale nodes, deploy a tiny agent (<20MB) on each host for read-only resource & container monitoring. Single-page dashboard for cluster health, AI workload badges, and highlighted anomalies — complements Grafana & Portainer for quick fault locating instead of replacing them.

![Dashboard](https://img.shields.io/badge/status-active-brightgreen)
![Go](https://img.shields.io/badge/Go-1.22%2B-blue)
![Tauri](https://img.shields.io/badge/Tauri-2.x-purple)

---

## Why ScaleObs

In parallel development of containerized microservices, developers commonly face tangled operational problems:

**Scattered microservices, slow fault location**  
Services run across cloud servers, local boxes, and edge devices. When a container goes down, there's no global overview — you have to SSH into each machine and run `docker ps` one by one.

**Isolated toolchains, messy remote management**  
Grafana for metrics, Portainer for containers, Headscale for node management. You switch between a dozen browser tabs with scattered credentials. Adding a new node means configuring every platform from scratch.

**Cross-region network jitter masks real failures**  
Tailscale / Headscale tunnels jitter and time out frequently. It's hard to tell whether a service actually crashed or the network just glitched, leading to false restarts and wasted time.

Traditional Prometheus + cAdvisor stack is bloated, heavy on memory, painful to configure per node, and overkill for small-scale teams or edge hardware.

### Positioning

ScaleObs is designed as a **top-level summary portal** rather than a replacement for any existing tool:

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

Each node runs a tiny agent (<20MB) to read host & container metrics in read-only mode. Integrated with Headscale, newly joined nodes are auto-discovered and appear automatically. Single-page dashboard displays overall node health, container status and AI workload tags with highlighted exceptions. One-click jump to Grafana for trends, Portainer for container ops, or Headscale for network config — solving observability fragmentation with minimal overhead.

### Problems other tools don't solve

| Problem | Existing tools | ScaleObs |
|---------|---------------|----------|
| "What's the overall state of my 10 servers?" | Grafana needs Prometheus per host + manual dashboards | Install one agent and it appears |
| "Which nodes are running AI coding agents?" | No tool can answer this | Purple badges at a glance |
| "I added a Mac Mini to Tailscale, I want to see it" | Manually add targets in Grafana, endpoints in Portainer | Auto-syncs from Headscale, appears instantly |
| "I want to see all services on one page" | 5 browser tabs open | One page, all covered |
| "My cheap 2C4G VPS can barely run, what monitoring fits?" | Prometheus + Grafana eat 1GB RAM | Agent is < 20MB |

### When ScaleObs truly shines

- You have **3+ machines** scattered across home, cloud, and office
- You use **Tailscale** / Headscale for networking
- You run **AI coding agents** (OpenCode / Codex / Claude Code)
- You want to know **"is everything OK right now"** — not query historical trends

## Core Features

- **Ultra-light agent** — Single binary, ~20MB memory footprint, zero dependencies. Perfect for edge nodes
- **Auto node discovery** — Nodes join Tailscale → appear on dashboard automatically, no manual setup
- **Host & container monitoring** — Real-time CPU, memory, disk, network; read-only Docker container status
- **AI workload identification** — Exclusive badges for OpenCode, Claude Code, and other coding agents
- **Global exception highlighting** — Spot crashes, OOM, and frequent restarts at a glance
- **One-click jump links** — Grafana for trends, Portainer for container ops, Headscale for node config
- **Simple YAML configuration** — Single config file handles everything

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
