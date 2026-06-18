import { defineStore } from 'pinia'
import { ref, computed, onUnmounted } from 'vue'
import { servicesApi, serversApi } from '@/api'
import type { Service, ServerStatus } from '@/types'

export const useGatewayStore = defineStore('gateway', () => {
  const services = ref<Service[]>([])
  const servers = ref<ServerStatus[]>([])
  const loading = ref(false)
  const error = ref('')
  let refreshTimer: ReturnType<typeof setInterval> | null = null

  const servicesByCategory = computed(() => {
    const groups: Record<string, Service[]> = {}
    for (const svc of services.value) {
      if (!groups[svc.category]) {
        groups[svc.category] = []
      }
      groups[svc.category].push(svc)
    }
    return groups
  })

  const categories = computed(() => {
    return Object.keys(servicesByCategory.value).sort()
  })

  async function fetchServices(): Promise<void> {
    try {
      services.value = await servicesApi.list()
    } catch (e: any) {
      console.error('Failed to fetch services:', e)
    }
  }

  async function fetchServers(): Promise<void> {
    try {
      servers.value = await serversApi.list()
    } catch (e: any) {
      console.error('Failed to fetch servers:', e)
    }
  }

  async function refresh(): Promise<void> {
    if (loading.value) return
    loading.value = true
    error.value = ''
    try {
      await Promise.all([fetchServices(), fetchServers()])
    } catch (e: any) {
      error.value = e.message || 'Failed to load data'
    } finally {
      loading.value = false
    }
  }

  function startAutoRefresh(intervalMs = 10000): void {
    stopAutoRefresh()
    refresh()
    refreshTimer = setInterval(refresh, intervalMs)
  }

  function stopAutoRefresh(): void {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }

  function getService(id: string): Service | undefined {
    return services.value.find((s) => s.id === id)
  }

  // Cleanup on store disposal
  onUnmounted(() => {
    stopAutoRefresh()
  })

  return {
    services,
    servers,
    loading,
    error,
    servicesByCategory,
    categories,
    refresh,
    startAutoRefresh,
    stopAutoRefresh,
    getService,
  }
})
