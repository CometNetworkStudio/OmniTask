import { defineStore } from 'pinia'
import { authApi } from '@/api/services/auth'

const TOKEN_KEY = 'omnitask_token'
const USER_KEY = 'omnitask_user'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem(TOKEN_KEY) || '',
    user: JSON.parse(localStorage.getItem(USER_KEY) || 'null'),
  }),
  getters: {
    isAuthenticated: (state) => Boolean(state.token),
    username: (state) => state.user?.username || '',
  },
  actions: {
    async login(payload) {
      this.setSession(await authApi.login(payload))
    },
    async register(payload) {
      this.setSession(await authApi.register(payload))
    },
    setSession(data) {
      this.token = data.token
      this.user = { id: data.user_id, username: data.username }
      localStorage.setItem(TOKEN_KEY, this.token)
      localStorage.setItem(USER_KEY, JSON.stringify(this.user))
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USER_KEY)
    },
  },
})
