<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { credentialsApi } from '@/api/services/credentials'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'
import DataTable from '@/components/DataTable.vue'

const platforms = usePlatformsStore()
const ui = useUiStore()

const credentials = ref([])
const loading = ref(false)
const dialog = ref(false)
const saving = ref(false)
const editingId = ref('')
const deleteTarget = ref(null)
const deleting = ref(false)

const form = reactive({ platform: '', label: '', isDefault: false, data: {} })
const platform = computed(() => platforms.platform(form.platform))
const fields = computed(() => platform.value?.credentials || [])
const isEditing = computed(() => Boolean(editingId.value))

const headers = [
  { title: '平台', key: 'platform' },
  { title: '账号', key: 'account' },
  { title: '默认', key: 'is_default', sortable: false },
  { title: '更新时间', key: 'updated_at' },
  { title: '', key: 'actions', sortable: false, align: 'end' },
]

onMounted(async () => {
  await platforms.load()
  load()
})

async function load() {
  loading.value = true
  try {
    credentials.value = await credentialsApi.list()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

function resetData() {
  form.data = {}
  for (const field of fields.value) {
    form.data[field.key] = ''
  }
}

function openCreate() {
  editingId.value = ''
  form.platform = platforms.list[0]?.id || ''
  form.label = ''
  form.isDefault = false
  resetData()
  dialog.value = true
}

function openEdit(item) {
  editingId.value = item.id
  form.platform = item.platform
  form.label = item.label || ''
  form.isDefault = item.isDefault
  resetData()
  dialog.value = true
}

function collectData() {
  const data = {}
  for (const [key, value] of Object.entries(form.data)) {
    if (value !== '' && value !== null && value !== undefined) data[key] = value
  }
  return data
}

async function save() {
  const data = collectData()
  if (!isEditing.value && Object.keys(data).length === 0) {
    ui.notify('请至少填写一项凭据', 'warning')
    return
  }
  saving.value = true
  try {
    if (isEditing.value) {
      await credentialsApi.update(editingId.value, { data, label: form.label, is_default: form.isDefault })
    } else {
      await credentialsApi.create({ platform: form.platform, data, label: form.label, is_default: form.isDefault })
    }
    ui.notify('已保存')
    dialog.value = false
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    saving.value = false
  }
}

async function setDefault(item) {
  try {
    await credentialsApi.update(item.id, { label: item.label, is_default: true })
    ui.notify('已设为默认')
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await credentialsApi.archive(deleteTarget.value.id)
    ui.notify('已删除（有历史任务时会先归档，任务删净后自动清除）')
    deleteTarget.value = null
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="text-headline-small">凭据</div>
        <div class="text-medium-emphasis">加密存储，仅任务下发时解密；推荐使用令牌 / Cookie，避免保存长期密码</div>
      </v-col>
      <v-col cols="auto">
        <div class="d-flex align-center ga-2">
          <v-btn variant="text" prepend-icon="mdi-history" to="/settings?tab=security">访问记录</v-btn>
          <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreate">添加凭据</v-btn>
        </div>
      </v-col>
    </v-row>

    <v-card>
      <DataTable :headers="headers" :items="credentials" :loading="loading" :paginated="false" item-value="id">
        <template #item.platform="{ item }">
          <v-chip size="small" variant="tonal">{{ platforms.name(item.platform) }}</v-chip>
        </template>
        <template #item.account="{ item }">
          <div class="d-flex flex-column">
            <span>{{ item.label || item.preview || '—' }}</span>
            <span v-if="item.label && item.preview" class="text-body-small text-medium-emphasis">{{ item.preview }}</span>
          </div>
        </template>
        <template #item.is_default="{ item }">
          <v-chip v-if="item.isDefault" size="small" color="primary" variant="tonal" prepend-icon="mdi-star">
            默认
          </v-chip>
          <v-btn v-else size="small" variant="text" @click="setDefault(item)">设为默认</v-btn>
        </template>
        <template #item.updated_at="{ item }">{{ formatTime(item.updated_at) }}</template>
        <template #item.actions="{ item }">
          <div class="d-flex align-center justify-end ga-1">
            <v-btn size="small" variant="text" color="primary" @click="openEdit(item)">编辑</v-btn>
            <v-btn size="small" variant="text" color="error" @click="deleteTarget = item">删除</v-btn>
          </div>
        </template>
        <template #no-data>
          <div class="pa-6 text-medium-emphasis">暂无凭据</div>
        </template>
      </DataTable>
    </v-card>

    <v-dialog v-model="dialog" max-width="520" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large">{{ isEditing ? '编辑凭据' : '添加凭据' }}</v-card-title>
          <v-card-subtitle v-if="isEditing">凭据字段留空则保持不变</v-card-subtitle>
        </v-card-item>
        <v-card-text>
          <v-form @submit.prevent="save">
            <v-select
              v-model="form.platform"
              :items="platforms.options"
              item-title="title"
              item-value="value"
              label="平台"
              :disabled="isEditing"
              @update:model-value="resetData"
            />
            <v-text-field v-model="form.label" label="名称（可选，便于区分账号）" />
            <template v-for="field in fields" :key="field.key">
              <v-textarea
                v-if="field.type === 'cookie'"
                v-model="form.data[field.key]"
                :label="field.label"
                rows="3"
                auto-grow
              />
              <v-text-field
                v-else
                v-model="form.data[field.key]"
                :label="field.label"
                :type="['password', 'token'].includes(field.type) ? 'password' : 'text'"
              />
            </template>
            <v-switch v-model="form.isDefault" label="设为该平台默认凭据" color="primary" hide-details />
            <div class="d-flex justify-end ga-2 mt-4">
              <v-btn variant="text" @click="dialog = false">取消</v-btn>
              <v-btn type="submit" color="primary" :loading="saving">保存</v-btn>
            </div>
          </v-form>
        </v-card-text>
      </v-card>
    </v-dialog>

    <v-dialog :model-value="Boolean(deleteTarget)" max-width="420" persistent @update:model-value="deleteTarget = null">
      <v-card>
        <v-card-item>
          <v-card-title class="text-title-large">删除凭据</v-card-title>
        </v-card-item>
        <v-card-text>
          确定删除「{{ deleteTarget?.label || deleteTarget?.preview || deleteTarget?.platform }}」？
          若该凭据有历史任务，会先归档并从列表隐藏，直到相关任务全部删除后再彻底清除。
        </v-card-text>
        <v-card-actions class="justify-end">
          <v-btn variant="text" @click="deleteTarget = null">取消</v-btn>
          <v-btn color="error" :loading="deleting" @click="confirmDelete">删除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>
