import { http, type ApiResponse } from './client'

export interface SearchEngine {
  id: number
  name: string
  url_template: string
  icon: string
  is_default: boolean
  sort: number
  created_at: string
  updated_at: string
}

export interface SearchEngineWritePayload {
  name?: string
  url_template?: string
  icon?: string
  is_default?: boolean
  sort?: number
}

export async function listSearchEngines(): Promise<SearchEngine[]> {
  const { data } = await http.get<ApiResponse<SearchEngine[]>>('/admin/search-engines')
  return data.data ?? []
}

export async function createSearchEngine(payload: SearchEngineWritePayload): Promise<SearchEngine> {
  const { data } = await http.post<ApiResponse<SearchEngine>>('/admin/search-engines', payload)
  return data.data!
}

export async function updateSearchEngine(id: number, payload: SearchEngineWritePayload): Promise<SearchEngine> {
  const { data } = await http.put<ApiResponse<SearchEngine>>(`/admin/search-engines/${id}`, payload)
  return data.data!
}

export async function deleteSearchEngine(id: number): Promise<void> {
  await http.delete(`/admin/search-engines/${id}`)
}

export interface SearchEngineReorderItem {
  id: number
  sort: number
}

// v0.2.16 P0 b: batch reorder for inline drag UX. Mirror reorderCards / reorderGroups.
export async function reorderSearchEngines(items: SearchEngineReorderItem[]): Promise<void> {
  await http.put('/admin/search-engines/reorder', { items })
}

export interface RestoreBuiltinsResult {
  /** Names of the engines that were actually inserted (empty = nothing missing). */
  added: string[]
  /** Full engine list after the restore, already sorted. */
  engines: SearchEngine[]
}

/**
 * v0.2.29: re-add builtin engines the user has deleted. Additive only — engines
 * that still exist (matched by name) are left untouched, so this never clobbers
 * a customized entry.
 */
export async function restoreBuiltins(): Promise<RestoreBuiltinsResult> {
  const { data } = await http.post<ApiResponse<RestoreBuiltinsResult>>(
    '/admin/search-engines/restore-builtins',
  )
  return data.data ?? { added: [], engines: [] }
}

/**
 * Substitute the query into a URL template. Supports both `{q}` and `{query}`
 * placeholders (admins can use either; same string is replaced for both).
 */
export function buildSearchURL(template: string, query: string): string {
  const encoded = encodeURIComponent(query)
  return template.replace(/\{q(?:uery)?\}/g, encoded)
}
