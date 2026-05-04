<script setup lang="ts">
import { computed } from 'vue'
import type { City } from '@/utils/citySearch'
import { weatherEmoji, formatTemp } from '@/utils/weatherEmoji'
import type { TempUnit } from '@/api/panel'

const props = defineProps<{
  city: City
  /** ISO ms timestamp the parent re-emits each minute. We don't keep our own
   *  timer — one shared timer in HomeHero re-renders all widgets in lockstep. */
  now: number
  /** Open-Meteo "current" payload (or null while loading / on error). */
  weather: { temperature_2m: number; weather_code: number; is_day: 0 | 1 } | null
  unit: TempUnit
}>()

// Format HH:MM and MM/DD in the city's IANA timezone using Intl. We don't need
// per-tick reformatting cost — each tick is once a minute.
const timeStr = computed(() =>
  new Intl.DateTimeFormat('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: props.city.tz,
  }).format(new Date(props.now)),
)

const dateStr = computed(() =>
  new Intl.DateTimeFormat('en-US', {
    month: '2-digit',
    day: '2-digit',
    timeZone: props.city.tz,
  }).format(new Date(props.now)),
)

const emojiInfo = computed(() => {
  if (!props.weather) return { emoji: '·', label: '' }
  return weatherEmoji(props.weather.weather_code, props.weather.is_day)
})

const tempStr = computed(() => {
  if (!props.weather) return '—'
  return formatTemp(props.weather.temperature_2m, props.unit)
})

const cityLabel = computed(() => props.city.name_cn || props.city.name_en)
</script>

<template>
  <div
    class="cw mp-acrylic-light"
    :data-loading="weather === null"
    :title="`${city.name_cn} / ${city.name_en} · ${city.tz}`"
  >
    <!-- v0.2.0: compact two-row layout. Was 4 stacked rows (~190px tall)
         which made hero dominate the home page; now ~90px tall so the
         card grid is the visual focus. Row 1 = identity, Row 2 = state. -->
    <div class="cw__row cw__row--top">
      <span class="cw__name">{{ cityLabel }}</span>
      <span class="cw__time">{{ timeStr }}</span>
    </div>
    <div class="cw__row cw__row--bottom">
      <span class="cw__weather">
        <span class="cw__emoji" :title="emojiInfo.label">{{ emojiInfo.emoji }}</span>
        <span class="cw__temp">{{ tempStr }}</span>
      </span>
      <span class="cw__date">{{ dateStr }}</span>
    </div>
    <!-- Phase 4a polish: pulse bar at the bottom while weather is loading. -->
    <div class="cw__loading-bar" aria-hidden="true" />
  </div>
</template>

<style scoped>
.cw {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  padding: 12px 16px;
  /* v0.2.2: theme-driven background + border. Moon resolves to the
     v0.2.0 hardcoded values; risen swaps to warm-brown / golden. */
  background: var(--mp-weather-bg);
  border: 1px solid var(--mp-weather-border);
  border-radius: 10px;
  min-width: 160px;
  flex: 1 1 0;
  position: relative;
  overflow: hidden;
}
.cw__row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}
.cw__name {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--mp-weather-text);
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cw__time {
  font-size: 1.4rem;
  font-weight: 300;
  font-variant-numeric: tabular-nums;
  color: var(--mp-weather-time);
  line-height: 1;
  flex-shrink: 0;
}
.cw__weather {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.cw__emoji {
  font-size: 1.05rem;
  line-height: 1;
}
.cw__temp {
  font-size: 0.95rem;
  font-variant-numeric: tabular-nums;
  color: var(--mp-weather-temp);
}
.cw__date {
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
  color: var(--mp-weather-date);
}
.cw__loading-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, rgba(91, 141, 239, 0.6), transparent);
  background-size: 200% 100%;
  opacity: 0;
  transition: opacity 200ms ease;
  pointer-events: none;
}
.cw[data-loading='true'] .cw__loading-bar {
  opacity: 1;
  animation: cw-pulse 1.6s ease-in-out infinite;
}
@keyframes cw-pulse {
  0%   { background-position: 200% 0; }
  100% { background-position: -100% 0; }
}
</style>
