import { http, type ApiResponse } from './client'

export interface RestoreResult {
  groups: number
  cards: number
  engines: number
  settings: number
  uploads_restored: number
}

/**
 * Trigger JSON download. Backend sets Content-Disposition; we just navigate
 * to the URL and the browser downloads. Cookies are sent automatically.
 */
export function exportBackupJSON(): void {
  const a = document.createElement('a')
  a.href = '/api/admin/backup'
  a.click()
}

export function exportBackupZip(): void {
  const a = document.createElement('a')
  a.href = '/api/admin/backup/zip'
  a.click()
}

export async function restoreBackup(file: File): Promise<RestoreResult> {
  const fd = new FormData()
  fd.append('backup', file)
  const { data } = await http.post<ApiResponse<RestoreResult>>(
    '/admin/backup/restore',
    fd,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return data.data!
}
