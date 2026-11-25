<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import { platformsApi } from '@/api/services/platforms'
import { tasksApi } from '@/api/services/tasks'
import { credentialsApi } from '@/api/services/credentials'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { buildParams, compareFieldValues, parseProgress } from '@/utils/taskParams'
import { formatAgo } from '@/utils/format'
import DataTable from '@/components/DataTable.vue'
import PlatformField from '@/components/PlatformField.vue'
import ParamFields from '@/components/ParamFields.vue'

const VIEW_KEY = 'omnitask_platform_view'

const route = useRoute()
const router = useRouter()
const { smAndDown } = useDisplay()
const platforms = usePlatformsStore()
const ui = useUiStore()

const items = ref([])
const refreshedAt = ref(null)
const loading = ref(false)
const discovering = ref(false)
const executing = ref(false)
const search = ref('')
const credentials = ref([])
const accountId = ref('')
const viewMode = ref(defaultView(smAndDown.value))

const execDialog = ref(false)
const execTarget = ref(null)
const execParams = reactive({})
const detailDialog = ref(false)
const detailTarget = ref(null)
const selected = ref([])
const filterDialog = ref(false)
const filterParams = reactive({})
const batchDialog = ref(false)
const batchDial = ref(false)
const batchActionId = ref('')
const batchParams = reactive({})
const batchSubmitting = ref(false)

const platformId = computed(() => route.params.id)
const platform = computed(() => platforms.platform(platformId.value))
const discovery = computed(() => platform.value?.discovery || null)
const columns = computed(() => discovery.value?.columns || [])
const execAllFields = computed(() => {
  const fields = execTarget.value?.action?.params || []
  return fields.length ? fields : platform.value?.params || []
})
const primaryKey = computed(() => discovery.value?.primary || columns.value[0]?.key || '')
const progressColumn = computed(() => columns.value.find((col) => col.type === 'progress') || null)
const hasImportantColumns = computed(() => columns.value.some((col) => col.important === true))
const importantColumns = computed(() =>
  hasImportantColumns.value ? columns.value.filter((col) => col.important === true) : columns.value,
)
const selectedItems = computed(() => items.value.filter((item) => selected.value.includes(item.key)))
const batchAction = computed(() => listActions.value.find((action) => action.id === batchActionId.value))
const batchAllFields = computed(() => {
  const fields = batchAction.value?.params || []
  return fields.length ? fields : platform.value?.params || []
})
const fixedParamKeys = computed(() => new Set(Object.keys(discovery.value?.param_map || {})))
const execFields = computed(() =>
  execAllFields.value.filter((field) => !execTarget.value?.item || !fixedParamKeys.value.has(field.key)),
)
const batchFields = computed(() =>
  batchAllFields.value.filter((field) => !selectedItems.value.length || !fixedParamKeys.value.has(field.key)),
)
const otherColumns = computed(() => columns.value.filter((col) => col.key !== primaryKey.value))
const detailColumns = computed(() => otherColumns.value.filter((col) => col.type !== 'progress'))
const summaryColumns = computed(() =>
  importantColumns.value.filter((col) => col.key !== primaryKey.value && col.type !== 'progress'),
)

const listActions = computed(() => {
  const all = platform.value?.actions || []
  const ids = discovery.value?.actions?.length ? discovery.value.actions : all.map((a) => a.id)
  return ids.map((id) => all.find((a) => a.id === id)).filter(Boolean)
})

const accountOptions = computed(() =>
  credentials.value.map((c) => ({
    title: c.label || c.preview || c.id,
    value: c.id,
  })),
)

const headers = computed(() => [
  ...importantColumns.value.map((col) => ({
    title: col.label,
    key: `f_${col.key}`,
    type: col.type,
    minWidth: col.type === 'progress' ? 180 : undefined,
    sortable: true,
    sortRaw: (a, b) => compareFieldValues(a.fields?.[col.key], b.fields?.[col.key], col),
  })),
  { title: '', key: 'actions', sortable: false, align: 'end' },
])

const filteredItems = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return items.value
  return items.value.filter((item) =>
    Object.values(item.fields || {}).some((v) => String(v).toLowerCase().includes(keyword)),
  )
})

function defaultView(small) {
  const saved = localStorage.getItem(`${VIEW_KEY}:${small ? 'small' : 'large'}`)
  return saved || (small ? 'grid' : 'table')
}

function fieldSlot(key) {
  return `item.f_${key}`
}

function progressValue(item) {
  return parseProgress(item?.fields?.[progressColumn.value?.key])
}

