<script setup lang="ts">
// Bottom status bar (v0.2.1). Visibility is gated purely by CSS via the
// --mp-status-bar-display variable: "moon" theme sets it to "none" so the
// component renders zero pixels even though it's mounted; "risen" sets
// "flex" so it appears.
//
// Why mount on both themes: keeps the data path simple — the component
// fetches /api/site/stats on mount + polls every 5 minutes. Switching
// themes via the picker doesn't unmount/remount, so cached stats stay
// fresh across toggles. The only "cost" of mounting on moon is the
// initial fetch, which costs maybe 200 bytes.

import { onBeforeUnmount, onMounted, ref } from 'vue'
import { getSiteStats, type SiteStats } from '@/api/site'

const stats = ref<SiteStats | null>(null)
let pollTimer: number | null = null

async function load() {
  try {
    stats.value = await getSiteStats()
  } catch {
    // Network / 5xx — leave the bar empty rather than show "—" loud,
    // it's decorative chrome not load-bearing data.
    stats.value = null
  }
}

function fmtUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const m = Math.floor(seconds / 60)
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ${m % 60}m`
  const d = Math.floor(h / 24)
  return `${d}d ${h % 24}h`
}

onMounted(() => {
  load()
  // Poll every 5 minutes. Non-aligned, non-jittered — the panel isn't a
  // metrics tool, fresh-enough is fine.
  pollTimer = window.setInterval(load, 5 * 60 * 1000)
})

onBeforeUnmount(() => {
  if (pollTimer !== null) clearInterval(pollTimer)
})
</script>

<template>
  <div class="status-bar" data-testid="status-bar">
    <div v-if="stats" class="sb__item">
      <span class="sb__label">VERSION</span>
      <span class="sb__value">v{{ stats.version }}</span>
    </div>
    <div v-if="stats" class="sb__item">
      <span class="sb__label">CARDS</span>
      <span class="sb__value">{{ stats.cards_count }}</span>
    </div>
    <div v-if="stats" class="sb__item">
      <span class="sb__label">GROUPS</span>
      <span class="sb__value">{{ stats.groups_count }}</span>
    </div>
    <div v-if="stats" class="sb__item">
      <span class="sb__label">UPTIME</span>
      <span class="sb__value">{{ fmtUptime(stats.uptime_seconds) }}</span>
    </div>
  </div>
</template>

<style scoped>
.status-bar {
  /* CSS-gated visibility: moon → none, risen → flex (set in main.css). */
  display: var(--mp-status-bar-display, none);
  position: fixed;
  bottom: 16px;
  left: 50%;
  transform: translateX(-50%);
  gap: 48px;
  padding: 12px 24px;
  background: rgba(20, 15, 10, 0.55);
  border: 1px solid rgba(212, 175, 122, 0.15);
  border-radius: 6px;
  /* backdrop-filter is acceptable here because the bar's painted area is
     ~30 px tall × ~600 px wide — small enough that the per-frame backdrop
     sample is cheap. The earlier v0.1.6 / v0.1.7 lessons were about
     fullscreen / repeated-element backdrop, not small fixed bars. */
  -webkit-backdrop-filter: blur(8px);
  backdrop-filter: blur(8px);
  z-index: 50;
  pointer-events: none;
}
.sb__item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}
.sb__label {
  font-size: 9px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(212, 175, 122, 0.5);
}
.sb__value {
  font-family: var(--mp-brand-font);
  font-size: 13px;
  color: rgba(245, 235, 215, 0.9);
  font-variant-numeric: tabular-nums;
}

@media (max-width: 768px) {
  .status-bar {
    gap: 24px;
    padding: 10px 16px;
  }
  .sb__label {
    font-size: 8px;
  }
  .sb__value {
    font-size: 12px;
  }
}
</style>
