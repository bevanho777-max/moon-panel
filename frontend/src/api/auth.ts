import { http, type ApiResponse } from './client'

export interface MeInfo {
  initialized: boolean
  authenticated: boolean
  username?: string
  totp_enabled?: boolean
}

export async function getMe(): Promise<MeInfo> {
  const { data } = await http.get<ApiResponse<MeInfo>>('/auth/me')
  return data.data!
}

export async function initAdmin(password: string): Promise<void> {
  await http.post('/auth/init', { password })
}

export interface LoginResult {
  /** True if backend issued a 2FA challenge cookie; client must call verifyTOTP next. */
  needs2FA: boolean
  /** Username on full-success path; empty when needs_2fa=true. */
  username: string
}

export async function login(
  password: string,
  options: { username?: string; rememberMe?: boolean } = {},
): Promise<LoginResult> {
  const { data } = await http.post<ApiResponse<{ username?: string; needs_2fa?: boolean }>>(
    '/auth/login',
    {
      username: options.username ?? 'admin',
      password,
      remember_me: options.rememberMe ?? false,
    },
  )
  return {
    needs2FA: !!data.data?.needs_2fa,
    username: data.data?.username ?? '',
  }
}

export async function logout(): Promise<void> {
  await http.post('/auth/logout')
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<void> {
  await http.put('/auth/password', {
    old_password: oldPassword,
    new_password: newPassword,
  })
}
