<script setup>
import { onMounted, ref } from 'vue'
import { questionsApi } from '@/api/services/questions'
import { PLATFORMS, platformByValue } from '@/config/platforms'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'

const ui = useUiStore()
const platform = ref('')
const questions = ref([])
const loading = ref(false)

const platformOptions = [{ title: '全部平台', value: '' }, ...PLATFORMS]

const headers = [
  { title: '平台', key: 'platform' },
  { title: '题干', key: 'stem' },
  { title: '来源', key: 'source' },
  { title: '置信度', key: 'confidence' },
  { title: '命中', key: 'hit_count' },
  { title: '更新时间', key: 'updated_at' },
]

onMounted(load)

async function load() {
  loading.value = true
  try {
    const data = await questionsApi.list(platform.value, 1, 100)
    questions.value = data.list || []
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="text-h5">题库</div>
        <div class="text-medium-emphasis">{{ questions.length }} 条</div>
      </v-col>
      <v-col cols="auto">
        <div class="d-flex align-center ga-2">
          <v-select
            v-model="platform"
            :items="platformOptions"
            item-title="title"
            item-value="value"
            label="平台"
            hide-details
            density="compact"
            @update:model-value="load"
          />
          <v-btn variant="text" prepend-icon="mdi-refresh" :loading="loading" @click="load">
            刷新
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-card>
      <v-data-table :headers="headers" :items="questions" :loading="loading" item-value="id">
        <template #item.platform="{ item }">
          <v-chip size="small" variant="tonal" :prepend-icon="platformByValue(item.platform)?.icon">
            {{ platformByValue(item.platform)?.title || item.platform }}
          </v-chip>
        </template>
        <template #item.stem="{ item }">
          <span class="text-truncate">{{ (item.stem || '').slice(0, 60) }}</span>
        </template>
        <template #item.source="{ item }">
          <v-chip size="small" variant="tonal">{{ item.source }}</v-chip>
        </template>
        <template #item.confidence="{ item }">
          {{ Number(item.confidence).toFixed(2) }}
        </template>
        <template #item.updated_at="{ item }">
          {{ formatTime(item.updated_at) }}
        </template>
        <template #no-data>
          <div class="pa-6 text-medium-emphasis">暂无题库数据</div>
        </template>
      </v-data-table>
    </v-card>
  </v-container>
</template>
