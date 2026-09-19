import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', {
  state: () => ({
    messages: [],
  }),
  actions: {
    notify(text, color = 'success') {
      this.messages.push({ text, color })
    },
    clear() {
      this.messages = []
    },
  },
})
