<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useGatewayStore } from '@/stores/gateway'
import { agentServersApi } from '@/api'
import ServerCard from '@/components/ServerCard.vue'
import TileGrid from '@/components/TileGrid.vue'
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
  PlusIcon,
  CableIcon,
  ExternalLinkIcon,
  TerminalIcon,
  XIcon,
  SunIcon,
  MoonIcon,
} from '@lucide/vue'
import type { ServerStatus, AgentServerStatus } from '@/types'
import { ref } from 'vue'

const router = useRouter()
const auth = useAuthStore()
const gw = useGatewayStore()

const agentServers = ref<AgentServerStatus[]>([])
const showAddModal = ref(false)
const addForm = ref({ name: '', url: '', type: 'opencode', user: '', password: '' })

const knownAgentPorts: Record<string, number> = {
  opencode: 4096,
}

const lastRefreshed = ref('')

async function fetchAgentServers() {
  try {
    const data = await agentServersApi.list()
    agentServers.value = data || []
  } catch {
    agentServers.value = []
  }
}

const activeGroupFilter = ref('')

// Extract unique groups from servers
const serverGroups = computed(() => {
  const groups = new Set<string>()
  for (const s of gw.servers) {
    if (s.group) groups.add(s.group)
  }
  return Array.from(groups).sort()
})

// Filter servers by active group
const filteredServers = computed(() => {
  if (!activeGroupFilter.value) return gw.servers
  return gw.servers.filter(s => s.group === activeGroupFilter.value)
})

// Gather detected agents from servers (auto-detected + host_agents annotations)
const detectedAgents = computed(() => {
  const list: { name: string; host: string; agent: string; online: boolean }[] = []
  for (const s of gw.servers) {
    const allAgents = new Set<string>()
    // From host_agents config annotation (top-level agents field)
    if (s.agents) s.agents.forEach((a) => allAgents.add(a))
    // From agent auto-detection via WebSocket metrics
    if (s.metrics?.agents) s.metrics.agents.forEach((a) => allAgents.add(a))
    if (allAgents.size > 0) {
      for (const agent of allAgents) {
        list.push({ name: s.name || s.id, host: s.host || '', agent, online: s.online })
      }
    }
  }
  return list
})

function openAddModal() {
  addForm.value = { name: '', url: '', type: 'opencode', user: '', password: '' }
  showAddModal.value = true
}

async function submitAddForm() {
  const f = addForm.value
  if (!f.name || !f.url) return
  try {
    const payload: { name: string; url: string; user?: string; password?: string } = { name: f.name, url: f.url }
    if (f.user) payload.user = f.user
    if (f.password) payload.password = f.password
    await agentServersApi.add(payload)
    showAddModal.value = false
    await fetchAgentServers()
  } catch {
    // silent
  }
}

// Derive network groups from server data
interface NetworkInfo {
  id: string
  label: string
  icon: string
  count: number
  type: 'headscale' | 'agent' | 'config' | 'docker' | 'service'
  filter?: (s: ServerStatus) => boolean
  link?: string // external URL
  panelId?: string // proxy panel ID
}

const networks = computed<NetworkInfo[]>(() => {
  const result: NetworkInfo[] = []

  // Headscale networks
  const headscaleNets = new Map<string, number>()
  for (const s of gw.servers) {
    if (s.source === 'headscale') {
      const net = s.network_name || 'default'
      headscaleNets.set(net, (headscaleNets.get(net) || 0) + 1)
    }
  }
  for (const [net, count] of headscaleNets) {
    result.push({
      id: `hs-${net}`,
      label: `Headscale · ${net}`,
      icon: 'globe',
      count,
      type: 'headscale',
    })
  }

  // Agent-connected servers
  const agentCount = gw.servers.filter(s => s.source === 'agent').length
  if (agentCount > 0) {
    result.push({
      id: 'agent',
      label: 'Agent 直连',
      icon: 'monitor',
      count: agentCount,
      type: 'agent',
    })
  }

  return result
})

