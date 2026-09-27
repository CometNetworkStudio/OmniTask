<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import { platformsApi } from '@/api/services/platforms'
import { tasksApi } from '@/api/services/tasks'
import { credentialsApi } from '@/api/services/credentials'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { buildParams } from '@/utils/taskParams'
import { formatAgo } from '@/utils/format'
import DataTable from '@/components/DataTable.vue'
import PlatformField from '@/components/PlatformField.vue'

const VIEW_KEY = 'omnitask_platform_view'

const route = useRoute()
const router = useRouter()
const { mobile, xs } = useDisplay()
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
const viewMode = ref(localStorage.getItem(VIEW_KEY) || (mobile.value ? 'grid' : 'table'))

const execDialog = ref(false)
const execTarget = ref(null)
const execParams = reactive({})

const platformId = computed(() => route.params.id)
const platform = computed(() => platforms.platform(platformId.value))
const discovery = computed(() => platform.value?.discovery || null)
const columns = computed(() => discovery.value?.columns || [])
const paramFields = computed(() => platform.value?.params || [])
const primaryKey = computed(() => discovery.value?.primary || columns.value[0]?.key || '')
const otherColumns = computed(() => columns.value.filter((c) => c.key !== primaryKey.value))

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
  ...columns.value.map((col) => ({ title: col.label, key: `f_${col.key}`, type: col.type })),
  { title: '', key: 'actions', sortable: false, align: 'end' },
])

const filteredItems = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return items.value
  return items.value.filter((item) =>
    Object.values(item.fields || {}).some((v) => String(v).toLowerCase().includes(keyword)),
  )
})

function fieldSlot(key) {
  return `item.f_${key}`
}

watch(viewMode, (mode) => localStorage.setItem(VIEW_KEY, mode))

onMounted(init)
watch(platformId, init)
watch(accountId, () => loadCached())

async function init() {
  await platforms.load()
  await loadCredentials()
  items.value = []
  refreshedAt.value = null
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
    const data = await platformsApi.discover(platformId.value, accountId.value)
    items.value = data.items || []
    refreshedAt.value = data.refreshed_at || null
    ui.notify(`已刷新：${items.value.length} 条`)
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    discovering.value = false
  }
}

function openExecute(action, item) {
  execTarget.value = { action, item }
  for (const key of Object.keys(execParams)) delete execParams[key]
  for (const field of paramFields.value) {
    execParams[field.key] = field.type === 'switch' ? false : ''
  }
  execDialog.value = true
}

