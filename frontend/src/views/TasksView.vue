<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { tasksApi } from '@/api/services/tasks'
import { PLATFORMS, defaultParams, platformByValue } from '@/config/platforms'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { formatTime, statusMeta } from '@/utils/format'

const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()

const tasks = ref([])
const loading = ref(false)
const dialog = ref(false)
const submitting = ref(false)

const form = reactive({
  platform: PLATFORMS[0].value,
  action: PLATFORMS[0].actions[0].value,
  params: defaultParams(PLATFORMS[0].value),
})

const platform = computed(() => platformByValue(form.platform))
const actionOptions = computed(() => platform.value?.actions || [])
const paramSchema = computed(() => platform.value?.params || [])

const headers = [
  { title: '平台', key: 'platform' },
  { title: '动作', key: 'action' },
  { title: '状态', key: 'status' },
  { title: '进度', key: 'progress' },
  { title: '创建时间', key: 'created_at' },
  { title: '', key: 'actions', sortable: false, align: 'end' },
]

onMounted(load)

async function load() {
  loading.value = true
  try {
    const data = await tasksApi.list(1, 100)
    tasks.value = data.list || []
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

function onPlatformChange() {
  form.action = actionOptions.value[0]?.value || ''
  form.params = defaultParams(form.platform)
}

function buildParams() {
  const out = {}
  paramSchema.value.forEach((item) => {
    const value = form.params[item.key]
    if (item.type === 'switch') {
      if (value) out[item.key] = 'true'
      return
    }
    if (value === '' || value === null || value === undefined) return
    out[item.key] = String(value)
  })
  return out
}

async function createTask() {
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
          <v-btn
            variant="text"
            prepend-icon="mdi-refresh"
            :loading="loading"
            @click="load"
          >
            刷新
          </v-btn>
          <v-btn color="primary" prepend-icon="mdi-plus" @click="dialog = true">
            新建任务
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-card>
      <v-data-table
        :headers="headers"
        :items="tasks"
        :loading="loading"
        item-value="id"
        hover
        @click:row="(_, { item }) => router.push(`/tasks/${item.id}`)"
      >
        <template #item.platform="{ item }">
          <v-chip size="small" variant="tonal" :prepend-icon="platformByValue(item.platform)?.icon">
            {{ platformByValue(item.platform)?.title || item.platform }}
          </v-chip>
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
        <template #item.created_at="{ item }">
          {{ formatTime(item.created_at) }}
        </template>
        <template #item.actions="{ item }">
          <v-btn
            size="small"
            variant="text"
            color="primary"
            @click.stop="router.push(`/tasks/${item.id}`)"
          >
            详情
          </v-btn>
        </template>
        <template #no-data>
          <div class="pa-6 text-medium-emphasis">暂无任务，点击右上角「新建任务」</div>
        </template>
      </v-data-table>
    </v-card>

    <v-dialog v-model="dialog" max-width="640">
      <v-card>
        <v-card-item>
          <v-card-title class="text-h6">新建任务</v-card-title>
        </v-card-item>
        <v-card-text>
          <v-form @submit.prevent="createTask">
            <v-select
              v-model="form.platform"
              :items="PLATFORMS"
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
            <v-divider class="my-4" />
            <div class="text-subtitle-2 mb-2">参数</div>
            <template v-for="item in paramSchema" :key="item.key">
              <v-switch
                v-if="item.type === 'switch'"
                v-model="form.params[item.key]"
                :label="item.label"
                color="primary"
                hide-details
                inset
              />
              <v-text-field
                v-else
                v-model="form.params[item.key]"
                :label="item.label"
                :type="item.type === 'number' ? 'number' : 'text'"
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
