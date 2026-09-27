<script setup>
import { onMounted, ref } from 'vue'
import { usersApi } from '@/api/services/users'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { formatTime } from '@/utils/format'
import DataTable from '@/components/DataTable.vue'

const auth = useAuthStore()
const ui = useUiStore()

const items = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(100)
const loading = ref(false)

const headers = [
  { title: '用户名', key: 'username' },
  { title: '角色', key: 'role' },
  { title: '状态', key: 'enabled', sortable: false },
  { title: '注册时间', key: 'created_at' },
]

onMounted(load)

async function load() {
  loading.value = true
  try {
    const data = await usersApi.list(page.value, size.value)
    items.value = data.list || []
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

function isSelf(item) {
  return item.id === auth.user?.id
}

async function setRole(item, role) {
  try {
    await usersApi.setRole(item.id, role)
    item.role = role
    ui.notify('角色已更新')
    auth.fetchMe().catch(() => {})
  } catch (err) {
    ui.notify(err.message, 'error')
    await load()
  }
}

async function setEnabled(item, enabled) {
  try {
    await usersApi.setEnabled(item.id, enabled)
    item.enabled = enabled
    ui.notify(enabled ? '已启用' : '已禁用')
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="text-headline-small">用户</div>
        <div class="text-medium-emphasis">管理员管理用户角色与状态</div>
      </v-col>
      <v-col cols="auto">
        <v-btn variant="text" prepend-icon="mdi-refresh" :loading="loading" @click="load">刷新</v-btn>
      </v-col>
    </v-row>

    <v-card>
      <DataTable
        :headers="headers"
        :items="items"
        :loading="loading"
        :total="total"
        :page="page"
        :size="size"
        @update:page="onPage"
        @update:size="onSize"
      >
        <template #item.role="{ item }">
          <v-select
            :model-value="item.role"
            :items="['admin', 'user']"
            density="compact"
            hide-details
            :disabled="isSelf(item)"
            @update:model-value="(value) => setRole(item, value)"
          />
        </template>
        <template #item.enabled="{ item }">
          <v-switch
            :model-value="item.enabled"
            color="primary"
            density="compact"
            hide-details
            :disabled="isSelf(item)"
            @update:model-value="(value) => setEnabled(item, value)"
          />
        </template>
        <template #item.created_at="{ item }">
          {{ formatTime(item.created_at) }}
        </template>
        <template #no-data>
          <div class="pa-6 text-medium-emphasis">暂无用户</div>
        </template>
      </DataTable>
    </v-card>
  </v-container>
</template>
