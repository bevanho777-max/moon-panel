<script setup lang="ts">
// IconAutoComplete — Phase 4c
//
// Reusable icon picker: combines the catalog-backed suggestion dropdown
// (dashboard-icons + lucide) with the StatefulAutoComplete 4-state UX.
// Used in Cards.vue (alongside IconUploader) and Groups.vue (standalone).
//
// Lifts state and helpers that previously lived inline in Cards.vue:
//   - lazy-loaded catalog (loadIconCatalog → ~136KB async chunk)
//   - searchIcons + highlightSegments rendering with thumb + source tag
//   - select handler maps lucide vs dashboard names to icon refs
//
// What's NOT in this component:
//   - URL-paste auto-fetch (tryAutoFetchIcon) — that's a Cards-specific
//     UX. Parent listens to @blur and decides. Groups doesn't bind it.
//   - File upload — Cards uses IconUploader as a sibling; not part of
//     the icon-picker contract.

import { computed, h, nextTick, ref, type VNode } from 'vue'
import type { AutoCompleteOption } from 'naive-ui'
import {
  type IconCandidate,
  loadIconCatalog,
  searchIcons,
  highlightSegments,
} from '@/utils/iconSearch'
import LucideIcon from '@/components/LucideIcon.vue'
import StatefulAutoComplete from '@/components/StatefulAutoComplete.vue'

interface Props {
  modelValue: string
  originalValue: string
  placeholder?: string
  disabled?: boolean
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '搜索图标名（jellyfin / shield-check）/ 粘贴 URL / lucide:foo',
  disabled: false,
  loading: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
  (e: 'select', v: string | number): void
  (e: 'blur'): void
}>()

const DASHBOARD_CDN = 'https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png'

const iconCatalog = ref<IconCandidate[]>([])
const catalogLoading = ref(false)

async function ensureCatalog() {
  if (iconCatalog.value.length > 0 || catalogLoading.value) return
  catalogLoading.value = true
  try {
    iconCatalog.value = await loadIconCatalog()
  } catch (e) {
    console.error('[Moon Panel] icon catalog load failed:', e)
  } finally {
    catalogLoading.value = false
  }
}

interface IconSuggestion extends AutoCompleteOption {
  source: 'dashboard' | 'lucide'
  hits: number[]
  rawName: string
}

const iconSuggestions = computed<IconSuggestion[]>(() => {
  if (iconCatalog.value.length === 0) return []
  const query = props.modelValue
  if (
    query.startsWith('lucide:') ||
    query.startsWith('upload:') ||
    /^https?:\/\//i.test(query)
  ) {
    return []
  }
  const hits = searchIcons(query, iconCatalog.value, 20)
  return hits.map((hit) => ({
    label: hit.name,
    value: hit.source === 'lucide'
      ? `lucide:${hit.name}`
      : `${DASHBOARD_CDN}/${hit.name}.png`,
    source: hit.source,
    hits: hit.hits,
    rawName: hit.name,
  }))
})

function renderSuggestion(option: unknown): VNode {
  const opt = option as IconSuggestion
  const segs = highlightSegments(opt.rawName, opt.hits)
  const thumbStyle = 'width:24px;height:24px;border-radius:4px;flex-shrink:0;display:inline-flex;align-items:center;justify-content:center;background:rgba(255,255,255,0.06)'
  const thumb = opt.source === 'dashboard'
    ? h('img', {
        src: `${DASHBOARD_CDN}/${opt.rawName}.png`,
        loading: 'lazy',
        style: `${thumbStyle};object-fit:contain`,
      })
    : h(
        'span',
        { style: `${thumbStyle};color:#5b8def;background:rgba(91,141,239,0.15)` },
        h(LucideIcon, { name: opt.rawName, size: 16 }),
      )
  const nameNodes = segs.map((seg) =>
    seg.match
      ? h('span', { style: 'color:#5b8def;font-weight:600' }, seg.text)
      : seg.text,
  )
  const sourceTag = h(
    'span',
    {
      style:
        'font-size:0.65rem;opacity:0.55;padding:1px 5px;background:rgba(255,255,255,0.08);border-radius:3px;flex-shrink:0',
    },
    opt.source === 'dashboard' ? 'D' : 'L',
  )
  return h(
    'div',
    {
      style: 'display:flex;align-items:center;gap:10px;padding:2px 0;width:100%',
    },
    [thumb, h('span', { style: 'flex:1;min-width:0' }, nameNodes), sourceTag],
  )
}

function onUpdate(v: string) {
  emit('update:modelValue', v)
}
function onSelect(v: string | number) {
  // NAutoComplete's select() handler does doSelect(option.value) for the
  // select event AND doUpdateValue(option.label) for the v-model — the
  // input commits the LABEL, not the value. Our options use label = bare
  // name ("vmware-esxi") and value = "lucide:vmware-esxi" or full URL,
  // so without intervention the saved icon would lose its prefix and
  // fail the icon-format check. Schedule the override in nextTick so it
  // lands AFTER NAutoComplete's synchronous label commit.
  nextTick(() => emit('update:modelValue', String(v)))
  emit('select', v)
}
function onBlur() {
  emit('blur')
}
</script>

<template>
  <StatefulAutoComplete
    :model-value="modelValue"
    :original-value="originalValue"
    :options="iconSuggestions"
    :render-label="renderSuggestion"
    :placeholder="placeholder"
    :disabled="disabled"
    :loading="loading || catalogLoading"
    @update:model-value="onUpdate"
    @focus="ensureCatalog"
    @select="onSelect"
    @blur="onBlur"
  />
</template>
