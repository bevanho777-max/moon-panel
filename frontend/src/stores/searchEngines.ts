import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const KEY_SELECTED = 'moon.search.engine.id'

function loadSelected(): number | null {
  const raw = localStorage.getItem(KEY_SELECTED)
  if (!raw) return null
  const n = Number(raw)
  return Number.isFinite(n) && n > 0 ? n : null
}

/**
 * Selection-only store: tracks which search engine the user has currently
 * picked in the home-page header. Engine list itself comes from
 * /api/public/panel.search_engines (already fetched by Home.vue).
 *
 * If selectedId is null, the panel falls back to the engine flagged
 * is_default in the list. Persisted to localStorage with moon. prefix
 * (see memory/feedback_localstorage_naming.md).
 */
export const useSearchEngineStore = defineStore('searchEngine', () => {
  const selectedId = ref<number | null>(loadSelected())

  watch(selectedId, (v) => {
    if (v === null) localStorage.removeItem(KEY_SELECTED)
    else localStorage.setItem(KEY_SELECTED, String(v))
  })

  function setSelected(id: number | null) {
    selectedId.value = id
  }

  function clear() {
    selectedId.value = null
  }

  return { selectedId, setSelected, clear }
})
