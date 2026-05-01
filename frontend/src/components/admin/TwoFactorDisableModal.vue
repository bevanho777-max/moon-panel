<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NInput,
  NModal,
  NSpace,
  useMessage,
} from 'naive-ui'
import { disableTOTP } from '@/api/totp'
import { ApiError } from '@/api/client'

defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'disabled'): void
}>()

const message = useMessage()

const password = ref('')
const code = ref('')
const submitting = ref(false)
const inlineError = ref('')

const canSubmit = computed(() => {
  if (submitting.value) return false
  if (!password.value || !code.value) return false
  // Code is either 6-digit TOTP or backup code (8 chars + optional dash).
  return code.value.length >= 6
})

function reset() {
  password.value = ''
  code.value = ''
  inlineError.value = ''
}

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  inlineError.value = ''
  try {
    await disableTOTP(password.value, code.value)
    message.success('2FA 已禁用')
    emit('disabled')
    emit('update:show', false)
    reset()
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      inlineError.value = '密码或验证码不正确'
    } else {
      message.error(e instanceof ApiError ? e.message : '禁用失败')
    }
  } finally {
    submitting.value = false
  }
}

watch(
  () => emit,
  () => {},
)
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="禁用两步验证"
    style="max-width: 460px"
    @update:show="(v: boolean) => { if (!v) { emit('update:show', false); reset() } }"
  >
    <NAlert type="warning" :show-icon="false" style="margin-bottom: 16px">
      禁用 2FA 后账户安全将仅依赖密码。需要原密码 + 当前 6 位验证码（或备份码）双重确认。
    </NAlert>

    <div style="display: flex; flex-direction: column; gap: 12px">
      <div>
        <div class="t2d__label">密码</div>
        <NInput
          v-model:value="password"
          type="password"
          show-password-on="click"
          placeholder="当前登录密码"
          :disabled="submitting"
        />
      </div>
      <div>
        <div class="t2d__label">验证码或备份码</div>
        <NInput
          v-model:value="code"
          placeholder="6 位 TOTP 或 ABCD-1234 备份码"
          :disabled="submitting"
          @keyup.enter="submit"
        />
      </div>
      <NAlert v-if="inlineError" type="error" :show-icon="false">
        {{ inlineError }}
      </NAlert>
    </div>

    <template #footer>
      <NSpace justify="end">
        <NButton :disabled="submitting" @click="emit('update:show', false)">取消</NButton>
        <NButton
          type="error"
          :loading="submitting"
          :disabled="!canSubmit"
          @click="submit"
        >
          确认禁用
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.t2d__label {
  font-size: 0.85rem;
  margin-bottom: 4px;
}
</style>
