<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NAlert, NCheckbox, NSpace, useMessage } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/api/client'
import { verifyTOTP } from '@/api/totp'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const message = useMessage()

type Stage = 'password' | 'totp'

const stage = ref<Stage>('password')

const password = ref('')
const passwordConfirm = ref('')
const rememberMe = ref(false)
const submitting = ref(false)

// TOTP step state
const totpCode = ref('')
const useBackupCode = ref(false)
const totpError = ref('')

const isInit = computed(() => auth.initialized === false)
const title = computed(() => {
  if (stage.value === 'totp') return '两步验证'
  return isInit.value ? '首次启动 · 设置管理员密码' : '管理员登录'
})

const reasonAlert = computed(() => {
  const reason = route.query.reason as string | undefined
  if (reason === 'expired') return { type: 'warning' as const, msg: '会话已过期，请重新登录。' }
  if (reason === 'password_changed') return { type: 'info' as const, msg: '密码已修改，请使用新密码登录。' }
  return null
})

onMounted(async () => {
  if (auth.initialized === null) {
    await auth.refresh()
  }
})

function redirectAfterAuth() {
  const redirect = (route.query.redirect as string | undefined) ?? '/admin'
  router.replace(redirect)
}

async function submit() {
  if (!password.value) {
    message.warning('请输入密码')
    return
  }
  if (isInit.value && password.value !== passwordConfirm.value) {
    message.warning('两次输入的密码不一致')
    return
  }
  submitting.value = true
  try {
    if (isInit.value) {
      await auth.initAdmin(password.value)
      message.success('管理员已创建')
      redirectAfterAuth()
      return
    }
    const result = await auth.login(password.value, rememberMe.value)
    if (result.needs2FA) {
      stage.value = 'totp'
      totpCode.value = ''
      totpError.value = ''
      return
    }
    message.success('登录成功')
    redirectAfterAuth()
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 429) {
        message.error(e.message || '登录尝试过多，已临时锁定')
      } else {
        message.error(e.message || '操作失败')
      }
    } else {
      message.error('未知错误')
    }
  } finally {
    submitting.value = false
  }
}

async function submitTOTP() {
  const trimmed = totpCode.value.trim()
  if (!trimmed) {
    totpError.value = '请输入验证码'
    return
  }
  if (!useBackupCode.value && trimmed.length !== 6) {
    totpError.value = '请输入 6 位 TOTP 验证码'
    return
  }
  submitting.value = true
  totpError.value = ''
  try {
    await verifyTOTP(trimmed, { isBackup: useBackupCode.value, rememberMe: rememberMe.value })
    await auth.completeTOTP()
    message.success('登录成功')
    redirectAfterAuth()
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 429) {
        totpError.value = e.message || '尝试过多，临时锁定'
      } else if (e.status === 401) {
        totpError.value = '验证码不正确'
      } else if (e.status === 400) {
        // Challenge expired — restart from password step
        totpError.value = e.message || '会话已过期，请重新输入密码'
        stage.value = 'password'
        password.value = ''
      } else {
        totpError.value = e.message || '验证失败'
      }
    } else {
      totpError.value = '未知错误'
    }
  } finally {
    submitting.value = false
  }
}

function backToPassword() {
  stage.value = 'password'
  totpCode.value = ''
  totpError.value = ''
  useBackupCode.value = false
}
</script>

<template>
  <div class="login">
    <NCard :title="title" class="login__card mp-acrylic-strong">
      <!-- Stage 1: password (or first-time init) -->
      <template v-if="stage === 'password'">
        <NAlert
          v-if="reasonAlert"
          :type="reasonAlert.type"
          :show-icon="false"
          style="margin-bottom: 1rem"
        >
          {{ reasonAlert.msg }}
        </NAlert>
        <NAlert v-if="isInit" type="info" :show-icon="false" style="margin-bottom: 1rem">
          系统未初始化。设置一个密码作为管理员，之后用这个密码进入管理后台。
        </NAlert>
        <NForm @submit.prevent="submit">
          <NFormItem label="密码">
            <NInput
              v-model:value="password"
              type="password"
              show-password-on="click"
              placeholder="至少 8 位（公网部署建议 14+）"
              :disabled="submitting"
            />
          </NFormItem>
          <NFormItem v-if="isInit" label="确认密码">
            <NInput
              v-model:value="passwordConfirm"
              type="password"
              show-password-on="click"
              :disabled="submitting"
            />
          </NFormItem>
          <NCheckbox
            v-if="!isInit"
            v-model:checked="rememberMe"
            :disabled="submitting"
            style="margin-bottom: 1rem"
          >
            记住我（30 天免登录；不勾选默认 7 天）
          </NCheckbox>
          <NButton type="primary" block :loading="submitting" @click="submit">
            {{ isInit ? '创建并登录' : '登录' }}
          </NButton>
        </NForm>
      </template>

      <!-- Stage 2: TOTP / backup code -->
      <template v-else>
        <NAlert type="info" :show-icon="false" style="margin-bottom: 1rem">
          密码验证通过。请输入 Authenticator 应用上当前显示的 6 位验证码完成登录。
        </NAlert>
        <NForm @submit.prevent="submitTOTP">
          <NFormItem :label="useBackupCode ? '备份码' : '验证码（6 位）'">
            <NInput
              v-model:value="totpCode"
              :placeholder="useBackupCode ? 'ABCD-1234' : '000000'"
              :disabled="submitting"
              :maxlength="useBackupCode ? 9 : 6"
              style="font-family: monospace; font-size: 1.1rem; letter-spacing: 0.2em"
              data-testid="totp-input"
              @keyup.enter="submitTOTP"
            />
          </NFormItem>
          <NAlert v-if="totpError" type="error" :show-icon="false" style="margin-bottom: 1rem">
            {{ totpError }}
          </NAlert>
          <NSpace vertical :size="8">
            <NButton type="primary" block :loading="submitting" @click="submitTOTP">
              确认
            </NButton>
            <NSpace justify="space-between" style="font-size: 0.82rem">
              <NButton text @click="useBackupCode = !useBackupCode">
                {{ useBackupCode ? '改用 6 位 TOTP 验证码' : '改用备份码' }}
              </NButton>
              <NButton text @click="backToPassword">返回密码步骤</NButton>
            </NSpace>
          </NSpace>
        </NForm>
      </template>
    </NCard>
  </div>
</template>

<style scoped>
.login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}
.login__card {
  width: 100%;
  max-width: 400px;
}
</style>
