<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSpace,
  useMessage,
} from 'naive-ui'
import { useRouter } from 'vue-router'
import { changePassword } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/api/client'

defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', v: boolean): void }>()

const message = useMessage()
const router = useRouter()
const auth = useAuthStore()

const oldPwd = ref('')
const newPwd = ref('')
const confirmPwd = ref('')
const submitting = ref(false)
const inlineError = ref('')

interface Strength {
  level: 0 | 1 | 2 | 3 // 0=空, 1=弱, 2=中, 3=强
  label: string
  color: string
}

// Strength heuristic — intentionally simple so users can predict the score.
//   Length is the dominant factor (modern guidance: NIST SP 800-63B).
//   Character-class diversity adds a tier when length is borderline.
//   We deliberately don't pull in zxcvbn (~400KB gzip) for one widget.
function rateStrength(p: string): Strength {
  if (!p) return { level: 0, label: '', color: 'transparent' }
  const len = p.length
  const hasLetter = /[a-zA-Z]/.test(p)
  const hasDigit = /\d/.test(p)
  const hasSymbol = /[^a-zA-Z0-9]/.test(p)
  const classes = Number(hasLetter) + Number(hasDigit) + Number(hasSymbol)

  if (len < 8) return { level: 1, label: '太短', color: '#e88080' }
  if (len < 10 || classes < 2) return { level: 1, label: '弱', color: '#e88080' }
  if (len >= 14 || (len >= 10 && classes >= 3)) return { level: 3, label: '强', color: '#5dba6f' }
  return { level: 2, label: '中', color: '#e0b85a' }
}

const strength = computed(() => rateStrength(newPwd.value))

const canSubmit = computed(() => {
  if (submitting.value) return false
  if (!oldPwd.value || !newPwd.value || !confirmPwd.value) return false
  if (newPwd.value.length < 8) return false
  if (newPwd.value !== confirmPwd.value) return false
  if (newPwd.value === oldPwd.value) return false
  return true
})

const validationHint = computed(() => {
  if (!newPwd.value || !confirmPwd.value) return ''
  if (newPwd.value.length < 8) return '新密码至少 8 字符'
  if (newPwd.value !== confirmPwd.value) return '两次输入的新密码不一致'
  if (newPwd.value === oldPwd.value) return '新密码不能与原密码相同'
  return ''
})

watch(() => oldPwd.value, () => { inlineError.value = '' })

function reset() {
  oldPwd.value = ''
  newPwd.value = ''
  confirmPwd.value = ''
  inlineError.value = ''
  submitting.value = false
}

function close() {
  emit('update:show', false)
  reset()
}

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  inlineError.value = ''
  try {
    await changePassword(oldPwd.value, newPwd.value)
    message.success('密码修改成功，请用新密码重新登录')
    // Backend cleared the cookie. Sync local auth state and navigate.
    await auth.logout().catch(() => { /* cookie already cleared, swallow */ })
    emit('update:show', false)
    reset()
    router.replace({ name: 'login' })
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      inlineError.value = '原密码不正确'
    } else {
      message.error(e instanceof ApiError ? e.message : '修改失败')
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="修改密码"
    style="max-width: 460px"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(v: boolean) => { if (!v) close() }"
  >
    <NForm size="medium" :show-feedback="false" label-placement="top">
      <NFormItem label="原密码">
        <NInput
          v-model:value="oldPwd"
          type="password"
          show-password-on="click"
          placeholder="当前登录使用的密码"
          :disabled="submitting"
          @keyup.enter="submit"
        />
      </NFormItem>
      <NAlert
        v-if="inlineError"
        type="error"
        :show-icon="false"
        style="margin: -8px 0 12px"
      >
        {{ inlineError }}
      </NAlert>

      <NFormItem label="新密码（最少 8 字符）" style="margin-top: 12px">
        <NInput
          v-model:value="newPwd"
          type="password"
          show-password-on="click"
          placeholder="使用强密码 — 此 panel 已公网部署"
          :disabled="submitting"
        />
      </NFormItem>
      <div v-if="newPwd" class="cpm__strength">
        <div class="cpm__bar" :data-level="strength.level">
          <span :style="{ background: strength.color }" />
        </div>
        <span class="cpm__strength-label" :style="{ color: strength.color }">
          强度：{{ strength.label }}
        </span>
      </div>

      <NFormItem label="确认新密码" style="margin-top: 12px">
        <NInput
          v-model:value="confirmPwd"
          type="password"
          show-password-on="click"
          placeholder="再次输入新密码"
          :disabled="submitting"
          @keyup.enter="submit"
        />
      </NFormItem>
      <div v-if="validationHint" class="cpm__hint">{{ validationHint }}</div>
    </NForm>

    <NAlert
      type="info"
      :show-icon="false"
      style="margin-top: 16px; font-size: 0.8rem"
    >
      修改成功后会自动退出登录，请用新密码重新进入管理后台。
    </NAlert>

    <template #footer>
      <NSpace justify="end">
        <NButton :disabled="submitting" @click="close">取消</NButton>
        <NButton
          type="primary"
          :loading="submitting"
          :disabled="!canSubmit"
          @click="submit"
        >
          确认修改
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.cpm__strength {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 6px;
}
.cpm__bar {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.08);
  overflow: hidden;
  position: relative;
}
.cpm__bar > span {
  display: block;
  height: 100%;
  width: 33%;
  border-radius: 2px;
  transition: width 180ms ease, background 180ms ease;
}
.cpm__bar[data-level='2'] > span { width: 66%; }
.cpm__bar[data-level='3'] > span { width: 100%; }
.cpm__strength-label {
  font-size: 0.78rem;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.cpm__hint {
  margin-top: 6px;
  font-size: 0.78rem;
  color: #e88080;
}
</style>
