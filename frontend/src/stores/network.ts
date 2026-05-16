import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'

export type NetworkMode = 'auto' | 'internal' | 'external'
export type NetworkOverride = 'internal' | 'external'

// v0.2.23: auto-detection state.
export type DetectedMode = 'lan' | 'wan' | 'detecting' | 'unknown'
export type ProbeStatus = 'idle' | 'probing' | 'success' | 'failed'
export type SessionOverride = 'lan' | 'wan' | null

// v0.2.23: resolved direction the frontend should serve cards from.
// 'lan' / 'wan' come from auto-detection; 'internal' / 'external' come from
// explicit user choice in NetworkSwitcher. cardUrl.ts maps these to the
// actual URL field (lan/internal → url_internal, wan/external → url_external).
export type EffectiveMode = 'lan' | 'wan' | 'internal' | 'external'

// Minimum shape probe() needs from a card to auto-sample a fallback URL.
export interface ProbeSampleCard {
  url_internal: string
}

const KEY_GLOBAL = 'moon.network.global'
const KEY_OVERRIDES = 'moon.network.overrides'
const KEY_SESSION_OVERRIDE = 'moon-panel.session-override'

const PROBE_TIMEOUT_MS = 1500
const PROBE_INTERVAL_MS = 60_000

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

function loadSessionOverride(): SessionOverride {
  try {
    const raw = sessionStorage.getItem(KEY_SESSION_OVERRIDE)
    if (raw === 'lan' || raw === 'wan') return raw
  } catch {
    // sessionStorage unavailable (iOS private browsing quota 0) — no-op
  }
  return null
}

// v0.2.23 patch-3: <img> probe sidesteps fetch's CORS+redirect minefield.
// Chrome forbids `mode:'no-cors' + redirect:'manual'` (TypeError synchronously),
// and `redirect:'follow'` chases LAN reverse-proxy 301→https into failure.
// An <img> tag has no CORS preflight, follows redirects opaquely, and signals
// load OR error the moment the server speaks — both count as reachable.
// Genuine network unreachability shows up as the timeout firing first.
function probeViaImg(url: string, timeoutMs: number): Promise<boolean> {
  return new Promise((resolve) => {
    const img = new Image()
    let done = false
    const finish = (ok: boolean) => {
      if (done) return
      done = true
      clearTimeout(t)
      img.src = '' // abort in-flight load
      resolve(ok)
    }
    const t = window.setTimeout(() => finish(false), timeoutMs)
    img.onload = () => finish(true)
    img.onerror = () => finish(true)
    // Cache-bust so repeated probes don't short-circuit on the disk cache.
    img.src = url + (url.includes('?') ? '&' : '?') + '_mp_probe=' + Date.now()
  })
}

