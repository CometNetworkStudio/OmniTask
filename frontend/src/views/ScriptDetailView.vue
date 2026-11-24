<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { scriptsApi } from '@/api/services/scripts'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'
import { buildParams } from '@/utils/taskParams'
import ParamFields from '@/components/ParamFields.vue'

const route = useRoute()
const router = useRouter()
const ui = useUiStore()

const script = ref(null)
const loading = ref(false)
const debugDialog = ref(false)
const debugging = ref(false)
const debugAction = ref('')
const debugParams = reactive({})
const debugCredentials = reactive({})
const debugProxy = ref('')
const debugEvents = ref([])
const uninstallDialog = ref(false)
const reinstallJob = ref(null)

const manifest = computed(() => {
  try {
    return JSON.parse(script.value?.manifest || '{}')
  } catch {
    return {}
  }
})
const actions = computed(() => manifest.value.actions || [])
const credentials = computed(() => manifest.value.credentials || [])
const debugFields = computed(() => {
  const action = actions.value.find((item) => item.id === debugAction.value)
  const fields = action?.params || []
  return fields.length ? fields : manifest.value.params || []
})
const debugSteps = computed(() => {
  const map = new Map()
  for (const event of debugEvents.value) {
    if (event.kind === 'step' && event.step) map.set(event.step, event)
  }
  return [...map.values()]
})
const manifestText = computed(() => JSON.stringify(manifest.value, null, 2))

const STATUS_META = {
  installed: { text: '可用', color: 'success' },
  missing_runtime: { text: '缺少运行时', color: 'warning' },
  missing: { text: '未加载', color: 'error' },
}
const STEP_META = {
  running: { text: '进行中', color: 'warning' },
  succeeded: { text: '已完成', color: 'success' },
  failed: { text: '失败', color: 'error' },
  skipped: { text: '已跳过', color: 'grey' },
}
const LEVEL_META = {
  debug: { text: '调试', color: 'grey' },
  info: { text: '信息', color: 'info' },
  warn: { text: '警告', color: 'warning' },
  error: { text: '错误', color: 'error' },
}
const KIND_META = {
  progress: '进度',
  step: '步骤',
  log: '日志',
  metric: '指标',
  started: '开始',
  succeeded: '成功',
  failed: '失败',
}

function statusMeta(status) {
  return STATUS_META[status] || { text: status || '-', color: 'grey' }
}
function stepMeta(status) {
  return STEP_META[status] || STEP_META.running
}
function levelMeta(level) {
  return LEVEL_META[level] || LEVEL_META.info
}
function shortHash(value) {
  return value ? `${value.slice(0, 8)}…${value.slice(-6)}` : '未记录'
}