// Count Docker hosts (from Headscale nodes that also have Docker exposed)
// For now just show as a static count if we detect Docker-enabled IPs
const dockerCount = computed(() => {
  return gw.servers.filter(s =>
    ['100.64.0.3', '100.64.0.4'].includes(s.host || '')
  ).length
})

function handleNetworkClick(net: NetworkInfo) {
  // Scroll to server section and filter? For now just navigate
  // Since servers are already filtered by the network card click context,
  // we'll implement filtering later with a ref
}

async function setWindowTitle() {
  try {
    const { getCurrentWindow } = await import('@tauri-apps/api/window')
    await getCurrentWindow().setTitle('Novascale')
  } catch {
    document.title = 'Novascale'
  }
}

onMounted(() => {
  setWindowTitle()
  fetchAgentServers()
  gw.startAutoRefresh(15000)
})

// Patch gw.refresh to also fetch agent servers and update timestamp
const origRefresh = gw.refresh
gw.refresh = async () => {
  await origRefresh()
  await fetchAgentServers()
  lastRefreshed.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
}

onUnmounted(() => {
  gw.refresh = origRefresh
  gw.stopAutoRefresh()
})

function handleLogout() {
  auth.logout()
  router.push('/login')
}

function viewServer(id: string) {
  router.push(`/server/${id}`)
}

function openURL(url: string) {
  window.open(url, '_blank')
}

// Theme toggle
const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
</script>

