<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { legalDoc } from '@/content/legal'
import LegalDocument from '@/components/LegalDocument.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const docId = computed(() => (route.meta.doc === 'privacy' ? 'privacy' : 'terms'))
const doc = computed(() => legalDoc(docId.value))
const tabs = [
  { title: '用户协议', value: 'terms' },
  { title: '隐私政策', value: 'privacy' },
]

function switchDoc(value) {
  if (value && value !== docId.value) {
    router.push({ name: value === 'privacy' ? 'privacy' : 'terms' })
  }
}
</script>

<template>
  <v-container class="py-8">
    <v-row align="center" class="mb-4">
      <v-col>
        <div class="text-headline-small">OmniTask 万务通</div>
        <div class="text-medium-emphasis">用户协议与隐私政策</div>
      </v-col>
      <v-col cols="auto" class="d-flex ga-2">
        <v-btn variant="text" to="/home">返回首页</v-btn>
        <v-btn v-if="!auth.isAuthenticated" color="primary" to="/login">登录</v-btn>
        <v-btn v-else color="primary" to="/home">回到应用</v-btn>
      </v-col>
    </v-row>

    <v-tabs :model-value="docId" color="primary" class="mb-4" @update:model-value="switchDoc">
      <v-tab v-for="tab in tabs" :key="tab.value" :value="tab.value">{{ tab.title }}</v-tab>
    </v-tabs>

    <v-card>
      <v-card-text class="pa-6 pa-md-8">
        <LegalDocument :doc="doc" />
      </v-card-text>
    </v-card>

    <div class="text-body-small text-medium-emphasis mt-4">
      当前版本 {{ doc.updated }}。协议内容同时可在官方门户 omnitasks.top 查看。
    </div>
  </v-container>
</template>
