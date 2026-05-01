import { http, type ApiResponse } from './client'

/** Settings store is a flat key→value map. Phase 3b-1 doesn't seed any keys;
 * future phases populate keys like `site.title`, `ui.theme`, etc. */
export type SettingsMap = Record<string, string>

export async function getSettings(): Promise<SettingsMap> {
  const { data } = await http.get<ApiResponse<SettingsMap>>('/admin/settings')
  return data.data ?? {}
}

/**
 * Batch upsert. Send only the keys you want to change — other keys remain
 * untouched. Send empty string as value to clear a key (server keeps the row;
 * Phase 4+ may add explicit DELETE).
 */
export async function updateSettings(payload: SettingsMap): Promise<{ updated: number }> {
  const { data } = await http.put<ApiResponse<{ updated: number }>>('/admin/settings', payload)
  return data.data!
}
