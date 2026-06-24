<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useConnectionStore } from '@/stores/connection'
import { getApiBase } from '@/api'
import { SettingsIcon, ArrowLeftIcon, DownloadIcon, ServerIcon, CogIcon, SunIcon, MoonIcon, WifiIcon, WifiOffIcon } from '@lucide/vue'

interface Platform {
  os: string
  arch: string
  name: string
}

const router = useRouter()
const auth = useAuthStore()
const conn = useConnectionStore()

const content = ref('')
const saved = ref(false)
const error = ref('')
const saving = ref(false)
const loading = ref(true)
const platforms = ref<Platform[]>([])

// ──────────── Gateway connection state ────────────
const connMode = ref(conn.mode)
const remoteUrl = ref(conn.gatewayUrl)
const connTesting = ref(false)
const connOk = ref(false)
const connError = ref('')
const connLatency = ref(0)
const justApplied = ref(false)

async function testRemoteConnection(url: string) {
  connTesting.value = true
  connOk.value = false
  connError.value = ''
  connLatency.value = 0
  try {
    const testUrl = `${url}/api/health`
    const start = performance.now()
    const res = await fetch(testUrl, { method: 'GET', signal: AbortSignal.timeout(8000) })
    connLatency.value = Math.round(performance.now() - start)
    if (res.ok) {
      connOk.value = true
      connError.value = ''
    } else {
      connError.value = `HTTP ${res.status}`
    }
  } catch (e: any) {
    connError.value = e.name === 'TimeoutError' ? '连接超时（8秒）' : (e.message || '连接失败')
    connOk.value = false
  } finally {
    connTesting.value = false
  }
}

async function applyConnection() {
  if (connMode.value === 'remote') {
    if (!remoteUrl.value || !remoteUrl.value.startsWith('http')) {
      connError.value = '请输入有效的 URL（以 http:// 或 https:// 开头）'
      return
    }
    // Test before applying
    await testRemoteConnection(remoteUrl.value)
    if (!connOk.value) return
    conn.setRemote(remoteUrl.value)
  } else {
    conn.setLocal()
    connOk.value = true
    connError.value = ''
    connLatency.value = 0
  }
  justApplied.value = true
  setTimeout(() => { justApplied.value = false }, 3000)
  // Re-fetch settings data from the new Gateway
  fetchConfig()
}

// ──────────── Config fetch ────────────
async function apiFetch(path: string, options?: RequestInit): Promise<Response> {
  const base = getApiBase()
  const url = base ? `${base}${path}` : path
  return fetch(url, {
    ...options,
    headers: {
      ...options?.headers,
      Authorization: `Bearer ${auth.token}`,
    },
  })
}

