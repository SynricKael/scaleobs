# Server Ops Portal (SOP) — 架构设计文档

> 基于网络的服务器监控平台，统一聚合第三方 Web 管理界面。
> 2026-06-17

---

## 1. 概述

### 1.1 目标

构建一个**统一运维入口平台**，解决"管理多台服务器的多个 Web 面板需要记大量 IP:端口、反复登录"的问题。

### 1.2 设计原则

- **Docker 优先** — 所有服务组件优先容器化部署
- **声明式集成** — 新增第三方面板只需改 YAML 配置，不改代码
- **统一认证** — 用户登录一次 Gateway，即可访问所有已授权的面板
- **C/S 架构** — Server 端提供 Web 服务，Client 端（Windows 桌面）驻留系统托盘
- **分阶段交付** — Phase 1 聚焦面板聚合 + 基础监控，后续扩展告警和自动化

---

## 2. 整体架构

```
┌─────────────────────────────────────────────┐
│           用户桌面 (Tauri Client)             │
│  ┌──────────┐    ┌──────────────────────┐   │
│  │ 系统托盘  │    │  WebView 窗口        │   │
│  │ - 状态图标│    │  - Dashboard 磁贴    │   │
│  │ - 右键菜单│    │  - iframe 内嵌面板   │   │
│  │ - 通知    │    │  - 独立窗口拖拽      │   │
│  └──────────┘    └──────────┬───────────┘   │
└──────────────────────────────┼───────────────┘
                               │ HTTPS
┌──────────────────────────────▼───────────────┐
│         Nginx (TLS 终止 + 静态资源)           │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│          Gateway (Go) — 路由中枢              │
│                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │ Auth     │ │ Proxy    │ │ REST API     │  │
│  │ (JWT)    │ │ (httputil│ │ (服务器状态)  │  │
│  │          │ │ Reverse  │ │              │  │
│  │          │ │ Proxy)   │ │              │  │
│  └──────────┘ └──────────┘ └──────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │ WebSocket│ │ Plugin   │ │ Config       │  │
│  │ (Agent   │ │ Registry │ │ Loader       │  │
│  │ 通信)    │ │          │ │ (YAML)       │  │
│  └──────────┘ └──────────┘ └──────────────┘  │
├──────────────────────────────────────────────┤
│  Dashboard (Vue 3 SPA) — 静态文件由 Nginx 托管│
└──────────────────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
┌────────▼──┐  ┌───────▼───┐  ┌────▼────────┐
│ frps      │  │ Portainer  │  │ Netdata     │
│ Dashboard │  │            │  │ / Grafana   │
│ (7500)    │  │ (9000)     │  │ 等第三方服务  │
└───────────┘  └───────────┘  └─────────────┘
         │             │             │
┌────────▼─────────────▼─────────────▼────────┐
│            Agent (Go, 每台服务器一个)          │
│  CPU / 内存 / 磁盘 / Docker / 进程           │
│  WebSocket → Gateway (推指标)                │
└──────────────────────────────────────────────┘
```

---

## 3. 组件详述

### 3.1 Gateway (Go)

**位置**：`gateway/`

**职责**：
- HTTP 反向代理：`/p/<plugin-id>/*` → 后端服务
- JWT 认证：所有 API 和代理路径统一鉴权
- REST API：`/api/*` 提供 Dashboard 数据
- Agent 通信：WebSocket 接收 Agent 指标上报
- 健康检查：定时检查各服务可用性

**路由设计**：

| 路径 | 功能 | 认证 |
|------|------|------|
| `GET /api/servers` | 服务器列表 + 实时指标 | 需要 |
| `GET /api/services` | 插件服务列表 + 状态 | 需要 |
| `POST /api/auth/login` | 登录获取 JWT | 不需要 |
| `GET /api/ws/agent` | Agent WebSocket 接入点 | Agent Token |
| `/p/<id>/*` | 反向代理到第三方面板 | 需要 |
| `/dashboard/*` | SPA 静态文件 | 需要 |

**关键代码结构**：