watch(viewMode, (mode) =>
  localStorage.setItem(`${VIEW_KEY}:${smAndDown.value ? 'small' : 'large'}`, mode),
)
watch(smAndDown, (small) => {
  viewMode.value = defaultView(small)
})
watch(
  () => selected.value.length,
  (count) => {
    if (!count) batchDial.value = false
  },
)

function toggleView() {
  viewMode.value = viewMode.value === 'table' ? 'grid' : 'table'
}

function clearSelection() {
  selected.value = []
  batchDial.value = false
}

onMounted(init)
watch(platformId, init)
watch(accountId, () => loadCached())

async function init() {
  await platforms.load()
  await loadCredentials()
  items.value = []
  refreshedAt.value = null
  selected.value = []
  initParams(filterParams, [])
  if (platform.value) loadCached()
}

async function loadCredentials() {
  try {
    const all = await credentialsApi.list()
    credentials.value = all.filter((c) => c.platform === platformId.value)
  } catch {
    credentials.value = []
  }
  const preferred = credentials.value.find((c) => c.isDefault) || credentials.value[0]
  accountId.value = preferred?.id || ''
}

async function loadCached() {
  loading.value = true
  try {
    const data = await platformsApi.items(platformId.value, accountId.value)
    items.value = data.items || []
    refreshedAt.value = data.refreshed_at || null
  } catch {
    // 无缓存时忽略
  } finally {
    loading.value = false
  }
}

async function refresh() {
  discovering.value = true
  try {
    const params = buildParams(filterFields.value, filterParams)
    const data = await platformsApi.discover(platformId.value, accountId.value, params)
    items.value = data.items || []
    refreshedAt.value = data.refreshed_at || null
    selected.value = []
    ui.notify(`已刷新：${items.value.length} 条`)
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    discovering.value = false
  }
}

const filterFields = computed(() => platform.value?.params || [])

function initParams(target, fields) {
  for (const key of Object.keys(target)) delete target[key]
  for (const field of fields) {
    target[field.key] = field.type === 'switch' ? Boolean(field.default) : field.default ?? ''
  }
}

function openFilter() {
  initParams(filterParams, filterFields.value)
  filterDialog.value = true
}

function applyFilter() {
  filterDialog.value = false
  refresh()
}

function clearFilter() {
  initParams(filterParams, [])
  filterDialog.value = false
  refresh()
}

function openBatch(actionId = '') {
  batchActionId.value = actionId || listActions.value[0]?.id || ''
  initParams(batchParams, batchFields.value)
  batchDialog.value = true
}

function onBatchActionChange() {
  initParams(batchParams, batchFields.value)
}

async function confirmBatch() {
  if (!selectedItems.value.length) return
  batchSubmitting.value = true
  try {
    let created = 0
    for (const item of selectedItems.value) {
      const params = buildParams(batchFields.value, batchParams, discovery.value?.param_map, item.fields)
      await tasksApi.create({
        platform: platformId.value,
        action: batchActionId.value,
        credential_id: accountId.value || undefined,
        params,
      })
      created += 1
    }
    batchDialog.value = false
    selected.value = []
    ui.notify(`已创建 ${created} 个任务`)
    router.push('/tasks')
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    batchSubmitting.value = false
  }
}

function openDetail(item) {
  detailTarget.value = item
  detailDialog.value = true
}

function openExecute(action, item) {
  execTarget.value = { action, item }
  for (const key of Object.keys(execParams)) delete execParams[key]
  const fields = action?.params?.length ? action.params : platform.value?.params || []
  for (const field of fields) {
    execParams[field.key] = field.type === 'switch' ? Boolean(field.default) : field.default ?? ''
  }
  execDialog.value = true
}

async function confirmExecute() {
  const { action, item } = execTarget.value || {}
  if (!action) return
  executing.value = true
  try {
    const params = buildParams(execFields.value, execParams, discovery.value?.param_map, item?.fields)
    const task = await tasksApi.create({
      platform: platformId.value,
      action: action.id,
      credential_id: accountId.value || undefined,
      params,
    })
    execDialog.value = false
    ui.notify('任务已创建')
    router.push(`/tasks/${task.id}`)
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    executing.value = false
  }
}
</script>

