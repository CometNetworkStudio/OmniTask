import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createVuetify } from 'vuetify'
import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import './style.css'
import App from './App.vue'
import router from './router'

const savedTheme = localStorage.getItem('omnitask_theme')
const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches

const vuetify = createVuetify({
  icons: {
    defaultSet: 'mdi',
  },
  theme: {
    defaultTheme: savedTheme || (systemDark ? 'dark' : 'light'),
    themes: {
      light: {
        colors: {
          primary: '#1867C0',
          'primary-light': '#E3F0FC',
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
          'primary-light': '#0E2A44',
          secondary: '#1867C0',
          surface: '#1E1E1E',
          background: '#121212',
        },
      },
    },
  },
  defaults: {
    VBtn: { variant: 'elevated', rounded: 'xl' },
    VCard: { variant: 'elevated', rounded: 'xl', elevation: 2 },
    VAlert: { variant: 'elevated', rounded: 'xl' },
    VTextField: { variant: 'outlined', density: 'comfortable', rounded: 'xl' },
    VSelect: { variant: 'outlined', density: 'comfortable', rounded: 'xl' },
    VList: { rounded: 'xl' },
    VListItem: { rounded: 'xl' },
    VSwitch: { inset: 'material' },
  },
})

createApp(App).use(createPinia()).use(router).use(vuetify).mount('#app')
