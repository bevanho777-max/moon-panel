<script setup lang="ts">
import { computed, h, onMounted, ref, watch, type VNode } from 'vue'
import {
  NButton,
  NCard,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NPopconfirm,
  NRadio,
  NRadioGroup,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTag,
  useMessage,
  type SelectOption,
} from 'naive-ui'
import { ApiError } from '@/api/client'
import {
  type Card,
  type CardWritePayload,
  createCard,
  deleteCard,
  getCard,
  listCards,
  reorderCards,
  updateCard,
} from '@/api/card'
import { fetchIconByURL } from '@/api/icon'
import { useGroupsStore } from '@/stores/groups'
import IconUploader from '@/components/IconUploader.vue'
import LucideIcon from '@/components/LucideIcon.vue'
import SortableTable from '@/components/SortableTable.vue'
import StatefulInput from '@/components/StatefulInput.vue'
import IconAutoComplete from '@/components/admin/IconAutoComplete.vue'
import { showStatefulInputHintOnce } from '@/utils/statefulInputHint'

const message = useMessage()
const groupsStore = useGroupsStore()

// v0.2.15 P0 b: localStorage 记忆上次新建卡片时选择的分组. 符合 moon.* 命名约定
// (see memory/feedback_localstorage_naming.md). 仅新建路径写入, 编辑不污染.
const STORAGE_KEY_LAST_GROUP = 'moon.admin.cards.last_group_id'

const cards = ref<Card[]>([])

// v0.2.15 P0 a + v0.2.18: per-group nested 数据 (跟 SortableTable interface 对齐,
// 字段名 items 而不是 cards). vuedraggable 需要 mutate :list array, 所以是 ref
// 不是 computed. refresh() 时 rebuild, 拖完 onCardReorder() 调 refresh() 同步 sort.
const cardsByGroup = ref<{ id: number; name: string; items: Card[] }[]>([])

// Snapshot of editorForm at open time. StatefulInput compares modelValue to
// originalValue to drive its 4-state UX. For create mode we use empty strings;
// for edit mode we copy the card's persisted values.
const editorOriginal = ref({
  title: '',
  description: '',
  icon: '',
  url_internal: '',
  url_external: '',
})
const loading = ref(false)

const searchQuery = ref('')

const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)
const editorForm = ref<Required<CardWritePayload>>(emptyForm())
const submitting = ref(false)
// v0.2.24 B.3: tracks whether the user manually picked a radio this edit
// session. False = watch auto-follows URL input. True = leave alone (respects
// what the user — or, in edit mode, the persisted record — already chose).
const userPickedUrlDefault = ref(false)

const editorTitle = computed(() => (editorMode.value === 'create' ? '新建卡片' : '编辑卡片'))

const groupOptions = computed<SelectOption[]>(() =>
  groupsStore.items.map((g) => ({ label: g.name, value: g.id })),
)

// v0.2.15 P0 a: search filter 仅 visual (per-item v-show via SortableTable
// itemFilter prop), draggable disabled while searching. search 清空后恢复.
const isSearching = computed(() => searchQuery.value.trim() !== '')

function cardMatchesSearch(card: Card): boolean {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return true
  return (
    card.title.toLowerCase().includes(q) ||
    card.description.toLowerCase().includes(q) ||
    card.url_internal.toLowerCase().includes(q) ||
    card.url_external.toLowerCase().includes(q) ||
    groupsStore.nameOf(card.group_id).toLowerCase().includes(q)
  )
}

const totalMatchedCount = computed(() =>
  cards.value.filter(cardMatchesSearch).length,
)

const ICON_LIKE_PREFIX = /^(lucide:|upload:|https?:\/\/)/i

// v0.2.24 B.3: A1 自动跟随. "实质空" 判断跟 submit handler (line ~293) 同一
// 逻辑, 避免新建表单的 protocol prefix 'http://' / 'https://' 被当作有内容
// 误触 (Flag #33).
function isUrlSubstantiallyEmpty(v: string): boolean {
  const trimmed = (v || '').trim()
  return trimmed === '' || trimmed === 'http://' || trimmed === 'https://'
}

