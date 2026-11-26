<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { credentialsApi } from '@/api/services/credentials'
import { securityApi } from '@/api/services/security'
import { useAuthStore } from '@/stores/auth'
import { useMetaStore } from '@/stores/meta'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'
import { CONTACT_QQ_NUMBER, CONTACT_QQ_URL, LEGAL_DOCS, TERMS_VERSION } from '@/content/legal'
import AdminConfigPanel from '@/components/AdminConfigPanel.vue'
import AdminSecurityPanel from '@/components/AdminSecurityPanel.vue'
import DataTable from '@/components/DataTable.vue'
import LegalDocument from '@/components/LegalDocument.vue'
import TabsPanel from '@/components/TabsPanel.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const meta = useMetaStore()
const platforms = usePlatformsStore()
const ui = useUiStore()

const sections = computed(() => {
  const list = [
    { value: 'profile', title: '个人', icon: 'mdi-account-outline' },
    { value: 'security', title: '账号安全', icon: 'mdi-shield-lock-outline' },
  ]
  if (auth.isAdmin) {
    list.push(
      { value: 'admin-security', title: '站点安全', icon: 'mdi-shield-account-outline' },
      { value: 'admin-config', title: '站点配置', icon: 'mdi-tune-variant' },
    )
  }
  list.push({ value: 'agreement', title: '协议', icon: 'mdi-file-document-outline' })
  return list
})
const values = computed(() => sections.value.map((item) => item.value))

const tab = computed({
  get: () => (values.value.includes(route.query.tab) ? route.query.tab : 'profile'),
  set: (value) => router.replace({ query: { ...route.query, tab: value } }),
})

const logs = ref([])
const logsLoading = ref(false)
const sessions = ref([])
const sessionsLoading = ref(false)

const roleLabel = computed(() => (auth.isAdmin ? '管理员' : '普通用户'))
const PURPOSE_LABELS = { task: '任务执行', discover: '平台列表刷新', update: '编辑凭据' }
const logHeaders = [
  { title: '时间', key: 'created_at' },
  { title: '平台', key: 'platform' },
  { title: '用途', key: 'purpose' },
  { title: '关联', key: 'ref_id' },
]
const sessionHeaders = [
  { title: '设备', key: 'user_agent' },
  { title: 'IP', key: 'ip' },
  { title: '最近活跃', key: 'last_seen_at' },
  { title: '到期', key: 'expires_at' },
  { title: '', key: 'actions', sortable: false, align: 'end' },
]

async function loadLogs() {
  logsLoading.value = true
  try {
    logs.value = await credentialsApi.accessLogs()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    logsLoading.value = false
  }
}

async function loadSessions() {
  sessionsLoading.value = true
  try {
    sessions.value = await securityApi.mySessions()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    sessionsLoading.value = false
  }
}

function loadSecurity() {
  loadLogs()
  loadSessions()
}

