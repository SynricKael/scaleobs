<template>
  <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="close">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl w-full max-w-lg mx-4 max-h-[85vh] overflow-hidden">
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
          <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          {{ server?.name || server?.id }} 设置
        </h2>
        <button class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" @click="close">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Tabs -->
      <div class="flex border-b border-gray-200 dark:border-gray-700 px-4">
        <button
          v-for="tab in tabs" :key="tab.key"
          class="px-4 py-3 text-sm font-medium border-b-2 transition-colors"
          :class="activeTab === tab.key
            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
            : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- Body -->
      <div class="overflow-y-auto px-6 py-4 space-y-4 max-h-[60vh]">
        <!-- Tab: SSH -->
        <div v-if="activeTab === 'ssh'" class="space-y-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            SSH 连接设置，用于远程管理和文件传输。此节点将成为数据中心。
          </p>
          <div class="space-y-3">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">主机</label>
              <input v-model="form.ssh_host" placeholder="如 100.64.0.4"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
            </div>
            <div class="flex gap-3">
              <div class="flex-1">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">端口</label>
                <input v-model.number="form.ssh_port" type="number" placeholder="22"
                  class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                />
              </div>
              <div class="flex-1">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">用户名</label>
                <input v-model="form.ssh_user" placeholder="root"
                  class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                />
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">密码</label>
              <input v-model="form.ssh_pass" type="password" placeholder="输入密码"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">SSH Key 路径</label>
              <input v-model="form.ssh_key_path" placeholder="如 ~/.ssh/id_rsa"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
              <p class="text-xs text-gray-400 mt-1">留空则使用密码登录</p>
            </div>
          </div>
        </div>

        <!-- Tab: 分组 -->
        <div v-if="activeTab === 'group'" class="space-y-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            将主机归入分组，便于在仪表盘上按组筛选和管理。
          </p>
          <div class="space-y-2">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">分组名称</label>
            <div class="flex gap-2 flex-wrap">
              <button
                v-for="g in groupOptions" :key="g"
                class="px-4 py-2 rounded-lg text-sm font-medium border transition-colors"
                :class="form.group === g
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                  : 'border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-400 hover:border-gray-400 dark:hover:border-gray-500'"
                @click="form.group = form.group === g ? '' : g"
              >
                {{ g }}
              </button>
            </div>
            <div class="flex gap-2 mt-2">
              <input v-model="newGroup" placeholder="新建分组..."
                class="flex-1 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                @keyup.enter="addNewGroup"
              />
              <button v-if="newGroup" @click="addNewGroup"
                class="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-sm font-medium transition-colors"
              >
                添加
              </button>
            </div>
          </div>
        </div>

          <!-- Tab: Docker -->
        <div v-if="activeTab === 'docker'" class="space-y-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            Docker 连接设置。选择"Agent（代理）"将使用 ScaleObs Agent 自动报告容器信息；选择"API"则直接连接 Docker 守护进程的 TCP 端口。
          </p>

          <!-- Mode selector -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">连接方式</label>
            <div class="flex gap-3">
              <button
                class="flex-1 px-4 py-3 rounded-lg border-2 text-sm font-medium transition-colors text-left"
                :class="form.docker_mode === 'agent'
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                  : 'border-gray-200 dark:border-gray-600 text-gray-600 dark:text-gray-400 hover:border-gray-400'"
                @click="form.docker_mode = 'agent'"
              >
                <div class="flex items-center gap-2 mb-1">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V9a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2z" />
                  </svg>
                  <span>Agent（代理）</span>
                </div>
                <p class="text-xs opacity-70">通过 ScaleObs Agent 自动上报</p>
              </button>
              <button
                class="flex-1 px-4 py-3 rounded-lg border-2 text-sm font-medium transition-colors text-left"
                :class="form.docker_mode === 'api'
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                  : 'border-gray-200 dark:border-gray-600 text-gray-600 dark:text-gray-400 hover:border-gray-400'"
                @click="form.docker_mode = 'api'"
              >
                <div class="flex items-center gap-2 mb-1">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                  <span>API（直连）</span>
                </div>
                <p class="text-xs opacity-70">直接连接 Docker TCP 端口</p>
              </button>
            </div>
          </div>

          <!-- API fields (shown only when mode=api) -->
          <div v-if="form.docker_mode === 'api'" class="space-y-3 border-t border-gray-200 dark:border-gray-700 pt-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">主机地址</label>
              <input v-model="form.docker_host" placeholder="如 100.64.0.4"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">端口</label>
              <input v-model.number="form.docker_port" type="number" placeholder="2375"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
            </div>

            <!-- TLS Toggle -->
            <div class="flex items-center justify-between pt-2">
              <label class="text-sm font-medium text-gray-700 dark:text-gray-300">TLS 加密</label>
              <button
                class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
                :class="form.docker_tls ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'"
                @click="form.docker_tls = !form.docker_tls"
              >
                <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                  :class="form.docker_tls ? 'translate-x-6' : 'translate-x-1'"
                />
              </button>
            </div>

            <!-- TLS fields (shown only when TLS is enabled) -->
            <div v-if="form.docker_tls" class="space-y-3 pl-2 border-l-2 border-blue-300 dark:border-blue-700">
              <p class="text-xs text-gray-400">支持文件路径或 PEM 格式内容</p>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">CA 证书</label>
                <input v-model="form.docker_tls_ca" placeholder="如 /etc/docker/ca.pem 或 PEM 内容"
                  class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm font-mono text-xs"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">客户端证书</label>
                <input v-model="form.docker_tls_cert" placeholder="如 /etc/docker/cert.pem 或 PEM 内容"
                  class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm font-mono text-xs"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">客户端密钥</label>
                <input v-model="form.docker_tls_key" placeholder="如 /etc/docker/key.pem 或 PEM 内容"
                  class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm font-mono text-xs"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Tab: AI Agent -->
        <div v-if="activeTab === 'aiagent'" class="space-y-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            为此服务器配置 AI 编程 Agent (Codex, Claude Code, OpenCode 等)。配置后该服务器可被纳管执行自动化任务。
          </p>
          <div v-if="currentAgents.length" class="space-y-2">
            <div v-for="(agent, idx) in currentAgents" :key="idx"
              class="flex items-center justify-between px-3 py-2 rounded-lg bg-gray-50 dark:bg-gray-700/50 border border-gray-200 dark:border-gray-600"
            >
              <div class="flex items-center gap-2">
                <span class="w-2 h-2 rounded-full"
                  :class="agent.online ? 'bg-green-500' : 'bg-gray-400'"
                ></span>
                <span class="text-sm text-gray-700 dark:text-gray-300">{{ agent.name }}</span>
                <span class="text-xs text-gray-400">{{ agent.url }}</span>
              </div>
              <span v-if="agent.error" class="text-xs text-red-400">{{ agent.error }}</span>
            </div>
          </div>
          <div v-else class="text-sm text-gray-400 italic">
            该服务器暂无 AI Agent 配置
          </div>

          <div class="border-t border-gray-200 dark:border-gray-700 pt-4 mt-4">
            <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">添加 Agent Server</h4>
            <div class="space-y-3">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">名称</label>
                <input v-model="agentForm.name" placeholder="如 My Codex"
                  class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">URL</label>
                <input v-model="agentForm.url" placeholder="如 http://100.64.0.4:3456"
                  class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                />
              </div>
              <div class="flex gap-3">
                <div class="flex-1">
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">用户名（可选）</label>
                  <input v-model="agentForm.user" placeholder="Basic Auth"
                    class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                  />
                </div>
                <div class="flex-1">
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">密码（可选）</label>
                  <input v-model="agentForm.pass" type="password" placeholder="密码"
                    class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                  />
                </div>
              </div>
              <button @click="addAgentServer"
                class="w-full px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
                :disabled="!agentForm.name || !agentForm.url"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                添加 Agent Server
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between gap-3 px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
        <button @click="remove"
          class="px-4 py-2 text-sm font-medium text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
        >
          移除主机
        </button>
        <div class="flex items-center gap-3">
          <button @click="close"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"
          >
            取消
          </button>
          <button @click="save"
            class="px-5 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
            :disabled="saving"
          >
            <svg v-if="saving" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            {{ saving ? '保存中...' : '保存设置' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from 'vue'
import { serversApi, agentServersApi } from '@/api'
import type { ServerStatus, SSHConfig, AgentServerStatus } from '@/types'

const props = defineProps<{
  visible: boolean
  server?: ServerStatus | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
}>()

const activeTab = ref<'ssh' | 'group' | 'docker' | 'aiagent'>('ssh')
const tabs: { key: 'ssh' | 'group' | 'docker' | 'aiagent'; label: string }[] = [
  { key: 'ssh', label: 'SSH' },
  { key: 'group', label: '分组' },
  { key: 'docker', label: 'Docker' },
  { key: 'aiagent', label: 'AI Agent' },
]

const saving = ref(false)
const newGroup = ref('')
const existingGroups = ref<string[]>([])
const agentServers = ref<AgentServerStatus[]>([])
const currentAgents = computed(() => agentServers.value)

const form = reactive({
  group: '',
  ssh_host: '',
  ssh_port: 22,
  ssh_user: '',
  ssh_pass: '',
  ssh_key_path: '',
  // Docker
  docker_mode: 'agent',
  docker_host: '',
  docker_port: 2375,
  docker_tls: false,
  docker_tls_ca: '',
  docker_tls_cert: '',
  docker_tls_key: '',
})

const agentForm = reactive({
  name: '',
  url: '',
  user: '',
  pass: '',
})

// Compute available group options from currently known servers plus existing groups
const groupOptions = computed(() => {
  const set = new Set(existingGroups.value)
  if (form.group) set.add(form.group)
  return Array.from(set).sort()
})

function addNewGroup() {
  const g = newGroup.value.trim()
  if (g && !existingGroups.value.includes(g)) {
    existingGroups.value.push(g)
    form.group = g
    newGroup.value = ''
  }
}

function loadFromServer() {
  if (!props.server) return
  form.group = props.server.group || ''
  if (props.server.ssh) {
    form.ssh_host = props.server.ssh.host || ''
    form.ssh_port = props.server.ssh.port || 22
    form.ssh_user = props.server.ssh.user || ''
    form.ssh_pass = props.server.ssh.password || ''
    form.ssh_key_path = props.server.ssh.key_path || ''
  }
  // Load Docker config
  if (props.server.docker_config) {
    form.docker_mode = props.server.docker_config.mode || 'agent'
    form.docker_host = props.server.docker_config.host || ''
    form.docker_port = props.server.docker_config.port || 2375
    form.docker_tls = props.server.docker_config.tls || false
    form.docker_tls_ca = props.server.docker_config.tls_ca_cert || ''
    form.docker_tls_cert = props.server.docker_config.tls_cert || ''
    form.docker_tls_key = props.server.docker_config.tls_key || ''
  } else {
    // Reset to defaults
    form.docker_mode = 'agent'
    form.docker_host = ''
    form.docker_port = 2375
    form.docker_tls = false
    form.docker_tls_ca = ''
    form.docker_tls_cert = ''
    form.docker_tls_key = ''
  }
}

function close() {
  emit('close')
}

async function save() {
  saving.value = true
  try {
    // Build payload
    const payload: any = {
      group: form.group || undefined,
      ssh_host: form.ssh_host || undefined,
      ssh_port: form.ssh_port > 0 ? form.ssh_port : undefined,
      ssh_user: form.ssh_user || undefined,
      ssh_pass: form.ssh_pass || undefined,
      ssh_key_path: form.ssh_key_path || undefined,
      docker_mode: form.docker_mode === 'api' ? 'api' : undefined,
    }
    if (form.docker_mode === 'api') {
      payload.docker_host = form.docker_host || undefined
      payload.docker_port = form.docker_port > 0 ? form.docker_port : undefined
      payload.docker_tls = form.docker_tls || undefined
      if (form.docker_tls) {
        payload.docker_tls_ca = form.docker_tls_ca || undefined
        payload.docker_tls_cert = form.docker_tls_cert || undefined
        payload.docker_tls_key = form.docker_tls_key || undefined
      }
    }
    await serversApi.updateSettings(props.server!.id, payload)
    emit('saved')
    close()
  } catch (e: any) {
    console.error('Failed to save server settings:', e)
    alert('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!props.server) return
  const id = props.server.id
  const name = props.server.name || id
  if (!confirm(`确定要从面板中移除 "${name}"？\n\n此操作仅从面板列表中移除，不会影响服务器上的 Agent 进程。如果 Agent 仍在运行，它会自动重新注册。`)) return
  try {
    await serversApi.remove(id)
    emit('saved')
    close()
  } catch (e: any) {
    alert('移除失败: ' + (e.response?.data?.error || e.message))
  }
}

async function addAgentServer() {
  try {
    await agentServersApi.add({
      name: agentForm.name,
      url: agentForm.url,
      user: agentForm.user || undefined,
      password: agentForm.pass || undefined,
    })
    agentForm.name = ''
    agentForm.url = ''
    agentForm.user = ''
    agentForm.pass = ''
    // Reload agent servers
    const list = await agentServersApi.list()
    agentServers.value = list
  } catch (e: any) {
    alert('添加失败: ' + (e.response?.data?.error || e.message))
  }
}

// Load data when dialog opens
watch(() => props.visible, async (val) => {
  if (!val) return
  activeTab.value = 'ssh'
  newGroup.value = ''
  loadFromServer()
  try {
    // Fetch all servers to extract existing groups
    const servers = await serversApi.list()
    existingGroups.value = [...new Set(servers.map(s => s.group).filter(Boolean))] as string[]
    // Fetch agent servers
    agentServers.value = await agentServersApi.list()
  } catch (e) {
    console.error('Failed to load reference data', e)
  }
})
</script>
