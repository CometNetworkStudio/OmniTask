<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useMetaStore } from '@/stores/meta'
import { useUiStore } from '@/stores/ui'
import { TERMS_VERSION } from '@/content/legal'

const auth = useAuthStore()
const meta = useMetaStore()
const ui = useUiStore()
const router = useRouter()

const form = reactive({
  username: '',
  password: '',
  confirm: '',
  invite_code: '',
  accepted: false,
})
const loading = ref(false)
const error = ref('')
const closed = computed(() => meta.registration === 'closed')

async function submit() {
  if (closed.value) return
  if (form.password !== form.confirm) {
    error.value = '两次密码不一致'
    return
  }
  if (meta.inviteRequired && !form.invite_code.trim()) {
    error.value = '请输入邀请码'
    return
  }
  if (!form.accepted) {
    error.value = '请先阅读并同意用户协议和隐私政策'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await auth.register({
      username: form.username,
      password: form.password,
      invite_code: form.invite_code.trim(),
      accept_terms: true,
      terms_version: meta.termsVersion || TERMS_VERSION,
    })
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
    <v-row class="fill-height" justify="center" align="center">
      <v-col cols="12" sm="8" md="5" lg="4">
        <v-card>
          <v-card-item>
            <v-card-title class="text-title-large">注册 OmniTask</v-card-title>
          </v-card-item>
          <v-card-text>
            <v-alert v-if="closed" type="info" variant="tonal">
              当前实例已关闭注册。如需账号，请联系管理员。
            </v-alert>
            <template v-else>
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
                  hint="至少 8 位"
                />
                <v-text-field
                  v-model="form.confirm"
                  label="确认密码"
                  type="password"
                  prepend-inner-icon="mdi-lock-check-outline"
                />
                <v-text-field
                  v-if="meta.inviteRequired"
                  v-model="form.invite_code"
                  label="邀请码"
                  prepend-inner-icon="mdi-ticket-confirmation-outline"
                  hint="当前实例需要邀请码"
                />
                <v-checkbox v-model="form.accepted" color="primary" hide-details class="mt-2">
                  <template #label>
                    <span class="text-body-small">
                      我已阅读并同意
                      <router-link class="text-primary" to="/terms" target="_blank" @click.stop>
                        《用户协议》
                      </router-link>
                      和
                      <router-link class="text-primary" to="/privacy" target="_blank" @click.stop>
                        《隐私政策》
                      </router-link>
                    </span>
                  </template>
                </v-checkbox>
                <v-btn type="submit" color="primary" block :loading="loading" class="mt-2">
                  注册
                </v-btn>
              </v-form>
            </template>
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
