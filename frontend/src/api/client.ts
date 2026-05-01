import axios from 'axios'

export const http = axios.create({
  baseURL: '/api',
  withCredentials: true,
  timeout: 15_000,
})

export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data?: T
}

export class ApiError extends Error {
  constructor(public status: number, public code: number, message: string) {
    super(message)
  }
}

// Routes that legitimately receive 401 as part of normal flow (e.g.
// /auth/login on wrong password, /auth/me when not logged in returns 200
// with authenticated:false but check anyway). For these the interceptor
// must NOT auto-redirect — the calling code handles the response.
const AUTH_PASSTHROUGH_PATHS = ['/auth/login', '/auth/me', '/auth/init']

function isAuthPassthrough(url: string | undefined): boolean {
  if (!url) return false
  return AUTH_PASSTHROUGH_PATHS.some((p) => url.includes(p))
}

http.interceptors.response.use(
  (resp) => resp,
  (err) => {
    if (err.response) {
      const status = err.response.status as number
      const body = err.response.data as ApiResponse | undefined
      const code = body?.code ?? status
      const msg = body?.msg ?? err.response.statusText
      // Global session-expired handling: any UNEXPECTED 401 (i.e. not on a
      // route that uses 401 as a signal) means the user's session is gone.
      // Redirect once, with reason=expired so the login page can show
      // "会话已过期，请重新登录". Avoid infinite loops if we're already at /login.
      if (status === 401 && !isAuthPassthrough(err.config?.url)) {
        if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
          const redirect = encodeURIComponent(window.location.pathname + window.location.search)
          window.location.replace(`/login?reason=expired&redirect=${redirect}`)
        }
      }
      return Promise.reject(new ApiError(status, code, msg))
    }
    return Promise.reject(new ApiError(0, 0, err.message ?? 'network error'))
  },
)
