<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGatewayStore } from '@/stores/gateway'
import { dockerApi } from '@/api'
import {
  MonitorIcon,
  ArrowLeftIcon,
  ServerIcon,
  HardDriveIcon,
  ActivityIcon,


  ContainerIcon,
  PlayIcon,
  SquareIcon,
  AlertCircleIcon,
  ArrowUpIcon,
  ArrowDownIcon,
  WifiIcon,
  RotateCwIcon,
  LoaderIcon,
} from '@lucide/vue'

const route = useRoute()
const router = useRouter()
const gw = useGatewayStore()

const serverId = route.params.id as string
const loadingContainers = ref<Set<string>>(new Set())

const server = computed(() => gw.servers.find(s => s.id === serverId))

function formatTime(unix: number): string {
  return new Date(unix * 1000).toLocaleString('zh-CN')
}

function formatSize(mb: number): string {
  if (mb > 1024) return `${(mb / 1024).toFixed(1)} GB`
  return `${mb} MB`
}

function formatBytesPerSec(bps: number): string {
  if (bps > 1_000_000) return `${(bps / 1_000_000).toFixed(1)} MB/s`
  if (bps > 1_000) return `${(bps / 1_000).toFixed(1)} KB/s`
  return `${bps.toFixed(0)} B/s`
}

function containerIcon(state: string) {
  return state === 'running' ? PlayIcon : state === 'exited' ? SquareIcon : AlertCircleIcon
}

function containerColor(state: string) {
  return state === 'running'
    ? 'text-emerald-500 bg-emerald-50 dark:bg-emerald-900/20'
    : 'text-gray-400 bg-gray-50 dark:bg-gray-800'
}

function containerAction(cid: string, action: 'start' | 'stop' | 'restart') {
  if (loadingContainers.value.has(cid)) return
  loadingContainers.value = new Set([...loadingContainers.value, cid])
  dockerApi.containerAction(serverId, cid, action)
    .catch(() => {
      const s = new Set(loadingContainers.value)
      s.delete(cid)
      loadingContainers.value = s
    })
}

