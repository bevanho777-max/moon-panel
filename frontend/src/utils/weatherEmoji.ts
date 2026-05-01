// WMO weather code → emoji. Codes per Open-Meteo / WMO 4677.
// Reference: https://open-meteo.com/en/docs (table at the bottom)

export interface WeatherEmoji {
  emoji: string
  label: string
}

export function weatherEmoji(code: number, isDay = 1): WeatherEmoji {
  if (code === 0) return isDay ? { emoji: '☀️', label: '晴' } : { emoji: '🌙', label: '晴夜' }
  if (code === 1 || code === 2) return isDay ? { emoji: '⛅', label: '多云' } : { emoji: '☁️', label: '多云' }
  if (code === 3) return { emoji: '☁️', label: '阴' }
  if (code === 45 || code === 48) return { emoji: '🌫️', label: '雾' }
  if (code >= 51 && code <= 57) return { emoji: '🌦️', label: '毛毛雨' }
  if (code >= 61 && code <= 67) return { emoji: '🌧️', label: '雨' }
  if (code >= 71 && code <= 77) return { emoji: '❄️', label: '雪' }
  if (code >= 80 && code <= 82) return { emoji: '🌦️', label: '阵雨' }
  if (code === 85 || code === 86) return { emoji: '🌨️', label: '阵雪' }
  if (code === 95) return { emoji: '⛈️', label: '雷雨' }
  if (code === 96 || code === 99) return { emoji: '⛈️', label: '雷雨冰雹' }
  return { emoji: '🌡️', label: '未知' }
}

export function formatTemp(celsius: number, unit: 'C' | 'F'): string {
  if (unit === 'F') {
    const f = celsius * 9 / 5 + 32
    return `${Math.round(f)}°F`
  }
  return `${Math.round(celsius)}°C`
}
