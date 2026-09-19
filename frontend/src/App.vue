<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTheme } from 'vuetify'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()
const theme = useTheme()

const drawer = ref(true)
const userMenu = ref(false)

const navItems = [
  { title: '任务', to: '/', icon: 'mdi-clipboard-list-outline' },
  { title: '凭据', to: '/credentials', icon: 'mdi-key-variant' },
  { title: '题库', to: '/questions', icon: 'mdi-database-outline' },
]

const avatarLetter = computed(() => (auth.username || '?').charAt(0).toUpperCase())
const isDark = computed(() => theme.global.name.value === 'dark')

function toggleTheme() {
  const next = isDark.value ? 'light' : 'dark'
  theme.global.name.value = next
  localStorage.setItem('omnitask_theme', next)
}

function logout() {
  userMenu.value = false
  auth.logout()
  ui.notify('已退出登录', 'info')
  router.push({ name: 'login' })
}

function go(path) {
  userMenu.value = false
  router.push(path)
}
</script>

<template>
  <v-app>
    <template v-if="auth.isAuthenticated">
      <v-navigation-drawer v-model="drawer">
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

        <template #append>
          <v-divider />
          <v-list density="compact" class="mt-2">
            <v-list-item
              :title="auth.username"
              subtitle="当前账号"
              prepend-icon="mdi-account-circle-outline"
            />
          </v-list>
          <div class="pa-3 pt-2 d-flex flex-column ga-2">
            <v-btn block color="primary" prepend-icon="mdi-logout" @click="logout">
              退出登录
            </v-btn>
            <div class="text-caption text-center text-medium-emphasis">© CometNetworkStudio</div>
          </div>
        </template>
      </v-navigation-drawer>

      <v-app-bar flat color="surface">
        <v-app-bar-nav-icon @click="drawer = !drawer" />
        <v-app-bar-title>OmniTask</v-app-bar-title>
        <v-spacer />

        <v-btn
          :icon="isDark ? 'mdi-weather-sunny' : 'mdi-weather-night'"
          variant="text"
          :title="isDark ? '切换浅色' : '切换深色'"
          @click="toggleTheme"
        />

        <v-menu v-model="userMenu" location="bottom end">
          <template #activator="{ props }">
            <v-avatar v-bind="props" color="primary" class="ms-2" size="40">
              <span class="text-body-1 font-weight-medium">{{ avatarLetter }}</span>
            </v-avatar>
          </template>
          <v-list>
            <v-list-item
              title="凭据管理"
              prepend-icon="mdi-key-variant"
              @click="go('/credentials')"
            />
            <v-list-item
              title="题库"
              prepend-icon="mdi-database-outline"
              @click="go('/questions')"
            />
            <v-divider />
            <v-list-item title="退出登录" prepend-icon="mdi-logout" @click="logout" />
          </v-list>
        </v-menu>
      </v-app-bar>
    </template>

    <v-main>
      <router-view />
    </v-main>

    <v-snackbar-queue
      v-model="ui.messages"
      location="top right"
      timeout="3000"
      total-visible="4"
      transition="bouncy-slide-auto"
      closable
      contained
    />
  </v-app>
</template>
