<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { configApi } from '@/api/services/config'
import { useUiStore } from '@/stores/ui'

const props = defineProps({
  active: Boolean,
})

const ui = useUiStore()
const loading = ref(false)
const saving = ref(false)
const snapshot = ref(null)
const values = reactive({})
const originals = ref({})

const groups = computed(() => {
  const map = new Map()
  for (const field of snapshot.value?.fields || []) {
    if (!map.has(field.group)) map.set(field.group, [])
    map.get(field.group).push(field)
  }
  return [...map.entries()].map(([title, fields]) => ({ title, fields }))
})

function apply(data) {
  snapshot.value = data
  for (const key of Object.keys(values)) delete values[key]
  const originalsNext = {}
  for (const field of data?.fields || []) {
    values[field.key] = field.value
    originalsNext[field.key] = field.value
  }
  originals.value = originalsNext
}

function changedValues() {
  const patch = {}
  for (const [key, value] of Object.entries(values)) {
    if (JSON.stringify(value) !== JSON.stringify(originals.value[key])) {
      patch[key] = value
    }
  }
  return patch
}

async function load() {
  loading.value = true
  try {
    apply(await configApi.get())
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

async function save() {
  const patch = changedValues()
  if (!Object.keys(patch).length) {
    ui.notify('没有需要保存的修改', 'info')
    return
  }
  saving.value = true
  try {
    apply(await configApi.update(patch))
    ui.notify('配置已保存并生效')
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    saving.value = false
  }
}

async function reload() {
  loading.value = true
  try {
    apply(await configApi.reload())
    ui.notify('已重新读取配置文件')
  } catch (err) {
    ui.notify(err.message, 'error')
  } finally {
    loading.value = false
  }
}

function restoreDefaults() {
  for (const field of snapshot.value?.fields || []) {
    values[field.key] = field.default
  }
  ui.notify('已填入默认值，保存后生效', 'info')
}

watch(
  () => props.active,
  (active) => {
    if (active && !snapshot.value) load()
  },
  { immediate: true },
)
</script>

<template>
  <div>
    <v-row align="center" class="mb-4">
      <v-col>
        <div class="text-title-large">站点配置</div>
        <div class="text-medium-emphasis">
          只包含可安全热更新的注册、会话和配额项，保存后立即应用
        </div>
      </v-col>
      <v-col cols="auto" class="d-flex ga-2">
        <v-btn variant="text" prepend-icon="mdi-restore" :loading="loading" @click="reload">
          重新读取文件
        </v-btn>
        <v-btn variant="text" prepend-icon="mdi-refresh" :loading="loading" @click="load">
          刷新
        </v-btn>
        <v-btn color="primary" prepend-icon="mdi-content-save-outline" :loading="saving" @click="save">
          保存并生效
        </v-btn>
      </v-col>
    </v-row>

    <v-alert v-if="snapshot" type="info" variant="tonal" class="mb-4">
      <div class="text-body-small">主配置：{{ snapshot.file }}</div>
      <div class="text-body-small">可视化覆盖：{{ snapshot.override_file }}</div>
      <div class="text-body-small">修改会写入覆盖文件，不会覆盖主配置里的注释；密码与数据库配置不在这里编辑。</div>
    </v-alert>

    <v-alert
      v-for="warning in snapshot?.warnings || []"
      :key="warning"
      type="warning"
      variant="tonal"
      class="mb-4"
    >
      {{ warning }}
    </v-alert>

    <v-card v-for="group in groups" :key="group.title" class="mb-6">
      <v-card-item>
        <v-card-title class="text-title-large">{{ group.title }}</v-card-title>
      </v-card-item>
      <v-card-text>
        <v-row>
          <v-col v-for="field in group.fields" :key="field.key" cols="12" md="6">
            <v-select
              v-if="field.type === 'select'"
              v-model="values[field.key]"
              :items="field.options"
              item-title="label"
              item-value="value"
              :label="field.label"
              :hint="[field.help, field.effect].filter(Boolean).join('；')"
              persistent-hint
            />
            <v-switch
              v-else-if="field.type === 'bool'"
              v-model="values[field.key]"
              :label="field.label"
              :hint="[field.help, field.effect].filter(Boolean).join('；')"
              persistent-hint
              color="primary"
            />
            <v-text-field
              v-else-if="field.type === 'int'"
              v-model.number="values[field.key]"
              type="number"
              :label="field.label"
              :min="field.min"
              :max="field.max"
              :hint="[field.help, field.effect].filter(Boolean).join('；')"
              persistent-hint
            />
            <v-text-field
              v-else
              v-model="values[field.key]"
              :label="field.label"
              :hint="[field.help, field.effect].filter(Boolean).join('；')"
              persistent-hint
              autocomplete="off"
            />
          </v-col>
        </v-row>
        <div class="d-flex justify-end">
          <v-btn variant="text" prepend-icon="mdi-restore" @click="restoreDefaults">恢复默认值</v-btn>
        </div>
      </v-card-text>
    </v-card>

    <v-card class="mb-6">
      <v-card-item>
        <v-card-title class="text-title-large">需要重启的配置</v-card-title>
        <v-card-subtitle>这些项涉及监听地址、数据库、Agent 或密钥，请修改主配置文件后重启服务</v-card-subtitle>
      </v-card-item>
      <v-card-text>
        <v-list lines="two">
          <v-list-item
            v-for="item in snapshot?.restart_items || []"
            :key="item.key"
            :title="item.label"
            :subtitle="item.note || item.key"
          />
        </v-list>
      </v-card-text>
    </v-card>
  </div>
</template>
