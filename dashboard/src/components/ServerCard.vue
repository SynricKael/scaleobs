<script setup lang="ts">
import { ref } from 'vue'
import type { ServerStatus } from '@/types'
import { ContainerIcon, PlayIcon, SquareIcon, RotateCwIcon, ChevronDownIcon, ChevronUpIcon, LoaderIcon, SettingsIcon } from '@lucide/vue'
import GaugeMeter from './GaugeMeter.vue'
import { dockerApi } from '@/api'
import ServerSettingsDialog from './ServerSettingsDialog.vue'

const props = defineProps<{
  server: ServerStatus
}>()

const emit = defineEmits<{
  click: [id: string]
}>()

const showContainers = ref(false)
const showSettings = ref(false)

function formatTime(unix: number): string {
  if (!unix) return '-'
  return new Date(unix * 1000).toLocaleString('zh-CN')
}

function formatMB(mb: number): string {
  if (mb > 1024) return `${(mb / 1024).toFixed(1)} GB`
  return `${mb} MB`
}

function formatBps(bps: number): string {
  if (bps > 1_000_000) return `${(bps / 1_000_000).toFixed(1)} MB/s`
  if (bps > 1_000) return `${(bps / 1_000).toFixed(1)} KB/s`
  return `${bps.toFixed(0)} B/s`
}

function diskPercent(m: { used_gb: number; total_gb: number; percent: number }): number {
  if (m.percent > 0) return m.percent
  if (m.total_gb > 0) return (m.used_gb / m.total_gb) * 100
  return 0
}

function containerStatusIcon(state: string) {
  if (state === 'running') return PlayIcon
  if (state === 'exited' || state === 'stopped') return SquareIcon
  return RotateCwIcon
}

function containerStatusColor(state: string) {
  return state === 'running' ? 'text-emerald-500' : 'text-gray-400'
}

const loadingContainers = ref<Set<string>>(new Set())

function containerAction(containerId: string, action: 'start' | 'stop' | 'restart', e: MouseEvent) {
  e.stopPropagation()
  if (loadingContainers.value.has(containerId)) return
  loadingContainers.value = new Set([...loadingContainers.value, containerId])
  dockerApi.containerAction(props.server.id, containerId, action)
    .then(() => {
      // Success - will be refreshed by auto-poll
      setTimeout(() => {
        const s = new Set(loadingContainers.value)
        s.delete(containerId)
        loadingContainers.value = s
      }, 2000)
    })
    .catch(() => {
      const s = new Set(loadingContainers.value)
      s.delete(containerId)
      loadingContainers.value = s
    })
}
</script>

