<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { MonitorIcon, UserIcon, LockIcon, LogInIcon, Loader2Icon, SunIcon, MoonIcon } from '@lucide/vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const errorMsg = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!username.value || !password.value) {
    errorMsg.value = '请输入用户名和密码'
    return
  }

  loading.value = true
  errorMsg.value = ''

  try {
    await auth.login(username.value, password.value)
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (e: any) {
    if (e.response?.data?.error) {
      errorMsg.value = e.response.data.error
    } else {
      errorMsg.value = '登录失败，请检查网络连接'
    }
  } finally {
    loading.value = false
  }
}

// Theme toggle
const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center px-4 bg-gradient-to-br from-gray-50 to-blue-50 dark:from-gray-900 dark:to-gray-800">
    <!-- Theme toggle -->
    <button
      @click="toggleTheme"
      class="fixed top-4 right-4 p-2 rounded-lg bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-sm text-gray-500 dark:text-gray-400 hover:text-amber-500 dark:hover:text-amber-400 transition-colors"
      :title="isDark ? '切换日间模式' : '切换夜间模式'"
    >
      <SunIcon v-if="isDark" class="w-5 h-5" />
      <MoonIcon v-else class="w-5 h-5" />
    </button>
    <div class="w-full max-w-sm">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-8">
        <!-- Logo / Title -->
        <div class="text-center mb-8">
          <div class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-blue-600 flex items-center justify-center">
            <MonitorIcon class="w-8 h-8 text-white" />
          </div>
          <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
            ScaleObs
          </h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
            请登录以继续
          </p>
        </div>

        <!-- Error message -->
        <div
          v-if="errorMsg"
          class="bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded-lg p-3 mb-4 flex items-center gap-2"
        >
          <span>⚠</span> {{ errorMsg }}
        </div>

        <!-- Login form -->
        <form @submit.prevent="handleLogin" class="space-y-4">
          <div>
            <label
              for="username"
              class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              用户名
            </label>
            <div class="relative">
              <UserIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                id="username"
                v-model="username"
                type="text"
                autocomplete="username"
                class="w-full pl-10 pr-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg
                       bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
                       focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none
                       transition-colors"
                placeholder="admin"
                required
              />
            </div>
          </div>

          <div>
            <label
              for="password"
              class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              密码
            </label>
            <div class="relative">
              <LockIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                id="password"
                v-model="password"
                type="password"
                autocomplete="current-password"
                class="w-full pl-10 pr-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg
                       bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
                       focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none
                       transition-colors"
                placeholder="••••••••"
                required
              />
            </div>
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full py-2.5 px-4 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400
                   text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2
                   focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                   dark:focus:ring-offset-gray-800"
          >
            <Loader2Icon v-if="loading" class="w-4 h-4 animate-spin" />
            <LogInIcon v-else class="w-4 h-4" />
            {{ loading ? '登录中...' : '登录' }}
          </button>
        </form>
      </div>

      <p class="text-center text-xs text-gray-400 dark:text-gray-500 mt-6">
        Server Ops Portal v0.1.0
      </p>
    </div>
  </div>
</template>
