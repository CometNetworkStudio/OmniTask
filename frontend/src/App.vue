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
      title: '概览',
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

const PAGE_TITLES = {
  tasks: '任务',
  'task-detail': '任务详情',
  credentials: '凭据',
  questions: '题库',
  scripts: '脚本',
  users: '用户',
}
const currentTitle = computed(() => PAGE_TITLES[route.name] || 'OmniTask')
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
      <v-navigation-drawer
        v-model="drawer"
        :temporary="mobile"
        :permanent="!mobile"
        color="surface"
        class="app-drawer"
      >
        <div class="drawer-scroll overflow-y-auto hide-scrollbar">
          <v-list nav bg-color="transparent" class="px-2 pt-3 app-drawer__nav">
            <template v-for="(group, gi) in navGroups" :key="group.title">
              <v-divider v-if="gi > 0" class="my-2" />
              <v-list-subheader class="text-caption text-medium-emphasis px-3">
                {{ group.title }}
              </v-list-subheader>
              <v-list-item
                v-for="item in group.items"
                :key="item.to"
                :to="item.to"
                :prepend-icon="item.icon"
                :title="item.title"
                :value="item.to"
                color="primary"
                rounded="xl"
              />
            </template>
          </v-list>

          <div class="drawer-legal-spacer" aria-hidden="true" />

          <div class="px-4 pb-8 pt-4 text-caption text-medium-emphasis drawer-legal">
            <div>© CometNetworkStudio</div>
          </div>
        </div>

        <template #append>
          <v-divider />
          <v-list nav bg-color="transparent" class="px-2 py-2 flex-shrink-0 app-drawer__nav">
            <v-list-item
              :prepend-icon="isDark ? 'mdi-weather-sunny' : 'mdi-weather-night'"
              :title="isDark ? '浅色模式' : '深色模式'"
              color="primary"
              rounded="xl"
              @click="toggleTheme"
            />
            <v-list-item
              prepend-icon="mdi-logout"
              title="退出登录"
              color="primary"
              rounded="xl"
              @click="logout"
            />
          </v-list>
        </template>
      </v-navigation-drawer>

      <v-app-bar flat :height="64" color="surface">
        <v-app-bar-nav-icon @click="drawer = !drawer" />
        <v-app-bar-title class="text-body-1 font-weight-medium text-truncate">
          {{ currentTitle }}
        </v-app-bar-title>
        <v-spacer />

        <v-btn icon variant="text" :title="isDark ? '切换浅色' : '切换深色'" @click="toggleTheme">
          <v-icon :icon="isDark ? 'mdi-weather-sunny' : 'mdi-weather-night'" />
        </v-btn>

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

<style scoped>
.app-drawer :deep(.v-navigation-drawer__content) {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.drawer-scroll {
  flex: 1 1 auto;
  min-height: 0;
}

.drawer-legal-spacer {
  min-height: calc(100% - 16rem);
  pointer-events: none;
}

.hide-scrollbar {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.hide-scrollbar::-webkit-scrollbar {
  display: none;
}

.app-drawer :deep(.app-drawer__nav .v-list-item),
.app-drawer .drawer-legal {
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
}
</style>
