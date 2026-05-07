<script setup lang="ts">
import { computed, h, onMounted, ref, type VNode } from 'vue'
import {
  NButton,
  NCard,
  NDataTable,
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
  type DataTableColumns,
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
  updateCard,
} from '@/api/card'
import { fetchIconByURL } from '@/api/icon'
import { useGroupsStore } from '@/stores/groups'
import IconUploader from '@/components/IconUploader.vue'
import LucideIcon from '@/components/LucideIcon.vue'
import CardsSortModal from '@/components/admin/CardsSortModal.vue'
import StatefulInput from '@/components/StatefulInput.vue'
import IconAutoComplete from '@/components/admin/IconAutoComplete.vue'
import { showStatefulInputHintOnce } from '@/utils/statefulInputHint'
// Phase 4c: catalog lookup + suggestion rendering moved into IconAutoComplete.
// Cards.vue keeps only URL-paste auto-fetch (tryAutoFetchIcon).

const message = useMessage()
const groupsStore = useGroupsStore()

const cards = ref<Card[]>([])
const sortOpen = ref(false)

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

const editorTitle = computed(() => (editorMode.value === 'create' ? '新建卡片' : '编辑卡片'))

const groupOptions = computed<SelectOption[]>(() =>
  groupsStore.items.map((g) => ({ label: g.name, value: g.id })),
)

// Pure-frontend filter; future Phase may push to backend if dataset grows huge.
const filteredCards = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return cards.value
  return cards.value.filter((c) =>
    c.title.toLowerCase().includes(q) ||
    c.description.toLowerCase().includes(q) ||
    c.url_internal.toLowerCase().includes(q) ||
    c.url_external.toLowerCase().includes(q) ||
    groupsStore.nameOf(c.group_id).toLowerCase().includes(q),
  )
})

const ICON_LIKE_PREFIX = /^(lucide:|upload:|https?:\/\/)/i

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
    url_default: 'internal',
    open_in_new_tab: true,
    sort: 0,
  }
}

