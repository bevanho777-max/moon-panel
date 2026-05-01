import { http, type ApiResponse } from './client'

export interface AuditLogEntry {
  id: number
  timestamp: string
  actor: string
  action: string
  target_type: string
  target_id: string
  ip: string
  user_agent: string
  status: number
  details: string // JSON-encoded blob; parse client-side for display
  created_at: string
}

export interface AuditLogList {
  items: AuditLogEntry[]
  total: number
  page: number
  size: number
}

export interface AuditLogQuery {
  page?: number
  size?: number
  action?: string
  actor?: string
  from?: string // ISO 8601
  to?: string
}

export async function listAuditLogs(query: AuditLogQuery = {}): Promise<AuditLogList> {
  const { data } = await http.get<ApiResponse<AuditLogList>>('/admin/audit-logs', {
    params: query,
  })
  return data.data!
}
