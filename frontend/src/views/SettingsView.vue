<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { credentialsApi } from '@/api/services/credentials'
import { useAuthStore } from '@/stores/auth'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'
import DataTable from '@/components/DataTable.vue'
import TabsPanel from '@/components/TabsPanel.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const platforms = usePlatformsStore()
const ui = useUiStore()

const SECTIONS = [
  { value: 'profile', title: '个人', icon: 'mdi-account-outline' },
  { value: 'security', title: '安全', icon: 'mdi-shield-lock-outline' },
  { value: 'agreement', title: '协议', icon: 'mdi-file-document-outline' },
]
const VALUES = SECTIONS.map((s) => s.value)

const tab = computed({
  get: () => (VALUES.includes(route.query.tab) ? route.query.tab : 'profile'),
  set: (value) => router.replace({ query: { ...route.query, tab: value } }),
})

const logs = ref([])
const logsLoading = ref(false)

const roleLabel = computed(() => (auth.isAdmin ? '管理员' : '普通用户'))
const PURPOSE_LABELS = { task: '任务执行', discover: '平台列表刷新' }
const logHeaders = [
  { title: '时间', key: 'created_at' },
  { title: '平台', key: 'platform' },
  { title: '用途', key: 'purpose' },
  { title: '关联', key: 'ref_id' },
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

onMounted(() => {
  platforms.load().catch(() => {})
  if (tab.value === 'security') loadLogs()
})

watch(tab, (value) => {
  if (value === 'security') loadLogs()
})
</script>

<template>
  <v-container>
    <div class="text-headline-small mb-4">设置</div>

    <TabsPanel v-model="tab" :tabs="SECTIONS">
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

      <template #agreement>
        <v-card class="mb-3">
          <v-card-item>
            <v-card-title class="text-title-large">免责声明</v-card-title>
          </v-card-item>
          <v-card-text class="text-body-medium">
            本项目仅供个人学习与技术研究使用。请遵守各平台服务条款与学校相关规定，因使用本项目产生的一切后果由使用者自行承担。本项目不包含第三方平台脚本代码，脚本由作者独立发布并自负责任。
          </v-card-text>
        </v-card>
        <v-card>
          <v-card-item>
            <v-card-title class="text-title-large">凭据与隐私</v-card-title>
          </v-card-item>
          <v-card-text class="text-body-medium">
            <p class="mb-2">OmniTask 需要代你登录目标平台，因此任何部署实例在任务运行时都能在技术上解密你的凭据。加密（AES-256-GCM）只能防止数据库/备份泄露，防不住恶意运维。</p>
            <p class="mb-2">请只使用你信任的实例，不要使用来路不明的公开部署；担心风险请自行本地部署。</p>
            <p>建议为各平台使用独立密码，避免与邮箱、银行等账号相同。每次凭据解密都会记录在「安全」标签页，供你自查。</p>
          </v-card-text>
        </v-card>
      </template>
    </TabsPanel>
  </v-container>
</template>
