# Phase 1 — 面板聚合 + 基础监控 实施计划

> **Goal:** 搭建 Server Ops Portal 核心骨架：Gateway 反向代理 + 统一认证 + 磁贴 Dashboard + Agent 基础采集

**Architecture:** Go Gateway 作为路由中枢（反向代理 + JWT 认证 + REST API）+ Vue 3 前端磁贴面板 + Go Agent 指标采集 + Docker Compose 编排

**Tech Stack:** Go (Gateway/Agent), Vue 3 + TypeScript + Tailwind + Vite (Dashboard), YAML (配置), Docker Compose

---

## 项目目录结构

```
E:\ProgramOps\Observer\
├── docker-compose.yml
├── .env.example
├── config/
│   └── services.yml
├── gateway/
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── config/
│   │   └── config.go
│   ├── model/
│   │   ├── server.go
│   │   ├── service.go
│   │   └── metrics.go
│   ├── proxy/
│   │   └── proxy.go
│   ├── auth/
│   │   └── jwt.go
│   ├── api/
│   │   ├── server_handler.go
│   │   ├── service_handler.go
│   │   ├── auth_handler.go
│   │   └── router.go
│   ├── monitor/
│   │   ├── agent_hub.go
│   │   └── health_check.go
│   └── plugin/
│       └── registry.go
├── dashboard/
│   ├── Dockerfile
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   ├── index.html
│   └── src/
│       ├── main.ts
│       ├── App.vue
│       ├── router/
│       │   └── index.ts
│       ├── stores/
│       │   ├── auth.ts
│       │   └── gateway.ts
│       ├── api/
│       │   └── index.ts
│       ├── types/
│       │   └── index.ts
│       ├── views/
│       │   ├── Login.vue
│       │   ├── Dashboard.vue
│       │   └── PanelView.vue
│       └── components/
│           ├── TileGrid.vue
│           ├── ServerCard.vue
│           └── StatusBadge.vue
├── agent/
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── collector/
│   │   ├── cpu.go
│   │   ├── memory.go
│   │   └── disk.go
│   ├── reporter/
│   │   └── ws.go
│   └── model/
│       └── metrics.go
└── nginx/
    └── nginx.conf
```

---

### Task 1: 项目骨架 + Docker Compose + 配置文件

**Files:**
- Create: `docker-compose.yml`
- Create: `.env.example`
- Create: `config/services.yml`
- Create: `nginx/nginx.conf`
- Create: `gateway/Dockerfile`
- Create: `dashboard/Dockerfile`
- Create: `agent/Dockerfile`

**Steps:**

**Step 1:** Create `.env.example`
```bash
JWT_SECRET=change-me-to-a-random-string
ADMIN_PASSWORD_HASH=<bcrypt hash of admin password>
FRPS_TOKEN=change-me
```

**Step 2:** Create `config/services.yml` with sample services (frps, portainer, netdata)

**Step 3:** Create `nginx/nginx.conf` — reverse proxy to gateway, serve dashboard static files

**Step 4:** Create `docker-compose.yml` — gateway + dashboard + nginx + frps services

**Step 5:** Create stub Dockerfiles for gateway/dashboard/agent

---

### Task 2: Gateway — 配置加载 + 插件注册表

**Files:**
- Create: `gateway/go.mod`
- Create: `gateway/main.go`
- Create: `gateway/config/config.go`
- Create: `gateway/model/server.go`
- Create: `gateway/model/service.go`
- Create: `gateway/model/metrics.go`
- Create: `gateway/plugin/registry.go`

**Step 1:** Initialize Go module
```bash
cd gateway
go mod init github.com/glrs/observer/gateway
```

**Step 2:** `model/` — define data structures:
- `Server` struct: ID, Name, Host, AgentPort, AgentToken, Tags
- `Service` struct: ID, Name, Icon, URLPath, TargetURL, Category, OpenMethod, HealthCheck
- `Metrics` struct: CPUPercent, Memory, Disk, Uptime, DockerContainers

