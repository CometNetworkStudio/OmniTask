import { defineStore } from 'pinia'
import { metaApi } from '@/api/services/meta'

let loading = null

export const useMetaStore = defineStore('meta', {
  state: () => ({
    data: null,
    loaded: false,
  }),
  getters: {
    registration: (state) => state.data?.registration || 'open',
    inviteRequired: (state) => Boolean(state.data?.invite_required),
    captcha: (state) => Boolean(state.data?.captcha),
    limits: (state) => state.data?.limits || {},
    features: (state) => state.data?.features || {},
  },
  actions: {
    async load(force = false) {
      if (this.loaded && !force) return this.data
      if (loading) return loading
      loading = metaApi
        .get()
        .then((data) => {
          this.data = data
          this.loaded = true
          return data
        })
        .catch(() => {
          this.data = null
          this.loaded = true
          return null
        })
        .finally(() => {
          loading = null
        })
      return loading
    },
  },
})
