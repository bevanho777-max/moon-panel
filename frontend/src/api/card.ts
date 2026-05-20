import { http, type ApiResponse } from './client'

export interface Card {
  id: number
  group_id: number
  title: string
  description: string
  icon: string
  icon_type: string // deprecated, ignore on frontend
  url_internal: string
  url_external: string
  url_default: '' | 'internal' | 'external'
  open_in_new_tab: boolean
  sort: number
  created_at: string
  updated_at: string
}

export interface CardWritePayload {
  group_id?: number
  title?: string
  description?: string
  icon?: string
  url_internal?: string
  url_external?: string
  url_default?: '' | 'internal' | 'external'
  open_in_new_tab?: boolean
  sort?: number
}

export async function listCards(groupId?: number): Promise<Card[]> {
  const url = groupId ? `/admin/cards?group_id=${groupId}` : '/admin/cards'
  const { data } = await http.get<ApiResponse<Card[]>>(url)
  return data.data ?? []
}

export async function getCard(id: number): Promise<Card> {
  const { data } = await http.get<ApiResponse<Card>>(`/admin/cards/${id}`)
  return data.data!
}

export async function createCard(payload: CardWritePayload): Promise<Card> {
  const { data } = await http.post<ApiResponse<Card>>('/admin/cards', payload)
  return data.data!
}

export async function updateCard(id: number, payload: CardWritePayload): Promise<Card> {
  const { data } = await http.put<ApiResponse<Card>>(`/admin/cards/${id}`, payload)
  return data.data!
}

export async function deleteCard(id: number): Promise<void> {
  await http.delete(`/admin/cards/${id}`)
}

export interface CardReorderItem {
  id: number
  sort: number
  group_id?: number // optional cross-group move
}

export async function reorderCards(items: CardReorderItem[]): Promise<void> {
  await http.put('/admin/cards/reorder', { items })
}
