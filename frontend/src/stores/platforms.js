import { defineStore } from 'pinia'
import { platformsApi } from '@/api/services/platforms'

export const usePlatformsStore = defineStore('platforms', {
  state: () => ({ list: [], loaded: false, loading: false }),
  getters: {
    options: (state) => state.list.map((p) => ({ title: p.name, value: p.id })),
  },
  actions: {
    async load(force = false) {
      if (this.loaded && !force) return this.list
      this.loading = true
      try {
        this.list = await platformsApi.list()
        this.loaded = true
      } finally {
        this.loading = false
      }
      return this.list
    },
    async refresh() {
      return this.load(true)
    },
    platform(id) {
      return this.list.find((p) => p.id === id) || null
    },
    name(id) {
      return this.platform(id)?.name || id
    },
    actionName(platformId, actionId) {
      const action = this.platform(platformId)?.actions?.find((item) => item.id === actionId)
      return action?.name || actionId
    },
    action(platformId, actionId) {
      return this.platform(platformId)?.actions?.find((item) => item.id === actionId) || null
    },
  },
})
