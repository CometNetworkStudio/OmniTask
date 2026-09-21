<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { tasksApi } from '@/api/services/tasks'
import { platformsApi } from '@/api/services/platforms'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { formatAgo, formatTime, statusMeta } from '@/utils/format'
import DataTable from '@/components/DataTable.vue'

const router = useRouter()
const platforms = usePlatformsStore()
const ui = useUiStore()

const tasks = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(100)
const filterStatus = ref('')
const filterPlatform = ref('')
const loading = ref(false)
const dialog = ref(false)
const submitting = ref(false)
const discovering = ref(false)
const discoveryItems = ref([])
const discoveryRefreshedAt = ref(null)

const statusOptions = [
  { title: '全部状态', value: '' },
  { title: '待执行', value: 'pending' },
  { title: '排队中', value: 'queued' },
  { title: '执行中', value: 'running' },
  { title: '成功', value: 'succeeded' },
  { title: '失败', value: 'failed' },
  { title: '已取消', value: 'canceled' },
  { title: '已中断', value: 'interrupted' },
]
const platformFilterOptions = computed(() => [{ title: '全部平台', value: '' }, ...platforms.options])

const form = reactive({ platform: '', action: '', params: {}, selectedKey: '' })
const platform = computed(() => platforms.platform(form.platform))
const actionOptions = computed(() => platform.value?.actions || [])
const paramSchema = computed(() => platform.value?.params || [])
const discovery = computed(() => platform.value?.discovery || null)

const headers = [
  { title: '平台', key: 'platform' },
  { title: '动作', key: 'action' },
  { title: '状态', key: 'status' },
  { title: '进度', key: 'progress' },
  { title: '创建时间', key: 'created_at' },
  { title: '', key: 'actions', sortable: false, align: 'end' },
]
const discoveryHeaders = computed(() => {
  const cols = (discovery.value?.columns || []).map((c) => ({ title: c.label, key: `f_${c.key}` }))
  return [{ title: '', key: 'select', sortable: false, width: 48 }, ...cols]
})

onMounted(async () => {
  await platforms.load()
  resetDialog()
  load()
})

