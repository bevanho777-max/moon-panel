import { http, type ApiResponse } from './client'

export interface TrustedIPEntry {
  cidr: string
  note?: string
  added_at: string
}

export interface LockedIPItem {
  ip: string
  source: 'login' | 'totp'
  failures: number
  remaining_seconds: number
  locked_until: string
}

export async function listTrustedIPs(): Promise<TrustedIPEntry[]> {
  const { data } = await http.get<ApiResponse<{ items: TrustedIPEntry[] }>>(
    '/admin/security/trusted-ips',
  )
  return data.data?.items ?? []
}

export async function addTrustedIP(cidr: string, note?: string): Promise<void> {
  await http.post('/admin/security/trusted-ips', { cidr, note })
}

export async function deleteTrustedIP(cidr: string): Promise<void> {
  // Path param needs encoding because CIDR contains '/'.
  await http.delete(`/admin/security/trusted-ips/${encodeURIComponent(cidr)}`)
}

export async function listLockedIPs(): Promise<LockedIPItem[]> {
  const { data } = await http.get<ApiResponse<{ items: LockedIPItem[] }>>(
    '/admin/security/locked-ips',
  )
  return data.data?.items ?? []
}

export async function unlockIP(ip: string, source: 'login' | 'totp'): Promise<void> {
  await http.post('/admin/security/unlock', { ip, source })
}
