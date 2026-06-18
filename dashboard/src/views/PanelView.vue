<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGatewayStore } from '@/stores/gateway'
import {
  ArrowLeftIcon,
  ExternalLinkIcon,
  RefreshCwIcon,
  Loader2Icon,
  FrownIcon,
} from '@lucide/vue'

const route = useRoute()
const router = useRouter()
const gw = useGatewayStore()

const iframeLoaded = ref(false)
const iframeError = ref(false)
const iframeUrl = ref('')
const iframeKey = ref(0) // Increment to force iframe full reload

const serviceId = computed(() => route.params.id as string)
const service = computed(() => gw.getService(serviceId.value))

// Set native window title when entering panel
async function setWindowTitle(name: string) {
  try {
    const { getCurrentWindow } = await import('@tauri-apps/api/window')
    await getCurrentWindow().setTitle(name + ' - 服务器运维平台')
  } catch {
    document.title = name + ' - 服务器运维平台'
  }
}

onMounted(() => {
  if (!service.value) {
    gw.refresh().then(() => {
      if (!gw.getService(serviceId.value)) {
        iframeError.value = true
        return
      }
      buildIframeUrl()
    })
  } else {
    buildIframeUrl()
  }
})

function buildIframeUrl() {
  if (!service.value) return
  iframeUrl.value = service.value.url
  setWindowTitle(service.value.name)
}

function refreshIframe() {
  iframeLoaded.value = false
  iframeError.value = false
  iframeKey.value++ // Vue destroys & recreates the iframe
}

function openInBrowser() {
  if (iframeUrl.value) {
    window.open(iframeUrl.value, '_blank')
  }
}

function goBack() {
  // Restore window title
  setWindowTitle('服务器运维平台')
  router.push('/')
}

function onIframeLoad() {
  iframeLoaded.value = true
}

function onIframeError() {
  iframeError.value = true
  iframeLoaded.value = true
}
</script>

<template>
  <div class="h-screen flex flex-col bg-white dark:bg-gray-900">
    <!-- Top bar — draggable like a native title bar -->
    <header
      data-tauri-drag-region
      class="flex items-center justify-between px-4 h-12 border-b border-gray-200 dark:border-gray-700
             bg-white dark:bg-gray-800 shrink-0 select-none"
    >
      <div class="flex items-center gap-2">
        <button
          @click="goBack"
          class="flex items-center justify-center w-8 h-8 rounded-md text-gray-500
                 hover:text-gray-700 hover:bg-gray-100 dark:text-gray-400
                 dark:hover:text-gray-200 dark:hover:bg-gray-700 transition-colors"
          title="返回"
        >
          <ArrowLeftIcon class="w-4 h-4" />
        </button>
        <span class="text-gray-300 dark:text-gray-600">|</span>
        <h1 class="text-sm font-medium text-gray-800 dark:text-gray-100">
          {{ service?.name || '加载中...' }}
        </h1>
        <!-- Status badge (compact) -->
        <span
          v-if="service"
          class="text-xs px-1.5 py-0.5 rounded-full ml-1"
          :class="{
            'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400': service.status === 'online',
            'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400': service.status === 'degraded',
            'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400': service.status === 'offline',
          }"
        >
          {{ service.status === 'online' ? '在线' : service.status === 'degraded' ? '告警' : '离线' }}
        </span>
      </div>

      <div class="flex items-center gap-1">
        <button
          @click="refreshIframe"
          class="flex items-center justify-center w-8 h-8 rounded-md text-gray-500
                 hover:text-blue-600 hover:bg-blue-50 dark:text-gray-400
                 dark:hover:text-blue-400 dark:hover:bg-blue-900/20 transition-colors"
          title="刷新面板"
        >
          <RefreshCwIcon class="w-4 h-4" />
        </button>
        <button
          @click="openInBrowser"
          class="flex items-center justify-center w-8 h-8 rounded-md text-gray-500
                 hover:text-blue-600 hover:bg-blue-50 dark:text-gray-400
                 dark:hover:text-blue-400 dark:hover:bg-blue-900/20 transition-colors"
          title="在浏览器中打开"
        >
          <ExternalLinkIcon class="w-4 h-4" />
        </button>
      </div>
    </header>

    <!-- Loading state -->
    <div
      v-if="!iframeLoaded && !iframeError"
      class="flex-1 flex items-center justify-center"
    >
      <div class="text-center">
        <Loader2Icon class="w-10 h-10 mx-auto mb-4 animate-spin text-gray-400" />
        <p class="text-gray-500 dark:text-gray-400">加载中...</p>
      </div>
    </div>

    <!-- Error state -->
    <div
      v-if="iframeError"
      class="flex-1 flex items-center justify-center"
    >
      <div class="text-center max-w-md">
        <FrownIcon class="w-16 h-16 mx-auto mb-4 text-gray-300 dark:text-gray-600" />
        <h2 class="text-xl font-semibold text-gray-700 dark:text-gray-300 mb-2">
          无法加载面板
        </h2>
        <p class="text-gray-500 dark:text-gray-400 mb-4">
          {{ service ? '该服务可能未运行或暂时不可用' : '未找到该服务' }}
        </p>
        <div class="flex gap-3 justify-center">
          <button
            @click="refreshIframe"
            class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm transition-colors"
          >
            重试
          </button>
          <button
            @click="goBack"
            class="px-4 py-2 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600
                   text-gray-700 dark:text-gray-300 rounded-lg text-sm transition-colors"
          >
            返回首页
          </button>
        </div>
      </div>
    </div>

    <!-- Iframe (key forces real reload not cache) -->
    <iframe
      v-if="iframeUrl && !iframeError"
      :key="iframeKey"
      :src="iframeUrl"
      class="flex-1 w-full border-none"
      :class="{ invisible: !iframeLoaded }"
      @load="onIframeLoad"
      @error="onIframeError"
      sandbox="allow-scripts allow-forms allow-same-origin allow-popups"
    />
  </div>
</template>
