// Public site stats — feeds the bottom status bar in the "risen" theme.
// Backend endpoint is GET /api/site/stats (no auth, light, polled by the
// status bar every ~5 minutes — well below the IP rate limit on the
// weather endpoint, since this isn't even rate-limited).

import { http, type ApiResponse } from './client'

export interface SiteStats {
  version: string
  cards_count: number
  groups_count: number
  uptime_seconds: number
}

export async function getSiteStats(): Promise<SiteStats> {
  const { data } = await http.get<ApiResponse<SiteStats>>('/site/stats')
  return data.data!
}
