<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useGatewayStore } from '@/stores/gateway'
import { useAuthStore } from '@/stores/auth'
import {
  MonitorIcon,
  LogOutIcon,
  RefreshCwIcon,
  Grid3X3Icon,
  UserIcon,
  SettingsIcon,
  GlobeIcon,
  ServerIcon,
  ContainerIcon,
  ExternalLinkIcon,
  SunIcon,
  MoonIcon,
  StarIcon,
  PinIcon,
  BoxIcon,
  LayoutDashboardIcon,
  NetworkIcon,
  FolderOpenIcon,
  ChevronRightIcon,
  ChevronDownIcon,
} from '@lucide/vue'
import type { Service, ServerStatus, DockerContainer } from '@/types'

const router = useRouter()
const auth = useAuthStore()
const gw = useGatewayStore()

// Pinning: store pinned identifiers in localStorage
const PINNED_KEY = 'pinned-apps'

interface PinnedItem {
  id: string          // "service:{id}" or "docker:{serverId}:{containerId}"
  type: 'service' | 'docker'
  label: string
  subtitle: string
  online: boolean
  icon: string
}

const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function getPinnedIds(): Set<string> {
  try {
    const raw = localStorage.getItem(PINNED_KEY)
    return new Set(raw ? JSON.parse(raw) : [])
  } catch {
    return new Set()
  }
}

function setPinnedIds(ids: Set<string>) {
  localStorage.setItem(PINNED_KEY, JSON.stringify([...ids]))
}

function isPinned(id: string): boolean {
  return getPinnedIds().has(id)
}

function togglePin(id: string) {
  const pinned = getPinnedIds()
  if (pinned.has(id)) {
    pinned.delete(id)
  } else {
    pinned.add(id)
  }
  setPinnedIds(pinned)
}

// All services with status
const allServices = computed(() => gw.services)

// Docker containers from all servers
interface DockerApp {
  serverId: string
  serverName: string
  container: DockerContainer
  host: string
}

const allDockerContainers = computed<DockerApp[]>(() => {
  const result: DockerApp[] = []
  for (const s of gw.servers) {
    if (s.metrics?.docker_containers) {
      for (const c of s.metrics.docker_containers) {
        result.push({
          serverId: s.id,
          serverName: s.name || s.host || s.id,
          container: c,
          host: s.host || '',
        })
      }
    }
  }
  return result
})

// ──────────── Network → Application grouping ────────────

// Infer application name from container labels or name
function inferApp(container: DockerContainer): string {
  if (container.labels) {
    const project = container.labels['com.docker.compose.project']
    if (project) return project
  }
  // Fallback: use first segment of name before '-'
  const name = container.name || ''
  const idx = name.indexOf('-')
  if (idx > 0) return name.substring(0, idx)
  return 'standalone'
}

interface NetworkGroup {
  name: string           // Docker network name
  apps: Record<string, DockerApp[]>  // appName → containers
}

interface ServerDockerGroup {
  serverId: string
  serverName: string
  host: string
  networkName: string    // Tailscale/Headscale network name
  networks: NetworkGroup[]
  collapsed: boolean     // UI state: server-level collapse
}

