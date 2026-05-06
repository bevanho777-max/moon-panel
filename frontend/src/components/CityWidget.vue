<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { City } from '@/utils/citySearch'
import { weatherEmoji, formatTemp } from '@/utils/weatherEmoji'
import type { TempUnit } from '@/api/panel'

const props = defineProps<{
  city: City
  /** ISO ms timestamp the parent re-emits each second (for HH:MM:SS display).
   *  We don't keep our own timer — one shared timer in HomeHero re-renders
   *  all widgets in lockstep. Mobile (≤768px) hides time/date via CSS so the
   *  per-second reactivity has no visual cost on small screens. */
  now: number
  /** Open-Meteo "current" payload (or null while loading / on error). */
  weather: { temperature_2m: number; weather_code: number; is_day: 0 | 1 } | null
  unit: TempUnit
}>()

// Format HH:MM:SS and MM/DD in the city's IANA timezone using Intl. PC keeps
// all elements; mobile hides time/date via @media. Per-tick (1Hz) reformat
// cost is < 1ms total across all widgets — negligible.
const timeStr = computed(() =>
  new Intl.DateTimeFormat('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
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

// v0.2.9 路径 A+: City name overflow detection for marquee scroll.
// Only active on mobile (≤768px) where .cw__name has max-width: 60px.
// PC has no max-width so scrollWidth always == clientWidth, no overflow,
// no class added — naturally PC-disabled.
const cityNameRef = ref<HTMLElement | null>(null)
const cityNameOverflow = ref(false)
const cityMarqueeDuration = ref('3s')

function checkCityNameOverflow() {
  const el = cityNameRef.value
  if (!el) return

  const inner = el.querySelector('.cw__name__inner') as HTMLElement | null
  if (!inner) return

  const innerWidth = inner.scrollWidth
  const containerWidth = el.clientWidth
  const isOverflow = innerWidth > containerWidth

  cityNameOverflow.value = isOverflow

  if (isOverflow) {
    // Marquee duration adapts to overflow distance: 0.04s per pixel,
    // clamped 2-8s so very-short and very-long names both stay readable.
    const distance = innerWidth - containerWidth
    const duration = Math.min(8, Math.max(2, distance * 0.04))
    cityMarqueeDuration.value = `${duration}s`
  }
}

let resizeObs: ResizeObserver | null = null

onMounted(() => {
  nextTick(checkCityNameOverflow)

  if (cityNameRef.value && typeof ResizeObserver !== 'undefined') {
    resizeObs = new ResizeObserver(() => {
      nextTick(checkCityNameOverflow)
    })
    resizeObs.observe(cityNameRef.value)
  }
})

// Re-check when city name content changes (language switch / city reorder).
// cityLabel is reactive (computed from props.city), so watch it.
watch(() => cityLabel.value, () => {
  nextTick(checkCityNameOverflow)
})

onBeforeUnmount(() => {
  if (resizeObs) {
    resizeObs.disconnect()
    resizeObs = null
  }
})
</script>

<template>
  <div
    class="cw mp-acrylic-light"
    :data-loading="weather === null"
    :title="`${city.name_cn} / ${city.name_en} · ${city.tz}`"
  >
    <!-- v0.2.9: single-row horizontal layout — name / emoji / temp / date / time.
         Replaces v0.2.0's 2-row stack. Mobile (≤768px) puts 3 cards per row at
         33% width each, so each card stays compact. Emoji kept as a separate
         span to preserve its :title (weather-condition label on hover). -->
    <span
      ref="cityNameRef"
      class="cw__name"
      :class="{ 'cw__name--overflow': cityNameOverflow }"
      :style="{ '--cw-marquee-duration': cityMarqueeDuration }"
    >
      <span class="cw__name__inner">{{ cityLabel }}</span>
    </span>
    <span class="cw__emoji" :title="emojiInfo.label">{{ emojiInfo.emoji }}</span>
    <span class="cw__temp">{{ tempStr }}</span>
    <span class="cw__date">{{ dateStr }}</span>
    <span class="cw__time">{{ timeStr }}</span>
    <!-- Phase 4a polish: pulse bar at the bottom while weather is loading. -->
    <div class="cw__loading-bar" aria-hidden="true" />
  </div>
</template>

<style scoped>
.cw {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
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
.cw__name {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--mp-weather-text);
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 0;
}
.cw__emoji {
  font-size: 1.05rem;
  line-height: 1;
  flex-shrink: 0;
}
.cw__temp {
  font-size: 0.95rem;
  font-variant-numeric: tabular-nums;
  color: var(--mp-weather-temp);
  flex-shrink: 0;
}
.cw__date {
  font-size: 0.95rem;
  font-variant-numeric: tabular-nums;
  color: var(--mp-weather-date);
  flex-shrink: 0;
  margin-left: auto;
}
.cw__time {
  font-size: 0.95rem;
  font-weight: 300;
  font-variant-numeric: tabular-nums;
  color: var(--mp-weather-time);
  line-height: 1;
  flex-shrink: 0;
}
.cw__loading-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  /* v0.2.5: theme-aware loading pulse. Moon resolves --mp-brand-accent to
     the previous brand-blue (#5b8def equivalent); risen resolves to
     warm golden, keeping the loading pulse visually consistent with the
     active theme rather than flashing blue under the golden palette. */
  background: linear-gradient(90deg, transparent, color-mix(in srgb, var(--mp-brand-accent) 60%, transparent), transparent);
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

/* v0.2.9: Mobile (≤768px) hides date+time — phone system already shows
   time/date in status bar, no need to duplicate inside widget. Mobile
   cards display: city + emoji + temperature only. PC (≥769px) keeps
   all 5 elements with seconds-precision time (handled by default
   styles + 1Hz tick from HomeHero). */
@media (max-width: 768px) {
  .cw__date,
  .cw__time {
    display: none;
  }

  /* v0.2.9 路径 A+: Mobile 字号缩 ~15% so 3 elements (city + emoji + temp)
     fit comfortably in ~120px (375 viewport / 3 cards). */
  .cw__name {
    font-size: 0.8rem;
    max-width: 60px;
    overflow: hidden;
    white-space: nowrap;
  }
  .cw__emoji {
    font-size: 0.9rem;
  }
  .cw__temp {
    font-size: 0.8rem;
  }

  /* v0.2.9 路径 A+: When city name content overflows the 60px container
     (e.g., "阿姆斯特丹" 5 chars or "San Francisco" 13 chars), JS adds
     --overflow modifier and the inner span scrolls. Container clips,
     inner span animates translateX. Speed adapts to length via CSS var. */
  .cw__name--overflow .cw__name__inner {
    display: inline-block;
    animation: cw-marquee var(--cw-marquee-duration, 3s) linear infinite;
    padding-right: 16px;
  }

  /* PC hover suspends scroll for readability. Mobile has no hover so
     the animation runs continuously — phones don't stay on a card long
     enough for this to be annoying. */
  .cw:hover .cw__name--overflow .cw__name__inner {
    animation-play-state: paused;
  }
}

/* v0.2.9 路径 A+: marquee keyframes — only used by .cw__name__inner
   when --overflow modifier is present (mobile only). */
@keyframes cw-marquee {
  0%   { transform: translateX(0); }
  100% { transform: translateX(calc(-100% + 60px)); }
}
</style>