<template>
  <v-container>
    <v-card class="mb-4">
      <v-card-text class="pa-3 d-flex align-center flex-nowrap ga-2">
        <div class="d-none d-sm-flex align-center ga-2 flex-shrink-0">
          <v-avatar size="36" color="primary-light">
            <v-icon :icon="platform?.icon || 'mdi-puzzle-outline'" />
          </v-avatar>
          <div class="platform-title d-none d-md-block">
            <div class="text-title-medium text-truncate">{{ platform?.name || platformId }}</div>
            <div class="d-none d-lg-block text-body-small text-medium-emphasis text-truncate">
              {{ platform?.author || '未知作者' }}
            </div>
          </div>
        </div>

        <v-text-field
          v-model="search"
          class="platform-search flex-grow-0 ms-auto"
          prepend-inner-icon="mdi-magnify"
          label="搜索"
          aria-label="搜索"
          density="compact"
          hide-details
          clearable
        />

        <div class="d-flex align-center ga-1 flex-shrink-0">
          <v-select
            v-if="accountOptions.length"
            v-model="accountId"
            :items="accountOptions"
            item-title="title"
            item-value="value"
            label="账号"
            density="compact"
            hide-details
            class="platform-account flex-shrink-0 d-none d-sm-block"
          />
          <v-menu v-if="accountOptions.length" location="bottom end">
            <template #activator="{ props: menuProps }">
              <v-btn
                v-bind="menuProps"
                class="d-sm-none"
                variant="text"
                icon
                aria-label="切换账号"
              >
                <v-icon icon="mdi-account-switch-outline" />
                <v-tooltip activator="parent" location="bottom">切换账号</v-tooltip>
              </v-btn>
            </template>
            <v-list density="compact">
              <v-list-item
                v-for="option in accountOptions"
                :key="option.value"
                :title="option.title"
                :active="accountId === option.value"
                @click="accountId = option.value"
              />
            </v-list>
          </v-menu>
          <v-btn
            v-if="discovery && filterFields.length"
            variant="text"
            icon
            aria-label="筛选"
            @click="openFilter"
          >
            <v-icon icon="mdi-filter-outline" />
            <v-tooltip activator="parent" location="bottom">筛选</v-tooltip>
          </v-btn>
          <v-btn
            v-if="discovery"
            variant="text"
            icon
            :aria-label="viewMode === 'table' ? '切换为卡片视图' : '切换为表格视图'"
            @click="toggleView"
          >
            <v-icon :icon="viewMode === 'table' ? 'mdi-view-grid-outline' : 'mdi-table'" />
            <v-tooltip activator="parent" location="bottom">
              {{ viewMode === 'table' ? '卡片视图' : '表格视图' }}
            </v-tooltip>
          </v-btn>
          <v-btn
            v-if="discovery"
            variant="text"
            icon
            :aria-label="refreshedAt ? `刷新于 ${formatAgo(refreshedAt)}` : '刷新'"
            :loading="discovering"
            @click="refresh"
          >
            <v-icon icon="mdi-refresh" />
            <v-tooltip activator="parent" location="bottom">
              {{ refreshedAt ? `刷新于 ${formatAgo(refreshedAt)}` : '刷新' }}
            </v-tooltip>
          </v-btn>
          <v-btn
            v-for="action in !discovery ? listActions : []"
            :key="action.id"
            color="primary"
            @click="openExecute(action, null)"
          >
            {{ action.name }}
          </v-btn>
        </div>
      </v-card-text>
    </v-card>

    <v-card v-if="!platform" class="pa-6 text-medium-emphasis">平台不存在或未启用</v-card>

    <template v-else-if="discovery">
      <v-alert v-if="!accountId" type="info" variant="tonal" class="mb-3">
        该平台还没有凭据，请先到「凭据」页添加后再刷新列表。
      </v-alert>

      <v-card v-if="viewMode === 'table'" class="overflow-x-auto">
        <DataTable
          v-model="selected"
          :headers="headers"
          :items="filteredItems"
          :loading="loading"
          :paginated="false"
          item-value="key"
          show-select
        >
          <template v-for="col in importantColumns" :key="col.key" #[fieldSlot(col.key)]="{ item }">
            <div v-if="col.type === 'progress'" class="d-flex align-center ga-3">
              <v-progress-linear
                :model-value="parseProgress(item.fields[col.key])"
                height="6"
                rounded
                color="primary"
                class="flex-grow-1"
              />
              <span class="text-body-small text-medium-emphasis">
                {{ parseProgress(item.fields[col.key]) }}%
              </span>
            </div>
            <PlatformField v-else :value="item.fields[col.key]" :type="col.type" />
          </template>
          <template #item.actions="{ item }">
            <div class="d-flex align-center justify-end ga-1">
              <v-menu v-if="listActions.length" location="bottom end">
                <template #activator="{ props: menuProps }">
                  <v-btn
                    v-bind="menuProps"
                    size="small"
                    variant="tonal"
                    color="primary"
                    prepend-icon="mdi-play"
                  >
                    动作
                  </v-btn>
                </template>
                <v-list density="compact">
                  <v-list-item
                    v-for="action in listActions"
                    :key="action.id"
                    :title="action.name"
                    prepend-icon="mdi-play-circle-outline"
                    @click="openExecute(action, item)"
                  />
                </v-list>
              </v-menu>
              <v-btn size="small" variant="text" icon aria-label="详细信息" @click.stop="openDetail(item)">
                <v-icon icon="mdi-information-outline" />
                <v-tooltip activator="parent" location="bottom">详细信息</v-tooltip>
              </v-btn>
            </div>
          </template>
          <template #no-data>
            <div class="pa-6 text-medium-emphasis">暂无数据，点击刷新按钮获取</div>
          </template>
        </DataTable>
      </v-card>

      <v-row v-else>
        <v-col v-for="item in filteredItems" :key="item.key" cols="12" sm="6" md="4">
          <v-card class="h-100 d-flex flex-column">
            <v-card-item>
              <template #append>
                <div class="d-flex align-center ga-3">
                  <v-progress-circular
                    v-if="progressColumn"
                    :model-value="progressValue(item)"
                    :color="progressValue(item) >= 100 ? 'success' : 'primary'"
                    :size="52"
                    :width="5"
                  >
                    <span class="text-label-small">{{ progressValue(item) }}%</span>
                  </v-progress-circular>
                  <v-checkbox-btn
                    v-model="selected"
                    :value="item.key"
                    color="primary"
                    density="compact"
                    hide-details
                    :aria-label="'选择 ' + item.fields[primaryKey]"
                  />
                </div>
              </template>
              <v-card-title class="text-title-medium text-wrap">{{ item.fields[primaryKey] }}</v-card-title>
            </v-card-item>
            <v-card-text class="flex-grow-1">
              <div
                v-for="col in detailColumns"
                :key="col.key"
                class="d-flex align-center justify-space-between ga-3 mb-2"
              >
                <span class="text-body-small text-medium-emphasis">{{ col.label }}</span>
                <PlatformField :value="item.fields[col.key]" :type="col.type" />
              </div>
            </v-card-text>
            <v-card-actions class="flex-wrap ga-1">
              <v-btn
                v-for="action in listActions"
                :key="action.id"
                size="small"
                variant="tonal"
                color="primary"
                @click="openExecute(action, item)"
              >
                {{ action.name }}
              </v-btn>
            </v-card-actions>
          </v-card>
        </v-col>
        <v-col v-if="!filteredItems.length" cols="12">
          <v-card class="pa-6 text-medium-emphasis">暂无数据，点击刷新按钮获取</v-card>
        </v-col>
      </v-row>
    </template>

    <v-card v-else-if="platform && platform.params?.length" class="pa-5">
      <div class="text-title-medium mb-3">执行</div>
      <v-btn
        v-for="action in listActions"
        :key="action.id"
        color="primary"
        class="me-2 mb-2"
        @click="openExecute(action, null)"
      >
        {{ action.name }}
      </v-btn>
    </v-card>

    <v-dialog v-model="filterDialog" max-width="560" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large">筛选</v-card-title>
        </v-card-item>
        <v-card-text class="d-flex flex-column ga-4">
          <div class="d-flex flex-column ga-4">
            <ParamFields :fields="filterFields" :values="filterParams" />
          </div>
        </v-card-text>
        <v-card-actions class="justify-end ga-2 pa-4">
          <v-btn variant="text" @click="clearFilter">清空</v-btn>
          <v-btn variant="text" @click="filterDialog = false">取消</v-btn>
          <v-btn color="primary" @click="applyFilter">应用</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="batchDialog" max-width="560" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large">批量执行</v-card-title>
          <v-card-subtitle>{{ selectedItems.length }} 个条目</v-card-subtitle>
        </v-card-item>
        <v-card-text class="d-flex flex-column ga-4">
          <v-select
            v-model="batchActionId"
            :items="listActions"
            item-title="name"
            item-value="id"
            label="动作"
            @update:model-value="onBatchActionChange"
          />
          <div class="d-flex flex-column ga-4">
            <ParamFields :fields="batchFields" :values="batchParams" />
          </div>
        </v-card-text>
        <v-card-actions class="justify-end ga-2 pa-4">
          <v-btn variant="text" @click="batchDialog = false">取消</v-btn>
          <v-btn color="primary" :loading="batchSubmitting" @click="confirmBatch">执行</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="execDialog" max-width="640" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large">{{ execTarget?.action?.name }}</v-card-title>
          <v-card-subtitle v-if="execTarget?.action?.description">
            {{ execTarget.action.description }}
          </v-card-subtitle>
        </v-card-item>
        <v-card-text class="d-flex flex-column ga-4">
          <v-sheet
            v-if="execTarget?.item"
            color="primary-light"
            rounded="xl"
            class="pa-4"
          >
            <div class="d-flex align-center justify-space-between ga-4">
              <div class="text-title-medium">{{ execTarget.item.fields[primaryKey] }}</div>
              <v-progress-circular
                v-if="progressColumn"
                :model-value="progressValue(execTarget.item)"
                :color="progressValue(execTarget.item) >= 100 ? 'success' : 'primary'"
                :size="44"
                :width="4"
              >
                <span class="text-label-small">{{ progressValue(execTarget.item) }}%</span>
              </v-progress-circular>
            </div>
            <div class="d-flex flex-wrap ga-4 mt-2">
              <div v-for="col in summaryColumns" :key="col.key">
                <div class="text-label-medium text-medium-emphasis">{{ col.label }}</div>
                <PlatformField :value="execTarget.item.fields[col.key]" :type="col.type" />
              </div>
            </div>
          </v-sheet>
          <v-expansion-panels v-if="execFields.length" variant="accordion" flat>
            <v-expansion-panel>
              <v-expansion-panel-title class="text-body-medium">执行参数</v-expansion-panel-title>
              <v-expansion-panel-text>
                <div class="d-flex flex-column ga-4">
                  <ParamFields :fields="execFields" :values="execParams" />
                </div>
              </v-expansion-panel-text>
            </v-expansion-panel>
          </v-expansion-panels>
        </v-card-text>
        <v-card-actions class="justify-end ga-2 pa-4">
          <v-btn variant="text" @click="execDialog = false">取消</v-btn>
          <v-btn color="primary" :loading="executing" @click="confirmExecute">执行</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="detailDialog" max-width="720">
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large text-wrap">
            {{ detailTarget?.fields[primaryKey] }}
          </v-card-title>
          <v-card-subtitle>详细信息</v-card-subtitle>
        </v-card-item>
        <v-card-text class="d-flex flex-column ga-4">
          <div v-if="progressColumn" class="d-flex align-center justify-space-between ga-4">
            <span class="text-body-medium text-medium-emphasis">{{ progressColumn.label }}</span>
            <v-progress-circular
              :model-value="progressValue(detailTarget)"
              :color="progressValue(detailTarget) >= 100 ? 'success' : 'primary'"
              :size="48"
              :width="5"
            >
              <span class="text-label-small">{{ progressValue(detailTarget) }}%</span>
            </v-progress-circular>
          </div>
          <v-row density="comfortable">
            <v-col v-for="col in detailColumns" :key="col.key" cols="12" sm="6">
              <div class="text-label-medium text-medium-emphasis">{{ col.label }}</div>
              <PlatformField :value="detailTarget?.fields[col.key]" :type="col.type" />
            </v-col>
          </v-row>
        </v-card-text>
        <v-card-actions class="justify-end pa-4">
          <v-btn variant="text" @click="detailDialog = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-speed-dial v-if="selectedItems.length" v-model="batchDial" location="top" transition="fade-transition">
      <template #activator="{ props: activatorProps }">
        <v-btn
          v-bind="activatorProps"
          class="batch-fab position-fixed bottom-0 right-0 ma-6"
          color="primary"
          size="large"
          prepend-icon="mdi-checkbox-multiple-marked"
          :aria-label="`批量操作，已选 ${selectedItems.length} 项`"
        >
          {{ selectedItems.length }}
        </v-btn>
      </template>
      <v-btn
        v-for="action in listActions"
        :key="action.id"
        color="primary"
        prepend-icon="mdi-play"
        @click="openBatch(action.id)"
      >
        {{ action.name }}
      </v-btn>
      <v-btn key="clear-selection" variant="tonal" prepend-icon="mdi-close" @click="clearSelection">
        清除选择
      </v-btn>
    </v-speed-dial>
  </v-container>
</template>

<style scoped>
.platform-search {
  width: 200px;
  min-width: 110px;
}

@media (max-width: 599px) {
  .platform-search {
    min-width: 48px;
  }
}

.platform-account {
  width: 160px;
}

.platform-title {
  min-width: 0;
  max-width: 120px;
}

@media (min-width: 1280px) {
  .platform-title {
    max-width: 220px;
  }
}

.batch-fab {
  z-index: 1000;
}
</style>
