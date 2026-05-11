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
/* v0.2.20 P0 a: 弃 flex, 改 CSS Grid 显式列数 (Bevan backlog 反馈 PC 3-up center,
   Mobile 2-up). 5 cities 时第 2/3 行 cards Grid 默认左对齐. max-width 900px ≈
   3×280px cards + 2×12px gap + buffer, margin auto 居中. */
.hero {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  max-width: 900px;
  margin: 0 auto 24px;
}

@media (max-width: 768px) {
  .hero {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    max-width: none;
  }
}
</style>