<template>
  <div class="min-h-screen bg-gray-900 text-gray-100">
    <!-- Top navigation bar -->
    <header
      data-tauri-drag-region
      class="sticky top-0 z-10 bg-white/80 dark:bg-gray-800/90 backdrop-blur-sm border-b border-gray-200 dark:border-gray-700 select-none"
    >
      <div class="flex items-center justify-between h-12 px-4">
        <div class="flex items-center gap-3 pl-1">
          <MonitorIcon class="w-5 h-5 text-indigo-400" />
          <h1 class="text-base font-bold">ScaleObs</h1>
        </div>

        <div class="flex items-center gap-3">
          <span v-if="lastRefreshed" class="text-[11px] text-gray-500 hidden sm:inline">
            上次刷新 {{ lastRefreshed }}
          </span>
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
            @click="handleLogout"
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

      <!-- Loading state -->
      <div v-if="gw.loading && gw.servers.length === 0" class="text-center py-20 text-gray-500">
        <MonitorIcon class="w-10 h-10 mx-auto mb-3 text-gray-600 animate-pulse" />
        <p class="text-sm">加载中...</p>
      </div>

      <!-- Server cards -->
      <section v-if="gw.servers.length > 0">
        <div class="flex items-center gap-2 mb-4">
          <ServerIcon class="w-5 h-5 text-indigo-400" />
          <h2 class="text-base font-semibold">服务器状态</h2>
          <span class="text-xs text-gray-500">{{ filteredServers.length }} / {{ gw.servers.length }} 台</span>
        </div>

        <!-- Group filter tabs -->
        <div v-if="serverGroups.length > 0" class="flex flex-wrap gap-1.5 mb-4">
          <button
            @click="activeGroupFilter = ''"
            class="px-3 py-1 text-xs rounded-lg border transition-colors"
            :class="!activeGroupFilter
              ? 'bg-blue-600 border-blue-500 text-white'
              : 'bg-gray-800 border-gray-700 text-gray-400 hover:border-gray-500'"
          >
            全部
          </button>
          <button
            v-for="g in serverGroups" :key="g"
            @click="activeGroupFilter = g"
            class="px-3 py-1 text-xs rounded-lg border transition-colors"
            :class="activeGroupFilter === g
              ? 'bg-blue-600 border-blue-500 text-white'
              : 'bg-gray-800 border-gray-700 text-gray-400 hover:border-gray-500'"
          >
            {{ g }}
          </button>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          <ServerCard
            v-for="srv in filteredServers"
            :key="srv.id"
            :server="srv"
            @click="viewServer"
          />
        </div>
      </section>

      <!-- ============ 网络栏目 ============ -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <GlobeIcon class="w-5 h-5 text-emerald-400" />
          <h2 class="text-base font-semibold">网络</h2>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <!-- Headscale networks -->
          <div
            v-for="net in networks"
            :key="net.id"
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-indigo-500 transition-colors cursor-pointer"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <GlobeIcon v-if="net.icon === 'globe'" class="w-5 h-5 text-blue-400" />
                <MonitorIcon v-else-if="net.icon === 'monitor'" class="w-5 h-5 text-green-400" />
                <ContainerIcon v-else class="w-5 h-5 text-cyan-400" />
                <span class="text-sm font-medium">{{ net.label }}</span>
              </div>
              <span class="text-xs bg-gray-700 text-gray-400 rounded-full px-2 py-0.5">
                {{ net.count }} 台
              </span>
            </div>
            <p class="text-xs text-gray-500 mt-1">
              <template v-if="net.type === 'headscale'">通过 Headscale API 自动发现</template>
              <template v-else-if="net.type === 'agent'">通过 Agent WebSocket 直连</template>
            </p>
          </div>

          <!-- Docker hosts card -->
          <div
            v-if="dockerCount > 0"
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-cyan-500 transition-colors cursor-pointer"
            @click="router.push('/settings')"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <ContainerIcon class="w-5 h-5 text-cyan-400" />
                <span class="text-sm font-medium">Docker</span>
              </div>
              <span class="text-xs bg-gray-700 text-gray-400 rounded-full px-2 py-0.5">{{ dockerCount }}</span>
            </div>
            <p class="text-xs text-gray-500 mt-1">通过 Docker TCP API 远程采集</p>
          </div>

          <!-- FRPS -->
          <div
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-orange-500 transition-colors cursor-pointer"
            @click="router.push('/panel/frps')"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <CableIcon class="w-5 h-5 text-orange-400" />
                <span class="text-sm font-medium">FRP 服务端</span>
              </div>
              <ExternalLinkIcon class="w-4 h-4 text-gray-500" />
            </div>
            <p class="text-xs text-gray-500 mt-1">打开 FRP 管理面板</p>
          </div>

          <!-- Headscale web admin -->
          <a
            :href="`https://8.135.39.171:8444/web/`"
            target="_blank"
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-blue-500 transition-colors cursor-pointer block"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <GlobeIcon class="w-5 h-5 text-blue-400" />
                <span class="text-sm font-medium">Headscale 管理</span>
              </div>
              <ExternalLinkIcon class="w-4 h-4 text-gray-500" />
            </div>
            <p class="text-xs text-gray-500 mt-1">打开 Headscale Web UI</p>
          </a>

          <!-- Add network card -->
          <div
            class="bg-gray-800/50 border border-dashed border-gray-600 rounded-lg p-4 hover:border-indigo-500 hover:bg-gray-800 transition-colors cursor-pointer flex items-center justify-center"
            @click="router.push('/settings')"
          >
            <div class="text-center">
              <PlusIcon class="w-6 h-6 text-gray-500 mx-auto mb-1" />
              <span class="text-sm text-gray-500">添加网络</span>
            </div>
          </div>
        </div>
      </section>

      <!-- ============ AI Agent Server ============ -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <TerminalIcon class="w-5 h-5 text-purple-400" />
          <h2 class="text-base font-semibold">AI Agent Server</h2>
          <span class="text-xs text-gray-500">{{ agentServers.length }} 个已配置</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <!-- Detected agents from servers -->
          <div
            v-for="da in detectedAgents"
            :key="da.host + '/' + da.agent"
            class="bg-gray-800/70 border border-purple-800/40 rounded-lg p-4 hover:border-purple-500 transition-colors"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <span
                  class="w-2 h-2 rounded-full inline-block shrink-0"
                  :class="da.online ? 'bg-emerald-500' : 'bg-gray-500'"
                />
                <span class="text-sm font-medium">{{ da.agent }}</span>
              </div>
              <span class="text-xs px-1.5 py-0.5 rounded bg-purple-900/30 text-purple-300">
                自动检测
              </span>
            </div>
            <p class="text-xs text-gray-500 truncate">{{ da.name }} ({{ da.host || 'localhost' }})</p>
          </div>

          <!-- Configured agent servers -->
          <div
            v-for="as in agentServers"
            :key="as.name"
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-purple-500 transition-colors cursor-pointer"
            @click="openURL(as.url)"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <span
                  class="w-2 h-2 rounded-full inline-block shrink-0"
                  :class="as.online ? 'bg-emerald-500' : 'bg-red-400'"
                />
                <span class="text-sm font-medium">{{ as.name }}</span>
              </div>
              <span
                class="text-xs px-1.5 py-0.5 rounded"
                :class="as.online ? 'text-emerald-400 bg-emerald-900/30' : 'text-red-400 bg-red-900/30'"
              >
                {{ as.online ? '在线' : '离线' }}
              </span>
            </div>
            <p class="text-xs text-gray-500 truncate">{{ as.url }}</p>
            <p v-if="as.error" class="text-xs text-red-400 mt-1 truncate">{{ as.error }}</p>
          </div>

          <!-- Add agent server card -->
          <div
            class="bg-gray-800/50 border border-dashed border-gray-600 rounded-lg p-4 hover:border-purple-500 hover:bg-gray-800 transition-colors cursor-pointer flex items-center justify-center"
            @click="openAddModal()"
          >
            <div class="text-center">
              <PlusIcon class="w-6 h-6 text-gray-500 mx-auto mb-1" />
              <span class="text-sm text-gray-500">添加 Server</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Add Agent Server Modal -->
      <div
        v-if="showAddModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
        @click.self="showAddModal = false"
      >
        <div class="bg-gray-800 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
          <div class="flex items-center justify-between mb-5">
            <h3 class="text-base font-semibold">添加 Agent Server</h3>
            <button @click="showAddModal = false" class="p-1 hover:bg-gray-700 rounded-lg transition-colors">
              <XIcon class="w-5 h-5 text-gray-400" />
            </button>
          </div>

          <div class="space-y-4">
            <div>
              <label class="block text-sm text-gray-400 mb-1">名称</label>
              <input
                v-model="addForm.name"
                type="text"
                placeholder="例如: Codex 开发机"
                class="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-purple-500"
              />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">URL 或 IP:端口</label>
              <input
                v-model="addForm.url"
                type="text"
                placeholder="http://100.64.0.4:8080 或 http://localhost:4096"
                class="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-purple-500"
              />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">类型</label>
              <div class="flex gap-2">
                <button
                  v-for="t in ['opencode', 'codex', 'claude code', 'other']"
                  :key="t"
                  @click="addForm.type = t"
                  class="px-3 py-1.5 text-xs rounded-lg border transition-colors"
                  :class="addForm.type === t ? 'bg-purple-600 border-purple-500 text-white' : 'bg-gray-900 border-gray-700 text-gray-400 hover:border-gray-500'"
                >{{ t }}</button>
              </div>
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">用户名 <span class="text-gray-600">（可选）</span></label>
              <input
                v-model="addForm.user"
                type="text"
                placeholder="Basic Auth 用户名"
                class="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-purple-500"
              />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">密码 <span class="text-gray-600">（可选）</span></label>
              <input
                v-model="addForm.password"
                type="password"
                placeholder="Basic Auth 密码"
                class="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-purple-500"
              />
            </div>
          </div>

          <div class="flex justify-end gap-3 mt-6">
            <button
              @click="showAddModal = false"
              class="px-4 py-2 text-sm text-gray-400 hover:text-gray-200 transition-colors"
            >取消</button>
            <button
              @click="submitAddForm()"
              :disabled="!addForm.name || !addForm.url"
              class="px-4 py-2 text-sm bg-purple-600 hover:bg-purple-500 disabled:opacity-40 rounded-lg transition-colors"
            >添加</button>
          </div>
        </div>
      </div>

      <!-- Service tile grid -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <Grid3X3Icon class="w-5 h-5 text-indigo-400" />
          <h2 class="text-base font-semibold">服务面板</h2>
        </div>
        <TileGrid
          :categories="gw.categories"
          :services-by-category="gw.servicesByCategory"
        />
      </section>
    </main>
  </div>
</template>