async function confirmExecute() {
  const { action, item } = execTarget.value || {}
  if (!action) return
  executing.value = true
  try {
    const params = buildParams(paramFields.value, execParams, discovery.value?.param_map, item?.fields)
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
      <v-card-text class="pa-4">
        <v-row align="center" class="flex-wrap flex-md-nowrap">
          <v-col cols="12" md="3">
            <div class="d-flex align-center ga-3">
              <v-avatar size="40" color="primary-light">
                <v-icon :icon="platform?.icon || 'mdi-puzzle-outline'" />
              </v-avatar>
              <div class="text-truncate">
                <div class="text-headline-small text-truncate">{{ platform?.name || platformId }}</div>
                <div class="text-body-small text-medium-emphasis text-truncate">
                  v{{ platform?.version || '?' }}{{ discovery?.label ? ` · ${discovery.label}` : '' }}
                  <span v-if="refreshedAt"> · 刷新于 {{ formatAgo(refreshedAt) }}</span>
                </div>
              </div>
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              v-model="search"
              prepend-inner-icon="mdi-magnify"
              label="搜索"
              density="compact"
              hide-details
              clearable
            />
          </v-col>
          <v-col v-if="accountOptions.length" cols="6" sm="4" md="2" class="ms-md-auto">
            <v-select
              v-model="accountId"
              :items="accountOptions"
              item-title="title"
              item-value="value"
              label="账号"
              density="compact"
              hide-details
            />
          </v-col>
          <v-col :cols="accountOptions.length ? 6 : 12" sm="8" md="auto" class="d-flex align-center justify-end ga-2 flex-wrap">
            <v-btn-toggle v-model="viewMode" mandatory variant="outlined" divided rounded="lg">
              <v-btn icon="mdi-table" value="table" />
              <v-btn icon="mdi-view-grid-outline" value="grid" />
            </v-btn-toggle>
            <v-btn
              v-if="discovery && xs"
              icon="mdi-refresh"
              variant="tonal"
              :loading="discovering"
              @click="refresh"
            />
            <v-btn
              v-else-if="discovery"
              variant="tonal"
              prepend-icon="mdi-refresh"
              :loading="discovering"
              @click="refresh"
            >
              刷新
            </v-btn>
            <v-btn
              v-for="action in !discovery ? listActions : []"
              :key="action.id"
              color="primary"
              @click="openExecute(action, null)"
            >
              {{ action.name }}
            </v-btn>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <v-card v-if="!platform" class="pa-6 text-medium-emphasis">平台不存在或未启用</v-card>

    <template v-else-if="discovery">
      <v-alert v-if="!accountId" type="info" variant="tonal" class="mb-3">
        该平台还没有凭据，请先到「凭据」页添加后再刷新列表。
      </v-alert>

      <v-card v-if="viewMode === 'table'">
        <DataTable :headers="headers" :items="filteredItems" :loading="loading" :paginated="false" item-value="key">
          <template v-for="col in columns" :key="col.key" #[fieldSlot(col.key)]="{ item }">
            <PlatformField :value="item.fields[col.key]" :type="col.type" />
          </template>
          <template #item.actions="{ item }">
            <div class="d-flex align-center justify-end ga-1">
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
            </div>
          </template>
          <template #no-data>
            <div class="pa-6 text-medium-emphasis">暂无数据，点击右上角「刷新列表」获取</div>
          </template>
        </DataTable>
      </v-card>

      <v-row v-else>
        <v-col v-for="item in filteredItems" :key="item.key" cols="12" sm="6" md="4">
          <v-card class="h-100 d-flex flex-column">
            <v-card-item>
              <v-card-title class="text-title-medium">{{ item.fields[primaryKey] }}</v-card-title>
            </v-card-item>
            <v-card-text class="flex-grow-1">
              <div
                v-for="col in otherColumns"
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
          <v-card class="pa-6 text-medium-emphasis">暂无数据，点击右上角「刷新列表」获取</v-card>
        </v-col>
      </v-row>
    </template>

    <v-card v-else-if="platform && paramFields.length" class="pa-5">
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

    <v-dialog v-model="execDialog" max-width="640" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large">{{ execTarget?.action?.name }}</v-card-title>
        </v-card-item>
        <v-card-text>
          <template v-if="execTarget?.item">
            <v-list density="compact" class="mb-2">
              <v-list-item v-for="col in columns" :key="col.key">
                <v-list-item-title class="text-body-small text-medium-emphasis">{{ col.label }}</v-list-item-title>
                <PlatformField :value="execTarget.item.fields[col.key]" :type="col.type" />
              </v-list-item>
            </v-list>
          </template>
          <v-expansion-panels v-if="paramFields.length" variant="accordion" flat>
            <v-expansion-panel>
              <v-expansion-panel-title class="text-body-medium">执行参数（可选，留空使用默认值）</v-expansion-panel-title>
              <v-expansion-panel-text>
                <template v-for="field in paramFields" :key="field.key">
                  <v-switch
                    v-if="field.type === 'switch'"
                    v-model="execParams[field.key]"
                    :label="field.label"
                    color="primary"
                    hide-details
                  />
                  <v-text-field
                    v-else
                    v-model="execParams[field.key]"
                    :label="field.label"
                    :placeholder="field.default != null ? String(field.default) : ''"
                    :type="field.type === 'number' ? 'number' : 'text'"
                  />
                </template>
              </v-expansion-panel-text>
            </v-expansion-panel>
          </v-expansion-panels>
        </v-card-text>
        <v-card-actions class="justify-end">
          <v-btn variant="text" @click="execDialog = false">取消</v-btn>
          <v-btn color="primary" :loading="executing" @click="confirmExecute">执行</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>