async function revokeSession(item) {
  try {
    await securityApi.revokeMySession(item.id)
    if (item.current) {
      auth.clear()
      router.push('/login')
      return
    }
    ui.notify('已退出该设备')
    loadSessions()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

async function revokeOtherSessions() {
  try {
    await securityApi.revokeOtherSessions()
    ui.notify('已退出其他设备')
    loadSessions()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

onMounted(() => {
  platforms.load().catch(() => {})
  if (tab.value === 'security') loadSecurity()
})

watch(tab, (value) => {
  if (value === 'security') loadSecurity()
})
</script>

<template>
  <v-container>
    <div class="text-headline-small mb-4">设置</div>

    <TabsPanel v-model="tab" :tabs="sections">
      <template #profile>
        <v-card>
          <v-list lines="two">
            <v-list-item title="用户名" :subtitle="auth.username || '-'" />
            <v-list-item title="角色" :subtitle="roleLabel" />
            <v-list-item title="用户 ID" :subtitle="auth.user?.id || '-'" />
          </v-list>
        </v-card>
      </template>

      <template #security>
        <v-card class="mb-6">
          <v-card-item>
            <v-card-title class="text-title-large">登录设备</v-card-title>
            <v-card-subtitle>会话保存在服务端，可随时退出任意设备</v-card-subtitle>
          </v-card-item>
          <v-card-text>
            <div class="d-flex justify-end mb-2">
              <v-btn
                variant="text"
                color="primary"
                size="small"
                :disabled="sessions.length <= 1"
                @click="revokeOtherSessions"
              >
                退出其他设备
              </v-btn>
            </div>
            <DataTable
              :headers="sessionHeaders"
              :items="sessions"
              :loading="sessionsLoading"
              :paginated="false"
              item-value="id"
            >
              <template #item.user_agent="{ item }">
                <div class="d-flex align-center ga-2">
                  <span class="text-truncate">{{ item.user_agent || '未知设备' }}</span>
                  <v-chip v-if="item.current" size="small" color="primary" variant="tonal">当前</v-chip>
                </div>
              </template>
              <template #item.ip="{ item }">{{ item.ip || '-' }}</template>
              <template #item.last_seen_at="{ item }">{{ formatTime(item.last_seen_at) }}</template>
              <template #item.expires_at="{ item }">{{ formatTime(item.expires_at) }}</template>
              <template #item.actions="{ item }">
                <v-btn size="small" variant="text" color="error" @click="revokeSession(item)">
                  退出
                </v-btn>
              </template>
              <template #no-data>
                <div class="pa-6 text-medium-emphasis">暂无登录设备</div>
              </template>
            </DataTable>
          </v-card-text>
        </v-card>

        <v-card>
          <v-card-item>
            <v-card-title class="text-title-large">凭据访问记录</v-card-title>
            <v-card-subtitle>每次解密凭据都会记录在此，仅含元数据，不含账号密码</v-card-subtitle>
          </v-card-item>
          <v-card-text>
            <DataTable :headers="logHeaders" :items="logs" :loading="logsLoading" :paginated="false" item-value="id">
              <template #item.created_at="{ item }">{{ formatTime(item.created_at) }}</template>
              <template #item.platform="{ item }">
                <v-chip size="small" variant="tonal">{{ platforms.name(item.platform) }}</v-chip>
              </template>
              <template #item.purpose="{ item }">{{ PURPOSE_LABELS[item.purpose] || item.purpose }}</template>
              <template #item.ref_id="{ item }">
                <v-btn
                  v-if="item.purpose === 'task' && item.ref_id"
                  size="small"
                  variant="text"
                  color="primary"
                  :to="`/tasks/${item.ref_id}`"
                >
                  {{ item.ref_id }}
                </v-btn>
                <span v-else class="text-medium-emphasis">—</span>
              </template>
              <template #no-data>
                <div class="pa-6 text-medium-emphasis">暂无访问记录</div>
              </template>
            </DataTable>
          </v-card-text>
        </v-card>
      </template>

      <template v-if="auth.isAdmin" #admin-security>
        <AdminSecurityPanel :active="tab === 'admin-security'" />
      </template>

      <template v-if="auth.isAdmin" #admin-config>
        <AdminConfigPanel :active="tab === 'admin-config'" />
      </template>

      <template #agreement>
        <v-alert type="info" variant="tonal" class="mb-4">
          当前协议版本 {{ meta.termsVersion || TERMS_VERSION }}。未登录时也可以通过
          <router-link class="text-primary" to="/terms">/terms</router-link>
          和
          <router-link class="text-primary" to="/privacy">/privacy</router-link>
          查看完整内容。
        </v-alert>

        <v-expansion-panels
          model-value="terms"
          variant="accordion"
          rounded="xl"
          elevation="2"
          class="mb-4 overflow-hidden"
        >
          <v-expansion-panel value="terms">
            <v-expansion-panel-title>用户协议</v-expansion-panel-title>
            <v-expansion-panel-text>
              <LegalDocument :doc="LEGAL_DOCS.terms" />
            </v-expansion-panel-text>
          </v-expansion-panel>
          <v-expansion-panel value="privacy">
            <v-expansion-panel-title>隐私政策</v-expansion-panel-title>
            <v-expansion-panel-text>
              <LegalDocument :doc="LEGAL_DOCS.privacy" />
            </v-expansion-panel-text>
          </v-expansion-panel>
        </v-expansion-panels>

        <v-card>
          <v-card-item>
            <v-card-title class="text-title-large">投诉与联系</v-card-title>
            <v-card-subtitle>滥用举报、隐私请求、账号申诉</v-card-subtitle>
          </v-card-item>
          <v-card-text class="text-body-medium">
            <p class="mb-2">
              QQ 频道：
              <a
                class="text-primary"
                :href="CONTACT_QQ_URL"
                target="_blank"
                rel="noopener noreferrer"
              >
                加入一起讨论
              </a>
            </p>
            <p class="mb-2">频道号：{{ CONTACT_QQ_NUMBER }}</p>
            <p class="mb-2">
              GitHub：
              <a
                class="text-primary"
                href="https://github.com/CometNetworkStudio/OmniTask/issues/new"
                target="_blank"
                rel="noopener noreferrer"
              >
                提交 Issue
              </a>
              ，请勿在公开渠道发布密码、Cookie、令牌或他人隐私。
            </p>
            <p>受理与封禁流程见用户协议第六、七节，数据保留与删除规则见隐私政策第四节。</p>
          </v-card-text>
        </v-card>
      </template>
    </TabsPanel>
  </v-container>
</template>