**Step 3:** `config/config.go` — load and validate services.yml:
- Use `gopkg.in/yaml.v3`
- `GatewayConfig` struct with Gateway, Servers, Services sections
- Validate required fields, set defaults

**Step 4:** `plugin/registry.go`:
- `Registry` struct holds loaded services
- `GetService(id)`, `ListServices()`, `ListByCategory()` methods
- Reload on SIGHUP support (optional)

**Step 5:** `main.go` — minimal:
- Load config from env var `CONFIG_PATH` (default: `/app/config.yml`)
- Print parsed config for verification
- HTTP server skeleton on `:8080`

---

### Task 3: Gateway — 反向代理 + 路径改写

**Files:**
- Create: `gateway/proxy/proxy.go`

**Step 1:** Implement `ServiceProxy` struct:
```go
type ServiceProxy struct {
    Service  *model.Service
    Proxy    *httputil.ReverseProxy
    BasePath string
}
```

**Step 2:** Core reverse proxy with path rewriting:
- Parse URL: `/p/<service-id>/<rest...>`
- Rewrite request URL path: strip `/p/<service-id>` prefix
- Set `X-Forwarded-*` headers
- Use `httputil.ReverseProxy` with custom `Director` and `ModifyResponse`

**Step 3:** HTML response rewriting in `ModifyResponse`:
- If `Content-Type` starts with `text/html`:
- Read response body
- Replace all `/static/...` → `/p/<service-id>/static/...`
- Replace all `/api/...` → `/p/<service-id>/api/...`
- Also rewrite relative paths like `./` and `../`
- Write modified body back

**Step 4:** Register proxy routes in `api/router.go`:
- For each registered service, create a route: `/p/{id}/` → ServiceProxy
- Catch-all for unmatched services → 404

---

### Task 4: Gateway — JWT 认证 + 登录 API

**Files:**
- Create: `gateway/auth/jwt.go`
- Create: `gateway/api/auth_handler.go`
- Modify: `gateway/api/router.go`

**Step 1:** JWT implementation:
```go
func GenerateToken(username string, secret string) (string, error)
func ValidateToken(tokenString string, secret string) (Claims, error)
```
- Use `golang-jwt/jwt/v5`
- Claims: username, exp (24h), iat
- Store bcrypt hashed password in config

**Step 2:** Login handler:
```
POST /api/auth/login
Body: { "username": "admin", "password": "..." }
Response: { "token": "...", "expires_at": "..." }
```
- Validate password against config's bcrypt hash
- Return JWT token

**Step 3:** Auth middleware:
```go
func AuthMiddleware(secret string) func(http.Handler) http.Handler
```
- Check `Authorization: Bearer <token>` header
- Skip auth for `/api/auth/login` and `/dashboard/*` (static files)
- Set user claim in request context

---

### Task 5: Gateway — 健康检查 + Dashboard REST API

**Files:**
- Create: `gateway/monitor/health_check.go`
- Create: `gateway/api/server_handler.go`
- Create: `gateway/api/service_handler.go`
- Create: `gateway/api/router.go`

**Step 1:** Health checker:
```go
type HealthChecker struct {
    services map[string]*ServiceStatus  // service_id → status
    mu       sync.RWMutex
}

func (hc *HealthChecker) Start(ctx context.Context, interval time.Duration)
```
- For each service, periodically check via HTTP GET or TCP dial
- Update status: online / degraded / offline
- Report in REST API responses

**Step 2:** Service API handler:
```
GET /api/services
Response: [{ id, name, icon, category, url, status, open_method }]
```

**Step 3:** Server API handler:
```
GET /api/servers
Response: [{ id, name, host, metrics: { cpu, mem, disk }, online }]
```
- Returns data from latest Agent report (initially empty/stub until Agent connects)

**Step 4:** Wire it all in `router.go`:
```go
mux := http.NewServeMux()
// Auth routes
mux.HandleFunc("POST /api/auth/login", authHandler.Login)
// Protected routes (with middleware)
mux.Handle("GET /api/services", authMiddleware(hc.ServicesHandler))
mux.Handle("GET /api/servers", authMiddleware(hc.ServersHandler))
// Proxy
mux.Handle("/p/{id}/", authMiddleware(proxyHandler))
// Agent WebSocket
mux.HandleFunc("/api/ws/agent", agentHub.HandleWS)
```

