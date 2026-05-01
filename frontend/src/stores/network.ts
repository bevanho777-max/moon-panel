import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type NetworkMode = 'auto' | 'internal' | 'external'
export type NetworkOverride = 'internal' | 'external'

const KEY_GLOBAL = 'moon.network.global'
const KEY_OVERRIDES = 'moon.network.overrides'

function loadGlobal(): NetworkMode {
  const raw = localStorage.getItem(KEY_GLOBAL)
  if (raw === 'internal' || raw === 'external' || raw === 'auto') return raw
  return 'auto'
}

function loadOverrides(): Record<number, NetworkOverride> {
  const raw = localStorage.getItem(KEY_OVERRIDES)
  if (!raw) return {}
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object') return parsed
  } catch {
    // corrupted — fall through to empty
  }
  return {}
}

// Network mode store. State is persisted to localStorage with the moon. prefix
// (see memory/feedback_localstorage_naming.md).
//
//   global = 'auto'     → each card uses its own url_default
//   global = 'internal' → all cards force internal
//   global = 'external' → all cards force external
//   overrides[cardId]   → per-card override, beats global
//
// Phase 2.3a only stores the state. UI (NetworkSwitcher, right-click menu) and
// the public homepage rendering land in Phase 2.4.
export const useNetworkStore = defineStore('network', () => {
  const global = ref<NetworkMode>(loadGlobal())
  const overrides = ref<Record<number, NetworkOverride>>(loadOverrides())

  watch(global, (v) => localStorage.setItem(KEY_GLOBAL, v))
  watch(
    overrides,
    (v) => localStorage.setItem(KEY_OVERRIDES, JSON.stringify(v)),
    { deep: true },
  )

  function setGlobal(mode: NetworkMode) {
    global.value = mode
  }

  function setOverride(cardId: number, mode: NetworkOverride) {
    overrides.value = { ...overrides.value, [cardId]: mode }
  }

  function clearOverride(cardId: number) {
    const next = { ...overrides.value }
    delete next[cardId]
    overrides.value = next
  }

  function clearAllOverrides() {
    overrides.value = {}
  }

  return {
    global,
    overrides,
    setGlobal,
    setOverride,
    clearOverride,
    clearAllOverrides,
  }
})