const serverDockerGroups = computed<ServerDockerGroup[]>(() => {
  const serverMap = new Map<string, {
    serverId: string
    serverName: string
    host: string
    networkName: string
    netMap: Map<string, Map<string, DockerApp[]>>
  }>()

  for (const da of allDockerContainers.value) {
    let server = serverMap.get(da.serverId)
    if (!server) {
      const srv = gw.servers.find(s => s.id === da.serverId)
      server = {
        serverId: da.serverId,
        serverName: da.serverName,
        host: da.host,
        networkName: srv?.network_name || '',
        netMap: new Map(),
      }
      serverMap.set(da.serverId, server)
    }

    // Determine networks for this container
    const nets = da.container.networks && da.container.networks.length > 0
      ? da.container.networks
      : ['bridge']   // fallback for containers without explicit network info

    // Determine application
    const app = inferApp(da.container)

    for (const net of nets) {
      if (!server.netMap.has(net)) {
        server.netMap.set(net, new Map())
      }
      const appMap = server.netMap.get(net)!
      if (!appMap.has(app)) {
        appMap.set(app, [])
      }
      appMap.get(app)!.push(da)
    }
  }

  return Array.from(serverMap.values()).map(s => {
    const networks: NetworkGroup[] = []
    for (const [netName, appMap] of s.netMap) {
      const apps: Record<string, DockerApp[]> = {}
      for (const [appName, containers] of appMap) {
        apps[appName] = containers
      }
      networks.push({ name: netName, apps })
    }
    // Sort networks: 'bridge' last, others alphabetically
    networks.sort((a, b) => {
      if (a.name === 'bridge') return 1
      if (b.name === 'bridge') return -1
      return a.name.localeCompare(b.name)
    })
    return {
      serverId: s.serverId,
      serverName: s.serverName,
      host: s.host,
      networkName: s.networkName,
      networks,
      collapsed: false,
    }
  }).sort((a, b) => a.serverName.localeCompare(b.serverName))
})

// Collapse/expand a server group
const collapsedServers = new Set<string>()
function toggleServerCollapse(serverId: string) {
  if (collapsedServers.has(serverId)) {
    collapsedServers.delete(serverId)
  } else {
    collapsedServers.add(serverId)
  }
}
function isServerCollapsed(serverId: string): boolean {
  return collapsedServers.has(serverId)
}

// Group containers by server (for the summary line)
const serversWithContainers = computed(() =>
  gw.servers.filter(s => s.metrics?.docker_containers && s.metrics.docker_containers.length > 0)
)

// Pinned items (computed live)
const pinnedItems = computed<PinnedItem[]>(() => {
  const ids = getPinnedIds()
  const items: PinnedItem[] = []

  for (const id of ids) {
    if (id.startsWith('service:')) {
      const svcId = id.slice('service:'.length)
      const svc = gw.services.find(s => s.id === svcId)
      if (!svc) continue
      items.push({
        id,
        type: 'service',
        label: svc.name,
        subtitle: svc.url,
        online: svc.status === 'online',
        icon: svc.icon || 'box',
      })
    } else if (id.startsWith('docker:')) {
      const parts = id.slice('docker:'.length).split(':')
      if (parts.length < 2) continue
      const serverId = parts[0]
      const containerId = parts.slice(1).join(':')
      const server = gw.servers.find(s => s.id === serverId)
      if (!server || !server.metrics?.docker_containers) continue
      const container = server.metrics.docker_containers.find(c => c.id.startsWith(containerId) || c.name === containerId)
      if (!container) continue
      items.push({
        id,
        type: 'docker',
        label: container.name || container.id.slice(0, 8),
        subtitle: `${server.name || server.host || serverId} · ${container.image}`,
        online: container.state === 'running',
        icon: 'container',
      })
    }
  }
  return items
})

function openURL(url: string) {
  window.open(url, '_blank')
}

function viewServer(id: string) {
  router.push(`/server/${id}`)
}

function serviceIcon(svc: Service): string {
  return svc.icon || 'box'
}

const iconComponents: Record<string, any> = {
  globe: GlobeIcon,
  container: ContainerIcon,
  server: ServerIcon,
  monitor: MonitorIcon,
  box: BoxIcon,
  grid: Grid3X3Icon,
}

function iconFor(name: string) {
  return iconComponents[name] || BoxIcon
}

onMounted(() => {
  if (!gw.servers.length && !gw.loading) gw.refresh()
})
</script>

