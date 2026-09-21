<script setup>
import { onMounted, reactive, ref } from 'vue'
import { scriptsApi } from '@/api/services/scripts'
import { useUiStore } from '@/stores/ui'
import DataTable from '@/components/DataTable.vue'

const ui = useUiStore()
const scripts = ref([])
const loading = ref(false)
const dialog = ref(false)
const installing = ref(false)

const ACCEL = [
  { title: '不使用加速', value: '' },
  { title: 'gh-proxy.com', value: 'https://gh-proxy.com/' },
  { title: 'gh-proxy.org', value: 'https://gh-proxy.org/' },
]

const PRESETS = [
  { title: '词达人 (cidaren)', value: 'https://github.com/CometNetworkStudio/cidaren' },
  { title: 'WeLearn (welearn)', value: 'https://github.com/CometNetworkStudio/welearn' },
]

const form = reactive({ url: '', accel: '' })

const headers = [
  { title: '脚本', key: 'name' },
  { title: '标识', key: 'id' },
  { title: '版本', key: 'version' },
  { title: '运行时', key: 'runtime' },
  { title: '状态', key: 'status' },
  { title: '启用', key: 'enabled' },
  { title: '', key: 'actions', sortable: false, align: 'end' },
]

onMounted(load)

async function load() {
  loading.value = true
  try {
    scripts.value = await scriptsApi.list()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

async function install() {
  installing.value = true
  try {
    const url = form.accel ? `${form.accel}${form.url}` : form.url
    await scriptsApi.install(url)
    ui.notify('脚本安装成功')
    dialog.value = false
    form.url = ''
    form.accel = ''
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    installing.value = false
  }
}

async function toggle(item) {
  try {
    await scriptsApi.toggle(item.id, item.enabled)
    ui.notify(item.enabled ? '已启用' : '已停用')
  } catch (err) {
    item.enabled = !item.enabled
    ui.notify(err.message, 'error')
  }
}

async function uninstall(item) {
  try {
    await scriptsApi.uninstall(item.id)
    ui.notify('已卸载')
    await load()
  } catch (err) {
    ui.notify(err.message, 'error')
  }
}

function statusColor(status) {
  return status === 'installed' ? 'success' : 'warning'
}
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="text-h5">脚本</div>
        <div class="text-medium-emphasis">安装与管理平台脚本（仅管理员）</div>
      </v-col>
      <v-col cols="auto">
        <div class="d-flex align-center ga-2">
          <v-btn variant="text" prepend-icon="mdi-refresh" :loading="loading" @click="load">
            刷新
          </v-btn>
          <v-btn color="primary" prepend-icon="mdi-github" @click="dialog = true">
            从 GitHub 安装
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-card>
      <DataTable
        :headers="headers"
        :items="scripts"
        :loading="loading"
        :paginated="false"
        item-value="id"
      >
        <template #item.status="{ item }">
          <v-chip size="small" :color="statusColor(item.status)" variant="tonal">
            {{ item.status === 'installed' ? '可用' : '缺少运行时' }}
          </v-chip>
        </template>
        <template #item.enabled="{ item }">
          <v-switch
            v-model="item.enabled"
            color="primary"
            hide-details
            density="compact"
            @update:model-value="toggle(item)"
          />
        </template>
        <template #item.actions="{ item }">
          <v-btn size="small" variant="text" color="error" @click="uninstall(item)">卸载</v-btn>
        </template>
        <template #no-data>
          <div class="pa-6 text-medium-emphasis">暂无脚本，点击右上角「从 GitHub 安装」</div>
        </template>
      </DataTable>
    </v-card>

    <v-dialog v-model="dialog" max-width="640" persistent>
      <v-card>
        <v-card-item>
          <v-card-title class="text-h6">安装脚本</v-card-title>
          <v-card-subtitle>填写符合规范的脚本仓库地址（zip 由后端下载）</v-card-subtitle>
        </v-card-item>
        <v-card-text>
          <v-form @submit.prevent="install">
            <div class="d-flex flex-wrap ga-2 mb-4">
              <v-btn
                v-for="preset in PRESETS"
                :key="preset.value"
                size="small"
                variant="tonal"
                @click="form.url = preset.value"
              >
                {{ preset.title }}
              </v-btn>
            </div>
            <v-text-field
              v-model="form.url"
              label="GitHub 仓库或 zip 地址"
              placeholder="https://github.com/owner/repo"
            />
            <v-select
              v-model="form.accel"
              :items="ACCEL"
              item-title="title"
              item-value="value"
              label="加速源"
            />
            <div class="d-flex justify-end ga-2 mt-4">
              <v-btn variant="text" @click="dialog = false">取消</v-btn>
              <v-btn type="submit" color="primary" :loading="installing">安装</v-btn>
            </div>
          </v-form>
        </v-card-text>
      </v-card>
    </v-dialog>
  </v-container>
</template>