---

### Task 6: Dashboard — Vue 3 项目 + 登录页

**Files:**
- Create: `dashboard/package.json`
- Create: `dashboard/vite.config.ts`
- Create: `dashboard/tsconfig.json`
- Create: `dashboard/tailwind.config.js`
- Create: `dashboard/postcss.config.js`
- Create: `dashboard/index.html`
- Create: `dashboard/src/main.ts`
- Create: `dashboard/src/App.vue`
- Create: `dashboard/src/router/index.ts`
- Create: `dashboard/src/stores/auth.ts`
- Create: `dashboard/src/api/index.ts`
- Create: `dashboard/src/types/index.ts`
- Create: `dashboard/src/views/Login.vue`

**Step 1:** Initialize Vue 3 project:
- Dependencies: vue, vue-router, pinia, axios, @heroicons/vue
- Dev deps: typescript, vite, tailwindcss, postcss, autoprefixer, @vitejs/plugin-vue

**Step 2:** Types (`types/index.ts`):
```typescript
interface Service {
  id: string; name: string; icon: string;
  category: string; url: string; status: 'online' | 'degraded' | 'offline';
  open_method: 'iframe' | 'browser';
}
interface ServerStatus {
  id: string; name: string; host: string;
  online: boolean; metrics: { cpu: number; mem: number; disk: number };
}
```

**Step 3:** API module:
```typescript
// api/index.ts
const BASE = '/api'
export const api = {
  login(username: string, password: string): Promise<{token: string}>
  getServices(): Promise<Service[]>
  getServers(): Promise<ServerStatus[]>
}
```

**Step 4:** Auth store (Pinia):
- `login()`, `logout()`, `token` state
- Persist token in localStorage
- Axios interceptor auto-attach Bearer token

**Step 5:** Login page:
- Centered card layout
- Username + password fields
- Error message display
- On success: redirect to `/`

---

### Task 7: Dashboard — 磁贴概览 + 服务器状态

**Files:**
- Create: `dashboard/src/views/Dashboard.vue`
- Create: `dashboard/src/components/TileGrid.vue`
- Create: `dashboard/src/components/ServerCard.vue`
- Create: `dashboard/src/components/StatusBadge.vue`
- Create: `dashboard/src/stores/gateway.ts`

**Step 1:** Gateway store (Pinia):
- Fetch services and servers from API
- Auto-refresh every 10s
- Group services by category

**Step 2:** `StatusBadge.vue` — simple colored dot:
- online → 🟢 green
- degraded → 🟡 yellow
- offline → 🔴 red

**Step 3:** `TileGrid.vue`:
- Receives services grouped by category
- Renders category sections with headers
- Each tile: icon + name + StatusBadge
- Click → navigate to `/panel/:id`
- Responsive grid: 4 cols → 2 cols → 1 col

**Step 4:** `ServerCard.vue`:
- Server name, IP, status indicator
- Mini progress bars for CPU/memory/disk
- Last heartbeat timestamp

**Step 5:** `Dashboard.vue`:
- Top bar: title + search input (filter services)
- Server cards row
- Tile grid by category
- Auto-refresh
- Responsive layout

---

### Task 8: Dashboard — iframe 面板视图

**Files:**
- Create: `dashboard/src/views/PanelView.vue`

**Step 1:** PanelView page:
- Route: `/panel/:serviceId`
- Get service details from store by ID
- Full-screen iframe pointing to `{gateway_base}/p/{serviceId}/`
- iframe styling: 100% width/height, no border
- Header bar: service name, back button, open-in-browser button
- Loading spinner while iframe loads

**Step 2:** Error handling:
- iframe `onError` → show error state
- Service not found → 404 page
- Handle `X-Frame-Options` denial gracefully with message

---

### Task 9: Agent — 指标采集

