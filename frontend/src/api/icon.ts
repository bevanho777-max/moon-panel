import { http, type ApiResponse } from './client'

export interface IconUploadResponse {
  path: string      // "public/icons/<hash>.<ext>"
  url: string       // "/uploads/public/icons/<hash>.<ext>"
  icon: string      // "upload:public/icons/<hash>.<ext>"  — ready for card.icon
  deduped: boolean
  size: number
  type: string
}

export async function uploadIcon(blob: Blob, filename = 'icon.webp'): Promise<IconUploadResponse> {
  const form = new FormData()
  form.append('file', blob, filename)
  // Don't set Content-Type manually — axios sets it with the multipart boundary.
  const { data } = await http.post<ApiResponse<IconUploadResponse>>(
    '/admin/icons/upload',
    form,
  )
  return data.data!
}

/**
 * Ask the backend to fetch a remote URL and cache it locally. Returns the
 * canonical upload: reference (always replaces the original URL in card.icon).
 * SSRF-protected server-side; private/loopback/metadata IPs rejected.
 */
export async function fetchIconByURL(sourceUrl: string): Promise<IconUploadResponse> {
  const { data } = await http.post<ApiResponse<IconUploadResponse>>(
    '/admin/icons/fetch',
    { url: sourceUrl },
  )
  return data.data!
}