async function load() {
  loading.value = true
  try {
    script.value = await scriptsApi.get(route.params.id)
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

async function toggle() {
  try {
    await scriptsApi.toggle(script.value.id, script.value.enabled)
    ui.notify(script.value.enabled ? '已启用' : '已停用')
  } catch (err) {
    script.value.enabled = !script.value.enabled
    ui.notify(err.message, 'error')
  }
}

async function reinstall() {
  try {
    const { job_id } = await scriptsApi.reinstall(script.value.id)
    reinstallJob.value = { status: 'running', percent: 0, message: '排队中' }
    for (;;) {
      await new Promise((resolve) => setTimeout(resolve, 700))
      const state = await scriptsApi.installStatus(job_id)
      reinstallJob.value = state
      if (state.status === 'succeeded') break
      if (state.status === 'failed') throw new Error(state.error || '重新安装失败')
    }
    ui.notify('脚本已更新')
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    setTimeout(() => {
      reinstallJob.value = null
    }, 1500)
  }
}

async function rollback() {
  try {
    script.value = await scriptsApi.rollback(script.value.id)
    ui.notify('已回滚到上一个版本')
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

async function uninstall() {
  try {
    await scriptsApi.uninstall(script.value.id)
    ui.notify('已卸载')
    router.push('/scripts')
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

function openDebug() {
  debugEvents.value = []
  debugAction.value = actions.value[0]?.id || ''
  initDebugParams()
  for (const key of Object.keys(debugCredentials)) delete debugCredentials[key]
  for (const field of credentials.value) debugCredentials[field.key] = ''
  debugProxy.value = ''
  debugDialog.value = true
}

function onDebugActionChange() {
  initDebugParams()
}

function initDebugParams() {
  for (const key of Object.keys(debugParams)) delete debugParams[key]
  for (const field of debugFields.value) {
    debugParams[field.key] = field.type === 'switch' ? Boolean(field.default) : field.default ?? ''
  }
}

async function runDebug() {
  debugging.value = true
  debugEvents.value = []
  try {
    const creds = {}
    for (const [key, value] of Object.entries(debugCredentials)) {
      if (value !== '') creds[key] = value
    }
    debugEvents.value = await scriptsApi.debug(script.value.id, {
      action: debugAction.value,
      params: buildParams(debugFields.value, debugParams),
      credentials: creds,
      proxy: debugProxy.value,
    })
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    debugging.value = false
  }
}

onMounted(load)
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-4">
      <v-col>
        <div class="d-flex align-center ga-3 flex-wrap">
          <v-btn variant="text" icon="mdi-arrow-left" aria-label="返回脚本列表" @click="router.push('/scripts')" />
          <v-avatar color="primary-light" size="40">
            <v-icon icon="mdi-puzzle-outline" />
          </v-avatar>
          <div>
            <div class="text-headline-small">{{ script?.name || route.params.id }}</div>
            <div class="d-flex align-center ga-2 mt-1 flex-wrap">
              <v-chip v-if="script" size="small" :color="statusMeta(script.status).color" variant="tonal">
                {{ statusMeta(script.status).text }}
              </v-chip>
              <v-chip v-if="script?.version" size="small" variant="tonal">v{{ script.version }}</v-chip>
            </div>
          </div>
        </div>
      </v-col>
      <v-col cols="auto">
        <div class="d-flex align-center ga-2 flex-wrap">
          <v-btn variant="tonal" prepend-icon="mdi-bug-outline" :disabled="!script" @click="openDebug">
            调试
          </v-btn>
          <v-btn
            color="primary"
            prepend-icon="mdi-refresh"
            :loading="!!reinstallJob"
            :disabled="!script"
            @click="reinstall"
          >
            重新安装
          </v-btn>
          <v-menu location="bottom end">
            <template #activator="{ props }">
              <v-btn v-bind="props" icon="mdi-dots-vertical" variant="text" aria-label="更多操作" />
            </template>
            <v-list density="compact">
              <v-list-item
                v-if="script?.has_backup"
                title="回滚到上一版本"
                prepend-icon="mdi-history"
                @click="rollback"
              />
              <v-list-item
                title="卸载脚本"
                prepend-icon="mdi-delete"
                class="text-error"
                @click="uninstallDialog = true"
              />
            </v-list>
          </v-menu>
        </div>
      </v-col>
    </v-row>

    <v-progress-linear
      v-if="reinstallJob"
      :model-value="reinstallJob.percent"
      color="primary"
      height="8"
      rounded
      class="mb-3"
    />
    <div v-if="reinstallJob" class="text-body-small text-medium-emphasis mb-3">
      {{ reinstallJob.message }} {{ reinstallJob.percent }}%
    </div>

    <v-alert v-if="script?.last_error" type="warning" variant="tonal" class="mb-3">
      {{ script.last_error }}
    </v-alert>

    <v-row v-if="script" density="comfortable">
      <v-col cols="12" md="8">
        <v-card>
          <v-card-item>
            <v-card-title class="text-title-medium">基本信息</v-card-title>
          </v-card-item>
          <v-card-text class="pa-5">
            <v-row density="comfortable">
              <v-col cols="12" sm="6" md="3">
                <div class="text-label-medium text-medium-emphasis">运行时</div>
                <div class="text-body-medium">{{ script.runtime || '-' }}</div>
              </v-col>
              <v-col cols="12" sm="6" md="3">
                <div class="text-label-medium text-medium-emphasis">版本</div>
                <div class="text-body-medium">{{ script.version || '-' }}</div>
              </v-col>
              <v-col cols="12" sm="6" md="3">
                <div class="text-label-medium text-medium-emphasis">安装时间</div>
                <div class="text-body-medium">{{ formatTime(script.installed_at) }}</div>
              </v-col>
              <v-col cols="12" sm="6" md="3">
                <div class="text-label-medium text-medium-emphasis">启用</div>
                <v-switch
                  v-model="script.enabled"
                  color="primary"
                  density="compact"
                  hide-details
                  @update:model-value="toggle"
                />
              </v-col>
              <v-col cols="12" sm="6" md="3">
                <div class="text-label-medium text-medium-emphasis">来源</div>
                <v-btn
                  v-if="script.source_url"
                  :href="script.source_url"
                  target="_blank"
                  rel="noopener"
                  variant="tonal"
                  color="primary"
                  icon
                  size="x-small"
                  aria-label="打开来源仓库"
                >
                  <v-icon icon="mdi-github" />
                  <v-tooltip activator="parent" location="bottom">打开来源仓库</v-tooltip>
                </v-btn>
                <div v-else class="text-body-medium text-medium-emphasis">未记录</div>
              </v-col>
              <v-col cols="12" sm="6" md="3">
                <div class="text-label-medium text-medium-emphasis">Ref</div>
                <div class="text-body-medium text-truncate">{{ script.ref || 'main' }}</div>
              </v-col>
              <v-col cols="12" sm="6" md="3">
                <div class="text-label-medium text-medium-emphasis">Commit</div>
                <div class="text-body-medium text-truncate">{{ script.commit_sha || '未固定' }}</div>
              </v-col>
              <v-col cols="12" sm="6" md="3">
                <div class="text-label-medium text-medium-emphasis">Manifest hash</div>
                <div class="text-body-medium">{{ shortHash(script.manifest_hash) }}</div>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="4">
        <v-card class="h-100">
          <v-card-item>
            <v-card-title class="text-title-medium">组件</v-card-title>
          </v-card-item>
          <v-card-text class="pa-5">
            <div class="d-flex flex-wrap ga-2">
              <v-chip size="small" variant="tonal">进度</v-chip>
              <v-chip size="small" variant="tonal">步骤</v-chip>
              <v-chip size="small" variant="tonal">日志</v-chip>
              <v-chip size="small" variant="tonal">指标</v-chip>
              <v-chip size="small" variant="tonal">结果</v-chip>
              <v-chip v-if="manifest.discovery" size="small" variant="tonal">发现列表</v-chip>
            </div>
            <div v-if="manifest.permissions?.length" class="d-flex flex-wrap ga-2 mt-3">
              <v-chip
                v-for="permission in manifest.permissions"
                :key="permission"
                size="small"
                variant="tonal"
                color="warning"
              >
                {{ permission }}
              </v-chip>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12">
        <v-card>
          <v-card-item>
            <v-card-title class="text-title-medium">动作</v-card-title>
          </v-card-item>
          <v-list v-if="actions.length" density="comfortable" class="pa-2">
            <v-list-item v-for="action in actions" :key="action.id" class="py-2">
              <template #prepend>
                <v-avatar color="primary-light" size="36">
                  <v-icon icon="mdi-play-circle-outline" />
                </v-avatar>
              </template>
              <v-list-item-title>{{ action.name }}</v-list-item-title>
              <div v-if="action.description" class="text-body-small text-medium-emphasis mt-1">
                {{ action.description }}
              </div>
              <div v-if="action.params?.length || action.result" class="d-flex flex-wrap ga-1 mt-1">
                <v-chip v-for="field in action.params" :key="field.key" size="x-small" variant="tonal">
                  {{ field.label }}{{ field.required ? ' *' : '' }}
                </v-chip>
                <v-chip v-if="action.result" size="x-small" variant="tonal" color="primary">
                  结果：{{ action.result.type }}
                </v-chip>
              </div>
            </v-list-item>
          </v-list>
          <v-card-text v-else class="text-medium-emphasis">未声明动作</v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12">
        <v-card>
          <v-expansion-panels variant="accordion" flat>
            <v-expansion-panel>
              <v-expansion-panel-title>
                <span class="text-title-medium">Manifest</span>
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <v-sheet color="surface-variant" class="pa-4 rounded-xl overflow-x-auto">
                  <pre class="text-body-small ma-0">{{ manifestText }}</pre>
                </v-sheet>
              </v-expansion-panel-text>
            </v-expansion-panel>
          </v-expansion-panels>
        </v-card>
      </v-col>
    </v-row>

    <v-dialog v-model="debugDialog" max-width="760" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large">调试脚本</v-card-title>
        </v-card-item>
        <v-card-text>
          <v-select
            v-model="debugAction"
            :items="actions"
            item-title="name"
            item-value="id"
            label="动作"
            @update:model-value="onDebugActionChange"
          />
          <ParamFields v-if="debugFields.length" :fields="debugFields" :values="debugParams" />
          <v-text-field
            v-model="debugProxy"
            label="代理（可选）"
            placeholder="http://user:pass@host:port"
            autocomplete="off"
          />
          <template v-if="credentials.length">
            <v-divider class="my-4" />
            <div class="text-title-small mb-2">凭据</div>
            <v-text-field
              v-for="field in credentials"
              :key="field.key"
              v-model="debugCredentials[field.key]"
              :label="field.label"
              :type="['password', 'token'].includes(field.type) ? 'password' : 'text'"
              autocomplete="off"
            />
          </template>

          <v-divider class="my-4" />
          <div class="d-flex align-center justify-space-between mb-2">
            <div class="text-title-small">运行结果</div>
            <v-btn size="small" color="primary" :loading="debugging" @click="runDebug">运行</v-btn>
          </div>

          <div v-if="debugSteps.length" class="mb-3">
            <v-chip
              v-for="step in debugSteps"
              :key="step.step"
              size="small"
              :color="stepMeta(step.step_status).color"
              variant="tonal"
              class="me-1 mb-1"
            >
              {{ step.label || step.step }}：{{ stepMeta(step.step_status).text }}
            </v-chip>
          </div>

          <v-list v-if="debugEvents.length" density="compact" max-height="320" class="overflow-y-auto">
            <v-list-item v-for="(event, index) in debugEvents" :key="index">
              <template #prepend>
                <v-chip
                  v-if="event.kind === 'log'"
                  size="x-small"
                  :color="levelMeta(event.level).color"
                  variant="tonal"
                  class="me-2"
                >
                  {{ levelMeta(event.level).text }}
                </v-chip>
                <v-chip v-else size="x-small" variant="tonal" class="me-2">
                  {{ KIND_META[event.kind] || event.kind }}
                </v-chip>
              </template>
              <v-list-item-subtitle>
                {{ event.message || event.label || event.step || event.key }}
                <span v-if="event.percent">，{{ event.percent }}%</span>
                <span v-if="event.total">，{{ event.current }}/{{ event.total }}{{ event.unit }}</span>
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
          <div v-else class="text-medium-emphasis py-4 text-center">尚未运行</div>
        </v-card-text>
        <v-card-actions class="justify-end">
          <v-btn variant="text" @click="debugDialog = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="uninstallDialog" max-width="480" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large">卸载脚本</v-card-title>
        </v-card-item>
        <v-card-text>确认卸载「{{ script?.name }}」？已创建的任务不会删除。</v-card-text>
        <v-card-actions class="justify-end">
          <v-btn variant="text" @click="uninstallDialog = false">取消</v-btn>
          <v-btn color="error" @click="uninstall">卸载</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>