<template>
  <div
    class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 cursor-pointer hover:shadow-md hover:border-blue-300 dark:hover:border-blue-600 transition-all duration-200 active:scale-[0.98]"
    @click="emit('click', server.id)"
  >
    <!-- Header -->
    <div class="flex items-center justify-between mb-2">
      <div class="flex items-center gap-2 min-w-0">
        <span
          class="w-2.5 h-2.5 rounded-full inline-block shrink-0"
          :class="server.online ? 'bg-emerald-500' : 'bg-red-400'"
        />
        <div class="min-w-0">
          <h3 class="font-medium text-gray-900 dark:text-gray-100 truncate">
            {{ server.name || server.id }}
          </h3>
          <p v-if="server.host" class="text-xs text-gray-400 font-mono truncate mt-0.5">
            {{ server.host }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-1 shrink-0">
        <button
          class="p-1 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          title="设置"
          @click.stop="showSettings = true"
        >
          <SettingsIcon class="w-4 h-4" />
        </button>
        <span
          class="text-xs ml-1 px-1.5 py-0.5 rounded"
          :class="server.online ? 'text-emerald-600 bg-emerald-50 dark:bg-emerald-900/30 dark:text-emerald-400' : 'text-red-500 bg-red-50 dark:bg-red-900/30 dark:text-red-400'"
        >
          {{ server.online ? '在线' : '离线' }}
        </span>
      </div>
    </div>

    <!-- Metrics -->
    <div v-if="server.metrics" class="space-y-3">
      <!-- 3 gauges -->
      <div class="grid grid-cols-3 gap-1 -mx-1">
        <GaugeMeter :value="server.metrics.cpu_percent" label="CPU" />
        <GaugeMeter :value="server.metrics.memory.percent" label="内存" />
        <GaugeMeter
          :value="server.metrics.disks && server.metrics.disks.length > 0
            ? diskPercent(server.metrics.disks[0]) : 0"
          label="磁盘"
        />
      </div>

      <!-- Bottom info: Memory + Docker + Network -->
      <div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 pt-1 border-t border-gray-100 dark:border-gray-700/50">
        <span>{{ formatMB(server.metrics.memory.used_mb) }} / {{ formatMB(server.metrics.memory.total_mb) }}</span>

        <!-- Docker badge (clickable to expand) -->
        <span
          v-if="server.metrics.docker_stats"
          class="flex items-center gap-1 cursor-pointer hover:text-blue-400 transition-colors select-none"
          @click.stop="showContainers = !showContainers"
        >
          <ContainerIcon class="w-3 h-3" />
          {{ server.metrics.docker_stats.running }}/{{ server.metrics.docker_stats.total }}
          <component :is="showContainers ? ChevronUpIcon : ChevronDownIcon" class="w-3 h-3 opacity-50" />
        </span>

        <span v-if="server.metrics.network" class="flex items-center gap-1">
          <span>⬇{{ formatBps(server.metrics.network.bytes_recv_per_sec) }}</span>
          <span>⬆{{ formatBps(server.metrics.network.bytes_sent_per_sec) }}</span>
        </span>
      </div>
    </div>

    <!-- No metrics -->
    <div v-else class="text-xs text-gray-400 dark:text-gray-500 py-2 text-center">
      {{ server.online ? '等待数据...' : '离线' }}
    </div>

    <!-- Expandable container list -->
    <div
      v-if="showContainers && server.metrics?.docker_containers?.length"
      class="mt-3 border-t border-gray-100 dark:border-gray-700/50 pt-2 space-y-1.5"
      @click.stop
    >
      <div
        v-for="c in server.metrics.docker_containers"
        :key="c.id"
        class="flex items-center gap-2 text-xs rounded-lg px-2 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors group"
      >
        <component :is="containerStatusIcon(c.state)" class="w-3 h-3 shrink-0" :class="containerStatusColor(c.state)" />
        <span class="flex-1 min-w-0 truncate font-medium text-gray-700 dark:text-gray-300">{{ c.name }}</span>
        <span class="text-gray-400 hidden sm:inline truncate max-w-[120px]">{{ c.image }}</span>
        <span class="text-gray-500 shrink-0">{{ c.status }}</span>
        <!-- Action buttons -->
        <div class="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
          <button
            v-if="c.state !== 'running'"
            @click="containerAction(c.id, 'start', $event)"
            :disabled="loadingContainers.has(c.id)"
            class="p-0.5 rounded hover:bg-emerald-100 dark:hover:bg-emerald-900/30 text-emerald-500 disabled:opacity-30"
            title="启动"
          >
            <LoaderIcon v-if="loadingContainers.has(c.id)" class="w-3 h-3 animate-spin" />
            <PlayIcon v-else class="w-3 h-3" />
          </button>
          <button
            v-if="c.state === 'running'"
            @click="containerAction(c.id, 'stop', $event)"
            :disabled="loadingContainers.has(c.id)"
            class="p-0.5 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-red-400 disabled:opacity-30"
            title="停止"
          >
            <LoaderIcon v-if="loadingContainers.has(c.id)" class="w-3 h-3 animate-spin" />
            <SquareIcon v-else class="w-3 h-3" />
          </button>
          <button
            @click="containerAction(c.id, 'restart', $event)"
            :disabled="loadingContainers.has(c.id)"
            class="p-0.5 rounded hover:bg-amber-100 dark:hover:bg-amber-900/30 text-amber-400 disabled:opacity-30"
            title="重启"
          >
            <LoaderIcon v-if="loadingContainers.has(c.id)" class="w-3 h-3 animate-spin" />
            <RotateCwIcon v-else class="w-3 h-3" />
          </button>
        </div>
      </div>
    </div>

    <!-- Coding Agent badges -->
    <div v-if="server.agents && server.agents.length > 0" class="flex flex-wrap gap-1 mt-2">
      <span
        v-for="agent in server.agents"
        :key="agent"
        class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium"
        :class="{
          'bg-purple-900/40 text-purple-300 border border-purple-700/50': agent === 'codex',
          'bg-amber-900/40 text-amber-300 border border-amber-700/50': agent === 'claude code' || agent === 'claude',
          'bg-cyan-900/40 text-cyan-300 border border-cyan-700/50': agent === 'opencode',
          'bg-gray-700/60 text-gray-300 border border-gray-600/50': !['codex','claude','claude code','opencode'].includes(agent),
        }"
      >
        {{ agent }}
      </span>
    </div>

    <!-- Source badge -->
    <div class="mt-2 flex items-center justify-between text-xs text-gray-400 dark:text-gray-500">
      <div class="flex items-center gap-2">
        <span v-if="server.group" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 font-medium">
          {{ server.group }}
        </span>
        <span v-if="server.source === 'headscale'" class="text-blue-400">
          Headscale · {{ server.network_name || '-' }}
        </span>
        <span v-else-if="server.source === 'agent'" class="text-emerald-400">Agent</span>
        <span v-else class="text-gray-400">配置</span>
      </div>
      <span>{{ formatTime(server.last_seen) }}</span>
    </div>
  </div>

  <!-- Server settings dialog -->
  <ServerSettingsDialog
    :visible="showSettings"
    :server="server"
    @close="showSettings = false"
    @saved="showSettings = false"
  />
</template>
