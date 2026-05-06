<script setup lang="ts">
// Home page hero band: 3-5 city widgets in a row showing time + date + weather.
//
// Lifecycle:
//   - One shared minute-aligned timer ticks `now` (so all widgets re-render in
//     lockstep and we don't drift). Aligns to the next minute boundary on mount.
//   - Weather fetched once per city on mount, then refreshed every 10 min via
//     setInterval. `cities` prop changes (admin updates list) re-trigger the
//     full fetch.
//   - On unmount, both timers are cleared.
//
// Why one shared timer instead of one per widget: with N widgets, N independent
// setInterval(..., 60000) would drift apart and re-render at slightly different
// times. One timer keeps the column-clock effect visually aligned.

import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { City } from '@/utils/citySearch'
import type { TempUnit } from '@/api/panel'
import { getWeather } from '@/api/weather'
import CityWidget from './CityWidget.vue'

const props = defineProps<{
  cities: City[]
  tempUnit: TempUnit
}>()

const now = ref(Date.now())
const weatherMap = ref<Record<string, { temperature_2m: number; weather_code: number; is_day: 0 | 1 } | null>>({})

let minuteTimer: number | null = null
let alignTimer: number | null = null
let weatherTimer: number | null = null

const visibleCities = computed(() => props.cities.slice(0, 5))

function cityKey(c: City): string {
  return `${c.lat.toFixed(2)},${c.lon.toFixed(2)}`
}

async function fetchOne(city: City) {
  try {
    const r = await getWeather(city.lat, city.lon)
    weatherMap.value[cityKey(city)] = {
      temperature_2m: r.current.temperature_2m,
      weather_code: r.current.weather_code,
      is_day: r.current.is_day,
    }
  } catch {
    // Leave entry as-is (null on first load, or last-good on refresh failure).
    if (!(cityKey(city) in weatherMap.value)) {
      weatherMap.value[cityKey(city)] = null
    }
  }
}

async function fetchAll() {
  await Promise.all(visibleCities.value.map(fetchOne))
}

function scheduleMinuteTick() {
  // Align first tick to the next wall-clock minute, then every 60s.
  const msToNextMinute = 60_000 - (Date.now() % 60_000)
  alignTimer = window.setTimeout(() => {
    now.value = Date.now()
    minuteTimer = window.setInterval(() => {
      now.value = Date.now()
    }, 60_000)
  }, msToNextMinute)
}

function startWeatherRefresh() {
  weatherTimer = window.setInterval(fetchAll, 10 * 60 * 1000)
}

// First-mount setup: tick + initial fetch + 10-min refresh.
scheduleMinuteTick()
fetchAll()
startWeatherRefresh()

// If admin edits city list mid-session, refetch (lat/lon may have changed).
watch(
  () => props.cities.map((c) => `${c.lat},${c.lon}`).join('|'),
  () => {
    fetchAll()
  },
)

onBeforeUnmount(() => {
  if (alignTimer !== null) clearTimeout(alignTimer)
  if (minuteTimer !== null) clearInterval(minuteTimer)
  if (weatherTimer !== null) clearInterval(weatherTimer)
})
</script>

<template>
  <section v-if="visibleCities.length > 0" class="hero" data-testid="home-hero">
    <CityWidget
      v-for="c in visibleCities"
      :key="cityKey(c)"
      :city="c"
      :now="now"
      :weather="weatherMap[cityKey(c)] ?? null"
      :unit="tempUnit"
    />
  </section>
</template>

<style scoped>
.hero {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 24px;
}
@media (max-width: 720px) {
  .hero {
    gap: 8px;
  }
  .hero > * {
    flex: 1 1 calc(50% - 8px);
    min-width: 0;
  }
}
@media (max-width: 420px) {
  .hero > * {
    flex: 1 1 100%;
  }
}

/* v0.2.8: PC desktop (≥769px) — center weather cards at fixed width
   instead of stretching across the container. Mobile (≤720px) and
   intermediate (721-768px) layouts unchanged.
   :deep(.cw) targets the CityWidget root class via Vue scoped CSS
   boundary penetration (same pattern as v0.2.7 AuditLog NCard fix). */
@media (min-width: 769px) {
  .hero {
    justify-content: center;
  }
  .hero :deep(.cw) {
    flex: 0 1 280px;
  }
}
</style>
