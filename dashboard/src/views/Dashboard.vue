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
  SearchIcon,
} from '@lucide/vue'
import type { ServerStatus, AgentServerStatus } from '@/types'
import { ref } from 'vue'

const router = useRouter()
const auth = useAuthStore()
const gw = useGatewayStore()

const agentServers = ref<AgentServerStatus[]>([])
const showAddModal = ref(false)
const showDiscoverModal = ref(false)
const discovering = ref(false)
const addingDiscovered = ref(false)
const discoveredList = ref<DiscoveredAgent[]>([])
const addForm = ref({ name: '', url: '', type: 'opencode', user: '', password: '' })

const lastRefreshed = ref('')

const knownAgentPorts: Record<string, number> = {
  opencode: 4096,
  codex: 8080,
  'claude code': 8080,
}

interface DiscoveredAgent {
  key: string
  agent: string
  type: string
  host: string
  serverName: string
  online: boolean
  checked: boolean
}

async function fetchAgentServers() {
  try {
    const data = await agentServersApi.list()
    agentServers.value = data || []
  } catch {
    agentServers.value = []
  }
}

function openDiscoverModal() {
  showDiscoverModal.value = true
  discovering.value = true
  discoveredList.value = []
  // Build discovered agent list from all servers
  const seen = new Set<string>()
  const list: DiscoveredAgent[] = []
  const configuredNames = new Set(agentServers.value.map(a => a.name + a.url))
  for (const s of gw.servers) {
    const allAgents = new Set<string>()
    if (s.agents) s.agents.forEach(a => allAgents.add(a))
    if (s.metrics?.agents) s.metrics.agents.forEach(a => allAgents.add(a))
    for (const agent of allAgents) {
      const key = `${s.host || s.id}/${agent}`
      if (seen.has(key)) continue
      seen.add(key)
      // Skip if already configured
      const guessName = `${s.name || s.host || s.id} · ${agent}`
      const guessUrl = `http://${s.host || 'localhost'}:${knownAgentPorts[agent] || 4096}`
      if (configuredNames.has(guessName + guessUrl)) continue
      list.push({
        key,
        agent: guessName,
        type: agent,
        host: s.host || '',
        serverName: s.name || s.id,
        online: s.online,
        checked: false,
      })
    }
  }
  discoveredList.value = list
  discovering.value = false
}

async function submitDiscovered() {
  const selected = discoveredList.value.filter(d => d.checked)
  if (selected.length === 0) return
  addingDiscovered.value = true
  let added = 0
  for (const da of selected) {
    try {
      await agentServersApi.add({
        name: da.agent,
        url: `http://${da.host}:${knownAgentPorts[da.type] || 4096}`,
      })
      added++
    } catch {
      // individual failures don't block the rest
    }
  }
  addingDiscovered.value = false
  showDiscoverModal.value = false
  if (added > 0) {
    await fetchAgentServers()
  }
}

const activeGroupFilter = ref('')
const sortKey = ref<'name' | 'host' | 'last_seen'>('last_seen')

// Extract unique groups from servers
const serverGroups = computed(() => {
  const groups = new Set<string>()
  for (const s of gw.servers) {
    if (s.group) groups.add(s.group)
  }
  return Array.from(groups).sort()
})

// Filter & sort servers
const filteredServers = computed(() => {
  let list = gw.servers
  if (activeGroupFilter.value) {
    list = list.filter(s => s.group === activeGroupFilter.value)
  }
  // Sort
  const sorted = [...list]
  sorted.sort((a, b) => {
    let cmp = 0
    if (sortKey.value === 'name') {
      const na = (a.name || a.id || '').toLowerCase()
      const nb = (b.name || b.id || '').toLowerCase()
      cmp = na.localeCompare(nb)
    } else if (sortKey.value === 'host') {
      const ha = (a.host || '').toLowerCase()
      const hb = (b.host || '').toLowerCase()
      cmp = ha.localeCompare(hb)
    } else {
      // last_seen descending (newest first)
      cmp = (b.last_seen || 0) - (a.last_seen || 0)
    }
    return cmp
  })
  return sorted
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
  type: 'headscale' | 'service'
  link?: string
  panelId?: string
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
  return result
})

// Docker hosts with containers, for the services area
const dockerHostsList = computed(() => {
  return gw.servers.filter(s =>
    s.metrics?.docker_containers && s.metrics.docker_containers.length > 0
  )
})