**Files:**
- Create: `agent/go.mod`
- Create: `agent/main.go`
- Create: `agent/model/metrics.go`
- Create: `agent/collector/cpu.go`
- Create: `agent/collector/memory.go`
- Create: `agent/collector/disk.go`

**Step 1:** Initialize Go module:
```bash
cd agent
go mod init github.com/glrs/observer/agent
```

**Step 2:** Metrics model:
```go
type Metrics struct {
    CPUPercent  float64   `json:"cpu_percent"`
    Memory      MemoryInfo `json:"memory"`
    Disks       []DiskInfo `json:"disk"`
    UptimeSec   int64     `json:"uptime_seconds"`
    Timestamp   time.Time `json:"timestamp"`
}
type MemoryInfo struct {
    TotalMB  int64   `json:"total_mb"`
    UsedMB   int64   `json:"used_mb"`
    Percent  float64 `json:"percent"`
}
type DiskInfo struct {
    Mount   string  `json:"mount"`
    TotalGB int64   `json:"total_gb"`
    UsedGB  int64   `json:"used_gb"`
    Percent float64 `json:"percent"`
}
```

**Step 3:** CPU collector:
- Read `/proc/stat` on Linux
- Calculate delta-based CPU percentage
- Cache previous values for delta calculation
- Fallback: use `github.com/shirou/gopsutil/cpu` as dependency

**Step 4:** Memory collector:
- Read `/proc/meminfo`
- Parse MemTotal, MemAvailable, MemFree
- Calculate used = total - available - buffers/cache

**Step 5:** Disk collector:
- Call `df -B1 --output=target,size,used,avail` on Linux
- Parse output
- Return array of mount points (skip pseudo-fs like tmpfs, devtmpfs)

---

### Task 10: Agent — WebSocket 上报

**Files:**
- Create: `agent/reporter/ws.go`
- Modify: `agent/main.go`

**Step 1:** WebSocket reporter:
```go
func Connect(gatewayURL, serverID, token string) (*websocket.Conn, error)
func ReportMetrics(conn *websocket.Conn, metrics *Metrics) error
```
- Use `github.com/gorilla/websocket`
- Connect to `ws://gateway:8080/api/ws/agent`
- Auth message: `{ "type": "auth", "server_id": "...", "token": "..." }`
- Metrics message: `{ "type": "metrics", "server_id": "...", "timestamp": "...", "data": {...} }`
- Auto-reconnect on disconnect (exponential backoff)

**Step 2:** Main loop:
```
Parse args: --gateway, --server-id, --token, --interval
Connect to Gateway via WebSocket
Loop every <interval> (default 10s):
    Collect metrics
    Send via WebSocket
    Sleep
Handle SIGTERM/SIGINT gracefully
```

---

## 验收标准 (Acceptance Criteria)

- [ ] `docker-compose up` 启动所有服务，Nginx 监听 80/443 端口
- [ ] 访问 `http://localhost/dashboard/` 看到登录页面
- [ ] 使用配置中的管理员凭据登录成功，获得 JWT Token
- [ ] 登录后看到磁贴概览页，frps 和 Portainer 显示在对应分类下
- [ ] 点击 frps 磁贴，通过 iframe 嵌入 frps Dashboard 页面
- [ ] 服务器状态卡片展示（最初可能为空，准备就绪）
- [ ] Agent 可以连接 Gateway WebSocket 并上报指标
- [ ] Gateway 健康检查标记在线/离线服务
- [ ] 所有 Go 代码通过 `go vet` 和 `golangci-lint`
- [ ] 所有 Docker 镜像可构建

---

## 测试计划

**Gateway:**
- Unit test: config loading from valid/invalid YAML
- Unit test: JWT token generation and validation
- Unit test: proxy path rewriting logic
- Integration: `go test ./...` with test server

**Dashboard:**
- TypeScript compilation: `tsc --noEmit`
- Vite build: `npm run build` succeeds

**Agent:**
- Unit test: metrics collection (mock /proc data)
- Integration: WebSocket connect to test server
