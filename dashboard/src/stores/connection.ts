import { defineStore } from 'pinia'
import { ref } from 'vue'

const STORAGE_KEY = 'scaleobs_gateway_url'

export const useConnectionStore = defineStore('connection', () => {
  const savedUrl = localStorage.getItem(STORAGE_KEY) || ''

  const mode = ref<'local' | 'remote'>(savedUrl ? 'remote' : 'local')
  const gatewayUrl = ref(savedUrl)
  const status = ref<'unknown' | 'connected' | 'disconnected' | 'testing'>('unknown')
  const error = ref('')
  const latency = ref(0) // ms

  function setLocal() {
    mode.value = 'local'
    gatewayUrl.value = ''
    localStorage.removeItem(STORAGE_KEY)
  }

  function setRemote(url: string) {
    // Normalize: strip trailing slash
    const normalized = url.replace(/\/+$/, '')
    mode.value = 'remote'
    gatewayUrl.value = normalized
    localStorage.setItem(STORAGE_KEY, normalized)
  }

  function getBaseUrl(): string {
    return mode.value === 'remote' ? gatewayUrl.value : ''
  }

  async function testConnection(url?: string): Promise<boolean> {
    const target = url || getBaseUrl() || 'http://localhost:8080'
    const testUrl = `${target}/api/health`

    status.value = 'testing'
    error.value = ''
    latency.value = 0

    const start = performance.now()
    try {
      const res = await fetch(testUrl, { method: 'GET', signal: AbortSignal.timeout(8000) })
      latency.value = Math.round(performance.now() - start)
      if (res.ok) {
        status.value = 'connected'
        return true
      }
      throw new Error(`HTTP ${res.status}`)
    } catch (e: any) {
      status.value = 'disconnected'
      error.value = e.name === 'TimeoutError'
        ? '连接超时（8秒）'
        : e.message || '连接失败'
      latency.value = 0
      return false
    }
  }

  return {
    mode,
    gatewayUrl,
    status,
    error,
    latency,
    setLocal,
    setRemote,
    testConnection,
    getBaseUrl,
  }
})