onMounted(() => {
  if (!server.value) {
    gw.refresh()
  }
})
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Top bar -->
    <header class="sticky top-0 z-10 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm
                   border-b border-gray-200 dark:border-gray-700">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-14">
          <div class="flex items-center gap-3">
            <button @click="router.push('/')"
              class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200
                     transition-colors p-1 rounded">
              <ArrowLeftIcon class="w-5 h-5" />
            </button>
            <ServerIcon class="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <h1 class="text-lg font-bold text-gray-800 dark:text-gray-100">
              {{ server?.name || serverId }} 详情
            </h1>
          </div>
          <span class="text-xs text-gray-400">
            ID: {{ serverId }}
          </span>
        </div>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <template v-if="server">
        <!-- Metrics overview -->
        <div v-if="server.metrics" class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
            <div class="flex items-center gap-2 text-sm text-gray-500 mb-2">
              <ActivityIcon class="w-4 h-4" /> CPU
            </div>
            <p class="text-2xl font-bold" :class="server.metrics.cpu_percent > 80 ? 'text-red-500' : server.metrics.cpu_percent > 60 ? 'text-amber-500' : 'text-blue-600'">
              {{ server.metrics.cpu_percent.toFixed(1) }}%
            </p>
          </div>
          <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
            <div class="flex items-center gap-2 text-sm text-gray-500 mb-2">
              <MonitorIcon class="w-4 h-4" /> 内存
            </div>
            <p class="text-2xl font-bold text-violet-600 dark:text-violet-400">
              {{ formatSize(server.metrics.memory.used_mb) }}
            </p>
            <p class="text-xs text-gray-400 mt-0.5">
              总计 {{ formatSize(server.metrics.memory.total_mb) }}
              ({{ server.metrics.memory.percent.toFixed(1) }}%)
            </p>
          </div>
          <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
            <div class="flex items-center gap-2 text-sm text-gray-500 mb-2">
              <WifiIcon class="w-4 h-4" /> 网络
            </div>
            <div class="flex items-center gap-4">
              <div class="flex items-center gap-1.5">
                <ArrowDownIcon class="w-4 h-4 text-emerald-500" />
                <span class="text-lg font-bold text-gray-700 dark:text-gray-300">
                  {{ formatBytesPerSec(server.metrics.network?.bytes_recv_per_sec || 0) }}
                </span>
              </div>
              <div class="flex items-center gap-1.5">
                <ArrowUpIcon class="w-4 h-4 text-blue-500" />
                <span class="text-lg font-bold text-gray-700 dark:text-gray-300">
                  {{ formatBytesPerSec(server.metrics.network?.bytes_sent_per_sec || 0) }}
                </span>
              </div>
            </div>
            <p class="text-xs text-gray-400 mt-0.5">下载 / 上传</p>
          </div>
          <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
            <div class="flex items-center gap-2 text-sm text-gray-500 mb-2">
              <ContainerIcon class="w-4 h-4" /> 容器
            </div>
            <p class="text-2xl font-bold text-emerald-600 dark:text-emerald-400">
              {{ server.metrics.docker_stats?.running || 0 }}
            </p>
            <p class="text-xs text-gray-400 mt-0.5">
              总计 {{ server.metrics.docker_stats?.total || 0 }} 容器
            </p>
          </div>
        </div>

        <!-- Disks -->
        <section v-if="server.metrics?.disks?.length" class="mb-6">
          <h2 class="flex items-center gap-2 text-lg font-semibold text-gray-700 dark:text-gray-300 mb-3">
            <HardDriveIcon class="w-5 h-5" /> 磁盘
          </h2>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            <div v-for="disk in server.metrics.disks" :key="disk.mount"
              class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-3">
              <div class="flex justify-between text-sm mb-1">
                <span class="font-medium text-gray-700 dark:text-gray-300">{{ disk.mount }}</span>
                <span class="text-gray-500">{{ disk.used_gb }}G / {{ disk.total_gb }}G</span>
              </div>
              <div class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                <div class="h-full rounded-full transition-all duration-500"
                  :class="disk.percent > 80 ? 'bg-red-500' : disk.percent > 60 ? 'bg-amber-500' : 'bg-blue-500'"
                  :style="{ width: `${Math.min(disk.percent, 100)}%` }" />
              </div>
              <p class="text-xs text-gray-400 mt-1">{{ disk.percent.toFixed(1) }}% 已用</p>
            </div>
          </div>
        </section>

        <!-- Docker containers -->
        <section v-if="server.metrics?.docker_containers?.length">
          <h2 class="flex items-center gap-2 text-lg font-semibold text-gray-700 dark:text-gray-300 mb-3">
            <ContainerIcon class="w-5 h-5" /> 容器列表
            <span class="text-xs text-gray-400 font-normal">({{ server.metrics.docker_containers.length }})</span>
          </h2>
          <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-gray-100 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
                  <th class="text-left py-2.5 px-4 text-gray-500 font-medium">状态</th>
                  <th class="text-left py-2.5 px-4 text-gray-500 font-medium">名称</th>
                  <th class="text-left py-2.5 px-4 text-gray-500 font-medium">镜像</th>
                  <th class="text-left py-2.5 px-4 text-gray-500 font-medium hidden md:table-cell">端口</th>
                  <th class="text-left py-2.5 px-4 text-gray-500 font-medium hidden md:table-cell">状态详情</th>
                  <th class="text-left py-2.5 px-4 text-gray-500 font-medium hidden lg:table-cell">创建时间</th>
                  <th class="text-left py-2.5 px-4 text-gray-500 font-medium w-24">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="c in server.metrics.docker_containers" :key="c.id"
                  class="border-b border-gray-50 dark:border-gray-700/50 hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                  <td class="py-2.5 px-4">
                    <component :is="containerIcon(c.state)"
                      class="w-4 h-4"
                      :class="c.state === 'running' ? 'text-emerald-500' : 'text-gray-400'" />
                  </td>
                  <td class="py-2.5 px-4 font-medium text-gray-800 dark:text-gray-200">{{ c.name }}</td>
                  <td class="py-2.5 px-4 text-gray-500 font-mono text-xs">{{ c.image }}</td>
                  <td class="py-2.5 px-4 text-gray-500 text-xs hidden md:table-cell">{{ c.ports || '-' }}</td>
                  <td class="py-2.5 px-4 text-gray-500 text-xs hidden md:table-cell">{{ c.status }}</td>
                  <td class="py-2.5 px-4 text-gray-400 text-xs hidden lg:table-cell">{{ formatTime(c.created) }}</td>
                  <td class="py-2.5 px-4">
                    <div class="flex gap-1 items-center">
                      <button
                        v-if="c.state !== 'running'"
                        @click="containerAction(c.id, 'start')"
                        :disabled="loadingContainers.has(c.id)"
                        class="p-1 rounded hover:bg-emerald-100 dark:hover:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400 disabled:opacity-30"
                        title="启动"
                      >
                        <LoaderIcon v-if="loadingContainers.has(c.id)" class="w-3.5 h-3.5 animate-spin" />
                        <PlayIcon v-else class="w-3.5 h-3.5" />
                      </button>
                      <button
                        v-if="c.state === 'running'"
                        @click="containerAction(c.id, 'stop')"
                        :disabled="loadingContainers.has(c.id)"
                        class="p-1 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-red-600 dark:text-red-400 disabled:opacity-30"
                        title="停止"
                      >
                        <LoaderIcon v-if="loadingContainers.has(c.id)" class="w-3.5 h-3.5 animate-spin" />
                        <SquareIcon v-else class="w-3.5 h-3.5" />
                      </button>
                      <button
                        @click="containerAction(c.id, 'restart')"
                        :disabled="loadingContainers.has(c.id)"
                        class="p-1 rounded hover:bg-amber-100 dark:hover:bg-amber-900/30 text-amber-600 dark:text-amber-400 disabled:opacity-30"
                        title="重启"
                      >
                        <LoaderIcon v-if="loadingContainers.has(c.id)" class="w-3.5 h-3.5 animate-spin" />
                        <RotateCwIcon v-else class="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- Empty state -->
        <div v-if="!server.metrics" class="text-center py-16 text-gray-400">
          <ServerIcon class="w-16 h-16 mx-auto mb-4 opacity-30" />
          <p class="text-lg">等待 Agent 上报数据...</p>
          <p class="text-sm mt-1">请确保 Agent 已部署并连接到 Gateway</p>
        </div>

        <div v-if="server.metrics && !server.metrics.docker_containers?.length" class="mt-6 text-center py-8 text-gray-400 border border-dashed border-gray-300 dark:border-gray-600 rounded-xl">
          <ContainerIcon class="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p class="text-sm">未检测到 Docker 容器</p>
          <p class="text-xs mt-0.5">Docker 可能未安装或 Agent 无权限访问 Docker 套接字</p>
        </div>
      </template>

      <!-- Server not found -->
      <div v-else class="text-center py-16 text-gray-400">
        <p class="text-lg">服务器未找到</p>
        <p class="text-sm mt-1">ID: {{ serverId }}</p>
      </div>
    </main>
  </div>
</template>
