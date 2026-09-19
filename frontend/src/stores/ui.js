import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', {
  state: () => ({
    snackbar: { show: false, text: '', color: 'success' },
  }),
  actions: {
    notify(text, color = 'success') {
      this.snackbar = { show: true, text, color }
    },
    closeSnackbar() {
      this.snackbar.show = false
    },
  },
})
