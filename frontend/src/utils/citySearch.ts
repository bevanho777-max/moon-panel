// City catalog search — substring + prefix scoring on both Chinese and English
// names. Catalog (~108 cities, ~10KB) is loaded lazily as an async chunk.

export interface City {
  name_cn: string
  name_en: string
  tz: string
  lat: number
  lon: number
}

export interface CitySearchHit extends City {
  score: number
}

let cached: City[] | null = null
let loadingPromise: Promise<City[]> | null = null

export async function loadCityCatalog(): Promise<City[]> {
  if (cached) return cached
  if (!loadingPromise) {
    loadingPromise = import('@/data/cities.json').then((m) => {
      cached = m.default as City[]
      return cached
    })
  }
  return loadingPromise
}

/**
 * Substring + prefix scoring across name_cn and name_en. Empty query → empty.
 * Higher score = better match. Top `limit` returned.
 */
export function searchCities(query: string, catalog: City[], limit = 12): CitySearchHit[] {
  const q = query.trim().toLowerCase()
  if (!q) return []

  const hits: CitySearchHit[] = []
  for (const c of catalog) {
    const cn = c.name_cn.toLowerCase()
    const en = c.name_en.toLowerCase()
    let score = -1
    // Prefer prefix matches; then any substring; pick the better of cn/en match.
    const cnIdx = cn.indexOf(q)
    const enIdx = en.indexOf(q)
    if (cnIdx === 0 || enIdx === 0) {
      score = 1000 - Math.min(cn.length, en.length)
    } else if (cnIdx > 0 || enIdx > 0) {
      const idx = Math.min(
        cnIdx >= 0 ? cnIdx : Number.POSITIVE_INFINITY,
        enIdx >= 0 ? enIdx : Number.POSITIVE_INFINITY,
      )
      score = 500 - idx - Math.min(cn.length, en.length)
    }
    if (score >= 0) {
      hits.push({ ...c, score })
    }
  }
  hits.sort((a, b) => b.score - a.score)
  return hits.slice(0, limit)
}

/** Validate a custom city entry (for the inline fallback when catalog has no match). */
export function validateCustomCity(c: Partial<City>): string | null {
  if (!c.name_cn?.trim() && !c.name_en?.trim()) return '城市名不能为空（中文名或英文名至少填一个）'
  if (!c.tz?.trim()) return '时区不能为空（如 Asia/Shanghai）'
  if (typeof c.lat !== 'number' || c.lat < -90 || c.lat > 90) return '纬度必须在 -90 到 90 之间'
  if (typeof c.lon !== 'number' || c.lon < -180 || c.lon > 180) return '经度必须在 -180 到 180 之间'
  return null
}
