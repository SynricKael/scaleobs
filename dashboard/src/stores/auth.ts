import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api'
import type { LoginResponse } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref(localStorage.getItem('username') || '')
  const expiresAt = ref(Number(localStorage.getItem('expiresAt') || '0'))

  const isAuthenticated = computed(() => {
    if (!token.value) return false
    // Check if token is expired
    if (expiresAt.value && Date.now() / 1000 > expiresAt.value) {
      logout()
      return false
    }
    return true
  })

  async function login(user: string, password: string): Promise<void> {
    const res: LoginResponse = await authApi.login({
      username: user,
      password,
    })
    token.value = res.token
    username.value = res.username
    expiresAt.value = res.expires_at

    localStorage.setItem('token', res.token)
    localStorage.setItem('username', res.username)
    localStorage.setItem('expiresAt', String(res.expires_at))
  }

  function logout(): void {
    token.value = ''
    username.value = ''
    expiresAt.value = 0
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('expiresAt')
  }

  return {
    token,
    username,
    expiresAt,
    isAuthenticated,
    login,
    logout,
  }
})
