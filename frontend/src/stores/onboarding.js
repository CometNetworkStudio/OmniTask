import { defineStore } from 'pinia'
import { useAuthStore } from './auth'

const BASE_KEY = 'omnitask_onboarding_done'

function storageKey() {
  const auth = useAuthStore()
  return `${BASE_KEY}:${auth.user?.id || auth.username || 'anon'}`
}

export const useOnboardingStore = defineStore('onboarding', {
  state: () => ({ done: true, step: 1 }),
  actions: {
    load() {
      this.done = localStorage.getItem(storageKey()) === '1'
      const step = Number(localStorage.getItem(`${storageKey()}:step`))
      this.step = step > 0 ? step : 1
    },
    finish() {
      this.done = true
      localStorage.setItem(storageKey(), '1')
    },
    setStep(step) {
      this.step = step
      localStorage.setItem(`${storageKey()}:step`, String(step))
    },
  },
})
