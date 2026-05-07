<script setup lang="ts">
// Home page hero band: 3-5 city widgets in a row showing time + date + weather.
//
// Lifecycle:
//   - One shared 1/min tick timer updates `now` so all widgets re-render in
//     lockstep (column-clock effect). v0.2.13 reverts v0.2.9's 1/sec rate back
//     to 1/min for HH:MM time display on PC (Bevan daily-use feedback: seconds
//     create visual noise, panel does not need timer-grade precision). Mobile
//     hides time via CSS regardless.
//   - Weather fetched once per city on mount, then refreshed every 10 min via
//     setInterval. `cities` prop changes (admin updates list) re-trigger the
//     full fetch.
//   - On unmount, both timers are cleared.
//
// Why one shared timer instead of one per widget: with N widgets, N independent
// setInterval would drift apart and re-render at slightly different times.
// One timer keeps the column-clock effect visually aligned.

import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
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

let tickTimer: ReturnType<typeof setInterval> | null = null
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

function startWeatherRefresh() {
  weatherTimer = window.setInterval(fetchAll, 10 * 60 * 1000)
}

// First-mount setup: 1/min tick + initial fetch + 10-min refresh. tick uses
// onMounted because it lazy-binds the timer; fetchAll/startWeatherRefresh are
// fire-and-forget at setup-script top level.
onMounted(() => {
  tickTimer = setInterval(() => {
    now.value = Date.now()
  }, 60000)
})
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
  if (tickTimer !== null) clearInterval(tickTimer)
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
/* v0.2.9: Mobile/tablet (≤768px) — 3 cards per row, evenly distributed.
   Replaces v0.2.8's 720/420 split AND v0.2.9 first attempt's single-column
   stacking. Single row keeps weather widgets compact rather than dominating
   mobile screen. Cards inside use single-line 4-element layout (city/temp/
   date/time) so 33% width per card is sufficient. */
@media (max-width: 768px) {
  .hero {
    gap: 6px;
  }
  .hero > * {
    flex: 0 1 calc(33.333% - 4px);
    min-width: 0;
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
