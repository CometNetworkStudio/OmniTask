<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()

const navItems = [
  { title: '任务', to: '/', icon: 'mdi-clipboard-list-outline' },
  { title: '凭据', to: '/credentials', icon: 'mdi-key-variant' },
  { title: '题库', to: '/questions', icon: 'mdi-database-outline' },
]

function logout() {
  auth.logout()
  ui.notify('已退出登录', 'info')
  router.push({ name: 'login' })
}
</script>

<template>
  <v-app>
    <template v-if="auth.isAuthenticated">
      <v-navigation-drawer permanent>
        <v-list nav>
          <v-list-item
            title="OmniTask"
            subtitle="万务通"
            prepend-icon="mdi-auto-fix"
            class="text-primary"
          />
          <v-divider class="my-2" />
          <v-list-item
            v-for="item in navItems"
            :key="item.to"
            :to="item.to"
            :prepend-icon="item.icon"
            :title="item.title"
            color="primary"
          />
        </v-list>
      </v-navigation-drawer>

      <v-app-bar flat color="surface">
        <v-app-bar-title>{{ auth.username }}</v-app-bar-title>
        <v-spacer />
        <v-btn variant="text" prepend-icon="mdi-logout" @click="logout">退出</v-btn>
      </v-app-bar>
    </template>

    <v-main>
      <router-view />
    </v-main>

    <v-snackbar v-model="ui.snackbar.show" :color="ui.snackbar.color" timeout="3000">
      {{ ui.snackbar.text }}
    </v-snackbar>
  </v-app>
</template>
