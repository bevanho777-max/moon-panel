<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NInput,
  NModal,
  NSpace,
  NSpin,
  useMessage,
} from 'naive-ui'
import QRCode from 'qrcode'
import { enrollTOTP, confirmTOTP, type EnrollResponse } from '@/api/totp'
import { ApiError } from '@/api/client'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'enabled'): void
}>()

const message = useMessage()

type Stage = 'loading' | 'scan' | 'backup-codes' | 'done'
const stage = ref<Stage>('loading')
const enrollment = ref<EnrollResponse | null>(null)
const qrDataUrl = ref('')
const code = ref('')
const submitting = ref(false)
const inlineError = ref('')

async function startEnrollment() {
  stage.value = 'loading'
  enrollment.value = null
  qrDataUrl.value = ''
  code.value = ''
  inlineError.value = ''
  try {
    const r = await enrollTOTP()
    enrollment.value = r
    qrDataUrl.value = await QRCode.toDataURL(r.otpauth_url, { width: 200, margin: 1 })
    stage.value = 'scan'
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      message.error('已经启用过 2FA — 请先在站点设置禁用再重新注册')
      emit('update:show', false)
    } else {
      message.error(e instanceof ApiError ? e.message : '启动注册失败')
      emit('update:show', false)
    }
  }
}

async function submitConfirm() {
  if (code.value.length !== 6) {
    inlineError.value = '请输入 6 位验证码'
    return
  }
  submitting.value = true
  inlineError.value = ''
  try {
    await confirmTOTP(code.value)
    stage.value = 'backup-codes'
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      inlineError.value = '验证码不正确，请确认手机时间准确后重新输入'
      code.value = ''
    } else {
      message.error(e instanceof ApiError ? e.message : '确认失败')
    }
  } finally {
    submitting.value = false
  }
}

function copySecret() {
  if (!enrollment.value) return
  navigator.clipboard.writeText(enrollment.value.secret).then(
    () => message.success('已复制密钥'),
    () => message.error('复制失败 — 请手动选中复制'),
  )
}

function copyAllBackupCodes() {
  if (!enrollment.value) return
  navigator.clipboard.writeText(enrollment.value.backup_codes.join('\n')).then(
    () => message.success('备份码已复制'),
    () => message.error('复制失败 — 请手动选中复制'),
  )
}

function downloadBackupCodes() {
  if (!enrollment.value) return
  const blob = new Blob(
    [
      `Moon Panel 2FA Backup Codes\n生成时间: ${new Date().toISOString()}\n\n` +
        enrollment.value.backup_codes.join('\n') +
        '\n\n每个码只能使用一次。妥善保存。\n',
    ],
    { type: 'text/plain' },
  )
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'moon-panel-backup-codes.txt'
  a.click()
  URL.revokeObjectURL(url)
}

function finish() {
  stage.value = 'done'
  emit('enabled')
  emit('update:show', false)
}

const title = computed(() => {
  if (stage.value === 'backup-codes') return '保存备份码（关键步骤）'
  return '启用两步验证 (2FA)'
})

