import { defineStore } from 'pinia'
import { authApi } from '@/api/services/auth'

let bootPromise = null

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null,
    loaded: false,
  }),
  getters: {
    isAuthenticated: (state) => Boolean(state.user),
    isAdmin: (state) => state.user?.role === 'admin',
    username: (state) => state.user?.username || '',
  },
  actions: {
    async login(payload) {
      const data = await authApi.login(payload)
      this.user = data.user
      this.loaded = true
    },
    async register(payload) {
      const data = await authApi.register(payload)
      this.user = data.user
      this.loaded = true
    },
    // bootstrap 在路由守卫中调用，通过服务端会话确认当前用户，避免信任本地缓存。
    async bootstrap(force = false) {
      if (this.loaded && !force) return this.user
      if (bootPromise) return bootPromise
      bootPromise = authApi
        .me()
        .then((data) => {
          this.user = { id: data.id, username: data.username, role: data.role || 'user' }
          return this.user
        })
        .catch(() => {
          this.user = null
          return null
        })
        .finally(() => {
          this.loaded = true
          bootPromise = null
        })
      return bootPromise
    },
    async logout() {
      try {
        await authApi.logout()
      } catch {
        // 会话可能已经过期，本地状态仍需清理
      }
      this.clear()
    },
    clear() {
      this.user = null
      this.loaded = true
    },
  },
})