```
gateway/
├── main.go
├── config/
│   └── config.go         # YAML 配置加载
├── model/
│   ├── server.go          # 服务器模型
│   ├── service.go         # 插件服务模型
│   └── metrics.go         # 指标数据结构
├── proxy/
│   └── proxy.go           # ReverseProxy 实现
│       ├── rewrite_url()        # 路径重写
│       ├── inject_auth()        # 注入认证信息
│       └── rewrite_html_links() # 修正 HTML 中的绝对路径
├── auth/
│   └── jwt.go             # JWT 签发/验证
├── api/
│   ├── server_handler.go  # 服务器相关 API
│   ├── service_handler.go # 服务相关 API
│   └── auth_handler.go    # 登录 API
├── monitor/
│   ├── agent_hub.go       # Agent 连接管理 (WebSocket)
│   └── health_check.go    # 服务健康检查
└── plugin/
    └── registry.go        # 插件注册表加载
```

### 3.2 插件注册表 (services.yml)

声明式配置，所有第三方面板在这里注册：

```yaml
gateway:
  title: "服务器运维平台"
  port: 8080
  auth:
    enabled: true
    jwt_secret: "${JWT_SECRET}"
    users:
      - username: admin
        password_hash: "${ADMIN_PASSWORD_HASH}"

servers:
  - id: main-server
    name: "主服务器"
    host: 192.168.1.100
    agent_port: 9090
    agent_token: "${AGENT_TOKEN}"
    tags: [prod, main]
    location: "北京"

services:
  - id: frps
    name: "FRP 服务端"
    icon: frps.svg
    url: /p/frps/
    target: http://frps:7500
    category: 网络
    open_method: iframe     # iframe 内嵌 或 browser 打开
    health_check:
      type: http
      path: /p/frps/api/status
      interval: 30s
    auth:
      type: inject_header
      header_name: "Cookie"
      header_value: "token=${FRPS_TOKEN}"

  - id: portainer
    name: "容器管理"
    icon: portainer.svg
    url: /p/portainer/
    target: http://portainer:9000
    category: 容器
    open_method: iframe
    health_check:
      type: tcp
      port: 9000
```

### 3.3 Dashboard 前端 (Vue 3)

**位置**：`dashboard/`

**页面设计**：

| 页面 | 说明 |
|------|------|
| `/` | 磁贴概览 — 所有注册服务按分类显示，带状态指示灯 |
| `/server/:id` | 服务器详情 — 实时 CPU/内存/磁盘 图表，运行的容器 |
| `/panel/:id` | 内嵌面板页 — iframe 嵌入第三方 Web 界面 |
| `/settings` | 设置 — 连接配置、主题切换 |

**磁贴布局**：

```
┌─────────────────────────────────────────────┐
│  🔍 搜索服务...        [主题切换] [设置]     │
├─────────────────────────────────────────────┤
│  📡 网络                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ FRP 服务端│ │ NPM      │ │ Cloudfl  │     │
│  │ 🟢 在线   │ │ 🟡 告警  │ │ 🔴 离线  │     │
│  └──────────┘ └──────────┘ └──────────┘     │
│                                              │
│  🐳 容器                                       │
│  ┌──────────┐ ┌──────────┐                   │
│  │ Portainer│ │ Adminer  │                   │
│  │ 🟢 在线   │ │ 🟢 在线   │                   │
│  └──────────┘ └──────────┘                   │
│                                              │
│  🖥️ 服务器状态                                 │
│  ┌────────────────────────────────────────┐  │
│  │ 主服务器            CPU ████████░░ 76% │  │
│  │ 192.168.1.100       MEM ██████░░░░ 58% │  │
│  │ 🟢 在线 2min        DSK ███████░░ 65% │  │
│  └────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────┐  │
│  │ 从服务器            CPU ██░░░░░░░░ 22% │  │
│  │ 192.168.1.101       MEM █████░░░░░ 45% │  │
│  │ 🟢 在线 5min        DSK ███░░░░░░░ 32% │  │
│  └────────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

**技术栈**：Vue 3 + TypeScript + Pinia + Tailwind CSS + Vite

### 3.4 Agent (Go)

**位置**：`agent/`

**职责**：
1. 每分钟采集服务器指标
2. WebSocket 连接 Gateway 实时上报
3. 接收 Gateway 下发的远程执行指令

**采集指标**：

| 指标 | 来源 | 采集间隔 |
|------|------|---------|
| CPU 使用率 | `/proc/stat` (Linux) | 10s |
| 内存 | `/proc/meminfo` | 10s |
| 磁盘 | `df` | 30s |
| 网络 IO | `/proc/net/dev` | 30s |
| 系统负载 | `/proc/loadavg` | 10s |
| 进程数 | `/proc` 计数 | 60s |
| Docker 容器 | Docker socket (可选) | 30s |
| 主机信息 | `uname`, `/etc/os-release` | 启动时 + 每 1h |

**通信协议**：

```
Agent → Gateway (WebSocket, JSON):
{
  "type": "metrics",
  "server_id": "main-server",
  "timestamp": "2026-06-17T14:30:00Z",
  "data": {
    "cpu_percent": 45.2,
    "memory": { "total_mb": 32768, "used_mb": 18432, "percent": 56.2 },
    "disk": [
      { "mount": "/", "total_gb": 512, "used_gb": 280, "percent": 54.7 }
    ],
    "uptime_seconds": 1209600,
    "docker_containers": { "total": 8, "running": 6, "stopped": 2 }
  }
}

