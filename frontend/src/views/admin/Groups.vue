<script setup lang="ts">
import { computed, h, onMounted, ref, type VNode } from 'vue'
import {
  NButton,
  NCard,
  NEmpty,
  NForm,
  NFormItem,
  NModal,
  NPopconfirm,
  NSpace,
  NSpin,
  useMessage,
} from 'naive-ui'
import draggable from 'vuedraggable'
import { GripVertical } from 'lucide-vue-next'
import { ApiError } from '@/api/client'
import {
  createGroup,
  deleteGroup,
  listGroups,
  reorderGroups,
  updateGroup,
  type Group,
} from '@/api/group'
import { useGroupsStore } from '@/stores/groups'
import LucideIcon from '@/components/LucideIcon.vue'
import StatefulInput from '@/components/StatefulInput.vue'
import IconAutoComplete from '@/components/admin/IconAutoComplete.vue'
import { showStatefulInputHintOnce } from '@/utils/statefulInputHint'

const message = useMessage()
const groupsStore = useGroupsStore()

const groups = ref<Group[]>([])
const loading = ref(false)
// Snapshot of editorForm at modal open time — fed into StatefulInput so it
// can compare current vs original and drive its 4-state UX.
const editorOriginal = ref({ name: '', icon: '' })

const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const editorForm = ref({ name: '', icon: '', sort: 0 })
const submitting = ref(false)

const editorTitle = computed(() => (editorMode.value === 'create' ? '新建分组' : '编辑分组'))