async function fetchConfig() {
  loading.value = true
  try {
    const [cfgRes, platRes] = await Promise.all([
      apiFetch('/api/config'),
      apiFetch('/api/agent/platforms'),
    ])
    if (cfgRes.ok) {
      const data = await cfgRes.json()
      content.value = data.content
    }
    if (platRes.ok) {
      const data = await platRes.json()
      platforms.value = data.platforms || []
    }
  } catch (e: any) {
    error.value = '加载失败: ' + e.message
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  saved.value = false
  error.value = ''
  try {
    const res = await apiFetch('/api/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: content.value }),
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    saved.value = true
    setTimeout(() => (saved.value = false), 3000)
  } catch (e: any) {
    error.value = '保存失败: ' + e.message
  } finally {
    saving.value = false
  }
}

function downloadAgent(p: Platform) {
  const a = document.createElement('a')
  const base = getApiBase()
  a.href = base ? `${base}/api/agent/download/${p.os}/${p.arch}` : `/api/agent/download/${p.os}/${p.arch}`
  a.download = `sop-agent-${p.os}-${p.arch}`
  a.click()
}

function platformIcon(os: string): string {
  switch (os) {
    case 'linux': return '🐧'
    case 'darwin': return '🍎'
    case 'windows': return '🪟'
    default: return '💻'
  }
}

function installCommand(p: Platform): string {
  const token = 'sop-agent-token-2024'
  const base = getApiBase() || 'http://localhost:8080'
  const wsBase = base.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:')
  if (p.os === 'linux') {
    return `# 下载\nwget ${base}/api/agent/download/${p.os}/${p.arch} -O /usr/local/bin/sop-agent\nchmod +x /usr/local/bin/sop-agent\n\n# 启动 (设为 systemd 服务可持久化)\nexport GATEWAY_URL=${wsBase}\nexport AGENT_TOKEN=${token}\nsop-agent &`
  }
  if (p.os === 'darwin') {
    return `# 下载\ncurl -o /usr/local/bin/sop-agent ${base}/api/agent/download/${p.os}/${p.arch}\nchmod +x /usr/local/bin/sop-agent\n\n# 启动\nexport GATEWAY_URL=${wsBase}\nexport AGENT_TOKEN=${token}\nsop-agent &`
  }
  if (p.os === 'windows') {
    return `# PowerShell (管理员)\nInvoke-WebRequest -Uri "${base}/api/agent/download/${p.os}/${p.arch}" -OutFile "sop-agent.exe"\n\n$env:GATEWAY_URL="${wsBase}"\n$env:AGENT_TOKEN="${token}"\n.\\sop-agent.exe`
  }
  return ''
}

// Theme toggle
const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(fetchConfig)
</script>

<template>
  <div class="min-h-screen bg-gray-900 text-gray-100">
    <!-- Header -->
    <header class="bg-white/80 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center gap-3">
      <button @click="router.push('/')" class="p-1 hover:bg-gray-700 rounded-lg transition-colors" title="返回仪表盘">
        <ArrowLeftIcon class="w-5 h-5 text-gray-400" />
      </button>
      <SettingsIcon class="w-6 h-6 text-indigo-400" />
      <h1 class="text-lg font-semibold">系统设置</h1>
      <div class="flex-1" />
      <button
        @click="toggleTheme"
        class="text-gray-400 hover:text-amber-400 dark:hover:text-amber-300 transition-colors p-1.5 rounded-lg"
        :title="isDark ? '切换日间模式' : '切换夜间模式'"
      >
        <SunIcon v-if="isDark" class="w-5 h-5" />
        <MoonIcon v-else class="w-5 h-5" />
      </button>
    </header>

    <div class="max-w-5xl mx-auto p-6 space-y-6">
      <!-- Load Error -->
      <div v-if="error && !platforms.length" class="bg-red-900/50 border border-red-700 rounded-lg p-4 text-red-200">
        {{ error }}
      </div>

      <!-- ========== Gateway 连接 ========== -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <WifiIcon class="w-5 h-5 text-green-400" />
          <h2 class="text-base font-semibold">Gateway 连接</h2>
          <span class="text-xs text-gray-500">— 选择连接到本地或远程 Gateway</span>
        </div>

        <div class="bg-gray-800/50 border border-gray-700 rounded-lg p-4 space-y-4">
          <!-- Mode selector -->
          <div class="flex gap-6">
            <label class="flex items-center gap-2 cursor-pointer group">
              <input type="radio" v-model="connMode" value="local" class="accent-green-500" />
              <span class="text-sm text-gray-300 group-hover:text-gray-100 transition-colors">本地 Gateway</span>
              <span class="text-xs text-gray-600">(127.0.0.1:8080)</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer group">
              <input type="radio" v-model="connMode" value="remote" class="accent-blue-500" />
              <span class="text-sm text-gray-300 group-hover:text-gray-100 transition-colors">远程 Gateway</span>
            </label>
          </div>

          <!-- Remote URL input -->
          <div v-if="connMode === 'remote'" class="space-y-3">
            <div class="flex items-center gap-2">
              <input
                v-model="remoteUrl"
                type="text"
                class="flex-1 bg-gray-900 border border-gray-600 rounded-lg px-3 py-2 text-sm font-mono text-gray-200 focus:outline-none focus:border-blue-500 placeholder-gray-600"
                placeholder="https://8.135.39.171:8444"
              />
              <button
                @click="testRemoteConnection(remoteUrl)"
                :disabled="connTesting || !remoteUrl"
                class="px-4 py-2 text-sm bg-gray-700 hover:bg-gray-600 disabled:opacity-50 rounded-lg transition-colors"
              >
                {{ connTesting ? '测试中...' : '测试' }}
              </button>
              <button
                @click="applyConnection"
                class="px-4 py-2 text-sm bg-blue-600 hover:bg-blue-500 disabled:opacity-50 rounded-lg transition-colors"
              >
                应用
              </button>
            </div>

            <!-- Connection status -->
            <div v-if="connOk || connError" class="flex items-center gap-2 text-xs">
              <template v-if="connOk">
                <span class="w-2 h-2 rounded-full bg-green-500 shrink-0" />
                <span class="text-green-400">
                  {{ connLatency > 0 ? `已连接（${connLatency}ms）` : '已连接' }}
                </span>
              </template>
              <template v-else>
                <span class="w-2 h-2 rounded-full bg-red-500 shrink-0" />
                <span class="text-red-400">{{ connError }}</span>
              </template>
            </div>
          </div>

          <!-- Local mode hint -->
          <div v-if="connMode === 'local'" class="flex items-center gap-2 text-xs text-gray-500">
            <WifiOffIcon class="w-3.5 h-3.5" />
            <span>使用本地 Gateway（默认）— 与本机 127.0.0.1:8080 通信</span>
          </div>

          <!-- Applied feedback -->
          <div
            v-if="justApplied"
            class="text-xs text-green-400 bg-green-900/20 rounded px-3 py-1.5 flex items-center gap-1.5"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-green-400" />
            {{ connMode === 'remote' ? `已切换到远程 Gateway: ${conn.gatewayUrl}` : '已切换到本地 Gateway' }}
          </div>
        </div>
      </section>

      <!-- ========== Agent 下载 ========== -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <DownloadIcon class="w-5 h-5 text-green-400" />
          <h2 class="text-base font-semibold">下载 Agent</h2>
          <span class="text-xs text-gray-500">— 在要监控的机器上安装，自动上报指标</span>
        </div>

        <div v-if="loading" class="text-center py-8 text-gray-500">加载中...</div>

        <div v-else-if="platforms.length === 0" class="bg-gray-800/50 border border-gray-700 rounded-lg p-6 text-center text-gray-400">
          未找到已编译的 Agent 二进制文件，请联系管理员编译后重试。
        </div>

        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          <div
            v-for="p in platforms"
            :key="p.os + '/' + p.arch"
            class="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-indigo-600 transition-colors"
          >
            <div class="flex items-center justify-between mb-2">
              <span class="text-2xl">{{ platformIcon(p.os) }}</span>
              <button
                @click="downloadAgent(p)"
                class="px-3 py-1 text-xs bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors flex items-center gap-1"
              >
                <DownloadIcon class="w-3 h-3" />
                下载
              </button>
            </div>
            <div class="text-sm font-medium text-gray-200">{{ p.name }}</div>
            <div class="text-xs text-gray-500 font-mono mt-1">{{ p.os }}/{{ p.arch }}</div>
          </div>
        </div>

        <!-- 安装说明（展开） -->
        <details class="mt-3 bg-gray-800/30 border border-gray-700 rounded-lg">
          <summary class="px-4 py-2 text-sm text-gray-400 cursor-pointer hover:text-gray-200 select-none">
            查看安装和启动命令
          </summary>
          <div class="px-4 pb-4">
            <p class="text-xs text-gray-500 mb-2">
              下载后，在目标机器上设置环境变量并启动 agent：
            </p>
            <div v-for="p in platforms" :key="'cmd-' + p.os + p.arch" class="mb-3">
              <div class="text-xs text-gray-400 font-mono mb-1">{{ p.name }} ({{ p.os }}/{{ p.arch }})</div>
              <pre class="bg-gray-950 rounded p-3 text-xs text-green-300 overflow-x-auto font-mono leading-relaxed">{{ installCommand(p) }}</pre>
            </div>
          </div>
        </details>
      </section>

      <!-- ========== 配置编辑器 ========== -->
      <section>
        <div class="flex items-center gap-2 mb-4">
          <CogIcon class="w-5 h-5 text-indigo-400" />
          <h2 class="text-base font-semibold">配置文件</h2>
          <span class="text-xs text-gray-500">— 添加 Headscale 网络、Docker 主机、手动服务器</span>
        </div>

        <div v-if="loading" class="text-center py-8 text-gray-500">加载中...</div>

        <div v-else>
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <span class="text-sm text-gray-400">services.yml</span>
              <span v-if="saved" class="px-2 py-0.5 text-xs bg-green-600/30 text-green-300 rounded">已保存 ✓</span>
            </div>
            <button
              @click="saveConfig"
              :disabled="saving"
              class="px-4 py-1.5 text-sm bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 rounded-lg transition-colors"
            >
              {{ saving ? '保存中...' : '保存配置' }}
            </button>
          </div>

          <div v-if="error" class="mb-3 p-3 bg-red-900/30 border border-red-700 rounded text-sm text-red-200">
            {{ error }}
          </div>

          <div class="mb-3 grid grid-cols-1 md:grid-cols-2 gap-3">
            <div class="bg-gray-800/50 border border-gray-700 rounded-lg p-3">
              <h3 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Headscale 网络</h3>
              <p class="text-xs text-gray-500">
                在 <code>headscale_networks:</code> 下添加条目。<br>
                每个网络: <code>name</code>、<code>url</code>、<code>api_key</code>。
              </p>
            </div>
            <div class="bg-gray-800/50 border border-gray-700 rounded-lg p-3">
              <h3 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">手动服务器</h3>
              <p class="text-xs text-gray-500">
                在 <code>servers:</code> 下添加。<br>
                需要: <code>id</code>、<code>name</code>、<code>host</code>、<code>agent_token</code>。
              </p>
            </div>
          </div>

          <textarea
            v-model="content"
            class="w-full h-[300px] bg-gray-950 border border-gray-700 rounded-lg p-4 font-mono text-sm text-gray-200 focus:outline-none focus:border-indigo-500 resize-none"
            spellcheck="false"
          ></textarea>

          <p class="mt-2 text-xs text-gray-500">
            修改后点「保存配置」立即生效。回到仪表盘刷新即可看到更新。
          </p>
        </div>
      </section>
    </div>
  </div>
</template>