async function load() {
  loading.value = true
  try {
    const data = await tasksApi.list({
      page: page.value,
      size: size.value,
      status: filterStatus.value,
      platform: filterPlatform.value,
    })
    tasks.value = data.list || []
    total.value = data.total || 0
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

function onPage(value) {
  page.value = value
  load()
}
function onSize(value) {
  size.value = value
  page.value = 1
  load()
}
function onFilter() {
  page.value = 1
  load()
}

function resetDialog() {
  form.platform = platforms.list[0]?.id || ''
  onPlatformChange()
}

function onPlatformChange() {
  const current = platform.value
  form.action = current?.actions?.[0]?.id || ''
  form.params = {}
  for (const field of current?.params || []) {
    form.params[field.key] = field.default
  }
  form.selectedKey = ''
  discoveryItems.value = []
  discoveryRefreshedAt.value = null
  if (discovery.value) {
    loadCachedItems()
  }
}

async function loadCachedItems() {
  try {
    const data = await platformsApi.items(platform.value.id)
    discoveryItems.value = data.items || []
    discoveryRefreshedAt.value = data.refreshed_at || null
  } catch {
    // 忽略
  }
}

async function refreshDiscovery() {
  discovering.value = true
  try {
    const data = await platformsApi.discover(platform.value.id)
    discoveryItems.value = data.items || []
    discoveryRefreshedAt.value = data.refreshed_at || null
    ui.notify(`已刷新：${discoveryItems.value.length} 条`)
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    discovering.value = false
  }
}

function fieldSlot(key) {
  return `item.f_${key}`
}

function buildParams() {
  const out = {}
  for (const field of paramSchema.value) {
    const value = form.params[field.key]
    if (field.type === 'switch') {
      if (value) out[field.key] = 'true'
      continue
    }
    if (value === '' || value === null || value === undefined) continue
    out[field.key] = String(value)
  }
  if (discovery.value && form.selectedKey) {
    const item = discoveryItems.value.find((it) => it.key === form.selectedKey)
    if (item) {
      for (const [paramKey, fieldKey] of Object.entries(discovery.value.param_map || {})) {
        const value = item.fields[fieldKey]
        if (value !== undefined && value !== null) out[paramKey] = String(value)
      }
    }
  }
  return out
}

async function createTask() {
  if (discovery.value && !form.selectedKey) {
    ui.notify('请先刷新并选择一条任务/课程', 'warning')
    return
  }
  submitting.value = true
  try {
    const task = await tasksApi.create({
      platform: form.platform,
      action: form.action,
      params: buildParams(),
    })
    dialog.value = false
    ui.notify('任务已创建')
    await load()
    router.push(`/tasks/${task.id}`)
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    submitting.value = false
  }
}

async function cancelRow(item) {
  try {
    await tasksApi.cancel(item.id)
    ui.notify('已取消')
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}
async function retryRow(item) {
  try {
    await tasksApi.retry(item.id)
    ui.notify('已重新排队')
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}
async function removeRow(item) {
  try {
    await tasksApi.remove(item.id)
    ui.notify('已删除')
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="text-h5">任务</div>
        <div class="text-medium-emphasis">创建并查看自动任务</div>
      </v-col>
      <v-col cols="auto">
        <div class="d-flex align-center ga-2">
          <v-btn variant="text" prepend-icon="mdi-refresh" :loading="loading" @click="load">刷新</v-btn>
          <v-btn color="primary" prepend-icon="mdi-plus" @click="dialog = true">新建任务</v-btn>
        </div>
      </v-col>
    </v-row>

    <v-row align="center" class="mb-2">
      <v-col cols="12" sm="6" md="3">
        <v-select
          v-model="filterStatus"
          :items="statusOptions"
          item-title="title"
          item-value="value"
          label="状态"
          density="compact"
          hide-details
          @update:model-value="onFilter"
        />
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-select
          v-model="filterPlatform"
          :items="platformFilterOptions"
          item-title="title"
          item-value="value"
          label="平台"
          density="compact"
          hide-details
          @update:model-value="onFilter"
        />
      </v-col>
    </v-row>

    <v-card>
      <DataTable
        :headers="headers"
        :items="tasks"
        :loading="loading"
        :total="total"
        :page="page"
        :size="size"
        item-value="id"
        @update:page="onPage"
        @update:size="onSize"
        @click:row="(_, { item }) => router.push(`/tasks/${item.id}`)"
      >
        <template #item.platform="{ item }">
          <v-chip size="small" variant="tonal">{{ platforms.name(item.platform) }}</v-chip>
        </template>
        <template #item.status="{ item }">
          <v-chip size="small" :color="statusMeta(item.status).color" variant="tonal">
            {{ statusMeta(item.status).text }}
          </v-chip>
        </template>
        <template #item.progress="{ item }">
          <v-progress-linear
            :model-value="item.progress"
            :color="statusMeta(item.status).color"
            height="8"
            rounded
            class="my-2"
          />
        </template>
        <template #item.created_at="{ item }">{{ formatTime(item.created_at) }}</template>
        <template #item.actions="{ item }">
          <div class="d-flex align-center justify-end ga-1">
            <v-btn size="small" variant="text" color="primary" @click.stop="router.push(`/tasks/${item.id}`)">
              详情
            </v-btn>
            <v-menu location="bottom end">
              <template #activator="{ props }">
                <v-btn size="small" variant="text" icon="mdi-dots-vertical" v-bind="props" @click.stop />
              </template>
              <v-list density="compact">
                <v-list-item
                  v-if="['pending', 'queued', 'running'].includes(item.status)"
                  title="取消"
                  prepend-icon="mdi-stop"
                  @click="cancelRow(item)"
                />
                <v-list-item
                  v-if="['failed', 'canceled', 'interrupted'].includes(item.status)"
                  title="重试"
                  prepend-icon="mdi-refresh"
                  @click="retryRow(item)"
                />
                <v-list-item
                  v-if="!['running', 'queued'].includes(item.status)"
                  title="删除"
                  prepend-icon="mdi-delete"
                  @click="removeRow(item)"
                />
              </v-list>
            </v-menu>
          </div>
        </template>
        <template #no-data>
          <div class="pa-6 text-medium-emphasis">暂无任务，点击右上角「新建任务」</div>
        </template>
      </DataTable>
    </v-card>

    <v-dialog v-model="dialog" max-width="720" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-h6">新建任务</v-card-title>
        </v-card-item>
        <v-card-text>
          <v-form @submit.prevent="createTask">
            <v-select
              v-model="form.platform"
              :items="platforms.options"
              item-title="title"
              item-value="value"
              label="平台"
              @update:model-value="onPlatformChange"
            />
            <v-select
              v-model="form.action"
              :items="actionOptions"
              item-title="title"
              item-value="value"
              label="动作"
            />

            <template v-if="discovery">
              <v-divider class="my-4" />
              <div class="d-flex align-center ga-2 mb-2">
                <div class="text-subtitle-2">{{ discovery.label }}（选择要处理的一项）</div>
                <v-spacer />
                <span v-if="discoveryRefreshedAt" class="text-caption text-medium-emphasis">
                  刷新于 {{ formatAgo(discoveryRefreshedAt) }}
                </span>
                <v-btn size="small" variant="tonal" :loading="discovering" @click="refreshDiscovery">
                  刷新列表
                </v-btn>
              </div>
              <DataTable
                :headers="discoveryHeaders"
                :items="discoveryItems"
                :paginated="false"
                item-value="key"
              >
                <template #item.select="{ item }">
                  <v-radio
                    :model-value="form.selectedKey"
                    :value="item.key"
                    color="primary"
                    hide-details
                    @update:model-value="form.selectedKey = item.key"
                  />
                </template>
                <template v-for="col in discovery.columns" #[fieldSlot(col.key)]="{ item }" :key="col.key">
                  <span>{{ item.fields[col.key] }}</span>
                </template>
                <template #no-data>
                  <div class="pa-4 text-medium-emphasis">暂无数据，点击「刷新列表」获取</div>
                </template>
              </DataTable>
            </template>

            <v-divider class="my-4" />
            <div class="text-subtitle-2 mb-2">执行参数</div>
            <template v-for="field in paramSchema" :key="field.key">
              <v-switch
                v-if="field.type === 'switch'"
                v-model="form.params[field.key]"
                :label="field.label"
                color="primary"
                hide-details
                inset
              />
              <v-text-field
                v-else
                v-model="form.params[field.key]"
                :label="field.label"
                :type="field.type === 'number' ? 'number' : 'text'"
              />
            </template>

            <div class="d-flex justify-end ga-2 mt-4">
              <v-btn variant="text" @click="dialog = false">取消</v-btn>
              <v-btn type="submit" color="primary" :loading="submitting">创建</v-btn>
            </div>
          </v-form>
        </v-card-text>
      </v-card>
    </v-dialog>
  </v-container>
</template>
