<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()

const form = reactive({ username: '', password: '', confirm: '' })
const loading = ref(false)
const error = ref('')

async function submit() {
  if (form.password !== form.confirm) {
    error.value = '两次密码不一致'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await auth.register({ username: form.username, password: form.password })
    ui.notify('注册成功')
    router.push('/')
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <v-container class="fill-height">
    <v-row justify="center" align="center" class="w-100">
      <v-col cols="12" sm="8" md="5" lg="4">
        <v-card>
          <v-card-item>
            <v-card-title class="text-h6">注册 OmniTask</v-card-title>
          </v-card-item>
          <v-card-text>
            <v-alert v-if="error" type="error" variant="tonal" class="mb-4">{{ error }}</v-alert>
            <v-form @submit.prevent="submit">
              <v-text-field
                v-model="form.username"
                label="用户名"
                prepend-inner-icon="mdi-account-outline"
                hint="3-64 个字符"
              />
              <v-text-field
                v-model="form.password"
                label="密码"
                type="password"
                prepend-inner-icon="mdi-lock-outline"
                hint="至少 6 位"
              />
              <v-text-field
                v-model="form.confirm"
                label="确认密码"
                type="password"
                prepend-inner-icon="mdi-lock-check-outline"
              />
              <v-btn type="submit" color="primary" block :loading="loading" class="mt-2">
                注册
              </v-btn>
            </v-form>
          </v-card-text>
          <v-card-actions class="justify-center">
            <span class="text-medium-emphasis">已有账号？</span>
            <v-btn variant="text" color="primary" to="/login">登录</v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>