Gateway → Agent (下发指令):
{
  "type": "exec",
  "id": "cmd-001",
  "command": "df -h",
  "timeout": 30
}

Agent → Gateway (执行结果):
{
  "type": "exec_result",
  "id": "cmd-001",
  "exit_code": 0,
  "stdout": "...",
  "stderr": "",
  "duration_ms": 120
}
```

### 3.5 桌面客户端 (Tauri)

**位置**：`tauri-client/`

**架构**：

```
tauri-client/
├── src-tauri/
│   ├── src/
│   │   ├── main.rs           # Tauri 入口
│   │   ├── tray.rs           # 系统托盘实现
│   │   ├── notify.rs         # Windows Toast 通知
│   │   └── settings.rs       # 本地配置存储
│   ├── icons/                # 托盘图标
│   └── Cargo.toml
├── src/
│   ├── App.vue
│   ├── main.ts
│   ├── views/
│   │   ├── Dashboard.vue     # 主面板（复用 Web Dashboard 组件）
│   │   ├── WebPanel.vue      # WebView 内嵌容器
│   │   └── Settings.vue      # 客户端设置
│   ├── composables/
│   │   └── useGateway.ts     # Gateway 连接管理
│   └── components/
│       ├── TileGrid.vue
│       ├── ServerCard.vue
│       └── StatusBadge.vue
├── tauri.conf.json
└── package.json
```

**托盘交互**：

| 操作 | 行为 |
|------|------|
| 启动 | 自动最小化到托盘，后台连接 Gateway |
| 左键单击图标 | 显示/隐藏主窗口 |
| 右键菜单 | 快速入口列表、设置、退出 |
| 托盘图标颜色 | 🟢 全部正常 / 🟡 有警告 / 🔴 有宕机 / ⚪ 连接中 |
| 服务器宕机 | Windows Toast 通知 + 图标变红 |

---

## 4. 数据流

### 4.1 用户访问面板

```
用户点击 Dashboard 磁贴
  → Tauri WebView 导航到 /panel/frps
  → Gateway 验证 JWT
  → Gateway 代理请求到 http://frps:7500
  → frps 响应 HTML
  → Gateway 改写 HTML 中资源路径 (/static/ → /p/frps/static/)
  → 返回给客户端渲染
```

### 4.2 Agent 指标上报

```
Agent 启动 → WebSocket 连接 Gateway (/api/ws/agent)
  → 认证 (Agent Token)
  → 每 10s 推送 metrics JSON
  → Gateway 更新内存中的服务器状态
  → Dashboard 轮询 /api/servers 获取最新状态
```

### 4.3 健康检查

```
Gateway 定时器 (每 30s)
  → 遍历所有注册服务
  → 对每个服务执行 health_check（HTTP GET 或 TCP 连接）
  → 更新服务状态（online / degraded / offline）
  → 状态变化时推送到 Dashboard WebSocket
