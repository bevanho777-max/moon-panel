<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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
import { listCards, reorderCards, type Card } from '@/api/card'
import { listGroups, type Group } from '@/api/group'
import { ApiError } from '@/api/client'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'saved'): void
}>()

const message = useMessage()

const groups = ref<Group[]>([])
const cardsByGroup = ref<Record<number, Card[]>>({})
const loading = ref(false)
const saving = ref(false)
const dirty = ref(false)

async function load() {
  loading.value = true
  dirty.value = false
  try {
    const [g, c] = await Promise.all([listGroups(), listCards()])
    groups.value = g
    const byGroup: Record<number, Card[]> = {}
    for (const grp of g) byGroup[grp.id] = []
    for (const card of c) {
      if (!byGroup[card.group_id]) byGroup[card.group_id] = []
      byGroup[card.group_id].push(card)
    }
    // Sort each group by current sort field — vuedraggable preserves array
    // order, so initial state matches what the user sees on the home page.
    for (const id in byGroup) {
      byGroup[id].sort((a, b) => a.sort - b.sort || a.id - b.id)
    }
    cardsByGroup.value = byGroup
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const groupedCount = computed(() =>
  Object.values(cardsByGroup.value).reduce((n, arr) => n + arr.length, 0),
)

function onChange() {
  dirty.value = true
}

async function save() {
  // Walk the per-group arrays in their current order, assigning sort = 10, 20, 30...
  // This produces clean integer gaps so future single-card sort edits don't
  // need an immediate global renumber. Cross-group moves aren't supported in
  // this modal — those belong in the card edit form (group_id field).
  const items: { id: number; sort: number }[] = []
  for (const id in cardsByGroup.value) {
    cardsByGroup.value[id].forEach((card, i) => {
      items.push({ id: card.id, sort: (i + 1) * 10 })
    })
  }
  if (items.length === 0) {
    emit('update:show', false)
    return
  }
  saving.value = true
  try {
    await reorderCards(items)
    message.success('已保存新的卡片排序')
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
    title="拖拽调整卡片排序"
    style="max-width: 720px"
    :mask-closable="!saving"
    @update:show="(v: boolean) => { if (!v) emit('update:show', false) }"
  >
    <NAlert type="info" :show-icon="false" style="margin-bottom: 12px; font-size: 0.82rem">
      在每个分组内拖动卡片调整顺序。跨分组移动请在卡片编辑对话框里改"所属分组"。
    </NAlert>

    <NSpin :show="loading">
      <NEmpty v-if="!loading && groupedCount === 0" description="还没有卡片" />
      <div v-else class="cs__groups">
        <div v-for="g in groups" :key="g.id" class="cs__group">
          <div class="cs__group-name">📁 {{ g.name }}</div>
          <div v-if="(cardsByGroup[g.id] ?? []).length === 0" class="cs__empty">
            该分组暂无卡片
          </div>
          <draggable
            v-else
            :list="cardsByGroup[g.id]"
            :group="`cards-${g.id}`"
            :animation="160"
            item-key="id"
            handle=".cs__handle"
            @change="onChange"
          >
            <template #item="{ element }">
              <div class="cs__card">
                <span class="cs__handle" title="拖动调整顺序">⋮⋮</span>
                <span class="cs__title">{{ element.title }}</span>
                <span class="cs__url">{{ element.url_internal || element.url_external || '—' }}</span>
              </div>
            </template>
          </draggable>
        </div>
      </div>
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
          保存排序
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<style scoped>
.cs__groups {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: 60vh;
  overflow-y: auto;
}
.cs__group {
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;
  padding: 10px 12px;
}
.cs__group-name {
  font-weight: 500;
  margin-bottom: 8px;
  opacity: 0.85;
}
.cs__empty {
  font-size: 0.78rem;
  opacity: 0.45;
  padding: 6px 0;
}
.cs__card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 5px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.06);
  margin-bottom: 4px;
  cursor: default;
}
.cs__card:hover {
  background: rgba(91, 141, 239, 0.08);
}
.cs__handle {
  cursor: grab;
  font-family: monospace;
  font-size: 0.95rem;
  opacity: 0.5;
  user-select: none;
  letter-spacing: -2px;
}
.cs__handle:active {
  cursor: grabbing;
}
.cs__title {
  font-weight: 500;
  flex-shrink: 0;
  min-width: 100px;
}
.cs__url {
  flex: 1;
  min-width: 0;
  font-size: 0.78rem;
  font-family: monospace;
  opacity: 0.55;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