async function setWindowTitle() {
  try {
    const { getCurrentWindow } = await import('@tauri-apps/api/window')
    await getCurrentWindow().setTitle('ScaleObs')
  } catch {
    document.title = 'ScaleObs'
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
          <nav class="ml-6 flex items-center gap-1">
            <button
              @click="router.push('/')"
              class="px-3 py-1.5 text-sm rounded-lg transition-colors"
              :class="router.currentRoute?.value?.path === '/' || !router.currentRoute?.value?.path?.startsWith('/apps')
                ? 'bg-blue-600 text-white'
                : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700'"
            >
              <ServerIcon class="w-4 h-4 inline-block mr-1" />
              设备
            </button>
            <button
              @click="router.push('/apps')"
              class="px-3 py-1.5 text-sm rounded-lg transition-colors"
              :class="router.currentRoute?.value?.path?.startsWith('/apps')
                ? 'bg-blue-600 text-white'
                : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700'"
            >
              <Grid3X3Icon class="w-4 h-4 inline-block mr-1" />
              应用
            </button>
          </nav>
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

        <!-- Group filter tabs + sort controls -->
        <div class="flex flex-wrap items-center gap-2 mb-4">
          <div v-if="serverGroups.length > 0" class="flex flex-wrap gap-1.5">
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
          <div class="flex-1"></div>
          <div class="flex items-center gap-1 text-xs">
            <span class="text-gray-500 mr-1">排序:</span>
            <button v-for="opt in [{k:'name',l:'名称'},{k:'host',l:'IP'},{k:'last_seen',l:'加入时间'}]" :key="opt.k"
              @click="sortKey = opt.k as any"
              class="px-2.5 py-1 rounded-lg border transition-colors"
              :class="sortKey === opt.k
                ? 'bg-indigo-600 border-indigo-500 text-white'
                : 'bg-gray-800 border-gray-700 text-gray-400 hover:border-gray-500'"
            >{{ opt.l }}</button>
          </div>
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
                <GlobeIcon class="w-5 h-5 text-blue-400" />
                <span class="text-sm font-medium">{{ net.label }}</span>
              </div>
              <span class="text-xs bg-gray-700 text-gray-400 rounded-full px-2 py-0.5">
                {{ net.count }} 台
              </span>
            </div>
            <p class="text-xs text-gray-500 mt-1">通过 Headscale API 自动发现</p>
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

      <!-- ============ Docker 容器 ============ -->
      <section v-if="dockerHostsList.length > 0">
        <div class="flex items-center gap-2 mb-4">
          <ContainerIcon class="w-5 h-5 text-cyan-400" />
          <h2 class="text-base font-semibold">容器</h2>
          <span class="text-xs text-gray-500">— 所有服务器的 Docker 容器概览</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div
            v-for="dh in dockerHostsList"
            :key="dh.id"
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-cyan-500 transition-colors cursor-pointer"
            @click="viewServer(dh.id)"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2 min-w-0">
                <ContainerIcon class="w-5 h-5 text-cyan-400 shrink-0" />
                <span class="text-sm font-medium truncate">{{ dh.name || dh.host }}</span>
              </div>
              <span v-if="dh.metrics?.docker_stats" class="text-xs bg-gray-700 text-gray-400 rounded-full px-2 py-0.5 shrink-0 ml-2">
                {{ dh.metrics.docker_stats.running }}/{{ dh.metrics.docker_stats.total }}
              </span>
            </div>
            <div class="flex flex-wrap gap-1 mt-1">
              <span
                v-for="c in dh.metrics?.docker_containers?.slice(0, 6)"
                :key="c.id"
                class="text-xs px-1.5 py-0.5 rounded"
                :class="c.state === 'running'
                  ? 'bg-emerald-900/30 text-emerald-300'
                  : 'bg-gray-700 text-gray-400'"
              >
                {{ c.name || c.id.slice(0, 8) }}
              </span>
              <span v-if="(dh.metrics?.docker_containers?.length || 0) > 6" class="text-xs text-gray-500 px-1">
                +{{ (dh.metrics?.docker_containers?.length || 0) - 6 }}
              </span>
            </div>
          </div>
        </div>
      </section>

      <!-- ============ AI Agent ============ -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <TerminalIcon class="w-5 h-5 text-purple-400" />
          <h2 class="text-base font-semibold">AI Agent</h2>
          <span class="text-xs text-gray-500">{{ agentServers.length }} 个已配置</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <!-- Configured agent servers (persistent even when offline) -->
          <div
            v-for="as in agentServers"
            :key="as.name"
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-purple-500 transition-colors cursor-pointer"
            @click="openURL(as.url)"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2 min-w-0">
                <span
                  class="w-2 h-2 rounded-full inline-block shrink-0"
                  :class="as.online ? 'bg-emerald-500' : 'bg-red-400'"
                />
                <span class="text-sm font-medium truncate">{{ as.name }}</span>
              </div>
              <span
                class="text-xs px-1.5 py-0.5 rounded shrink-0 ml-2"
                :class="as.online ? 'text-emerald-400 bg-emerald-900/30' : 'text-red-400 bg-red-900/30'"
              >
                {{ as.online ? '在线' : '离线' }}
              </span>
            </div>
            <p class="text-xs text-gray-500 truncate">{{ as.url }}</p>
            <p v-if="as.error" class="text-xs text-red-400 mt-1 truncate">{{ as.error }}</p>
          </div>

          <!-- Search & discover agents card -->
          <div
            class="bg-gray-800 border border-dashed border-purple-700/50 rounded-lg p-4 hover:border-purple-500 hover:bg-gray-800 transition-colors cursor-pointer flex flex-col items-center justify-center"
            @click="openDiscoverModal()"
          >
            <SearchIcon class="w-6 h-6 text-purple-400 mx-auto mb-1" />
            <span class="text-sm text-purple-400">搜索发现</span>
            <p class="text-xs text-gray-500 mt-1 text-center">扫描所有服务器上的 AI Agent</p>
          </div>

          <!-- Manual add -->
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

      <!-- ============ Discover Modal ============ -->
      <div
        v-if="showDiscoverModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
        @click.self="showDiscoverModal = false"
      >
        <div class="bg-gray-800 border border-gray-700 rounded-xl p-6 w-full max-w-lg mx-4 max-h-[80vh] overflow-hidden flex flex-col">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-base font-semibold">搜索发现 AI Agent</h3>
            <button @click="showDiscoverModal = false" class="p-1 hover:bg-gray-700 rounded-lg transition-colors">
              <XIcon class="w-5 h-5 text-gray-400" />
            </button>
          </div>

          <p class="text-sm text-gray-400 mb-4">
            扫描 {{ gw.servers.length }} 台服务器的 Agent 数据和配置，勾选要添加的 AI Agent，确认后固化到配置中。
          </p>

          <!-- Loading -->
          <div v-if="discovering" class="flex items-center justify-center py-8 text-gray-400">
            <RefreshCwIcon class="w-5 h-5 animate-spin mr-2" />
            正在扫描服务器...
          </div>

          <!-- Discovered agents list -->
          <div v-else class="overflow-y-auto flex-1 space-y-2">
            <div
              v-for="da in discoveredList"
              :key="da.key"
              class="flex items-center gap-3 px-3 py-2.5 rounded-lg border transition-colors cursor-pointer"
              :class="da.checked
                ? 'bg-purple-900/20 border-purple-700/50'
                : 'bg-gray-800/50 border-gray-700 hover:border-gray-600'"
              @click="da.checked = !da.checked"
            >
              <div
                class="w-4 h-4 rounded border-2 flex items-center justify-center shrink-0 transition-colors"
                :class="da.checked
                  ? 'bg-purple-500 border-purple-500'
                  : 'border-gray-500'"
              >
                <svg v-if="da.checked" class="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium text-gray-200">{{ da.agent }}</span>
                  <span class="text-xs px-1.5 py-0.5 rounded bg-gray-700 text-gray-400">{{ da.type }}</span>
                </div>
                <p class="text-xs text-gray-500 truncate">{{ da.serverName }} ({{ da.host }})</p>
              </div>
              <span
                class="w-2 h-2 rounded-full shrink-0"
                :class="da.online ? 'bg-emerald-500' : 'bg-gray-500'"
              />
            </div>

            <!-- Empty state -->
            <div v-if="discoveredList.length === 0" class="text-center py-8 text-gray-500">
              <SearchIcon class="w-8 h-8 mx-auto mb-2 text-gray-600" />
              <p class="text-sm">未发现新的 AI Agent</p>
              <p class="text-xs text-gray-600 mt-1">所有服务器上的 Agent 已全部配置</p>
            </div>
          </div>

          <!-- Footer -->
          <div class="flex items-center justify-between mt-4 pt-4 border-t border-gray-700">
            <span class="text-sm text-gray-400">
              {{ discoveredList.filter(d => d.checked).length }} 个已选
            </span>
            <div class="flex gap-3">
              <button
                @click="showDiscoverModal = false"
                class="px-4 py-2 text-sm text-gray-400 hover:text-gray-200 transition-colors"
              >取消</button>
              <button
                @click="submitDiscovered()"
                :disabled="discoveredList.filter(d => d.checked).length === 0 || addingDiscovered"
                class="px-4 py-2 text-sm bg-purple-600 hover:bg-purple-500 disabled:opacity-40 rounded-lg transition-colors flex items-center gap-2"
              >
                <RefreshCwIcon v-if="addingDiscovered" class="w-3 h-3 animate-spin" />
                添加所选
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- ============ Manual Add Modal ============ -->
      <div
        v-if="showAddModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
        @click.self="showAddModal = false"
      >
        <div class="bg-gray-800 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
          <div class="flex items-center justify-between mb-5">
            <h3 class="text-base font-semibold">手动添加 Agent Server</h3>
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