// Network mode store.
//
// State spans two persistence tiers and one in-memory tier:
//   localStorage    global mode + per-card permanent overrides (v0.2.x)
//   sessionStorage  session-scoped manual override (v0.2.23 D6)
//   in-memory       probe state (detectedMode / probeStatus / lastProbeAt /
//                   probeUrl) — re-derived on every page load
//
//   global = 'auto'     → effective = sessionOverride ?? detectedMode
//                          (v0.2.23: detection replaces v0.2.22's "follow
//                           each card's url_default" semantics. cardUrl.ts
//                           still uses url_default as a last-resort fallback
//                           when the chosen side has no URL.)
//   global = 'internal' → all cards force internal (unchanged from v0.2.x)
//   global = 'external' → all cards force external (unchanged from v0.2.x)
//   overrides[cardId]   → per-card permanent override, beats everything else
export const useNetworkStore = defineStore('network', () => {
  // existing state (v0.2.x)
  const global = ref<NetworkMode>(loadGlobal())
  const overrides = ref<Record<number, NetworkOverride>>(loadOverrides())

  // v0.2.23 probe state
  const detectedMode = ref<DetectedMode>('detecting')
  const probeStatus = ref<ProbeStatus>('idle')
  const lastProbeAt = ref<number | null>(null)
  const sessionOverride = ref<SessionOverride>(loadSessionOverride())
  const probeUrl = ref<string>('')

  watch(global, (v) => localStorage.setItem(KEY_GLOBAL, v))
  watch(
    overrides,
    (v) => localStorage.setItem(KEY_OVERRIDES, JSON.stringify(v)),
    { deep: true },
  )

  // 'detecting' → optimistic 'lan' (keep internal cards clickable during first
  // 1.5s probe window); 'unknown' → conservative 'wan' (no probe ever ran).
  const effectiveMode = computed<EffectiveMode>(() => {
    if (global.value === 'internal') return 'internal'
    if (global.value === 'external') return 'external'
    if (sessionOverride.value) return sessionOverride.value
    if (detectedMode.value === 'detecting') return 'lan'
    if (detectedMode.value === 'unknown') return 'wan'
    return detectedMode.value
  })

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

  function setProbeUrl(url: string) {
    probeUrl.value = url
  }

  function setSessionOverride(value: SessionOverride) {
    sessionOverride.value = value
    try {
      if (value) sessionStorage.setItem(KEY_SESSION_OVERRIDE, value)
      else sessionStorage.removeItem(KEY_SESSION_OVERRIDE)
    } catch {
      // iOS Safari private browsing: setItem throws QuotaExceededError.
      // sessionOverride.value still holds for the lifetime of this tab.
    }
  }

  function clearSessionOverride() {
    setSessionOverride(null)
  }

  // Returns the URL to probe in priority order:
  //   1. Admin-configured network.probe_url
  //   2. First non-empty url_internal across the cards passed in
  //   3. null → caller treats as "no probe possible" → 'wan' conservative
  function pickProbeURL(cards: ProbeSampleCard[]): string | null {
    if (probeUrl.value) return probeUrl.value
    for (const c of cards) {
      if (c.url_internal && c.url_internal.trim() !== '') return c.url_internal
    }
    return null
  }

  // probe runs once. Reachable (img load OR error) → LAN; timeout → WAN.
  // See probeViaImg above for why an <img> tag instead of fetch().
  async function probe(cards: ProbeSampleCard[]): Promise<void> {
    probeStatus.value = 'probing'

    const url = pickProbeURL(cards)
    if (!url) {
      detectedMode.value = 'wan'
      probeStatus.value = 'failed'
      lastProbeAt.value = Date.now()
      return
    }

    try {
      const reachable = await probeViaImg(url, PROBE_TIMEOUT_MS)
      detectedMode.value = reachable ? 'lan' : 'wan'
      probeStatus.value = reachable ? 'success' : 'failed'
    } finally {
      lastProbeAt.value = Date.now()
    }
  }

  // Watcher state lives in the store so any component can start/stop without
  // coordinating local refs. Only one watcher at a time (startWatcher is
  // idempotent on top of itself — calling it twice stops the old one first).
  let intervalId: number | null = null
  let currentCards: ProbeSampleCard[] = []
  let onlineHandler: (() => void) | null = null
  let offlineHandler: (() => void) | null = null
  let visHandler: (() => void) | null = null

  function startInterval() {
    if (intervalId !== null) return
    intervalId = window.setInterval(() => {
      void probe(currentCards)
    }, PROBE_INTERVAL_MS)
  }

  function stopInterval() {
    if (intervalId === null) return
    window.clearInterval(intervalId)
    intervalId = null
  }

  function startWatcher(cards: ProbeSampleCard[]) {
    stopWatcher() // re-entrant safety
    currentCards = cards
    void probe(currentCards)

    onlineHandler = () => {
      void probe(currentCards)
    }
    offlineHandler = () => {
      detectedMode.value = 'wan'
      probeStatus.value = 'failed'
      lastProbeAt.value = Date.now()
    }
    visHandler = () => {
      if (document.hidden) {
        stopInterval()
      } else {
        void probe(currentCards)
        startInterval()
      }
    }
    window.addEventListener('online', onlineHandler)
    window.addEventListener('offline', offlineHandler)
    document.addEventListener('visibilitychange', visHandler)
    startInterval()
  }

  function stopWatcher() {
    stopInterval()
    if (onlineHandler) window.removeEventListener('online', onlineHandler)
    if (offlineHandler) window.removeEventListener('offline', offlineHandler)
    if (visHandler) document.removeEventListener('visibilitychange', visHandler)
    onlineHandler = null
    offlineHandler = null
    visHandler = null
    currentCards = []
  }

  return {
    // existing (v0.2.x)
    global,
    overrides,
    setGlobal,
    setOverride,
    clearOverride,
    clearAllOverrides,
    // v0.2.23 — auto detection
    detectedMode,
    probeStatus,
    lastProbeAt,
    sessionOverride,
    probeUrl,
    effectiveMode,
    setProbeUrl,
    setSessionOverride,
    clearSessionOverride,
    probe,
    startWatcher,
    stopWatcher,
  }
})
