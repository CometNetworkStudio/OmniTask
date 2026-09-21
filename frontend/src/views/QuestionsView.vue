<script setup>
import { computed, onMounted, ref } from 'vue'
import { questionsApi } from '@/api/services/questions'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'
import DataTable from '@/components/DataTable.vue'

const ui = useUiStore()
const platforms = usePlatformsStore()
const platform = ref('')
const questions = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(100)
const loading = ref(false)

const platformOptions = computed(() => [{ title: '全部平台', value: '' }, ...platforms.options])

const headers = [
  { title: '平台', key: 'platform' },
  { title: '题干', key: 'stem' },
  { title: '来源', key: 'source' },
  { title: '置信度', key: 'confidence' },
  { title: '命中', key: 'hit_count' },
  { title: '更新时间', key: 'updated_at' },
]

onMounted(async () => {
  await platforms.load()
  load()
})

async function load() {
  loading.value = true
  try {
    const data = await questionsApi.list(platform.value, page.value, size.value)
    questions.value = data.list || []
    total.value = data.total || 0
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

function onPlatform() {
  page.value = 1
  load()
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
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="text-h5">题库</div>
        <div class="text-medium-emphasis">{{ total }} 条</div>
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
            @update:model-value="onPlatform"
          />
          <v-btn variant="text" prepend-icon="mdi-refresh" :loading="loading" @click="load">
            刷新
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-card>
      <DataTable
        :headers="headers"
        :items="questions"
        :loading="loading"
        :total="total"
        :page="page"
        :size="size"
        item-value="id"
        @update:page="onPage"
        @update:size="onSize"
      >
        <template #item.platform="{ item }">
          <v-chip size="small" variant="tonal">{{ platforms.name(item.platform) }}</v-chip>
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
      </DataTable>
    </v-card>
  </v-container>
</template>
