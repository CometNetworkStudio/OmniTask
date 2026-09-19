<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { tasksApi } from '@/api/services/tasks'
import { platformByValue } from '@/config/platforms'
import { useUiStore } from '@/stores/ui'
import { formatTime, statusMeta } from '@/utils/format'
import { connectTaskSocket } from '@/ws/taskSocket'

const route = useRoute()
const router = useRouter()
const ui = useUiStore()

const task = ref(null)
const events = ref([])
let socket = null

onMounted(async () => {
  try {
    task.value = await tasksApi.get(route.params.id)
  } catch (err) {
    ui.notify(err.message, 'error')
  }
  socket = connectTaskSocket(route.params.id, {
    onEvent: (event) => {
      events.value.unshift(event)
      if (task.value) {
        task.value.status = event.status || task.value.status
        if (typeof event.percent === 'number') task.value.progress = event.percent
      }
    },
  })
})

onBeforeUnmount(() => socket?.close())
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="d-flex align-center ga-2">
          <v-btn variant="text" icon="mdi-arrow-left" @click="router.push('/')" />
          <div class="text-h5">任务详情</div>
        </div>
      </v-col>
    </v-row>

    <v-card v-if="task">
      <v-card-text>
        <v-row>
          <v-col cols="12" md="6">
            <div class="text-caption text-medium-emphasis">平台</div>
            <div>{{ platformByValue(task.platform)?.title || task.platform }}</div>
          </v-col>
          <v-col cols="12" md="6">
            <div class="text-caption text-medium-emphasis">动作</div>
            <div>{{ task.action }}</div>
          </v-col>
          <v-col cols="12" md="6">
            <div class="text-caption text-medium-emphasis">状态</div>
            <v-chip size="small" :color="statusMeta(task.status).color" variant="tonal">
              {{ statusMeta(task.status).text }}
            </v-chip>
          </v-col>
          <v-col cols="12" md="6">
            <div class="text-caption text-medium-emphasis">创建时间</div>
            <div>{{ formatTime(task.created_at) }}</div>
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

        <template v-if="Object.keys(task.params || {}).length">
          <div class="text-subtitle-2 mt-6 mb-2">参数</div>
          <v-chip
            v-for="(value, key) in task.params"
            :key="key"
            size="small"
            variant="tonal"
            class="ma-1"
          >
            {{ key }}={{ value }}
          </v-chip>
        </template>
      </v-card-text>
    </v-card>

    <v-card class="mt-4">
      <v-card-item>
        <v-card-title class="text-subtitle-1">实时进度</v-card-title>
      </v-card-item>
      <v-card-text>
        <v-timeline v-if="events.length" density="compact" side="end">
          <v-timeline-item
            v-for="(event, index) in events"
            :key="index"
            :dot-color="statusMeta(event.status).color"
            size="small"
          >
            <div class="text-body-2">{{ event.message || event.status }}</div>
            <div class="text-caption text-medium-emphasis">
              {{ statusMeta(event.status).text }} · {{ event.percent ?? 0 }}%
            </div>
          </v-timeline-item>
        </v-timeline>
        <div v-else class="text-medium-emphasis py-6 text-center">
          等待进度事件…
        </div>
      </v-card-text>
    </v-card>
  </v-container>
</template>
