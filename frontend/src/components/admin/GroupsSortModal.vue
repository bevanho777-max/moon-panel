<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NEmpty,
  NModal,
  NSpace,
  NSpin,
  useMessage,
} from 'naive-ui'
import draggable from 'vuedraggable'
import { listGroups, reorderGroups, type Group } from '@/api/group'
import { ApiError } from '@/api/client'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'saved'): void
}>()

const message = useMessage()

const items = ref<Group[]>([])
const loading = ref(false)
const saving = ref(false)
const dirty = ref(false)

async function load() {
  loading.value = true
  dirty.value = false
  try {
    const list = await listGroups()
    list.sort((a, b) => a.sort - b.sort || a.id - b.id)
    items.value = list
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function onChange() {
  dirty.value = true
}

async function save() {
  const payload = items.value.map((g, i) => ({ id: g.id, sort: (i + 1) * 10 }))
  if (payload.length === 0) {
    emit('update:show', false)
    return
  }
  saving.value = true
  try {
    await reorderGroups(payload)
    message.success('已保存新的分组顺序')
    emit('saved')
    emit('update:show', false)
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

watch(
  () => props.show,
  (visible) => {
    if (visible) load()
  },
)
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    title="拖拽调整分组顺序"
    style="max-width: 520px"
    :mask-closable="!saving"
    @update:show="(v: boolean) => { if (!v) emit('update:show', false) }"
  >
    <NAlert type="info" :show-icon="false" style="margin-bottom: 12px; font-size: 0.82rem">
      拖动每行左侧的把手调整分组在主页的展示顺序。
    </NAlert>

    <NSpin :show="loading">
      <NEmpty v-if="!loading && items.length === 0" description="还没有分组" />
      <draggable
        v-else
        :list="items"
        :animation="160"
        item-key="id"
        handle=".gs__handle"
        @change="onChange"
      >
        <template #item="{ element }">
          <div class="gs__group">
            <span class="gs__handle" title="拖动调整顺序">⋮⋮</span>
            <span class="gs__name">📁 {{ element.name }}</span>
          </div>
        </template>
      </draggable>
    </NSpin>

    <template #footer>
      <NSpace justify="end">
        <NButton :disabled="saving" @click="emit('update:show', false)">取消</NButton>
        <NButton
          type="primary"
          :loading="saving"
          :disabled="!dirty"
          @click="save"
        >
          保存顺序
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.gs__group {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  margin-bottom: 6px;
}
.gs__group:hover {
  background: rgba(91, 141, 239, 0.08);
}
.gs__handle {
  cursor: grab;
  font-family: monospace;
  font-size: 1rem;
  opacity: 0.5;
  letter-spacing: -2px;
  user-select: none;
}
.gs__handle:active {
  cursor: grabbing;
}
.gs__name {
  font-weight: 500;
}
</style>
