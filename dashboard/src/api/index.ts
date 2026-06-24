import axios, { AxiosInstance } from 'axios'
import type { LoginRequest, LoginResponse, Service, ServerStatus, AgentServerStatus } from '@/types'

const GW_URL_KEY = 'scaleobs_gateway_url'

// Resolve the effective base URL for API calls.
// Returns empty string (same-origin) for local mode, or the remote Gateway URL.
export function getApiBase(): string {
  const saved = localStorage.getItem(GW_URL_KEY)
  if (saved && saved.trim()) return saved.trim()
  return import.meta.env.DEV ? '' : 'http://localhost:8080'
}

// Create axios instance (baseURL is set dynamically in the request interceptor)
const api: AxiosInstance = axios.create({
  baseURL: '',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: add auth token + dynamic base URL
api.interceptors.request.use((config) => {
  const base = getApiBase()
  if (base) {
    config.baseURL = base
  } else {
    // In dev mode the Vite proxy handles it; otherwise use localhost
    config.baseURL = import.meta.env.DEV ? '' : 'http://localhost:8080'
  }
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      localStorage.removeItem('username')
      localStorage.removeItem('expiresAt')
      const base = getApiBase()
      window.location.href = base ? `${base}/login` : '/login'
    }
    return Promise.reject(error)
  }
)

export const authApi = {
  login(data: LoginRequest): Promise<LoginResponse> {
    return api.post('/api/auth/login', data).then((res) => res.data)
  },
}

export const servicesApi = {
  list(): Promise<Service[]> {
    return api.get('/api/services').then((res) => res.data)
  },
}

export const serversApi = {
  list(): Promise<ServerStatus[]> {
    return api.get('/api/servers').then((res) => res.data)
  },
  updateSettings(serverId: string, data: {
    group?: string
    ssh_host?: string
    ssh_port?: number
    ssh_user?: string
    ssh_pass?: string
    ssh_key_path?: string
    docker_mode?: string
    docker_host?: string
    docker_port?: number
    docker_tls?: boolean
    docker_tls_ca?: string
    docker_tls_cert?: string
    docker_tls_key?: string
  }): Promise<void> {
    return api.patch(`/api/servers/${serverId}/settings`, data)
  },
  remove(serverId: string): Promise<void> {
    return api.delete(`/api/servers/${serverId}`)
  },
}

export const agentServersApi = {
  list(): Promise<AgentServerStatus[]> {
    return api.get('/api/agent-servers').then((res) => res.data)
  },
  add(data: { name: string; url: string; user?: string; password?: string }): Promise<void> {
    return api.post('/api/agent-servers', data)
  },
}

export const dockerApi = {
  containerAction(serverId: string, containerId: string, action: 'start' | 'stop' | 'restart'): Promise<void> {
    return api.post(`/api/docker/${serverId}/containers/${containerId}/${action}`)
  },
}

export default api
