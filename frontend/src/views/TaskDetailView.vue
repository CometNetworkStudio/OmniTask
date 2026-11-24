<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { tasksApi } from '@/api/services/tasks'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { formatTime, statusMeta } from '@/utils/format'
import { connectTaskSocket } from '@/ws/taskSocket'
import PlatformField from '@/components/PlatformField.vue'

const route = useRoute()
const router = useRouter()
const ui = useUiStore()
const platforms = usePlatformsStore()

const task = ref(null)
const events = ref([])
const loading = ref(false)
let socket = null

const canCancel = computed(() => ['pending', 'queued', 'running'].includes(task.value?.status))
const canRetry = computed(() => ['failed', 'canceled', 'interrupted'].includes(task.value?.status))
const canDelete = computed(() => !['running', 'queued'].includes(task.value?.status))
const actionName = computed(() => platforms.actionName(task.value?.platform, task.value?.action))
const action = computed(() => platforms.action(task.value?.platform, task.value?.action))
const resultSchema = computed(() => action.value?.result || null)

const steps = computed(() => {
  const map = new Map()
  for (const event of events.value) {
    if (event.kind === 'step' && event.step) map.set(event.step, event)
  }
  return [...map.values()]
})

const metrics = computed(() => {
  const map = new Map()
  for (const event of events.value) {
    if (event.kind === 'metric' && event.key) map.set(event.key, event)
  }
  return [...map.values()]
})

const logs = computed(() =>
  events.value
    .filter((event) => event.kind === 'log')
    .slice(-200)
    .reverse(),
)

const timeline = computed(() =>
  events.value
    .filter((event) => ['started', 'progress', 'succeeded', 'failed', 'canceled', 'interrupted'].includes(event.kind))
    .slice(-200)
    .reverse(),
)

