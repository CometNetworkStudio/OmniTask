import { defineStore } from 'pinia'

const KEY = 'omnitask_onboarding_done'

export const useOnboardingStore = defineStore('onboarding', {
  state: () => ({ done: localStorage.getItem(KEY) === '1' }),
  actions: {
    finish() {
      this.done = true
      localStorage.setItem(KEY, '1')
    },
    reset() {
      this.done = false
      localStorage.removeItem(KEY)
    },
  },
})
