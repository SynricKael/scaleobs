import axios, { AxiosInstance } from 'axios'
import type { LoginRequest, LoginResponse, Service, ServerStatus, AgentServerStatus } from '@/types'

// In dev (Vite + Tauri devUrl): requests go to Vite dev server which proxies /api to Gateway
// In prod (Tauri embedded tauri:// protocol): use absolute URL so requests reach Gateway
const BASE = import.meta.env.DEV ? '' : 'http://localhost:8080'

// Create axios instance
const api: AxiosInstance = axios.create({
  baseURL: BASE,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: add auth token
api.interceptors.request.use((config) => {
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
      window.location.href = '/login'
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
  }): Promise<void> {
    return api.patch(`/api/servers/${serverId}/settings`, data)
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
