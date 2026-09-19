<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { credentialsApi } from '@/api/services/credentials'
import { PLATFORMS, platformByValue } from '@/config/platforms'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'

const ui = useUiStore()
const credentials = ref([])
const loading = ref(false)
const dialog = ref(false)
const saving = ref(false)

const form = reactive({ platform: PLATFORMS[0].value, data: {} })
const platform = computed(() => platformByValue(form.platform))
const fields = computed(() => platform.value?.credentialFields || [])

const headers = [
  { title: '平台', key: 'platform' },
  { title: '更新时间', key: 'updated_at' },
  { title: '', key: 'actions', sortable: false, align: 'end' },
]

onMounted(load)

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
  fields.value.forEach((field) => {
    form.data[field.key] = ''
  })
}

function openDialog(platformValue) {
  form.platform = platformValue || PLATFORMS[0].value
  resetData()
  dialog.value = true
}

async function save() {
  saving.value = true
  try {
    await credentialsApi.upsert(form.platform, { ...form.data })
    ui.notify('凭据已保存')
    dialog.value = false
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="text-h5">凭据</div>
        <div class="text-medium-emphasis">加密存储，仅任务下发时解密</div>
      </v-col>
      <v-col cols="auto">
        <v-btn color="primary" prepend-icon="mdi-plus" @click="openDialog()">添加凭据</v-btn>
      </v-col>
    </v-row>

    <v-card>
      <v-data-table :headers="headers" :items="credentials" :loading="loading" item-value="id">
        <template #item.platform="{ item }">
          <v-chip size="small" variant="tonal" :prepend-icon="platformByValue(item.platform)?.icon">
            {{ platformByValue(item.platform)?.title || item.platform }}
          </v-chip>
        </template>
        <template #item.updated_at="{ item }">
          {{ formatTime(item.updated_at) }}
        </template>
        <template #item.actions="{ item }">
          <v-btn size="small" variant="text" color="primary" @click="openDialog(item.platform)">
            更新
          </v-btn>
        </template>
        <template #no-data>
          <div class="pa-6 text-medium-emphasis">暂无凭据</div>
        </template>
      </v-data-table>
    </v-card>

    <v-dialog v-model="dialog" max-width="520" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-h6">凭据</v-card-title>
        </v-card-item>
        <v-card-text>
          <v-form @submit.prevent="save">
            <v-select
              v-model="form.platform"
              :items="PLATFORMS"
              item-title="title"
              item-value="value"
              label="平台"
              @update:model-value="resetData"
            />
            <v-text-field
              v-for="field in fields"
              :key="field.key"
              v-model="form.data[field.key]"
              :label="field.label"
              :type="field.type === 'password' ? 'password' : 'text'"
            />
            <div class="d-flex justify-end ga-2 mt-4">
              <v-btn variant="text" @click="dialog = false">取消</v-btn>
              <v-btn type="submit" color="primary" :loading="saving">保存</v-btn>
            </div>
          </v-form>
        </v-card-text>
      </v-card>
    </v-dialog>
  </v-container>
</template>
