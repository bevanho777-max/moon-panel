// Icon catalog search — substring + prefix scoring + per-character hit indices
// for highlight rendering. Catalogs (dashboard-icons + lucide) are loaded
// lazily from async chunks so they don't bloat the main bundle.

export type IconSource = 'dashboard' | 'lucide'

export interface IconCandidate {
  source: IconSource
  name: string  // bare name, e.g. "jellyfin" or "shield-check"
}

export interface SearchHit {
  source: IconSource
  name: string
  /** Match score; higher = better. Prefix matches > substring > none. */
  score: number
  /** Character indices in `name` that matched the query (for highlighting). */
  hits: number[]
}

const DASHBOARD_BOOST = 5  // dashboard-icons preferred over lucide for branded apps

let cachedCatalog: IconCandidate[] | null = null
let loadingPromise: Promise<IconCandidate[]> | null = null

/**
 * Lazy-load both icon catalogs as async chunks. First call triggers two
 * dynamic imports (~70 KB raw / ~18 KB gzip total). Subsequent calls reuse
 * the cached array.
 */
export async function loadIconCatalog(): Promise<IconCandidate[]> {
  if (cachedCatalog) return cachedCatalog
  if (!loadingPromise) {
    loadingPromise = (async () => {
      const [dash, luc] = await Promise.all([
        import('@/data/dashboard-icons.json'),
        import('@/data/lucide-icons.json'),
      ])
      const dashboard = (dash.default as string[]).map<IconCandidate>((name) => ({
        source: 'dashboard',
        name,
      }))
      const lucide = (luc.default as string[]).map<IconCandidate>((name) => ({
        source: 'lucide',
        name,
      }))
      cachedCatalog = [...dashboard, ...lucide]
      return cachedCatalog
    })()
  }
  return loadingPromise
}

/**
 * Search the catalog for matches. Substring match with prefix boost; hits
 * indices returned for highlight. Top `limit` results.
 */
export function searchIcons(query: string, catalog: IconCandidate[], limit = 30): SearchHit[] {
  const q = query.trim().toLowerCase()
  if (!q) return []

  const out: SearchHit[] = []
  for (const item of catalog) {
    const lname = item.name.toLowerCase()
    const idx = lname.indexOf(q)
    if (idx < 0) continue
    // Prefix matches (idx === 0) score highest. Substring matches score by
    // how close to the start they are. Dashboard slightly preferred.
    let score = idx === 0 ? 1000 : 500 - idx
    if (item.source === 'dashboard') score += DASHBOARD_BOOST
    // Shorter names with the same query prefix should rank higher
    // (e.g., "plex" matches "plex" tighter than "plexamp").
    score -= lname.length
    out.push({
      source: item.source,
      name: item.name,
      score,
      hits: Array.from({ length: q.length }, (_, i) => idx + i),
    })
  }
  out.sort((a, b) => b.score - a.score)
  return out.slice(0, limit)
}

/**
 * Returns segments for highlighted rendering: alternating plain / matched.
 * Use in template: <span v-for="seg in segs"><mark v-if="seg.match">{{seg.text}}</mark><template v-else>{{seg.text}}</template></span>
 */
export function highlightSegments(name: string, hits: number[]): Array<{ text: string; match: boolean }> {
  if (hits.length === 0) return [{ text: name, match: false }]
  const hitSet = new Set(hits)
  const segs: Array<{ text: string; match: boolean }> = []
  let buf = ''
  let lastMatch = false
  for (let i = 0; i < name.length; i++) {
    const isHit = hitSet.has(i)
    if (i === 0 || isHit === lastMatch) {
      buf += name[i]
    } else {
      segs.push({ text: buf, match: lastMatch })
      buf = name[i]
    }
    lastMatch = isHit
  }
  if (buf) segs.push({ text: buf, match: lastMatch })
  return segs
}
