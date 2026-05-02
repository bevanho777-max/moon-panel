// Version metadata + GitHub Releases helper for the bottom-left badge.
//
// /api/version is the binary's own LDFLAGS-injected version. The GitHub
// Releases list is fetched directly from the public GitHub REST API
// (no auth needed; 60 req/h per IP rate-limit is plenty for "user clicks
// the badge once or twice per session"), cached in localStorage for 30
// minutes so re-opens of the popover don't re-hit the API.

import { http } from './client'

export interface VersionInfo {
  version: string
  build_date: string
  commit: string
}

export async function getVersion(): Promise<VersionInfo> {
  const { data } = await http.get<VersionInfo>('/version')
  return data
}

export interface GitHubRelease {
  tag_name: string
  name: string
  published_at: string
  html_url: string
  body: string
  prerelease: boolean
  draft: boolean
}

const GITHUB_API =
  'https://api.github.com/repos/bevanho777-max/moon-panel/releases'
const CACHE_KEY = 'moon-panel-releases'
const CACHE_TTL_MS = 30 * 60 * 1000 // 30 min

interface CachedReleases {
  fetched_at: number
  releases: GitHubRelease[]
}

/** Return the most recent N stable releases (drafts + prereleases filtered
 *  out). Reads localStorage cache when fresh, otherwise hits api.github.com.
 *  On network/429 errors falls back to stale cache when available; if no
 *  cache and the request fails, throws. */
export async function getRecentReleases(limit = 3): Promise<GitHubRelease[]> {
  const cached = readCache()
  if (cached && Date.now() - cached.fetched_at < CACHE_TTL_MS) {
    return cached.releases.slice(0, limit)
  }
  try {
    const res = await fetch(GITHUB_API, {
      headers: { Accept: 'application/vnd.github+json' },
    })
    if (!res.ok) throw new Error(`GitHub API ${res.status}`)
    const all = (await res.json()) as GitHubRelease[]
    const stable = all.filter((r) => !r.prerelease && !r.draft)
    writeCache(stable)
    return stable.slice(0, limit)
  } catch (err) {
    if (cached) return cached.releases.slice(0, limit)
    throw err
  }
}

function readCache(): CachedReleases | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY)
    if (!raw) return null
    return JSON.parse(raw) as CachedReleases
  } catch {
    return null
  }
}

function writeCache(releases: GitHubRelease[]) {
  try {
    const cached: CachedReleases = { fetched_at: Date.now(), releases }
    localStorage.setItem(CACHE_KEY, JSON.stringify(cached))
  } catch {
    // localStorage quota exceeded / disabled — fail silently. The popover
    // still renders fine on this session; next session takes the network
    // path again.
  }
}
