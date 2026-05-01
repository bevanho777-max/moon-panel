import { http, type ApiResponse } from './client'

export interface Group {
  id: number
  name: string
  icon: string
  sort: number
  created_at: string
  updated_at: string
}

export interface GroupWritePayload {
  name: string
  icon?: string
  sort?: number
}

export async function listGroups(): Promise<Group[]> {
  const { data } = await http.get<ApiResponse<Group[]>>('/admin/groups')
  return data.data ?? []
}

export async function createGroup(payload: GroupWritePayload): Promise<Group> {
  const { data } = await http.post<ApiResponse<Group>>('/admin/groups', payload)
  return data.data!
}

export async function updateGroup(id: number, payload: GroupWritePayload): Promise<Group> {
  const { data } = await http.put<ApiResponse<Group>>(`/admin/groups/${id}`, payload)
  return data.data!
}

export async function deleteGroup(id: number): Promise<void> {
  await http.delete(`/admin/groups/${id}`)
}

export interface GroupReorderItem {
  id: number
  sort: number
}

export async function reorderGroups(items: GroupReorderItem[]): Promise<void> {
  await http.put('/admin/groups/reorder', { items })
}
