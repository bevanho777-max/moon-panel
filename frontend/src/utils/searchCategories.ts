import type { SearchEngine } from '@/api/searchEngine'

/**
 * Category display order. Mirrors the backend's valid set
 * (api.SearchEngineCategories in backend/internal/api/search_engine.go) —
 * adding or renaming a category means changing both sides.
 */
export const CATEGORY_ORDER = ['web', 'image', 'music', 'video'] as const

export type SearchCategory = (typeof CATEGORY_ORDER)[number]

export const CATEGORY_LABELS: Record<string, string> = {
  web: '网页',
  image: '图片',
  music: '音乐',
  video: '影视',
}

/** Bucket for rows carrying a category the frontend doesn't know about. */
export const OTHER_CATEGORY_KEY = 'other'
export const OTHER_CATEGORY_LABEL = '其它'

export function categoryLabel(key: string): string {
  return CATEGORY_LABELS[key] ?? OTHER_CATEGORY_LABEL
}

export interface EngineCategoryGroup {
  key: string
  label: string
  engines: SearchEngine[]
}

function bySortThenId(a: SearchEngine, b: SearchEngine): number {
  return a.sort - b.sort || a.id - b.id
}

/**
 * Group engines for the home-page picker and the admin table.
 *
 * Known categories come first in CATEGORY_ORDER; anything else (an engine
 * written by a newer backend, or a row the boot migration hasn't stamped yet)
 * collapses into a single trailing "其它" group rather than disappearing.
 * Empty groups are omitted so the dropdown has no dead headers.
 */
export function groupEnginesByCategory(engines: SearchEngine[]): EngineCategoryGroup[] {
  const buckets = new Map<string, SearchEngine[]>()
  for (const engine of engines) {
    const key = CATEGORY_LABELS[engine.category] ? engine.category : OTHER_CATEGORY_KEY
    const bucket = buckets.get(key)
    if (bucket) bucket.push(engine)
    else buckets.set(key, [engine])
  }

  const out: EngineCategoryGroup[] = []
  for (const key of CATEGORY_ORDER) {
    const bucket = buckets.get(key)
    if (bucket && bucket.length > 0) {
      out.push({ key, label: CATEGORY_LABELS[key], engines: [...bucket].sort(bySortThenId) })
    }
  }
  const other = buckets.get(OTHER_CATEGORY_KEY)
  if (other && other.length > 0) {
    out.push({
      key: OTHER_CATEGORY_KEY,
      label: OTHER_CATEGORY_LABEL,
      engines: [...other].sort(bySortThenId),
    })
  }
  return out
}
