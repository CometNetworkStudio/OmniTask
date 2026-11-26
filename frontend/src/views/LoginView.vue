<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useMetaStore } from '@/stores/meta'
import { useUiStore } from '@/stores/ui'

const auth = useAuthStore()
const meta = useMetaStore()
const ui = useUiStore()
const router = useRouter()
const route = useRoute()

const form = reactive({ username: '', password: '' })
const loading = ref(false)
const error = ref('')

async function submit() {
  loading.value = true
  error.value = ''
  try {
    await auth.login({ ...form })
    ui.notify('登录成功')
    router.push(route.query.redirect || '/')
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <v-container class="fill-height">
    <v-row class="fill-height" justify="center" align="center">
      <v-col cols="12" sm="8" md="5" lg="4">
        <v-card>
          <v-card-item>
            <v-card-title class="text-title-large">登录 OmniTask</v-card-title>
            <v-card-subtitle>一处托管，万务自达</v-card-subtitle>
          </v-card-item>
          <v-card-text>
            <v-alert v-if="error" type="error" variant="tonal" class="mb-4">{{ error }}</v-alert>
            <v-form @submit.prevent="submit">
              <v-text-field
                v-model="form.username"
                label="用户名"
                prepend-inner-icon="mdi-account-outline"
                autocomplete="username"
              />
              <v-text-field
                v-model="form.password"
                label="密码"
                type="password"
                prepend-inner-icon="mdi-lock-outline"
                autocomplete="current-password"
              />
              <v-btn type="submit" color="primary" block :loading="loading" class="mt-2">
                登录
              </v-btn>
            </v-form>
          </v-card-text>
          <v-card-actions v-if="meta.registration !== 'closed'" class="justify-center">
            <span class="text-medium-emphasis">还没有账号？</span>
            <v-btn variant="text" color="primary" to="/register">
              {{ meta.registration === 'invite' ? '使用邀请码注册' : '注册' }}
            </v-btn>
          </v-card-actions>
          <v-card-text class="pt-0 text-center text-body-small text-medium-emphasis">
            登录即表示你已阅读并同意
            <router-link class="text-primary" to="/terms">《用户协议》</router-link>
            和
            <router-link class="text-primary" to="/privacy">《隐私政策》</router-link>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>