```

---

## 5. 部署架构 (Docker Compose)

```yaml
version: '3.8'
services:
  gateway:
    build: ./gateway
    ports:
      - "8080:8080"
    volumes:
      - ./config/services.yml:/app/config.yml:ro
      - gateway_data:/data
    environment:
      JWT_SECRET: "${JWT_SECRET}"
    depends_on:
      - frps
    restart: unless-stopped

  dashboard:
    build: ./dashboard
    # 由 Nginx 托管，或直接由 Gateway 提供

  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./dashboard/dist:/var/www/dashboard:ro
    depends_on:
      - gateway
    restart: unless-stopped

  frps:
    image: fatedier/frps
    volumes:
      - ./config/frps.toml:/etc/frp/frps.toml:ro
    restart: unless-stopped

  # 可选集成
  portainer:
    image: portainer/portainer-ce
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer_data:/data
    restart: unless-stopped

volumes:
  gateway_data:
  portainer_data:
```

---

## 6. 分阶段规划

### Phase 1 — 面板聚合 + 基础监控（MVP）

目标：核心功能跑通，至少能接入 frps + 1-2 个第三方面板

- [ ] Gateway 基础框架（路由、配置加载、反向代理）
- [ ] 插件注册表 (services.yml) 解析
- [ ] JWT 统一认证
- [ ] Dashboard 磁贴概览页面 (Vue 3)
- [ ] iframe 面板嵌入 + 路径改写
- [ ] Agent 基础：CPU/内存/磁盘采集 + WebSocket 上报
- [ ] Docker Compose 编排
- [ ] 健康检查（服务在线/离线）

### Phase 2 — 桌面客户端

- [ ] Tauri 项目初始化 + 系统托盘
- [ ] 内嵌 WebView 窗口
- [ ] 托盘状态图标（绿/黄/红）
- [ ] 右键菜单快速入口
- [ ] 宕机通知

### Phase 3 — 监控增强

- [ ] 历史指标存储（SQLite/PostgreSQL）
- [ ] 时序图表（CPU/内存历史曲线）
- [ ] 告警规则引擎
- [ ] 告警通知（飞书/邮件）

### Phase 4 — 自动化运维

- [ ] Agent 远程命令执行
- [ ] 脚本管理 + 定时任务
- [ ] 文件管理（上传/下载/编辑）
- [ ] 日志查看

---

## 7. 目录结构（完整项目）

```
observer/
├── README.md
├── docker-compose.yml
├── .env.example
├── gateway/               # Go Gateway 服务
│   ├── main.go
│   ├── config/
│   ├── proxy/
│   ├── auth/
│   ├── api/
│   ├── monitor/
│   └── plugin/
├── dashboard/             # Vue 3 前端
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── vite.config.ts
├── agent/                 # Go Agent
│   ├── main.go
│   ├── collector/
│   ├── reporter/
│   └── executor/
├── tauri-client/          # Tauri 桌面端
│   ├── src-tauri/
│   └── src/
├── config/
│   ├── services.yml       # 插件注册表
│   └── frps.toml          # frp 配置示例
├── nginx/
│   └── nginx.conf
├── docs/
│   ├── plans/
│   └── manual/
└── scripts/
    └── setup.sh
```

---

## 8. 技术选型理由

| 选择 | 理由 |
|------|------|
| **Go** (Gateway/Agent) | 单二进制部署、高性能反向代理、并发模型优秀、跨平台编译 |
| **Vue 3 + TypeScript** | 组件化开发、生态丰富，中文社区支持好 |
| **Tauri** | 比 Electron 轻 10 倍（~5MB vs ~150MB），原生托盘支持，Rust 性能 |
| **Tailwind CSS** | 快速构建 UI，不写自定义 CSS，保持一致性 |
| **SQLite** | Phase 1 无需独立数据库，文件级存储，零运维 |
| **YAML** | 人类可读的声明式配置，Ops 人员友好 |
| **WebSocket** | 实时推送 Agent 指标，比 HTTP 轮询高效 |

---

## 9. 开放问题

- [ ] Gateway 对第三方 HTML 的路径改写（URL rewriting）如何做得健壮？需要处理 JS 动态生成的路径吗？
- [ ] 统一认证对于需要独立登录的第三方服务（如某些不支持免密的公版软件），如何通过代理层处理？
- [ ] WebView 中 iframe 嵌入某些服务（如 Grafana）可能有 X-Frame-Options 限制，是否需要所有服务通过代理转发来规避？
- [ ] 是否需要考虑多用户/权限分离（不同用户只能看到授权给自己的面板）？
