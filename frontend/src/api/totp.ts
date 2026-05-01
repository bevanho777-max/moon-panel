import { http, type ApiResponse } from './client'

export interface EnrollResponse {
  secret: string
  otpauth_url: string
  backup_codes: string[]
}

export async function enrollTOTP(): Promise<EnrollResponse> {
  const { data } = await http.post<ApiResponse<EnrollResponse>>('/auth/2fa/enroll')
  return data.data!
}

export async function confirmTOTP(code: string): Promise<void> {
  await http.post('/auth/2fa/confirm', { code })
}

export async function disableTOTP(password: string, code: string): Promise<void> {
  await http.post('/auth/2fa/disable', { password, code })
}

export async function verifyTOTP(
  code: string,
  options: { isBackup?: boolean; rememberMe?: boolean } = {},
): Promise<void> {
  await http.post('/auth/2fa/verify', {
    code,
    is_backup: options.isBackup ?? false,
    remember_me: options.rememberMe ?? false,
  })
}