watch(() => editorForm.value.url_internal, (newVal) => {
  if (userPickedUrlDefault.value) return
  const intEmpty = isUrlSubstantiallyEmpty(newVal)
  const extEmpty = isUrlSubstantiallyEmpty(editorForm.value.url_external)
  if (intEmpty && extEmpty) {
    editorForm.value.url_default = ''
  } else if (!intEmpty && extEmpty) {
    editorForm.value.url_default = 'internal'
  } else if (!intEmpty && !extEmpty) {
    // 同时有值 → 跟最后输入 (Bevan X1, watch 顺序决定)
    editorForm.value.url_default = 'internal'
  }
})

watch(() => editorForm.value.url_external, (newVal) => {
  if (userPickedUrlDefault.value) return
  const intEmpty = isUrlSubstantiallyEmpty(editorForm.value.url_internal)
  const extEmpty = isUrlSubstantiallyEmpty(newVal)
  if (intEmpty && extEmpty) {
    editorForm.value.url_default = ''
  } else if (intEmpty && !extEmpty) {
    editorForm.value.url_default = 'external'
  } else if (!intEmpty && !extEmpty) {
    // 同时有值 → 跟最后输入 (Bevan X1)
    editorForm.value.url_default = 'external'
  }
})

function onUrlDefaultPick() {
  // 用户点 radio → 后续 watch 跳过自动跟随 (Bevan X1: 尊重历史)
  userPickedUrlDefault.value = true
}

function emptyForm(): Required<CardWritePayload> {
  return {
    group_id: 0,
    title: '',
    description: '',
    icon: '',
    // Phase 4a: prefill protocols on NEW cards. Internal services typically
    // use http://, external https://. User can edit either freely. Edit-mode
    // skips this entirely (existing url_internal/url_external loaded as-is).
    url_internal: 'http://',
    url_external: 'https://',
    url_default: '',
    open_in_new_tab: true,
    sort: 0,
  }
}

// v0.2.15 P0 a: build per-group cards bucket (groups 顺序 + 组内 sort 排).
function rebuildCardsByGroup(allCards: Card[]) {
  const byGroup: Record<number, Card[]> = {}
  for (const g of groupsStore.items) byGroup[g.id] = []
  for (const card of allCards) {
    if (!byGroup[card.group_id]) byGroup[card.group_id] = []
    byGroup[card.group_id].push(card)
  }
  for (const id in byGroup) {
    byGroup[id].sort((a, b) => a.sort - b.sort || a.id - b.id)
  }
  cardsByGroup.value = groupsStore.items.map((g) => ({
    id: g.id,
    name: g.name,
    items: byGroup[g.id] ?? [],
  }))
}

