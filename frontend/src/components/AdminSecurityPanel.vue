<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { securityApi } from '@/api/services/security'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'
import DataTable from '@/components/DataTable.vue'

const props = defineProps({
  active: Boolean,
})

const ui = useUiStore()

const overview = ref(null)
const sessions = ref([])
const attempts = ref([])
const loading = ref(false)

const REGISTRATION_LABELS = {
  open: '开放注册',
  invite: '邀请码注册',
  closed: '关闭注册',
}
const REASON_LABELS = {
  success: '成功',
  bad_password: '密码错误',
  unknown_user: '用户不存在',
  disabled: '账号已禁用',
  locked: '失败次数过多',
}

const registrationLabel = computed(() => REGISTRATION_LABELS[overview.value?.registration] || '-')
const lockoutLabel = computed(() => {
  if (!overview.value?.login_max_failures) return '未启用'
  return `${overview.value.login_max_failures} 次 / ${overview.value.login_window_minutes} 分钟`
})
const quotaLabel = computed(() => {
  const data = overview.value
  if (!data) return '-'
  return `${data.max_active_tasks || '不限'} / ${data.daily_tasks || '不限'} / ${data.max_pending_tasks || '不限'}`
})

const sessionHeaders = [
  { title: '用户', key: 'username' },
  { title: '设备', key: 'user_agent' },
  { title: 'IP', key: 'ip' },
  { title: '最近活跃', key: 'last_seen_at' },
  { title: '到期', key: 'expires_at' },
  { title: '', key: 'actions', sortable: false, align: 'end' },
]
const attemptHeaders = [
  { title: '时间', key: 'created_at' },
  { title: '用户名', key: 'username' },
  { title: '来源 IP', key: 'ip' },
  { title: '结果', key: 'success' },
  { title: '原因', key: 'reason' },
]

async function load() {
  loading.value = true
  try {
    const [overviewData, sessionList, attemptList] = await Promise.all([
      securityApi.overview(),
      securityApi.sessions(),
      securityApi.attempts(),
    ])
    overview.value = overviewData
    sessions.value = sessionList || []
    attempts.value = attemptList || []
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

async function revokeSession(item) {
  try {
    await securityApi.revokeSession(item.id)
    ui.notify('已吊销该会话')
    load()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

async function revokeUserSessions(item) {
  try {
    await securityApi.revokeUserSessions(item.user_id)
    ui.notify(`已吊销 ${item.username} 的全部会话`)
    load()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

onMounted(() => {
  if (props.active) load()
})

watch(
  () => props.active,
  (active) => {
    if (active && !overview.value) load()
  },
)
</script>

<template>
  <div>
    <v-row align="center" class="mb-4">
      <v-col>
        <div class="text-title-large">站点安全</div>
        <div class="text-medium-emphasis">登录会话、注册策略与登录失败记录</div>
      </v-col>
      <v-col cols="auto">
        <v-btn variant="text" prepend-icon="mdi-refresh" :loading="loading" @click="load">刷新</v-btn>
      </v-col>
    </v-row>

    <v-row class="mb-6">
      <v-col cols="12" sm="6" md="3">
        <v-card class="h-100">
          <v-card-item>
            <v-card-subtitle>注册方式</v-card-subtitle>
            <v-card-title class="text-title-medium">{{ registrationLabel }}</v-card-title>
          </v-card-item>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-card class="h-100">
          <v-card-item>
            <v-card-subtitle>活跃会话</v-card-subtitle>
            <v-card-title class="text-title-medium">{{ overview?.active_sessions ?? '-' }}</v-card-title>
          </v-card-item>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-card class="h-100">
          <v-card-item>
            <v-card-subtitle>24 小时失败登录</v-card-subtitle>
            <v-card-title class="text-title-medium">{{ overview?.failed_logins_24h ?? '-' }}</v-card-title>
          </v-card-item>
        </v-card>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-card class="h-100">
          <v-card-item>
            <v-card-subtitle>会话有效期</v-card-subtitle>
            <v-card-title class="text-title-medium">{{ overview?.session_ttl_hours ?? '-' }} 小时</v-card-title>
          </v-card-item>
        </v-card>
      </v-col>
    </v-row>

    <v-card class="mb-6">
      <v-list lines="two">
        <v-list-item title="登录失败锁定" :subtitle="lockoutLabel" />
        <v-list-item title="任务配额（活跃 / 每日 / 待处理）" :subtitle="quotaLabel" />
        <v-list-item title="注册用户数" :subtitle="String(overview?.user_count ?? '-')" />
      </v-list>
    </v-card>

    <v-card class="mb-6">
      <v-card-item>
        <v-card-title class="text-title-large">登录会话</v-card-title>
        <v-card-subtitle>会话存储在服务端，吊销后立即失效</v-card-subtitle>
      </v-card-item>
      <v-card-text>
        <DataTable
          :headers="sessionHeaders"
          :items="sessions"
          :loading="loading"
          :paginated="false"
          item-value="id"
        >
          <template #item.username="{ item }">
            <div class="d-flex align-center ga-2">
              <span>{{ item.username || item.user_id }}</span>
              <v-chip v-if="item.current" size="small" color="primary" variant="tonal">当前</v-chip>
            </div>
          </template>
          <template #item.user_agent="{ item }">
            <span class="text-truncate">{{ item.user_agent || '未知设备' }}</span>
          </template>
          <template #item.ip="{ item }">{{ item.ip || '-' }}</template>
          <template #item.last_seen_at="{ item }">{{ formatTime(item.last_seen_at) }}</template>
          <template #item.expires_at="{ item }">{{ formatTime(item.expires_at) }}</template>
          <template #item.actions="{ item }">
            <v-btn size="small" variant="text" color="error" @click="revokeSession(item)">吊销</v-btn>
            <v-btn size="small" variant="text" @click="revokeUserSessions(item)">该用户全部</v-btn>
          </template>
          <template #no-data>
            <div class="pa-6 text-medium-emphasis">暂无活跃会话</div>
          </template>
        </DataTable>
      </v-card-text>
    </v-card>

    <v-card class="mb-6">
      <v-card-item>
        <v-card-title class="text-title-large">登录记录</v-card-title>
        <v-card-subtitle>最近 50 次登录尝试，不包含密码</v-card-subtitle>
      </v-card-item>
      <v-card-text>
        <DataTable
          :headers="attemptHeaders"
          :items="attempts"
          :loading="loading"
          :paginated="false"
          item-value="id"
        >
          <template #item.created_at="{ item }">{{ formatTime(item.created_at) }}</template>
          <template #item.ip="{ item }">{{ item.ip || '-' }}</template>
          <template #item.success="{ item }">
            <v-chip size="small" :color="item.success ? 'success' : 'error'" variant="tonal">
              {{ item.success ? '成功' : '失败' }}
            </v-chip>
          </template>
          <template #item.reason="{ item }">{{ REASON_LABELS[item.reason] || item.reason || '-' }}</template>
          <template #no-data>
            <div class="pa-6 text-medium-emphasis">暂无登录记录</div>
          </template>
        </DataTable>
      </v-card-text>
    </v-card>
  </div>
</template>