<template>
  <div class="min-h-screen bg-gray-900 text-gray-100">
    <!-- Top navigation bar -->
    <header data-tauri-drag-region
      class="sticky top-0 z-10 bg-white/80 dark:bg-gray-800/90 backdrop-blur-sm border-b border-gray-200 dark:border-gray-700 select-none"
    >
      <div class="flex items-center justify-between h-12 px-4">
        <div class="flex items-center gap-3 pl-1">
          <MonitorIcon class="w-5 h-5 text-indigo-400" />
          <h1 class="text-base font-bold">ScaleObs</h1>
          <nav class="ml-6 flex items-center gap-1">
            <button
              @click="router.push('/')"
              class="px-3 py-1.5 text-sm rounded-lg transition-colors"
              :class="!$route.path.startsWith('/apps')
                ? 'bg-blue-600 text-white'
                : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700'"
            >
              <ServerIcon class="w-4 h-4 inline-block mr-1" />
              设备
            </button>
            <button
              @click="router.push('/apps')"
              class="px-3 py-1.5 text-sm rounded-lg transition-colors"
              :class="$route.path.startsWith('/apps')
                ? 'bg-blue-600 text-white'
                : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700'"
            >
              <Grid3X3Icon class="w-4 h-4 inline-block mr-1" />
              应用
            </button>
          </nav>
          <span class="ml-2 text-xs text-gray-500">
            {{ allServices.length }} 个服务 · {{ allDockerContainers.length }} 个容器
          </span>
        </div>

        <div class="flex items-center gap-3">
          <button
            @click="gw.refresh()"
            :disabled="gw.loading"
            class="text-gray-400 hover:text-gray-200 transition-colors disabled:opacity-50 p-1 rounded"
            title="刷新"
          >
            <RefreshCwIcon class="w-4 h-4" :class="{ 'animate-spin': gw.loading }" />
          </button>

          <button
            @click="toggleTheme"
            class="text-gray-400 hover:text-amber-400 dark:hover:text-amber-300 transition-colors p-1 rounded"
            :title="isDark ? '切换日间模式' : '切换夜间模式'"
          >
            <SunIcon v-if="isDark" class="w-4 h-4" />
            <MoonIcon v-else class="w-4 h-4" />
          </button>

          <button
            @click="router.push('/settings')"
            class="text-gray-400 hover:text-gray-200 transition-colors p-1 rounded"
            title="设置"
          >
            <SettingsIcon class="w-4 h-4" />
          </button>

          <span class="text-xs text-gray-500 flex items-center gap-1">
            <UserIcon class="w-3 h-3" />
            {{ auth.username }}
          </span>

          <button
            @click="auth.logout(); router.push('/login')"
            class="text-gray-400 hover:text-red-400 transition-colors p-1 rounded"
            title="退出"
          >
            <LogOutIcon class="w-4 h-4" />
          </button>
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-8">
      <!-- Error banner -->
      <div
        v-if="gw.error"
        class="bg-red-900/30 text-red-400 text-sm rounded-lg p-3 flex items-center gap-2"
      >
        <span>⚠</span> {{ gw.error }}
      </div>

      <!-- Loading -->
      <div v-if="gw.loading && gw.services.length === 0 && gw.servers.length === 0" class="text-center py-20 text-gray-500">
        <Grid3X3Icon class="w-10 h-10 mx-auto mb-3 text-gray-600 animate-pulse" />
        <p class="text-sm">加载中...</p>
      </div>

      <!-- ============ 置顶 ============ -->
      <section v-if="pinnedItems.length > 0">
        <div class="flex items-center gap-2 mb-4">
          <PinIcon class="w-5 h-5 text-amber-400" />
          <h2 class="text-base font-semibold">置顶</h2>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          <div
            v-for="item in pinnedItems"
            :key="item.id"
            class="bg-gray-800 border border-amber-700/40 rounded-lg p-4 hover:border-amber-500 transition-colors group"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2 min-w-0">
                <component :is="iconFor(item.icon)" class="w-5 h-5 text-amber-400 shrink-0" />
                <span class="text-sm font-medium truncate">{{ item.label }}</span>
              </div>
              <button
                @click.stop="togglePin(item.id)"
                class="text-amber-500 hover:text-amber-400 p-1 rounded opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                title="取消置顶"
              >
                <StarIcon class="w-4 h-4 fill-amber-500" />
              </button>
            </div>
            <div class="flex items-center gap-2">
              <span
                class="w-2 h-2 rounded-full shrink-0"
                :class="item.online ? 'bg-emerald-500' : 'bg-gray-500'"
              />
              <p class="text-xs text-gray-500 truncate">{{ item.subtitle }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- ============ 所有服务 ============ -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <GlobeIcon class="w-5 h-5 text-blue-400" />
          <h2 class="text-base font-semibold">所有服务</h2>
          <span class="text-xs text-gray-500">{{ allServices.length }} 个</span>
        </div>

        <div v-if="allServices.length === 0" class="text-center py-10 text-gray-600">
          <BoxIcon class="w-10 h-10 mx-auto mb-2" />
          <p class="text-sm">暂无服务配置</p>
          <p class="text-xs mt-1">在 services.yml 中添加服务</p>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          <div
            v-for="svc in allServices"
            :key="'service:' + svc.id"
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-blue-500 transition-colors group cursor-pointer"
            @click="openURL(svc.url)"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2 min-w-0">
                <component :is="iconFor(serviceIcon(svc))" class="w-5 h-5 text-blue-400 shrink-0" />
                <span class="text-sm font-medium truncate">{{ svc.name }}</span>
              </div>
              <button
                @click.stop="togglePin('service:' + svc.id)"
                class="p-1 rounded opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                :class="isPinned('service:' + svc.id) ? 'text-amber-500' : 'text-gray-500 hover:text-amber-400'"
                :title="isPinned('service:' + svc.id) ? '取消置顶' : '置顶'"
              >
                <StarIcon class="w-4 h-4" :class="{ 'fill-amber-500': isPinned('service:' + svc.id) }" />
              </button>
            </div>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 min-w-0">
                <span
                  class="w-2 h-2 rounded-full shrink-0"
                  :class="svc.status === 'online' ? 'bg-emerald-500' : svc.status === 'degraded' ? 'bg-amber-500' : 'bg-gray-500'"
                />
                <span class="text-xs text-gray-500 truncate">{{ svc.url }}</span>
              </div>
              <ExternalLinkIcon class="w-3 h-3 text-gray-600 shrink-0 ml-1" />
            </div>
          </div>
        </div>
      </section>

      <!-- ============ Docker 容器 — 按网络→应用分组 ============ -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <ContainerIcon class="w-5 h-5 text-cyan-400" />
          <h2 class="text-base font-semibold">容器</h2>
          <span class="text-xs text-gray-500">{{ allDockerContainers.length }} 个 · {{ serverDockerGroups.length }} 台服务器</span>
        </div>

        <div v-if="allDockerContainers.length === 0" class="text-center py-10 text-gray-600">
          <ContainerIcon class="w-10 h-10 mx-auto mb-2" />
          <p class="text-sm">暂无容器数据</p>
          <p class="text-xs mt-1">容器由各服务器的 Agent 或 Docker API 上报</p>
        </div>

        <!-- Hierarchical: Server → Network → Application → Containers -->
        <div
          v-for="sg in serverDockerGroups"
          :key="sg.serverId"
          class="mb-5 bg-gray-800/40 border border-gray-700/50 rounded-xl overflow-hidden"
        >
          <!-- Server header (clickable to collapse/expand) -->
          <div
            class="flex items-center justify-between px-4 py-3 bg-gray-800/80 cursor-pointer hover:bg-gray-700/60 transition-colors select-none"
            @click="toggleServerCollapse(sg.serverId)"
          >
            <div class="flex items-center gap-2 min-w-0">
              <button class="text-gray-500 hover:text-gray-300 shrink-0">
                <ChevronDownIcon v-if="!isServerCollapsed(sg.serverId)" class="w-4 h-4" />
                <ChevronRightIcon v-else class="w-4 h-4" />
              </button>
              <ServerIcon class="w-4 h-4 text-gray-400 shrink-0" />
              <span class="text-sm font-medium text-gray-200">{{ sg.serverName }}</span>
              <span v-if="sg.networkName" class="text-[11px] bg-indigo-900/40 text-indigo-300 rounded px-1.5 py-0.5 ml-1">
                {{ sg.networkName }}
              </span>
              <span class="text-[11px] text-gray-600 ml-1">{{ sg.host }}</span>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <span class="text-xs text-gray-500">{{ allDockerContainers.filter(d => d.serverId === sg.serverId).length }} 个容器</span>
              <span
                class="text-xs px-1.5 py-0.5 rounded"
                :class="allDockerContainers.filter(d => d.serverId === sg.serverId && d.container.state === 'running').length > 0
                  ? 'bg-emerald-900/30 text-emerald-300'
                  : 'bg-gray-700 text-gray-400'"
              >
                {{ allDockerContainers.filter(d => d.serverId === sg.serverId && d.container.state === 'running').length }} 运行中
              </span>
            </div>
          </div>

          <!-- Network groups (collapsible body) -->
          <div v-if="!isServerCollapsed(sg.serverId)" class="divide-y divide-gray-700/30">
            <div
              v-for="net in sg.networks"
              :key="sg.serverId + ':' + net.name"
              class="px-4 py-3"
            >
              <!-- Network header -->
              <div class="flex items-center gap-2 mb-2">
                <NetworkIcon class="w-4 h-4 text-emerald-400 shrink-0" />
                <span class="text-xs font-semibold text-gray-300 uppercase tracking-wide">{{ net.name }}</span>
                <span class="text-[11px] text-gray-600">
                  {{ Object.values(net.apps).reduce((sum, arr) => sum + arr.length, 0) }} 个容器
                </span>
              </div>

              <!-- Application groups within this network -->
              <div class="space-y-2 ml-1">
                <div
                  v-for="(containers, appName) in net.apps"
                  :key="sg.serverId + ':' + net.name + ':' + appName"
                >
                  <!-- Application header -->
                  <div class="flex items-center gap-1.5 mb-1.5">
                    <FolderOpenIcon class="w-3.5 h-3.5 text-amber-400/70 shrink-0" />
                    <span class="text-xs font-medium text-gray-400">{{ appName }}</span>
                    <span class="text-[11px] text-gray-600">{{ containers.length }} 个</span>
                  </div>

                  <!-- Container cards -->
                  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2">
                    <div
                      v-for="da in containers"
                      :key="'docker:' + da.serverId + ':' + da.container.id"
                      class="bg-gray-800/70 border border-gray-700 rounded-lg p-2.5 hover:border-cyan-600 transition-colors group"
                    >
                      <div class="flex items-start justify-between">
                        <div class="flex items-center gap-2 min-w-0 flex-1">
                          <span
                            class="w-2 h-2 rounded-full shrink-0 mt-0.5"
                            :class="da.container.state === 'running' ? 'bg-emerald-500' : 'bg-gray-500'"
                          />
                          <div class="min-w-0">
                            <p class="text-xs font-medium truncate">{{ da.container.name || da.container.id.slice(0, 8) }}</p>
                            <p class="text-[11px] text-gray-500 truncate">{{ da.container.image }}</p>
                          </div>
                        </div>
                        <div class="flex items-center gap-1 shrink-0 ml-1">
                          <span
                            class="text-[11px] px-1.5 py-0.5 rounded"
                            :class="da.container.state === 'running'
                              ? 'bg-emerald-900/30 text-emerald-300'
                              : 'bg-gray-700 text-gray-400'"
                          >
                            {{ da.container.state }}
                          </span>
                          <button
                            @click.stop="togglePin('docker:' + da.serverId + ':' + da.container.id)"
                            class="p-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity"
                            :class="isPinned('docker:' + da.serverId + ':' + da.container.id) ? 'text-amber-500' : 'text-gray-500 hover:text-amber-400'"
                          >
                            <StarIcon class="w-3 h-3" :class="{ 'fill-amber-500': isPinned('docker:' + da.serverId + ':' + da.container.id) }" />
                          </button>
                        </div>
                      </div>
                      <div class="flex items-center gap-2 mt-1 ml-4">
                        <p v-if="da.container.status" class="text-[11px] text-gray-600 truncate">{{ da.container.status }}</p>
                        <p v-if="da.container.ports" class="text-[11px] text-gray-600 font-mono truncate">{{ da.container.ports }}</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>
