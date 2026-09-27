<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import { credentialsApi } from '@/api/services/credentials'
import { tasksApi } from '@/api/services/tasks'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingStore } from '@/stores/onboarding'
import { usePlatformsStore } from '@/stores/platforms'
import { randomQuote } from '@/config/quotes'

const router = useRouter()
const { mobile } = useDisplay()
const auth = useAuthStore()
const platforms = usePlatformsStore()
const onboarding = useOnboardingStore()
onboarding.load()

const credentialsCount = ref(0)
const tasksTotal = ref(0)
const current = ref(onboarding.step || 1)
const quote = ref(randomQuote())

watch(current, (value) => onboarding.setStep(value))

const displayName = computed(() => auth.username || '同学')

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 5) return '夜深了'
  if (hour < 9) return '早上好'
  if (hour < 12) return '上午好'
  if (hour < 14) return '中午好'
  if (hour < 18) return '下午好'
  if (hour < 23) return '晚上好'
  return '夜深了'
})

const steps = computed(() => {
  const list = []
  if (auth.isAdmin) {
    list.push({
      title: '安装脚本',
      desc: '在「脚本」页从 GitHub 安装平台脚本，例如词达人、WeLearn、学习通。',
      to: '/scripts',
      action: '去安装脚本',
      complete: platforms.list.length > 0,
    })
  }
  list.push(
    {
      title: '添加凭据',
      desc: '在「凭据」页录入平台账号密码或 Token，凭据加密存储，仅在任务下发时解密。',
      to: '/credentials',
      action: '去添加凭据',
      complete: credentialsCount.value > 0,
    },
    {
      title: '选择任务',
      desc: '进入「任务」页新建任务：选择平台与动作，刷新列表并勾选要处理的课程或作业。',
      to: '/tasks',
      action: '去选择任务',
      complete: tasksTotal.value > 0,
    },
    {
      title: '创建并查看',
      desc: '确认参数后创建任务，在任务详情页实时查看执行进度与结果。',
      to: '/tasks',
      action: '去创建任务',
      complete: tasksTotal.value > 0,
    },
  )
  return list.map((step, index) => ({ ...step, value: index + 1 }))
})

function stepSlot(value) {
  return `item.${value}`
}

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
  if (!onboarding.done) {
    const firstIncomplete = steps.value.findIndex((step) => !step.complete)
    const target = firstIncomplete >= 0 ? steps.value[firstIncomplete].value : steps.value.length
    if (target !== current.value) current.value = target
  }
})
</script>

<template>
  <v-container>
    <v-card class="mb-4">
      <v-card-text class="pa-6 pb-3">
        <div class="text-headline-small font-weight-medium">{{ greeting }}，{{ displayName }}</div>
        <div class="text-body-large mt-2">
          {{ quote.text }}<span class="text-medium-emphasis">{{ quote.source }}</span>
        </div>
      </v-card-text>
      <v-card-actions class="px-6 pb-6 pt-0 flex-wrap ga-2">
        <v-btn color="primary" prepend-icon="mdi-plus" to="/tasks">新建任务</v-btn>
        <v-btn prepend-icon="mdi-key-variant" to="/credentials">凭据</v-btn>
        <v-btn prepend-icon="mdi-database-outline" to="/questions">题库</v-btn>
        <v-btn v-if="auth.isAdmin" prepend-icon="mdi-puzzle-outline" to="/scripts">脚本</v-btn>
      </v-card-actions>
    </v-card>

    <v-stepper
      v-if="!onboarding.done"
      v-model="current"
      :items="steps"
      :item-props="(item) => ({ complete: item.complete })"
      :mobile="mobile"
      color="primary"
      bg-color="primary-light"
      rounded="xl"
      elevation="2"
      hide-actions
      class="mb-4"
    >
      <template #title="{ title }">
        <span class="text-title-small">{{ title }}</span>
      </template>
      <template v-for="step in steps" :key="step.value" #[stepSlot(step.value)]>
        <div class="text-body-medium text-medium-emphasis">{{ step.desc }}</div>
        <div class="d-flex align-center mt-3">
          <v-btn color="primary" size="small" :to="step.to">{{ step.action }}</v-btn>
          <v-btn
            icon="mdi-close"
            variant="text"
            size="small"
            class="ms-1"
            aria-label="隐藏引导"
            @click="onboarding.finish()"
          />
        </div>
      </template>
    </v-stepper>

    <v-row>
      <v-col cols="12" sm="4">
        <v-card @click="router.push('/tasks')">
          <v-card-item>
            <v-card-title class="text-title-large">{{ tasksTotal }}</v-card-title>
            <v-card-subtitle>任务</v-card-subtitle>
          </v-card-item>
        </v-card>
      </v-col>
      <v-col cols="12" sm="4">
        <v-card @click="router.push('/credentials')">
          <v-card-item>
            <v-card-title class="text-title-large">{{ credentialsCount }}</v-card-title>
            <v-card-subtitle>凭据</v-card-subtitle>
          </v-card-item>
        </v-card>
      </v-col>
      <v-col cols="12" sm="4">
        <v-card @click="router.push('/tasks')">
          <v-card-item>
            <v-card-title class="text-title-large">{{ platforms.list.length }}</v-card-title>
            <v-card-subtitle>已启用平台</v-card-subtitle>
          </v-card-item>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>
