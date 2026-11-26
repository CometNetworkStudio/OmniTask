<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay, useTheme } from 'vuetify'
import { useAuthStore } from '@/stores/auth'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { preferencesApi } from '@/api/services/preferences'

const auth = useAuthStore()
const platforms = usePlatformsStore()
const ui = useUiStore()
const router = useRouter()
const route = useRoute()
const theme = useTheme()
const { mobile } = useDisplay()

const drawer = ref(!mobile.value)
const userMenu = ref(false)
const openGroups = reactive({ '/tasks': false })
const pinnedPlatformIds = ref([])
const activeNavKey = ref('')
let navSource = ''

function isPinned(platformId) {
  return pinnedPlatformIds.value.includes(platformId)
}

const pinnedItems = computed(() =>
  pinnedPlatformIds.value
    .map((id) => platforms.list.find((p) => p.id === id))
    .filter(Boolean)
    .map((p) => ({
      title: p.name,
      to: `/platforms/${p.id}`,
      icon: p.icon || 'mdi-puzzle-outline',
      platformId: p.id,
    })),
)

function toggleGroup(key) {
  openGroups[key] = !openGroups[key]
}

async function loadPreferences() {
  try {
    const data = await preferencesApi.get()
    pinnedPlatformIds.value = Array.isArray(data?.pinned_platforms) ? data.pinned_platforms : []
  } catch {
    pinnedPlatformIds.value = []
  }
}

async function togglePin(platformId) {
  const wasPinned = isPinned(platformId)
  const next = wasPinned
    ? pinnedPlatformIds.value.filter((id) => id !== platformId)
    : [...pinnedPlatformIds.value, platformId]
  const previous = [...pinnedPlatformIds.value]
  pinnedPlatformIds.value = next
  if (wasPinned && activeNavKey.value === `pin:${platformId}`) {
    activeNavKey.value = route.path === `/platforms/${platformId}` ? `task:${platformId}` : ''
  }
  try {
    await preferencesApi.update(next)
  } catch (err) {
    pinnedPlatformIds.value = previous
    ui.notify(err.message, 'error')
  }
}

function navigatePlatform(platformId, source) {
  activeNavKey.value = `${source}:${platformId}`
  const target = `/platforms/${platformId}`
  if (route.path !== target) {
    navSource = source
    router.push(target)
  }
}

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
      items: [{ title: '首页', to: '/home', icon: 'mdi-home-outline' }],
    },
  ]
  if (pinnedItems.value.length) {
    groups.push({ title: 'Pin', items: pinnedItems.value })
  }
  groups.push({
    title: '任务',
    items: [
      {
        title: '任务',
        to: '/tasks',
        icon: 'mdi-clipboard-list-outline',
        children: platforms.list.map((p) => ({
          title: p.name,
          to: `/platforms/${p.id}`,
          icon: p.icon || 'mdi-puzzle-outline',
          platformId: p.id,
        })),
      },
      { title: '凭据', to: '/credentials', icon: 'mdi-key-variant' },
      { title: '题库', to: '/questions', icon: 'mdi-database-outline' },
    ],
  })
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
  home: '首页',
  tasks: '任务',
  'task-detail': '任务详情',
  platform: '平台',
  credentials: '凭据',
  questions: '题库',
  scripts: '脚本',
  'script-detail': '脚本详情',
  users: '用户',
  settings: '设置',
}
const currentTitle = computed(() => {
  if (route.name === 'platform') {
    const platform = platforms.platform(route.params.id)
    if (!platform) return '平台'
    return platform.version ? `${platform.name} v${platform.version}` : platform.name
  }
  return PAGE_TITLES[route.name] || 'OmniTask'
})