async function refresh() {
  loading.value = true
  try {
    const list = await listGroups()
    list.sort((a, b) => a.sort - b.sort || a.id - b.id)
    groups.value = list
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

// v0.2.16 P0 a: 拖拽结束 → 重算 sort = (i+1)*10, 立即 PUT (auto-save). 失败时
// reload (server state rollback). Mirrors v0.2.15 Cards.vue onCardReorder.
async function onGroupReorder() {
  const items = groups.value.map((g, i) => ({
    id: g.id,
    sort: (i + 1) * 10,
  }))
  if (items.length === 0) return
  try {
    await reorderGroups(items)
    await refresh()
    // Keep groupsStore (used by Cards.vue dropdown) in sync.
    await groupsStore.invalidate()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '排序保存失败, 已撤销')
    await refresh()
  }
}

function openCreate() {
  editorMode.value = 'create'
  editingId.value = null
  editorForm.value = { name: '', icon: '', sort: 0 }
  editorOriginal.value = { name: '', icon: '' }
  showStatefulInputHintOnce(message)
  editorOpen.value = true
}

function openEdit(g: Group) {
  editorMode.value = 'edit'
  editingId.value = g.id
  editorForm.value = { name: g.name, icon: g.icon, sort: g.sort }
  editorOriginal.value = { name: g.name, icon: g.icon }
  showStatefulInputHintOnce(message)
  editorOpen.value = true
}

async function submit() {
  const name = editorForm.value.name.trim()
  if (!name) {
    message.warning('分组名不能为空')
    return
  }
  submitting.value = true
  try {
    const payload = {
      name,
      icon: editorForm.value.icon,
      // sort 字段保留 reactive 维持 backend 协议向后兼容. 编辑表单不再含
      // NInputNumber 排序权重 (v0.2.16 删, 跟 v0.2.13 admin Cards 一致),
      // sort 由 inline drag onGroupReorder 重算.
    }
    if (editorMode.value === 'create') {
      await createGroup(payload)
      message.success('已创建')
    } else if (editingId.value !== null) {
      await updateGroup(editingId.value, payload)
      message.success('已更新')
    }
    editorOpen.value = false
    await refresh()
    // Invalidate the shared groups store so Cards.vue dropdown shows the
    // new name/sort immediately (4b polish — fixes the "rename group, then
    // open card editor → still shows old name" stale-cache bug).
    await groupsStore.invalidate()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(g: Group) {
  try {
    await deleteGroup(g.id)
    message.success(`已删除"${g.name}"`)
    await refresh()
    await groupsStore.invalidate()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

// v0.2.16: icon thumbnail render helper (mirror v0.2.15 Cards.vue renderIconThumb).
// size 参数化 (PC 默认 28, mobile-list 22). LucideIcon 内 size ~0.65 比例.
function renderGroupIcon(icon: string, size = 22): VNode {
  const dim = `${size}px`
  const lucideSize = Math.round(size * 0.65)
  const baseStyle = `width:${dim};height:${dim};border-radius:4px;flex-shrink:0;display:inline-flex;align-items:center;justify-content:center;font-size:11px;font-weight:600`
  if (!icon) {
    return h('div', {
      style: `${baseStyle};background:rgba(255,255,255,0.05);color:rgba(255,255,255,0.3)`,
    }, '—')
  }
  if (/^https?:\/\//i.test(icon)) {
    return h('img', {
      src: icon,
      style: `${baseStyle};object-fit:cover;background:rgba(255,255,255,0.05)`,
      onError: (e: Event) => {
        const img = e.target as HTMLImageElement
        img.style.display = 'none'
      },
    })
  }
  if (icon.startsWith('lucide:')) {
    return h(
      'div',
      {
        style: `${baseStyle};background:rgba(91,141,239,0.15);color:#5b8def`,
        title: icon,
      },
      h(LucideIcon, { name: icon.slice('lucide:'.length), size: lucideSize }),
    )
  }
  if (icon.startsWith('upload:')) {
    return h('img', {
      src: '/uploads/' + icon.slice('upload:'.length),
      style: `${baseStyle};object-fit:cover;background:rgba(255,255,255,0.05)`,
      title: icon,
      onError: (e: Event) => {
        const img = e.target as HTMLImageElement
        img.style.display = 'none'
      },
    })
  }
  return h('div', {
    style: `${baseStyle};background:rgba(255,193,77,0.15);color:#ffc14d`,
    title: icon,
  }, '?')
}

onMounted(refresh)
</script>

<template>
  <NCard>
    <template #header>
      <NSpace align="center" justify="space-between" style="width: 100%">
        <span>分组管理</span>
        <NButton type="primary" @click="openCreate">新建分组</NButton>
      </NSpace>
    </template>

    <NSpin :show="loading">
      <!-- v0.2.16 P0 a: groups-list 统一 PC + mobile (X3 inline drag, 弃 NDataTable
           + 弃 GroupsSortModal). per-row ⋮⋮ handle, 拖完立即 reorderGroups API.
           Mirrors v0.2.15 Cards.vue .cards-list pattern (single list, 无跨容器). -->
      <div v-if="groups.length > 0" class="groups-list">
        <draggable
          :list="groups"
          :group="'groups'"
          :animation="160"
          item-key="id"
          handle=".groups-list__handle"
          class="groups-list__items"
          @end="onGroupReorder"
        >
          <template #item="{ element: group }">
            <div class="groups-list__item">
              <button
                type="button"
                class="groups-list__handle"
                title="拖动调整顺序"
              >
                <GripVertical :size="16" />
              </button>
              <component :is="renderGroupIcon(group.icon, 22)" />
              <span class="groups-list__title">{{ group.name }}</span>
              <span class="groups-list__id">ID: {{ group.id }}</span>
              <span class="groups-list__sort">{{ group.sort }}</span>
              <div class="groups-list__actions">
                <NButton size="small" @click="openEdit(group)">编辑</NButton>
                <NPopconfirm
                  :positive-text="'删除'"
                  :negative-text="'取消'"
                  @positive-click="handleDelete(group)"
                >
                  <template #trigger>
                    <NButton size="small" type="error" ghost>删除</NButton>
                  </template>
                  删除分组"{{ group.name }}"？组内卡片会一并删除。
                </NPopconfirm>
              </div>
            </div>
          </template>
        </draggable>
      </div>

      <NEmpty v-else description="还没有分组，点右上角「新建分组」开始添加" />
    </NSpin>
  </NCard>

  <NModal
    v-model:show="editorOpen"
    preset="card"
    :title="editorTitle"
    style="max-width: 480px"
    :mask-closable="!submitting"
  >
    <NForm @submit.prevent="submit">
      <NFormItem label="名称" required>
        <StatefulInput
          v-model="editorForm.name"
          :original-value="editorOriginal.name"
          placeholder="例如：影音、工具、开发"
          :maxlength="128"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="图标">
        <IconAutoComplete
          v-model="editorForm.icon"
          :original-value="editorOriginal.icon"
          :disabled="submitting"
        />
      </NFormItem>
      <!-- v0.2.16: 排序权重 NInputNumber 移除 (X3 inline drag fully replace; 跟
           v0.2.13 admin Cards 编辑表单删除一致). editorForm.sort 保留 reactive
           default 0 维持 backend 协议向后兼容. -->
      <NSpace justify="end">
        <NButton @click="editorOpen = false" :disabled="submitting">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="submit">
          {{ editorMode === 'create' ? '创建' : '保存' }}
        </NButton>
      </NSpace>
    </NForm>
  </NModal>
</template>

<style scoped>
/* v0.2.16 P0 a: groups-list 统一 PC + mobile 模板 (X3 inline drag, 跟 v0.2.15
   .cards-list 一致). single list (groups 无 group concept), per-row handle. */
.groups-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.groups-list__items {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.groups-list__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 8px;
  background: var(--mp-card-bg);
  border: 1px solid var(--mp-card-border);
}
.groups-list__handle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: none;
  cursor: grab;
  color: var(--mp-text-secondary);
  flex-shrink: 0;
  border-radius: 4px;
  transition: color 0.15s;
}
.groups-list__handle:hover {
  color: var(--mp-brand-primary);
}
.groups-list__handle:active {
  cursor: grabbing;
}
.groups-list__title {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--mp-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1 1 auto;
  min-width: 0;
}
.groups-list__id {
  font-size: 0.85rem;
  color: var(--mp-text-secondary);
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}
.groups-list__sort {
  font-size: 0.85rem;
  color: var(--mp-text-secondary);
  width: 32px;
  text-align: center;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}
.groups-list__actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

/* Mobile @media: PC-only 元素隐藏, font 缩小 (跟 .cards-list 一致). */
@media (max-width: 768px) {
  .groups-list__id {
    display: none;
  }
  .groups-list__sort {
    display: none;
  }
  .groups-list__title {
    font-size: 0.85rem;
    line-height: 1.2;
  }
}
</style>
