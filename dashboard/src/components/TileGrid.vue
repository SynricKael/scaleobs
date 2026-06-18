<script setup lang="ts">
import { useRouter } from 'vue-router'
import { computed } from 'vue'
import type { Service } from '@/types'
import StatusBadge from './StatusBadge.vue'

const GATEWAY_URL = 'http://localhost:8080'
import {
  GlobeIcon,
  BoxIcon,
  BarChart3Icon,
  PieChartIcon,
  DatabaseIcon,
  HardDriveIcon,
  ShieldCheckIcon,
  WrenchIcon,
  ContainerIcon,
  ServerIcon,
  MonitorIcon,
  NetworkIcon,
} from '@lucide/vue'

const props = defineProps<{
  categories: string[]
  servicesByCategory: Record<string, Service[]>
}>()

const router = useRouter()

async function openService(svc: Service) {
  if (svc.open_method === 'browser') {
    window.open(svc.url, '_blank')
    return
  }

  // Tauri mode: invoke native open_panel command via window.__TAURI_INTERNALS__
  const tauriInvoke = (window as any).__TAURI_INTERNALS__?.invoke
  if (tauriInvoke) {
    // Use panel_url if set (direct URL bypassing proxy), otherwise build proxy URL
    const panelUrl = svc.panel_url || (svc.url.startsWith('http') ? svc.url : `${GATEWAY_URL}${svc.url}`)
    try {
      await tauriInvoke('open_panel', { url: panelUrl, title: svc.name })
      return
    } catch (err) {
      console.error('[TileGrid] open_panel failed:', err)
      // Fall through to iframe route
    }
  }

  // Browser fallback: build proxy URL
  const fullUrl = svc.url.startsWith('http') ? svc.url : `${GATEWAY_URL}${svc.url}`

  // Browser fallback: navigate to iframe panel view
  router.push(`/panel/${svc.id}`)
}

function getCategoryIcon(cat: string): string {
  const icons: Record<string, string> = {
    '网络': 'globe',
    '容器': 'cube',
    '监控': 'chart-bar',
    '数据库': 'database',
    '存储': 'hard-drive',
    '安全': 'shield',
    '开发': 'wrench',
    '其他': 'box',
  }
  return icons[cat] || 'box'
}

function getServiceIcon(svc: Service): string {
  const iconMap: Record<string, string> = {
    'server': 'server',
    'cube': 'cube',
    'chart-bar': 'chart-bar',
    'chart-pie': 'chart-pie',
    'network': 'network',
    'database': 'database',
    'shield': 'shield',
    'wrench': 'wrench',
    'hard-drive': 'hard-drive',
    'box': 'box',
    'monitor': 'monitor',
  }
  return iconMap[svc.icon] || 'box'
}

const iconComponents: Record<string, any> = {
  'globe': GlobeIcon,
  'cube': ContainerIcon,
  'chart-bar': BarChart3Icon,
  'chart-pie': PieChartIcon,
  'database': DatabaseIcon,
  'hard-drive': HardDriveIcon,
  'shield': ShieldCheckIcon,
  'wrench': WrenchIcon,
  'box': BoxIcon,
  'server': ServerIcon,
  'monitor': MonitorIcon,
  'network': NetworkIcon,
}
</script>

<template>
  <div class="space-y-8"><!-- BUILD-v3: Tauri native WebView panels -->
    <div v-for="cat in categories" :key="cat" class="category-section">
      <div class="flex items-center gap-2 mb-4">
        <component
          :is="iconComponents[getCategoryIcon(cat)] || iconComponents['box']"
          class="w-5 h-5 text-gray-500 dark:text-gray-400"
        />
        <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
          {{ cat }}
        </h2>
        <span class="text-xs text-gray-400 dark:text-gray-500 ml-1">
          ({{ servicesByCategory[cat]?.length || 0 }})
        </span>
      </div>

      <div
        class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"
      >
        <div
          v-for="svc in servicesByCategory[cat]"
          :key="svc.id"
          @click="openService(svc)"
          class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700
                 p-4 cursor-pointer hover:shadow-md hover:border-blue-300 dark:hover:border-blue-600
                 transition-all duration-200 active:scale-[0.98]"
        >
          <div class="flex items-start justify-between mb-3">
            <div
              class="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/30
                     flex items-center justify-center"
            >
              <component
                :is="iconComponents[getServiceIcon(svc)] || iconComponents['box']"
                class="w-5 h-5 text-blue-600 dark:text-blue-400"
              />
            </div>
            <StatusBadge :status="svc.status" />
          </div>

          <h3 class="font-medium text-gray-900 dark:text-gray-100 text-sm mb-1">
            {{ svc.name }}
          </h3>
          <p class="text-xs text-gray-400 dark:text-gray-500 truncate">
            {{ svc.url }}
          </p>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="categories.length === 0"
      class="text-center py-16 text-gray-400 dark:text-gray-500"
    >
      <CubeIcon class="w-16 h-16 mx-auto mb-4 opacity-30" />
      <p class="text-lg">暂无注册的服务</p>
      <p class="text-sm mt-1">请在 services.yml 中添加服务配置</p>
    </div>
  </div>
</template>