watch(
  () => props.show,
  (visible) => {
    if (visible) startEnrollment()
  },
)
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="title"
    style="max-width: 520px"
    :mask-closable="false"
    :close-on-esc="false"
    @update:show="(v: boolean) => { if (!v) emit('update:show', false) }"
  >
    <!-- Stage 1: loading -->
    <div v-if="stage === 'loading'" style="text-align: center; padding: 40px 0">
      <NSpin />
      <div style="margin-top: 12px; opacity: 0.6">正在生成密钥...</div>
    </div>

    <!-- Stage 2: scan QR + confirm code -->
    <div v-else-if="stage === 'scan' && enrollment">
      <NAlert type="info" :show-icon="false" style="margin-bottom: 12px">
        用 Authenticator 应用（Google Authenticator / 1Password / Bitwarden 等）扫描二维码，
        然后输入应用显示的 6 位验证码。
      </NAlert>

      <div class="t2f__qr-row">
        <div class="t2f__qr-wrap">
          <img :src="qrDataUrl" alt="2FA QR code" class="t2f__qr" />
        </div>
        <div class="t2f__manual">
          <div class="t2f__manual-label">无法扫码？手动输入：</div>
          <div class="t2f__secret-row">
            <code class="t2f__secret">{{ enrollment.secret }}</code>
            <NButton size="tiny" tertiary @click="copySecret">复制</NButton>
          </div>
          <div class="t2f__manual-tip">
            在 app 中选"手动输入密钥"，账户填 admin@MoonPanel，类型选 TOTP / 6 位 / 30 秒。
          </div>
        </div>
      </div>

      <div class="t2f__confirm">
        <div class="t2f__confirm-label">输入应用上的 6 位验证码：</div>
        <NInput
          v-model:value="code"
          placeholder="000000"
          :maxlength="6"
          :disabled="submitting"
          style="font-family: monospace; font-size: 1.1rem; letter-spacing: 0.3em"
          @keyup.enter="submitConfirm"
        />
        <NAlert
          v-if="inlineError"
          type="error"
          :show-icon="false"
          style="margin-top: 8px"
        >
          {{ inlineError }}
        </NAlert>
      </div>
    </div>

    <!-- Stage 3: show backup codes (one-time) -->
    <div v-else-if="stage === 'backup-codes' && enrollment">
      <NAlert type="warning" :show-icon="false" style="margin-bottom: 12px">
        <b>2FA 已启用。</b>
        以下 8 个备份码每个只能使用一次，丢失手机时用来登录。
        <span style="color: #e88080">这是唯一一次显示机会，立即保存。</span>
      </NAlert>

      <div class="t2f__codes">
        <div v-for="(c, i) in enrollment.backup_codes" :key="i" class="t2f__code">
          {{ c }}
        </div>
      </div>

      <NSpace style="margin-top: 12px">
        <NButton size="small" @click="copyAllBackupCodes">复制全部到剪贴板</NButton>
        <NButton size="small" @click="downloadBackupCodes">下载为 txt 文件</NButton>
      </NSpace>

      <NAlert type="info" :show-icon="false" style="margin-top: 12px; font-size: 0.8rem">
        建议保存到密码管理器（1Password / Bitwarden 的备注字段），或打印出来放保险柜。
      </NAlert>
    </div>

    <!-- Footer always at top level — slot binding can't be inside v-if -->
    <template #footer>
      <NSpace v-if="stage === 'scan'" justify="end">
        <NButton :disabled="submitting" @click="emit('update:show', false)">取消</NButton>
        <NButton
          type="primary"
          :loading="submitting"
          :disabled="code.length !== 6"
          @click="submitConfirm"
        >
          确认启用
        </NButton>
      </NSpace>
      <NSpace v-else-if="stage === 'backup-codes'" justify="end">
        <NButton type="primary" @click="finish">我已保存，完成</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.t2f__qr-row {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 16px;
}
.t2f__qr-wrap {
  background: white;
  padding: 8px;
  border-radius: 6px;
  flex-shrink: 0;
}
.t2f__qr {
  display: block;
  width: 200px;
  height: 200px;
}
.t2f__manual {
  flex: 1;
  min-width: 0;
}
.t2f__manual-label {
  font-size: 0.85rem;
  font-weight: 500;
  margin-bottom: 6px;
}
.t2f__secret-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.t2f__secret {
  flex: 1;
  font-family: monospace;
  font-size: 0.85rem;
  background: rgba(255, 255, 255, 0.06);
  padding: 6px 8px;
  border-radius: 4px;
  word-break: break-all;
  user-select: all;
}
.t2f__manual-tip {
  font-size: 0.75rem;
  opacity: 0.55;
  line-height: 1.5;
}
.t2f__confirm-label {
  font-size: 0.85rem;
  margin-bottom: 6px;
}
.t2f__codes {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}
.t2f__code {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 8px 12px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 0.95rem;
  letter-spacing: 0.05em;
  text-align: center;
  user-select: all;
}
</style>