const resultValue = computed(() => task.value?.result)
const resultRows = computed(() => {
  const value = resultValue.value
  if (Array.isArray(value)) return value.map((row, index) => normalizeRow(row, index))
  if (value && typeof value === 'object') return [normalizeRow(value, 0)]
  return []
})
const resultHeaders = computed(() => {
  const columns = resultSchema.value?.columns || []
  if (columns.length) return columns.map((col) => ({ title: col.label, key: col.key }))
  const keys = new Set()
  for (const row of resultRows.value) Object.keys(row).forEach((key) => keys.add(key))
  return [...keys]
    .filter((key) => !key.startsWith('__'))
    .map((key) => ({ title: prettyKey(key), key }))
})
const isTableResult = computed(
  () => resultSchema.value?.type === 'table' || Array.isArray(resultValue.value),
)
const resultLinks = computed(() => {
  const value = resultValue.value
  if (!value) return []
  if (Array.isArray(value)) {
    return value
      .filter((item) => item && typeof item === 'object')
      .map((item) => ({ label: item.label || item.name || item.url, url: item.url || item.link }))
      .filter((item) => item.url)
  }
  if (typeof value === 'object') {
    return Object.entries(value)
      .filter(([, url]) => typeof url === 'string' && /^https?:\/\//.test(url))
      .map(([label, url]) => ({ label: prettyKey(label), url }))
  }
  return []
})
const isLinksResult = computed(() => resultSchema.value?.type === 'links')
const resultEntries = computed(() => {
  const value = resultValue.value
  if (!value || typeof value !== 'object' || Array.isArray(value)) return []
  return Object.entries(value)
})
const hasResult = computed(
  () => (isTableResult.value && resultRows.value.length) || (isLinksResult.value && resultLinks.value.length) || resultEntries.value.length,
)
const paramsEntries = computed(() => Object.entries(task.value?.params || {}))

const LEVEL_META = {
  debug: { text: '调试', color: 'grey' },
  info: { text: '信息', color: 'info' },
  warn: { text: '警告', color: 'warning' },
  error: { text: '错误', color: 'error' },
}

const STEP_META = {
  running: { text: '进行中', color: 'warning', icon: 'mdi-progress-clock' },
  succeeded: { text: '已完成', color: 'success', icon: 'mdi-check-circle' },
  failed: { text: '失败', color: 'error', icon: 'mdi-close-circle' },
  skipped: { text: '已跳过', color: 'grey', icon: 'mdi-minus-circle-outline' },
}

function stepMeta(status) {
  return STEP_META[status] || STEP_META.running
}

function levelMeta(level) {
  return LEVEL_META[level] || LEVEL_META.info
}

function prettyKey(key) {
  return String(key).replace(/[_-]+/g, ' ')
}

function normalizeRow(row, index = 0) {
  const out = { __rowKey: index }
  for (const [key, value] of Object.entries(row || {})) {
    if (value === null || value === undefined) out[key] = '-'
    else if (typeof value === 'object') out[key] = JSON.stringify(value)
    else out[key] = value
  }
  return out
}

let refreshing = false
async function refreshTask() {
  if (refreshing) return
  refreshing = true
  try {
    task.value = await tasksApi.get(route.params.id)
  } catch {
    // 保留当前状态
  } finally {
    refreshing = false
  }
}

async function cancelTask() {
  try {
    await tasksApi.cancel(route.params.id)
    if (task.value) task.value.status = 'canceled'
    ui.notify('已取消')
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

async function retryTask() {
  try {
    task.value = await tasksApi.retry(route.params.id)
    events.value = []
    await loadEvents()
    ui.notify('已重新排队')
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

async function removeTask() {
  try {
    await tasksApi.remove(route.params.id)
    ui.notify('已删除')
    router.push('/')
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

function kindOf(event) {
  if (event.kind) return event.kind
  if (event.step) return 'step'
  if (event.key) return 'metric'
  if (event.level) return 'log'
  return 'progress'
}

function normalizeEvent(event) {
  return {
    id: event.id,
    kind: kindOf(event),
    type: event.type || event.status,
    percent: event.percent ?? 0,
    message: event.message || '',
    level: event.level || 'info',
    step: event.step || '',
    label: event.label || '',
    step_status: event.step_status || event.status || 'running',
    current: event.current ?? 0,
    total: event.total ?? 0,
    unit: event.unit || '',
    stage: event.stage || '',
    key: event.key || '',
    value: event.value || '',
    time: event.created_at || (event.timestamp ? new Date(event.timestamp * 1000).toISOString() : ''),
  }
}

function mergeEvent(raw) {
  if (!raw) return
  const event = normalizeEvent(raw)
  if (event.id && events.value.some((item) => item.id === event.id)) return
  events.value.push(event)
  if (task.value) {
    if (event.type) task.value.status = event.type
    if (typeof event.percent === 'number') task.value.progress = event.percent
    if (event.result && typeof event.result === 'object') task.value.result = event.result
    if (['succeeded', 'failed', 'canceled', 'interrupted'].includes(event.type)) refreshTask()
  }
}

async function loadEvents() {
  loading.value = true
  try {
    const list = await tasksApi.events(route.params.id)
    for (const event of list || []) mergeEvent(event)
  } catch {
    // 历史不可用时仍保留 WebSocket 实时事件
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  platforms.load().catch(() => {})
  try {
    task.value = await tasksApi.get(route.params.id)
  } catch (err) {
    ui.notify(err.message, 'error')
    return
  }
  socket = connectTaskSocket(route.params.id, { onEvent: mergeEvent })
  await loadEvents()
})

onBeforeUnmount(() => socket?.close())
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="d-flex align-center ga-2">
          <v-btn variant="text" icon="mdi-arrow-left" @click="router.push('/')" />
          <div class="text-headline-small">任务详情</div>
          <v-chip v-if="task" size="small" :color="statusMeta(task.status).color" variant="tonal">
            {{ statusMeta(task.status).text }}
          </v-chip>
        </div>
      </v-col>
      <v-col cols="auto">
        <div class="d-flex align-center ga-2">
          <v-btn v-if="canCancel" color="warning" prepend-icon="mdi-stop" @click="cancelTask">取消</v-btn>
          <v-btn v-if="canRetry" color="primary" prepend-icon="mdi-refresh" @click="retryTask">重试</v-btn>
          <v-btn v-if="canDelete" color="error" variant="tonal" prepend-icon="mdi-delete" @click="removeTask">
            删除
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-card v-if="task">
      <v-card-text>
        <v-row>
          <v-col cols="12" md="3">
            <div class="text-body-small text-medium-emphasis">平台</div>
            <div>{{ platforms.name(task.platform) }}</div>
          </v-col>
          <v-col cols="12" md="3">
            <div class="text-body-small text-medium-emphasis">动作</div>
            <div>{{ actionName }}</div>
          </v-col>
          <v-col cols="12" md="3">
            <div class="text-body-small text-medium-emphasis">创建时间</div>
            <div>{{ formatTime(task.created_at) }}</div>
          </v-col>
          <v-col cols="12" md="3">
            <div class="text-body-small text-medium-emphasis">结束时间</div>
            <div>{{ formatTime(task.finished_at) }}</div>
          </v-col>
        </v-row>

        <div class="mt-4">
          <v-progress-linear
            :model-value="task.progress"
            :color="statusMeta(task.status).color"
            height="10"
            rounded
          />
          <div class="text-medium-emphasis mt-1">{{ task.progress }}%</div>
        </div>

        <v-alert v-if="task.error" type="error" variant="tonal" class="mt-4">
          {{ task.error }}
        </v-alert>

        <template v-if="paramsEntries.length">
          <div class="text-title-small mt-6 mb-2">参数</div>
          <v-chip
            v-for="[key, value] in paramsEntries"
            :key="key"
            size="small"
            variant="tonal"
            class="me-1 mb-1"
          >
            {{ prettyKey(key) }}: {{ value }}
          </v-chip>
        </template>
      </v-card-text>
    </v-card>

    <v-row v-if="steps.length || metrics.length || hasResult" class="mt-4" density="comfortable">
      <v-col v-if="steps.length" cols="12" md="6">
        <v-card class="h-100">
          <v-card-item>
            <v-card-title class="text-title-medium">步骤</v-card-title>
          </v-card-item>
          <v-list density="compact">
            <v-list-item v-for="step in steps" :key="step.step">
              <template #prepend>
                <v-icon :color="stepMeta(step.step_status).color" :icon="stepMeta(step.step_status).icon" />
              </template>
              <v-list-item-title>{{ step.label || step.step }}</v-list-item-title>
              <template #append>
                <v-chip size="small" :color="stepMeta(step.step_status).color" variant="tonal">
                  {{ stepMeta(step.step_status).text }}
                </v-chip>
              </template>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <v-col v-if="metrics.length" cols="12" md="6">
        <v-card class="h-100">
          <v-card-item>
            <v-card-title class="text-title-medium">指标</v-card-title>
          </v-card-item>
          <v-card-text>
            <v-chip
              v-for="metric in metrics"
              :key="metric.key"
              size="small"
              variant="tonal"
              color="primary"
              class="me-1 mb-1"
            >
              {{ metric.label || prettyKey(metric.key) }} {{ metric.value }}{{ metric.unit }}
            </v-chip>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col v-if="hasResult" cols="12">
        <v-card>
          <v-card-item>
            <v-card-title class="text-title-medium">结果</v-card-title>
          </v-card-item>
          <v-card-text>
            <v-data-table
              v-if="isTableResult && resultRows.length"
              :headers="resultHeaders"
              :items="resultRows"
              item-value="__rowKey"
              :items-per-page="-1"
              hide-default-footer
              density="compact"
            >
              <template
                v-for="col in resultSchema?.columns || []"
                :key="col.key"
                #[`item.${col.key}`]="{ item }"
              >
                <PlatformField :value="item[col.key]" :type="col.type" />
              </template>
            </v-data-table>
            <template v-else-if="isLinksResult && resultLinks.length">
              <v-btn
                v-for="link in resultLinks"
                :key="link.url"
                :href="link.url"
                target="_blank"
                variant="tonal"
                color="primary"
                prepend-icon="mdi-open-in-new"
                class="me-1 mb-1"
              >
                {{ link.label }}
              </v-btn>
            </template>
            <template v-else>
              <v-chip
                v-for="[key, value] in resultEntries"
                :key="key"
                size="small"
                variant="tonal"
                color="success"
                class="me-1 mb-1"
              >
                {{ prettyKey(key) }}: {{ typeof value === 'object' ? JSON.stringify(value) : value }}
              </v-chip>
            </template>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-card class="mt-4">
      <v-card-item>
        <v-card-title class="text-title-medium">执行记录</v-card-title>
      </v-card-item>
      <v-card-text>
        <v-timeline v-if="timeline.length" density="compact" side="end">
          <v-timeline-item
            v-for="event in timeline"
            :key="event.id || `${event.kind}-${event.time}-${event.message}`"
            :dot-color="statusMeta(event.type).color"
            size="small"
          >
            <div class="text-body-medium">{{ event.message || statusMeta(event.type).text }}</div>
            <div class="text-body-small text-medium-emphasis">
              {{ statusMeta(event.type).text }} {{ event.percent }}%
              <span v-if="event.total">，{{ event.current }}/{{ event.total }}{{ event.unit }}</span>
              <span v-if="event.time">，{{ formatTime(event.time) }}</span>
            </div>
          </v-timeline-item>
        </v-timeline>
        <div v-else-if="loading" class="py-6 text-center text-medium-emphasis">加载中</div>
        <div v-else class="py-6 text-center text-medium-emphasis">暂无记录</div>
      </v-card-text>
    </v-card>

    <v-card v-if="logs.length" class="mt-4">
      <v-card-item>
        <v-card-title class="text-title-medium">日志</v-card-title>
      </v-card-item>
      <v-card-text>
        <v-list density="compact" max-height="360" class="overflow-y-auto">
          <v-list-item v-for="log in logs" :key="log.id || `${log.time}-${log.message}`">
            <template #prepend>
              <v-chip size="x-small" :color="levelMeta(log.level).color" variant="tonal" class="me-2">
                {{ levelMeta(log.level).text }}
              </v-chip>
            </template>
            <v-list-item-subtitle>{{ log.message }}</v-list-item-subtitle>
            <template #append>
              <span class="text-body-small text-medium-emphasis">{{ formatTime(log.time) }}</span>
            </template>
          </v-list-item>
        </v-list>
      </v-card-text>
    </v-card>
  </v-container>
</template>
