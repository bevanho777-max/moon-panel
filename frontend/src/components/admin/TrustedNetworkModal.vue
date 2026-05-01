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
import { addTrustedIP } from '@/api/security'
import { ApiError } from '@/api/client'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'added'): void
}>()

const message = useMessage()

const cidr = ref('')
const note = ref('')
const submitting = ref(false)
const inlineError = ref('')

// Lightweight client-side hint. Server is the source of truth — backend
// rejects /0 and parse errors. This just gives instant feedback for typos.
const looksValid = computed(() => {
  const m = /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$|^[0-9a-fA-F:]+\/\d{1,3}$/.exec(cidr.value.trim())
  return !!m && !cidr.value.trim().startsWith('0.0.0.0/0')
})

function reset() {
  cidr.value = ''
  note.value = ''
  inlineError.value = ''
}

async function submit() {
  if (!cidr.value.trim()) {
    inlineError.value = 'CIDR 不能为空'
    return
  }
  submitting.value = true
  inlineError.value = ''
  try {
    await addTrustedIP(cidr.value.trim(), note.value.trim() || undefined)
    message.success(`已添加 ${cidr.value.trim()}`)
    emit('added')
    emit('update:show', false)
    reset()
  } catch (e) {
    if (e instanceof ApiError) {
      inlineError.value = e.message || '添加失败'
    } else {
      message.error('未知错误')
    }
  } finally {
    submitting.value = false
  }
}

watch(
  () => props.show,
  (visible) => {
    if (!visible) reset()
  },
)
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="添加受信 CIDR"
    style="max-width: 460px"
    @update:show="(v: boolean) => { if (!v) emit('update:show', false) }"
  >
    <NAlert type="info" :show-icon="false" style="margin-bottom: 16px; font-size: 0.85rem">
      <div>常见示例：</div>
      <code>192.168.0.0/16</code> · 家庭局域网<br />
      <code>10.0.0.0/8</code> · 公司内网<br />
      <code>203.0.113.45/32</code> · 单一公网 IP
    </NAlert>

    <div style="display: flex; flex-direction: column; gap: 12px">
      <div>
        <div class="tn__label">CIDR</div>
        <NInput
          v-model:value="cidr"
          placeholder="例如 192.168.1.0/24"
          :disabled="submitting"
          style="font-family: monospace"
          @keyup.enter="submit"
        />
        <div v-if="cidr.trim() && !looksValid" class="tn__warn">
          格式看起来不像合法 CIDR — 后端校验为准，但建议复查
        </div>
      </div>
      <div>
        <div class="tn__label">备注（可选）</div>
        <NInput
          v-model:value="note"
          placeholder="例如：家里 WAN"
          :disabled="submitting"
          maxlength="64"
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
          type="primary"
          :loading="submitting"
          :disabled="!cidr.trim()"
          @click="submit"
        >
          添加
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.tn__label {
  font-size: 0.85rem;
  margin-bottom: 4px;
}
.tn__warn {
  margin-top: 4px;
  font-size: 0.75rem;
  color: #e0b85a;
}
</style>