async function refresh() {
  loading.value = true
  try {
    const [c] = await Promise.all([listCards(), groupsStore.ensureLoaded()])
    cards.value = c
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editorMode.value = 'create'
  editingId.value = null
  editorForm.value = emptyForm()
  if (groupsStore.items.length > 0) {
    editorForm.value.group_id = groupsStore.items[0].id
  }
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
// Track URLs that we've already attempted to fetch in this editor session.
// Prevents re-fetching when user blurs the field repeatedly.
const fetchedSet = ref<Set<string>>(new Set())
const iconFetching = ref(false)

// IconAutoComplete handles catalog loading + suggestions internally; Cards.vue
// only needs the URL-paste auto-fetch path. The component emits @select
// with the chosen value (jsdelivr URL or lucide:name); we ignore lucide:
// names and trigger fetch for http(s) URLs.
async function handleSuggestionSelect(value: string | number) {
  if (typeof value === 'string' && /^https?:\/\//i.test(value)) {
    await tryAutoFetchIcon()
  }
}

async function tryAutoFetchIcon() {
  const v = editorForm.value.icon.trim()
  if (!v || !/^https?:\/\//i.test(v)) return
  if (fetchedSet.value.has(v)) return // already tried this URL
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
  // Strip the Phase 4a protocol-only prefills. If user didn't add a host
  // after "http://" / "https://", treat as empty so we don't persist a
  // broken URL.
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

// Icon thumbnail in title cell. URL → <img>; lucide:/upload: → placeholder
// box with letter; empty → muted box. Phase 3 will swap lucide/upload to real
// icon rendering.
// v0.2.13 Patch 3: size 参数化 — PC NDataTable 默认 28, mobile 1-行卡片 22
// (Bevan: "图标可以再缩小一些"). LucideIcon 内 size 按 ~0.65 比例缩 (18/28),
// mobile 22 box 内 lucide 14, 视觉留边一致.
function renderIconThumb(icon: string, size = 28): VNode {
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

// Inline SVG paths derived from Lucide MIT-licensed source (lucide.dev/icons).
// Phase 3 will replace these with the proper lucide-vue-next components.
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

const columns = computed<DataTableColumns<Card>>(() => [
  {
    title: '标题',
    key: 'title',
    minWidth: 200,
    render: (row) =>
      h(
        'div',
        { style: 'display:flex;align-items:center;gap:10px' },
        [renderIconThumb(row.icon), h('span', row.title)],
      ),
  },
  {
    title: '分组',
    key: 'group_id',
    width: 140,
    render: (row) => h(NTag, { size: 'small' }, () => groupsStore.nameOf(row.group_id)),
  },
  {
    title: '内/外网',
    key: 'url_status',
    width: 120,
    render: renderDualURL,
  },
  { title: '排序', key: 'sort', width: 80 },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    render: (row) =>
      h(NSpace, { size: 'small' }, () => [
        h(NButton, { size: 'small', onClick: () => openEdit(row) }, () => '编辑'),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDelete(row),
            positiveText: '删除',
            negativeText: '取消',
          },
          {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, () => '删除'),
            default: () => `删除卡片"${row.title}"？`,
          },
        ),
      ]),
  },
])

onMounted(refresh)
</script>

<template>
  <NCard>
    <template #header>
      <NSpace align="center" justify="space-between" style="width: 100%">
        <span>卡片管理</span>
        <NSpace>
          <NButton :disabled="cards.length === 0" @click="sortOpen = true">调整顺序</NButton>
          <NButton type="primary" :disabled="groupsStore.items.length === 0" @click="openCreate">
            新建卡片
          </NButton>
        </NSpace>
      </NSpace>
    </template>

    <NSpace vertical size="medium">
      <NInput
        v-model:value="searchQuery"
        placeholder="搜索标题 / 描述 / URL / 分组名"
        clearable
      />
      <NSpin :show="loading">
        <NDataTable
          v-if="filteredCards.length > 0"
          :columns="columns"
          :data="filteredCards"
          :row-key="(row: Card) => row.id"
          :bordered="false"
          class="cards-table-pc"
        />

        <!-- v0.2.13: mobile card list. NDataTable 5 列 (标题/分组/内外网/排序/操作)
             横向放不下手机宽度, 改为竖向单卡片堆叠. PC (≥769px) 由 .cards-table-pc
             显示 NDataTable, 手机由 @media 隐藏 NDataTable + 显示此列表.
             Pattern mirrors v0.2.6 AuditLog mobile card list. -->
        <div v-if="filteredCards.length > 0" class="cards-mobile-list">
          <div v-for="c in filteredCards" :key="c.id" class="cards-mobile-list__item">
            <!-- v0.2.13 Patch 2+3: 1 行 flex (Bevan "可以直接一行吗?" → "图标和字
                 可以再缩小"). 删除 mobile 分组 NTag 显示 (PC NDataTable 仍显示).
                 icon 22px (PC 默认 28), 高度 ~40-42px (NButton small 28 锁定). -->
            <component :is="renderIconThumb(c.icon, 22)" />
            <span class="cards-mobile-list__title">{{ c.title }}</span>
            <span class="cards-mobile-list__icons">
              <component :is="renderDualURL(c)" />
            </span>
            <div class="cards-mobile-list__actions">
              <NButton size="small" @click="openEdit(c)">编辑</NButton>
              <NPopconfirm
                :positive-text="'删除'"
                :negative-text="'取消'"
                @positive-click="handleDelete(c)"
              >
                <template #trigger>
                  <NButton size="small" type="error" ghost>删除</NButton>
                </template>
                删除卡片"{{ c.title }}"？
              </NPopconfirm>
            </div>
          </div>
        </div>

        <NEmpty
          v-else-if="groupsStore.items.length === 0"
          description="还没有分组。请先到「分组」页面创建一个分组，再回来添加卡片。"
        />
        <NEmpty
          v-else-if="cards.length === 0"
          description="还没有卡片，点右上角「新建卡片」开始添加"
        />
        <NEmpty
          v-else
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
        <NRadioGroup v-model:value="editorForm.url_default" :disabled="submitting">
          <NRadio value="internal">内网</NRadio>
          <NRadio value="external">外网</NRadio>
        </NRadioGroup>
      </NFormItem>
      <NFormItem label="新标签页打开">
        <NSwitch v-model:value="editorForm.open_in_new_tab" :disabled="submitting" />
      </NFormItem>
      <!-- v0.2.13: 排序权重 NInputNumber 移除. CardsSortModal 已通过 vuedraggable
           提供拖拽排序, NInputNumber 数字编辑属于 10 年前过时方案. editorForm.sort
           保留在 reactive 定义中以维持 backend 协议向后兼容 (默认 0, 拖拽时由
           CardsSortModal 重写). -->
      <NSpace justify="end">
        <NButton @click="editorOpen = false" :disabled="submitting">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="submit">
          {{ editorMode === 'create' ? '创建' : '保存' }}
        </NButton>
      </NSpace>
    </NForm>
  </NModal>

  <CardsSortModal v-model:show="sortOpen" @saved="refresh" />
</template>

<style scoped>
/* v0.2.13 P0 a + Patch 1+2: Mobile-only card list, 1-row 极致紧凑.
   PC NDataTable hidden ≤768px. Patch 2 (Bevan: "可以直接一行吗?"): 2 行合并
   为 1 行, 删 NTag 分组显示, 高度 ~50px (vs Patch 1 ~82px, vs 原 3 行 ~150-180px).
   Trade-off: 长卡片名 (>65px) mobile ellipsis; PC NDataTable 仍完整可见. */
.cards-mobile-list {
  display: none;
}

@media (max-width: 768px) {
  .cards-table-pc {
    display: none;
  }
  .cards-mobile-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  /* Item 直接成 flex row container: icon + title (flex 1, ellipsis) + 内外网 + actions */
  .cards-mobile-list__item {
    /* v0.2.13 Patch 3: padding 8→6 上下 (省 4px), 横向 10 维持呼吸感 */
    padding: 6px 10px;
    border-radius: 8px;
    background: var(--mp-card-bg);
    border: 1px solid var(--mp-card-border);
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .cards-mobile-list__title {
    /* v0.2.13 Patch 3: font 0.95→0.85rem + line-height 1.3→1.2 */
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--mp-text-primary);
    line-height: 1.2;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1 1 auto;
    min-width: 0;
  }
  .cards-mobile-list__icons {
    display: flex;
    gap: 4px;
    align-items: center;
    flex-shrink: 0;
  }
  .cards-mobile-list__actions {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }
}
</style>
