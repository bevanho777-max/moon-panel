import { http, type ApiResponse } from './client'
import type { Group } from './group'
import type { Card } from './card'
import type { SearchEngine } from './searchEngine'
import type { City } from '@/utils/citySearch'

export type { SearchEngine }

export type TempUnit = 'C' | 'F'

/** UI customization (Phase 2.5c). All fields nullable — null means "use the
 *  built-in default" (no wallpaper / blur=0 / NaiveUI default blue). */
export interface UISettings {
  /** Wallpaper reference: "builtin:<id>" | "upload:<path>" | null. */
  wallpaper: string | null
  /** Backdrop blur 0–20 px. Stored as integer. */
  wallpaper_blur: number
  /** Theme primary color as #RRGGBB hex, or null for default. */
  theme_primary: string | null
  /** IDs of wallpapers shipped inside the binary. Stable across versions of the
   *  same release; frontend uses these to render the builtin grid without a
   *  separate API call. */
  builtins: string[]
}

/** v0.2.23: network auto-detection settings exposed publicly. */
export interface NetworkSettings {
  /** Admin-configured probe URL. Empty string = frontend auto-samples from
   *  card internal URLs. */
  probe_url: string
}

export interface SiteInfo {
  public_mode: boolean
  /** v0.2.0: admin-customizable site title; falls back to "Moon Panel". */
  title: string
  /** v0.2.1: theme preset id; "moon" (default, current visual) or "risen". */
  theme_preset: 'moon' | 'risen'
  cities: City[]
  temp_unit: TempUnit
  ui: UISettings
  network: NetworkSettings
}

export interface PanelData {
  site: SiteInfo
  groups: Array<Group & { cards?: Card[] }>
  search_engines: SearchEngine[]
}

export async function getPanel(): Promise<PanelData> {
  const { data } = await http.get<ApiResponse<PanelData>>('/public/panel')
  return data.data!
}
