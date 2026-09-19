import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createVuetify } from 'vuetify'
import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import './style.css'
import App from './App.vue'
import router from './router'

const vuetify = createVuetify({
  theme: {
    defaultTheme: 'light',
    themes: {
      light: {
        colors: {
          primary: '#1867C0',
          secondary: '#5CBBF6',
          surface: '#FFFFFF',
          background: '#F4F6F8',
          success: '#2E7D32',
          error: '#C62828',
          warning: '#EF6C00',
          info: '#0277BD',
        },
      },
      dark: {
        colors: {
          primary: '#5CBBF6',
          secondary: '#1867C0',
          surface: '#1E1E1E',
          background: '#121212',
        },
      },
    },
  },
  defaults: {
    VCard: { rounded: 'lg' },
    VBtn: { variant: 'flat' },
    VTextField: { variant: 'outlined', density: 'comfortable' },
    VSelect: { variant: 'outlined', density: 'comfortable' },
  },
})

createApp(App).use(createPinia()).use(router).use(vuetify).mount('#app')