function githubIssuesUrl(source) {
  const match = String(source || '').match(/github\.com[/:]([^/]+)\/([^/#?]+)/i)
  if (!match) return ''
  const repo = match[2].replace(/\.git$/i, '')
  return `https://github.com/${match[1]}/${repo}/issues/new`
}

const feedbackUrl = computed(() => {
  if (route.name !== 'platform') return ''
  return githubIssuesUrl(platforms.platform(route.params.id)?.source_url)
})

const avatarLetter = computed(() => (auth.username || '?').charAt(0).toUpperCase())
const isDark = computed(() => theme.global.name.value === 'dark')

function toggleTheme() {
  const next = isDark.value ? 'light' : 'dark'
  theme.global.name.value = next
  localStorage.setItem('omnitask_theme', next)
}

async function logout() {
  userMenu.value = false
  await auth.logout()
  ui.notify('已退出登录', 'info')
  router.push({ name: 'login' })
}

function go(path) {
  userMenu.value = false
  router.push(path)
}

onMounted(() => {
  if (auth.isAuthenticated) {
    platforms.load().catch(() => {})
  }
})

watch(
  () => auth.user?.id,
  (id) => {
    if (id) loadPreferences()
    else pinnedPlatformIds.value = []
  },
  { immediate: true },
)

watch(
  () => route.path,
  (path) => {
    if (path.startsWith('/platforms')) {
      if (navSource !== 'pin') {
        openGroups['/tasks'] = true
      }
      navSource = ''
    } else if (path !== '/tasks') {
      openGroups['/tasks'] = false
      navSource = ''
    } else {
      navSource = ''
    }
    if (path.startsWith('/platforms/') && route.params.id) {
      const id = String(route.params.id)
      const pinKey = `pin:${id}`
      const taskKey = `task:${id}`
      if (activeNavKey.value !== pinKey && activeNavKey.value !== taskKey) {
        activeNavKey.value = taskKey
      }
    } else {
      activeNavKey.value = ''
    }
  },
  { immediate: true },
)
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
              <v-list-subheader class="text-body-small text-medium-emphasis px-3">
                {{ group.title }}
              </v-list-subheader>
              <template v-for="item in group.items" :key="item.to">
                <template v-if="item.children">
                  <v-list-item
                    :to="item.to"
                    :prepend-icon="item.icon"
                    :title="item.title"
                    :value="item.to"
                    color="primary"
                    rounded="xl"
                    @click="toggleGroup(item.to)"
                  >
                    <template #append>
                      <v-icon
                        :icon="openGroups[item.to] ? 'mdi-chevron-up' : 'mdi-chevron-down'"
                        size="small"
                      />
                    </template>
                  </v-list-item>
                  <v-expand-transition>
                    <div v-show="openGroups[item.to]">
                      <v-list-item
                        v-for="child in item.children"
                        :key="child.to"
                        :prepend-icon="child.icon"
                        :title="child.title"
                        :active="child.platformId ? activeNavKey === 'task:' + child.platformId : undefined"
                        color="primary"
                        rounded="xl"
                        class="ps-8"
                        @click="child.platformId && navigatePlatform(child.platformId, 'task')"
                      >
                        <template #append>
                          <v-btn
                            icon
                            size="small"
                            variant="text"
                            :color="isPinned(child.platformId) ? 'primary' : undefined"
                            @click.stop.prevent="togglePin(child.platformId)"
                          >
                            <v-icon
                              :icon="isPinned(child.platformId) ? 'mdi-pin' : 'mdi-pin-outline'"
                              size="small"
                            />
                          </v-btn>
                        </template>
                      </v-list-item>
                    </div>
                  </v-expand-transition>
                </template>
                <v-list-item
                  v-else
                  :to="item.platformId ? undefined : item.to"
                  :prepend-icon="item.icon"
                  :title="item.title"
                  :value="item.platformId ? undefined : item.to"
                  :active="item.platformId ? activeNavKey === 'pin:' + item.platformId : undefined"
                  color="primary"
                  rounded="xl"
                  @click="item.platformId && navigatePlatform(item.platformId, 'pin')"
                >
                  <template v-if="item.platformId" #append>
                    <v-btn
                      icon
                      size="small"
                      variant="text"
                      :color="isPinned(item.platformId) ? 'primary' : undefined"
                      @click.stop.prevent="togglePin(item.platformId)"
                    >
                      <v-icon
                        :icon="isPinned(item.platformId) ? 'mdi-pin' : 'mdi-pin-outline'"
                        size="small"
                      />
                    </v-btn>
                  </template>
                </v-list-item>
              </template>
            </template>
          </v-list>

          <div class="drawer-legal-spacer" aria-hidden="true" />

          <div class="px-4 pb-8 pt-4 text-medium-emphasis drawer-legal">
            <div>© CometNetworkStudio</div>
          </div>
        </div>

        <template #append>
          <v-divider />
          <v-list nav bg-color="transparent" class="px-2 py-2 flex-shrink-0 app-drawer__nav">
            <v-list-item
              prepend-icon="mdi-cog-outline"
              title="设置"
              to="/settings"
              color="primary"
              rounded="xl"
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
        <v-app-bar-title class="text-body-large font-weight-medium text-truncate">
          {{ currentTitle }}
        </v-app-bar-title>
        <v-spacer />

        <v-btn
          v-if="feedbackUrl"
          icon
          variant="text"
          :href="feedbackUrl"
          target="_blank"
          rel="noopener noreferrer"
          aria-label="反馈脚本问题"
          title="反馈脚本问题"
        >
          <v-icon icon="mdi-bug-outline" />
        </v-btn>

        <v-btn icon variant="text" :title="isDark ? '切换浅色' : '切换深色'" @click="toggleTheme">
          <v-icon :icon="isDark ? 'mdi-weather-sunny' : 'mdi-weather-night'" />
        </v-btn>

        <v-menu v-model="userMenu" location="bottom end">
          <template #activator="{ props }">
            <v-btn icon variant="text" v-bind="props" class="me-2" aria-label="用户菜单">
              <v-avatar color="primary" size="36">
                <span class="text-body-medium font-weight-medium">{{ avatarLetter }}</span>
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
      rounded="xl"
      contained
    >
      <template #actions="{ props }">
        <v-btn icon="mdi-close" variant="text" size="small" aria-label="关闭" v-bind="props" />
      </template>
    </v-snackbar-queue>
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
