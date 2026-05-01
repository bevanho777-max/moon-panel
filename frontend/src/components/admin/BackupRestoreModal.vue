<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NInput,
  NModal,
  NSpace,
  NUpload,
  NUploadDragger,
  useMessage,
  type UploadFileInfo,
} from 'naive-ui'
import { restoreBackup, type RestoreResult } from '@/api/backup'
import { ApiError } from '@/api/client'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'restored'): void
}>()

const message = useMessage()

type Stage = 'select' | 'confirm' | 'restoring' | 'done'

const stage = ref<Stage>('select')
const file = ref<File | null>(null)
const fileName = ref('')
const confirmText = ref('')
const result = ref<RestoreResult | null>(null)

function reset() {
  stage.value = 'select'
  file.value = null
  fileName.value = ''
  confirmText.value = ''
  result.value = null
}

function close() {
  emit('update:show', false)
  reset()
}

function handleFileChange(opts: { fileList: UploadFileInfo[] }) {
  if (opts.fileList.length > 0) {
    const f = opts.fileList[0].file
    if (f) {
      file.value = f
      fileName.value = f.name
      stage.value = 'confirm'
    }
  }
}

async function performRestore() {
  if (!file.value) return
  if (confirmText.value.trim() !== 'RESTORE') {
    message.warning('请输入大写 RESTORE 确认操作')
    return
  }
  stage.value = 'restoring'
  try {
    const r = await restoreBackup(file.value)
    result.value = r
    stage.value = 'done'
    emit('restored')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '恢复失败')
    stage.value = 'confirm'
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
    :title="stage === 'done' ? '恢复完成' : '从备份恢复'"
    style="max-width: 520px"
    :mask-closable="stage !== 'restoring'"
    @update:show="(v: boolean) => { if (!v) close() }"
  >
    <!-- Stage 1: select file -->
    <div v-if="stage === 'select'">
      <NAlert type="warning" :show-icon="false" style="margin-bottom: 12px">
        <b>恢复将替换所有现有的分组、卡片、搜索引擎和站点设置。</b>
        admin 账号 / 2FA / 审计日志保留不变。建议先点上方"导出 JSON"备份当前数据。
      </NAlert>
      <NUpload
        :default-upload="false"
        accept=".json,.zip"
        :max="1"
        @change="handleFileChange"
      >
        <NUploadDragger>
          <div style="padding: 20px; text-align: center">
            <div style="font-size: 1.4rem">📦</div>
            <div style="margin-top: 8px; opacity: 0.85">点击或拖拽 .json / .zip 备份文件</div>
            <div style="margin-top: 4px; font-size: 0.78rem; opacity: 0.5">
              JSON：仅元数据 · ZIP：含 uploads/ 文件
            </div>
          </div>
        </NUploadDragger>
      </NUpload>
    </div>

    <!-- Stage 2: confirm with type-to-confirm -->
    <div v-else-if="stage === 'confirm'">
      <NAlert type="error" :show-icon="false" style="margin-bottom: 12px">
        将用 <code>{{ fileName }}</code> 的内容**完全替换**当前 panel 数据。
        现有 groups / cards / search_engines / 设置（非 auth.*）会**全部删除**后从备份重建。
        <br /><br />
        如需保留当前数据请先取消并到备份导出。
      </NAlert>
      <div style="font-size: 0.85rem; margin-bottom: 6px">
        请输入大写 <code style="font-family:monospace; background:rgba(255,255,255,0.06); padding:2px 6px; border-radius:3px">RESTORE</code> 确认：
      </div>
      <NInput
        v-model:value="confirmText"
        placeholder="RESTORE"
        :maxlength="10"
        style="font-family: monospace"
        @keyup.enter="performRestore"
      />
    </div>

    <!-- Stage 3: restoring -->
    <div v-else-if="stage === 'restoring'" style="text-align: center; padding: 30px 0">
      <div style="font-size: 1.6rem">⏳</div>
      <div style="margin-top: 10px; opacity: 0.7">正在恢复，请勿关闭...</div>
    </div>

    <!-- Stage 4: done -->
    <div v-else-if="stage === 'done' && result">
      <NAlert type="success" :show-icon="false" style="margin-bottom: 12px">
        恢复完成。建议刷新页面或重新登录以加载新数据。
      </NAlert>
      <ul class="br__stats">
        <li>分组：{{ result.groups }} 条</li>
        <li>卡片：{{ result.cards }} 条</li>
        <li>搜索引擎：{{ result.engines }} 条</li>
        <li>设置：{{ result.settings }} 项</li>
        <li v-if="result.uploads_restored > 0">上传文件：{{ result.uploads_restored }} 个</li>
      </ul>
    </div>

    <template #footer>
      <NSpace v-if="stage === 'select'" justify="end">
        <NButton @click="close">取消</NButton>
      </NSpace>
      <NSpace v-else-if="stage === 'confirm'" justify="end">
        <NButton @click="reset">重选文件</NButton>
        <NButton
          type="error"
          :disabled="confirmText.trim() !== 'RESTORE'"
          @click="performRestore"
        >
          确认恢复
        </NButton>
      </NSpace>
      <NSpace v-else-if="stage === 'done'" justify="end">
        <NButton type="primary" @click="close">关闭</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.br__stats {
  margin: 0;
  padding-left: 18px;
  font-size: 0.88rem;
  line-height: 1.7;
}
.br__stats li {
  font-variant-numeric: tabular-nums;
}
</style>
