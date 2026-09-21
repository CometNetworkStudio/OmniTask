<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay, useTheme } from 'vuetify'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()
const route = useRoute()
const theme = useTheme()
const { mobile } = useDisplay()

const drawer = ref(!mobile.value)
const userMenu = ref(false)

watch(mobile, (isMobile) => {
  drawer.value = !isMobile
})
watch(
  () => route.fullPath,
  () => {
    if (mobile.value) drawer.value = false
  },
)

const navGroups = computed(() => {
  const groups = [
    {
      title: '任务',
      items: [
        { title: '任务', to: '/', icon: 'mdi-clipboard-list-outline' },
        { title: '凭据', to: '/credentials', icon: 'mdi-key-variant' },
        { title: '题库', to: '/questions', icon: 'mdi-database-outline' },
      ],
    },
  ]
  if (auth.isAdmin) {
    groups.push({
      title: '管理',
      items: [
        { title: '脚本', to: '/scripts', icon: 'mdi-puzzle-outline' },
        { title: '用户', to: '/users', icon: 'mdi-account-group-outline' },
      ],
    })
  }
  return groups
})

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

onMounted(() => {
  if (auth.isAuthenticated) {
    auth.fetchMe().catch(() => {})
  }
})
</script>

<template>
  <v-app>
    <template v-if="auth.isAuthenticated">
      <v-navigation-drawer v-model="drawer" :temporary="mobile">
        <v-list nav>
          <v-list-item
            title="OmniTask"
            subtitle="万务通"
            prepend-icon="mdi-auto-fix"
            class="text-primary"
          />
          <template v-for="(group, gi) in navGroups" :key="group.title">
            <v-divider v-if="gi > 0" class="my-2" />
            <v-list-subheader>{{ group.title }}</v-list-subheader>
            <v-list-item
              v-for="item in group.items"
              :key="item.to"
              :to="item.to"
              :prepend-icon="item.icon"
              :title="item.title"
              color="primary"
            />
          </template>
        </v-list>

        <template #append>
          <v-divider />
          <v-list density="compact">
            <v-list-item
              :title="auth.username"
              subtitle="当前账号"
              prepend-icon="mdi-account-circle-outline"
            />
          </v-list>
          <div class="pa-3 pt-1 d-flex flex-column ga-2">
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
            <v-btn icon variant="text" v-bind="props" class="me-2" aria-label="用户菜单">
              <v-avatar color="primary" size="36">
                <span class="text-body-2 font-weight-medium">{{ avatarLetter }}</span>
              </v-avatar>
            </v-btn>
          </template>
          <v-list>
            <v-list-item title="凭据管理" prepend-icon="mdi-key-variant" @click="go('/credentials')" />
            <v-list-item title="题库" prepend-icon="mdi-database-outline" @click="go('/questions')" />
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
