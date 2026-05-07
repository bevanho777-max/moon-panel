<script setup lang="ts">
import { computed, h, ref, type VNode } from 'vue'
import { NButton, NDropdown, NInput, type DropdownOption } from 'naive-ui'
import { buildSearchURL, type SearchEngine } from '@/api/searchEngine'
import { useSearchEngineStore } from '@/stores/searchEngines'

const props = defineProps<{
  engines: SearchEngine[]
}>()

const emit = defineEmits<{
  /** Fired on every keystroke. Parent uses this for in-place card filtering. */
  (e: 'update:query', value: string): void
}>()

const store = useSearchEngineStore()

const query = ref('')

// Active engine: user's pick if still in the engine list, else the default flag,
// else first available, else null. Falling back keeps the picker working when
// admins delete the previously-selected engine.
const activeEngine = computed<SearchEngine | null>(() => {
  if (store.selectedId !== null) {
    const picked = props.engines.find((e) => e.id === store.selectedId)
    if (picked) return picked
  }
  const fallback = props.engines.find((e) => e.is_default)
  if (fallback) return fallback
  return props.engines[0] ?? null
})

function onInput(v: string) {
  query.value = v
  emit('update:query', v)
}

function handleEnter() {
  const q = query.value.trim()
  if (!q || !activeEngine.value) return
  const url = buildSearchURL(activeEngine.value.url_template, q)
  window.open(url, '_blank', 'noopener,noreferrer')
}

function handleEngineSelect(key: string | number) {
  const id = Number(key)
  store.setSelected(id)
}

function renderEngineIcon(icon: string, size: number): VNode {
  const dim = `${size}px`
  const baseStyle = `width:${dim};height:${dim};border-radius:3px;flex-shrink:0;display:inline-flex;align-items:center;justify-content:center;background:rgba(255,255,255,0.05)`
  if (!icon) {
    return h('div', { style: `${baseStyle};color:rgba(255,255,255,0.3);font-size:10px` }, '?')
  }
  if (/^https?:\/\//i.test(icon)) {
    return h('img', {
      src: icon,
      style: `${baseStyle};object-fit:contain`,
      onError: (e: Event) => {
        const img = e.target as HTMLImageElement
        img.style.display = 'none'
      },
    })
  }
  if (icon.startsWith('upload:')) {
    return h('img', {
      src: '/uploads/' + icon.slice('upload:'.length),
      style: `${baseStyle};object-fit:contain`,
      onError: (e: Event) => {
        const img = e.target as HTMLImageElement
        img.style.display = 'none'
      },
    })
  }
  return h('div', { style: `${baseStyle};color:rgba(255,193,77,0.7);font-size:10px` }, '?')
}

const dropdownOptions = computed<DropdownOption[]>(() => {
  return props.engines.map((e) => ({
    key: e.id,
    label: e.name + (e.is_default ? ' · 默认' : '') + (activeEngine.value?.id === e.id ? ' ✓' : ''),
    icon: () => renderEngineIcon(e.icon, 16),
  }))
})

const triggerTitle = computed(() => {
  if (!activeEngine.value) return '没有可用的搜索引擎'
  return `搜索引擎：${activeEngine.value.name}（点击切换）`
})
</script>

<template>
  <div class="header-search">
    <NInput
      :value="query"
      placeholder="过滤卡片 / 回车跳搜索引擎"
      clearable
      size="small"
      class="header-search__input"
      @update:value="onInput"
      @keydown.enter="handleEnter"
    />
    <NDropdown
      :options="dropdownOptions"
      placement="bottom-end"
      trigger="click"
      :disabled="engines.length === 0"
      @select="handleEngineSelect"
    >
      <NButton
        size="small"
        circle
        :title="triggerTitle"
        :disabled="!activeEngine"
        class="header-search__trigger"
      >
        <component v-if="activeEngine" :is="renderEngineIcon(activeEngine.icon, 16)" />
        <span v-else style="font-size: 11px; opacity: 0.5">?</span>
      </NButton>
    </NDropdown>
  </div>
</template>

<style scoped>
.header-search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.header-search__input {
  width: 280px;
  max-width: 100%;
}
@media (max-width: 768px) {
  .header-search__input {
    width: 100px;
  }
  /* v0.2.10: Mobile shrinks NInput height + font to match circle buttons
     (~28px) and reduce visual weight in cramped header. --n-height is
     NaiveUI's internal CSS var pairing with height to scale NInput overall. */
  .header-search :deep(.n-input__input-el) {
    height: 24px;
    font-size: 12px;
  }
  .header-search :deep(.n-input) {
    --n-height: 24px;
  }
}
@media (max-width: 480px) {
  .header-search__input {
    width: 80px;
  }
}
</style>
