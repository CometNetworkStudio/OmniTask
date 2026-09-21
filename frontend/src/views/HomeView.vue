<script setup>
import { computed, onMounted, ref } from 'vue'
import { credentialsApi } from '@/api/services/credentials'
import { tasksApi } from '@/api/services/tasks'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import { usePlatformsStore } from '@/stores/platforms'
import { useUiStore } from '@/stores/ui'
import { randomQuote } from '@/config/quotes'

const auth = useAuthStore()
const platforms = usePlatformsStore()
const onboarding = useOnboardingStore()
const ui = useUiStore()

const credentialsCount = ref(0)
const tasksTotal = ref(0)
const current = ref(1)
const quote = ref(randomQuote())

const steps = computed(() => {
  if (!auth.isAuthenticated) return []
  const list = []
  if (auth.isAdmin) {
    list.push({
      key: 'scripts',
      title: '安装脚本',
      desc: '在「脚本」页从 GitHub 安装平台脚本（如词达人、WeLearn、学习通）。',
      to: '/scripts',
      action: '去安装脚本',
      complete: platforms.list.length > 0,
    })
  }
  list.push(
    {
      key: 'credentials',
      title: '添加凭据',
      desc: '在「凭据」页录入平台账号密码或 Token（加密存储，仅任务下发时解密）。',
      to: '/credentials',
      action: '去添加凭据',
      complete: credentialsCount.value > 0,
    },
    {
      key: 'tasks',
      title: '选择任务',
      desc: '在「任务」页新建任务：选择平台与动作，刷新并选择要处理的任务/课程。',
      to: '/tasks',
      action: '去选择任务',
      complete: tasksTotal.value > 0,
    },
    {
      key: 'create',
      title: '创建并查看',
      desc: '确认执行参数后创建任务，即可在任务详情页实时查看进度。',
      to: '/tasks',
      action: '去创建任务',
      complete: tasksTotal.value > 0,
    },
  )
  return list
})

const stepperItems = computed(() =>
  steps.value.map((step, index) => ({ title: step.title, value: index + 1, complete: step.complete })),
)

onMounted(async () => {
  try {
    const [creds, taskPage] = await Promise.all([
      credentialsApi.list(),
      tasksApi.list({ page: 1, size: 1 }),
    ])
    credentialsCount.value = (creds || []).length
    tasksTotal.value = taskPage?.total || 0
  } catch {
    // 未登录或接口失败时忽略
  }
  platforms.load().catch(() => {})
  const firstIncomplete = steps.value.findIndex((step) => !step.complete)
  current.value = firstIncomplete >= 0 ? firstIncomplete + 1 : steps.value.length
})

function stepSlot(index) {
  return `item.${index + 1}`
}

function finish() {
  onboarding.finish()
  ui.notify('引导已完成，开始使用吧')
}
</script>

<template>
  <v-container>
    <template v-if="!onboarding.done">
      <v-row class="mb-2">
        <v-col>
          <div class="text-h5">欢迎使用 OmniTask</div>
          <div class="text-medium-emphasis">工欲善其事，必先利其器 —— 跟着引导，两三步就能开始</div>
        </v-col>
      </v-row>

      <v-card class="mb-4">
        <v-stepper v-model="current" alt-labels hide-actions :items="stepperItems">
          <template v-for="(step, index) in steps" :key="step.key" #[stepSlot(index)]>
            <v-card flat>
              <v-card-text>
                <div class="text-h6 mb-1">{{ step.title }}</div>
                <div class="text-medium-emphasis mb-4">{{ step.desc }}</div>
                <v-btn color="primary" :to="step.to">{{ step.action }}</v-btn>
              </v-card-text>
            </v-card>
          </template>
        </v-stepper>
        <v-card-actions class="justify-end">
          <v-btn variant="text" @click="onboarding.finish()">跳过引导</v-btn>
          <v-btn color="success" @click="finish">完成引导</v-btn>
        </v-card-actions>
      </v-card>
    </template>

    <template v-else>
      <v-card class="mb-4">
        <v-card-text>
          <div class="text-h5 mb-2">{{ quote.text }}</div>
          <div class="text-medium-emphasis text-right">—— {{ quote.source }}</div>
        </v-card-text>
      </v-card>

      <v-row>
        <v-col cols="12" sm="4">
          <v-card @click="$router.push('/tasks')">
            <v-card-item>
              <v-card-title class="text-h6">{{ tasksTotal }}</v-card-title>
              <v-card-subtitle>任务</v-card-subtitle>
            </v-card-item>
          </v-card>
        </v-col>
        <v-col cols="12" sm="4">
          <v-card @click="$router.push('/credentials')">
            <v-card-item>
              <v-card-title class="text-h6">{{ credentialsCount }}</v-card-title>
              <v-card-subtitle>凭据</v-card-subtitle>
            </v-card-item>
          </v-card>
        </v-col>
        <v-col cols="12" sm="4">
          <v-card>
            <v-card-item>
              <v-card-title class="text-h6">{{ platforms.list.length }}</v-card-title>
              <v-card-subtitle>已启用平台</v-card-subtitle>
            </v-card-item>
          </v-card>
        </v-col>
      </v-row>

      <div class="d-flex justify-end mt-2">
        <v-btn variant="text" size="small" @click="onboarding.reset()">重新观看引导</v-btn>
      </div>
    </template>
  </v-container>
</template>