async function refresh() {
  loading.value = true
  try {
    const [c] = await Promise.all([listCards(), groupsStore.ensureLoaded()])
    cards.value = c
    rebuildCardsByGroup(c)
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

// v0.2.15 P0 a + Patch 1: 拖拽结束 → 收集所有 group 当前状态, 重算 sort + group_id,
// 立即 PUT (auto-save). 跨分组拖时 vuedraggable 已 splice from-group + push to-group
// (因为各 group 同 :group="'cards'", sortablejs 跨容器移动 nested array). 遍历
// cardsByGroup 即可拿到最新状态, group_id 用 group.id (拖入目标分组). 失败时
// reload (server state rollback). Bevan 决策 C.1 (无 debounce, 即时反馈) + Patch 1
// B.1b (跨分组允许).
async function onCardReorder() {
  const items: { id: number; sort: number; group_id: number }[] = []
  for (const g of cardsByGroup.value) {
    g.items.forEach((card, i) => {
      items.push({
        id: card.id,
        sort: (i + 1) * 10,
        group_id: g.id,
      })
    })
  }
  if (items.length === 0) return
  try {
    await reorderCards(items)
    // Patch 1: 跨分组拖后重 fetch, 同步 cards.value 内 group_id 字段 (vuedraggable
    // 仅 mutate cardsByGroup nested array, cards ref 内的 group_id 是旧值)
    await refresh()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '排序保存失败, 已撤销')
    await refresh()
  }
}

function readLastGroupId(): number | null {
  const raw = localStorage.getItem(STORAGE_KEY_LAST_GROUP)
  if (!raw) return null
  const n = Number(raw)
  if (!Number.isFinite(n) || n <= 0) return null
  // Validate: group 仍存在 (防被删后无效)
  return groupsStore.items.some((g) => g.id === n) ? n : null
}

function openCreate() {
  editorMode.value = 'create'
  editingId.value = null
  editorForm.value = emptyForm()
  // v0.2.24 B.3: 新建态允许 watch 自动跟随 URL 输入
  userPickedUrlDefault.value = false
  // v0.2.15 P0 b: 优先 localStorage (有效), fallback 第一个分组.
  editorForm.value.group_id =
    readLastGroupId() ?? groupsStore.items[0]?.id ?? 0
  // Create mode: original values are all empty (StatefulInput skips B state
  // when originalValue is empty, so users see normal "click and type" UX).
  editorOriginal.value = { title: '', description: '', icon: '', url_internal: '', url_external: '' }
  // Phase 4b: refetch groups on open. Cheap insurance against stale dropdown
  // when admin renamed a group in another tab between page load and now.
  groupsStore.invalidate()
  showStatefulInputHintOnce(message)
  editorOpen.value = true
}

async function openEdit(c: Card) {
  editorMode.value = 'edit'
  editingId.value = c.id
  submitting.value = true
  try {
    const fresh = await getCard(c.id)
    editorForm.value = {
      group_id: fresh.group_id,
      title: fresh.title,
      description: fresh.description,
      icon: fresh.icon,
      url_internal: fresh.url_internal,
      url_external: fresh.url_external,
      url_default: fresh.url_default,
      open_in_new_tab: fresh.open_in_new_tab,
      sort: fresh.sort,
    }
    // v0.2.24 B.3: 编辑态尊重已存 url_default, 不自动跟随 (Bevan X1)
    userPickedUrlDefault.value = true
    editorOriginal.value = {
      title: fresh.title,
      description: fresh.description,
      icon: fresh.icon,
      url_internal: fresh.url_internal,
      url_external: fresh.url_external,
    }
    groupsStore.invalidate()
    showStatefulInputHintOnce(message)
    editorOpen.value = true
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载卡片失败')
  } finally {
    submitting.value = false
  }
}

// Soft-warn but don't block; user retains final say (per Phase 2.3b prereq).
const fetchedSet = ref<Set<string>>(new Set())
const iconFetching = ref(false)

async function handleSuggestionSelect(value: string | number) {
  if (typeof value === 'string' && /^https?:\/\//i.test(value)) {
    await tryAutoFetchIcon()
  }
}

async function tryAutoFetchIcon() {
  const v = editorForm.value.icon.trim()
  if (!v || !/^https?:\/\//i.test(v)) return
  if (fetchedSet.value.has(v)) return
  fetchedSet.value.add(v)
  iconFetching.value = true
  try {
    const result = await fetchIconByURL(v)
    editorForm.value.icon = result.icon
    if (result.deduped) {
      message.success('已缓存（命中已有图标，复用本地副本）')
    } else {
      message.success(`已下载缓存到本地 (${formatBytes(result.size)})`)
    }
  } catch (e) {
    if (e instanceof ApiError) {
      message.warning(`URL fetch 失败：${e.message}（保留原 URL，保存时仍会用此 URL）`)
    } else {
      message.warning('URL fetch 失败（保留原 URL）')
    }
  } finally {
    iconFetching.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function emitFieldShapeWarnings(f: Required<CardWritePayload>) {
  if (ICON_LIKE_PREFIX.test(f.title)) {
    message.warning(`标题"${f.title.slice(0, 20)}"看起来像图标值或 URL，是不是填错位置了？已继续保存。`)
  }
  if (f.icon !== '' && !ICON_LIKE_PREFIX.test(f.icon)) {
    message.warning(`图标"${f.icon.slice(0, 20)}"格式不像 lucide: / upload: / http(s) URL，可能填错？已继续保存。`)
  }
}

async function submit() {
  const f = editorForm.value
  if (f.url_internal.trim() === 'http://' || f.url_internal.trim() === 'https://') {
    f.url_internal = ''
  }
  if (f.url_external.trim() === 'http://' || f.url_external.trim() === 'https://') {
    f.url_external = ''
  }
  if (!f.group_id) {
    message.warning('请选择分组')
    return
  }
  if (!f.title.trim()) {
    message.warning('标题不能为空')
    return
  }
  if (!f.url_internal.trim() && !f.url_external.trim()) {
    message.warning('内网地址和外网地址至少填一个')
    return
  }
  emitFieldShapeWarnings(f)
  submitting.value = true
  try {
    if (editorMode.value === 'create') {
      await createCard(f)
      // v0.2.15 P0 b: 仅新建成功后存 localStorage (编辑不污染).
      if (f.group_id) {
        localStorage.setItem(STORAGE_KEY_LAST_GROUP, String(f.group_id))
      }
      message.success('已创建')
    } else if (editingId.value !== null) {
      await updateCard(editingId.value, f)
      message.success('已更新')
    }
    editorOpen.value = false
    await refresh()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(c: Card) {
  try {
    await deleteCard(c.id)
    message.success(`已删除"${c.title}"`)
    await refresh()
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '删除失败')
  }
}

// ---------- Render helpers (icon thumbnail + dual-URL status) ----------

// v0.2.13 Patch 3: size 参数化 — 默认 22 (sortable-table cells). LucideIcon 内
// size 按 ~0.65 比例缩 (22→14).
function renderIconThumb(icon: string, size = 22): VNode {
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

function homeSvg(active: boolean, isDefault: boolean): VNode {
  return h(
    'svg',
    {
      width: 18,
      height: 18,
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: active ? '#5b8def' : 'rgba(255,255,255,0.25)',
      'stroke-width': isDefault ? 2.5 : 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
    },
    [
      h('path', { d: 'm3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z' }),
      h('polyline', { points: '9 22 9 12 15 12 15 22' }),
    ],
  )
}

function globeSvg(active: boolean, isDefault: boolean): VNode {
  return h(
    'svg',
    {
      width: 18,
      height: 18,
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: active ? '#63e2b7' : 'rgba(255,255,255,0.25)',
      'stroke-width': isDefault ? 2.5 : 2,
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
    },
    [
      h('circle', { cx: 12, cy: 12, r: 10 }),
      h('path', { d: 'M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20' }),
      h('path', { d: 'M2 12h20' }),
    ],
  )
}

function renderDualURL(card: Card): VNode {
  const internalSet = card.url_internal.trim() !== ''
  const externalSet = card.url_external.trim() !== ''
  const internalIsDefault = card.url_default === 'internal'
  const externalIsDefault = card.url_default === 'external'

  const slot = (icon: VNode, isDefault: boolean, label: string): VNode =>
    h(
      'div',
      {
        style: `display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;border-radius:4px;border:1.5px solid ${isDefault ? '#5b8def' : 'transparent'}`,
        title: label,
      },
      icon,
    )

  return h(
    'div',
    { style: 'display:flex;gap:6px;align-items:center' },
    [
      slot(homeSvg(internalSet, internalIsDefault), internalIsDefault, internalSet ? `内网: ${card.url_internal}` : '内网未设'),
      slot(globeSvg(externalSet, externalIsDefault), externalIsDefault, externalSet ? `外网: ${card.url_external}` : '外网未设'),
    ],
  )
}

onMounted(refresh)
</script>

<template>
  <NCard>
    <template #header>
      <NSpace align="center" justify="space-between" style="width: 100%">
        <span>卡片管理</span>
        <NButton type="primary" :disabled="groupsStore.items.length === 0" @click="openCreate">
          新建卡片
        </NButton>
      </NSpace>
    </template>

    <NSpace vertical size="medium">
      <NInput
        v-model:value="searchQuery"
        placeholder="搜索标题 / 描述 / URL / 分组名"
        clearable
      />
      <NSpin :show="loading">
        <!-- v0.2.18: SortableTable 抽象 (Rule of Three, 复用跨 Cards/Groups/Search Engines).
             保留 v0.2.15 P0 a (跨分组拖, :group-name="cards" 同名), P0 b (localStorage
             分组记忆 — openCreate/submit 内), search disable + v-show 隐藏 (itemFilter prop). -->
        <SortableTable
          v-if="cards.length > 0"
          :groups="cardsByGroup"
          group-name="cards"
          :disabled="isSearching"
          :show-group-headers="true"
          :item-filter="cardMatchesSearch"
          @reorder="onCardReorder"
        >
          <template #item="{ item: card }">
            <component :is="renderIconThumb(card.icon, 22)" />
            <span class="cards-cell__title">{{ card.title }}</span>
            <NTag size="small" class="cards-cell__group-tag">
              {{ groupsStore.nameOf(card.group_id) }}
            </NTag>
            <span class="cards-cell__icons">
              <component :is="renderDualURL(card)" />
            </span>
            <span class="cards-cell__sort">{{ card.sort }}</span>
            <div class="cards-cell__actions">
              <NButton size="small" @click="openEdit(card)">编辑</NButton>
              <NPopconfirm
                :positive-text="'删除'"
                :negative-text="'取消'"
                @positive-click="handleDelete(card)"
              >
                <template #trigger>
                  <NButton size="small" type="error" ghost>删除</NButton>
                </template>
                删除卡片"{{ card.title }}"？
              </NPopconfirm>
            </div>
          </template>
        </SortableTable>

        <NEmpty
          v-if="groupsStore.items.length === 0"
          description="还没有分组。请先到「分组」页面创建一个分组，再回来添加卡片。"
        />
        <NEmpty
          v-else-if="cards.length === 0"
          description="还没有卡片，点右上角「新建卡片」开始添加"
        />
        <NEmpty
          v-else-if="isSearching && totalMatchedCount === 0"
          :description="`没有匹配「${searchQuery}」的卡片`"
        />
      </NSpin>
    </NSpace>
  </NCard>

  <NModal
    v-model:show="editorOpen"
    preset="card"
    :title="editorTitle"
    style="max-width: 560px"
    :mask-closable="!submitting"
  >
    <NForm @submit.prevent="submit">
      <NFormItem label="分组" required>
        <NSelect
          v-model:value="editorForm.group_id"
          :options="groupOptions"
          :disabled="submitting"
          placeholder="选择分组"
        />
      </NFormItem>
      <NFormItem label="标题" required>
        <StatefulInput
          v-model="editorForm.title"
          :original-value="editorOriginal.title"
          placeholder="例如：Jellyfin"
          :maxlength="128"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="描述">
        <StatefulInput
          v-model="editorForm.description"
          :original-value="editorOriginal.description"
          type="textarea"
          placeholder="可选，鼠标悬停时显示"
          :maxlength="512"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="图标">
        <NSpace vertical :size="8" style="width: 100%">
          <IconAutoComplete
            v-model="editorForm.icon"
            :original-value="editorOriginal.icon"
            :disabled="submitting || iconFetching"
            :loading="iconFetching"
            @select="handleSuggestionSelect"
            @blur="tryAutoFetchIcon"
          />
          <IconUploader
            :current="editorForm.icon"
            @uploaded="(p) => { editorForm.icon = p.iconRef }"
          />
        </NSpace>
      </NFormItem>
      <NFormItem label="内网地址">
        <StatefulInput
          v-model="editorForm.url_internal"
          :original-value="editorOriginal.url_internal"
          placeholder="http://192.168.1.10:8096"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="外网地址">
        <StatefulInput
          v-model="editorForm.url_external"
          :original-value="editorOriginal.url_external"
          placeholder="https://media.example.com"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem label="默认指向">
        <NRadioGroup
          v-model:value="editorForm.url_default"
          :disabled="submitting"
          @update:value="onUrlDefaultPick"
        >
          <NRadio value="internal">内网</NRadio>
          <NRadio value="external">外网</NRadio>
        </NRadioGroup>
      </NFormItem>
      <NFormItem label="新标签页打开">
        <NSwitch v-model:value="editorForm.open_in_new_tab" :disabled="submitting" />
      </NFormItem>
      <!-- v0.2.13: 排序权重 NInputNumber 移除 (v0.2.15 主表格直拖, v0.2.18 SortableTable
           抽象). editorForm.sort 保留 reactive 维持 backend 协议向后兼容. -->
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
/* v0.2.18: cells-only styles (.sortable-table 共性已抽到 SortableTable.vue 内).
   Cards 独有 cell: title + group-tag (PC) + icons (dual-URL) + sort (PC) + actions. */
.cards-cell__title {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--mp-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1 1 auto;
  min-width: 0;
}
.cards-cell__group-tag {
  flex-shrink: 0;
}
.cards-cell__icons {
  display: flex;
  gap: 4px;
  align-items: center;
  flex-shrink: 0;
}
.cards-cell__sort {
  font-size: 0.85rem;
  color: var(--mp-text-secondary);
  width: 32px;
  text-align: center;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}
.cards-cell__actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .cards-cell__group-tag {
    display: none;
  }
  .cards-cell__sort {
    display: none;
  }
  .cards-cell__icons {
    display: none;
  }
  .cards-cell__title {
    font-size: 0.85rem;
    line-height: 1.2;
  }
}
</style>
