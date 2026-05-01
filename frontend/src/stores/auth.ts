import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as api from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const initialized = ref<boolean | null>(null)
  const authenticated = ref(false)
  const username = ref<string | null>(null)
  const loading = ref(false)

  async function refresh() {
    loading.value = true
    try {
      const me = await api.getMe()
      initialized.value = me.initialized
      authenticated.value = me.authenticated
      username.value = me.username ?? null
    } finally {
      loading.value = false
    }
  }

  async function login(password: string, rememberMe = false): Promise<{ needs2FA: boolean }> {
    const result = await api.login(password, { rememberMe })
    if (!result.needs2FA) {
      // Full success — refresh /me to populate authenticated state.
      await refresh()
    }
    // 2FA branch: cookie state is "challenge only"; caller routes to TOTP step.
    return { needs2FA: result.needs2FA }
  }

  async function completeTOTP() {
    // Called after verifyTOTP — backend swapped challenge cookie for full
    // session. Refresh /me so the store reflects authenticated state.
    await refresh()
  }

  async function initAdmin(password: string) {
    await api.initAdmin(password)
    await refresh()
  }

  async function logout() {
    await api.logout()
    authenticated.value = false
    username.value = null
  }

  return {
    initialized,
    authenticated,
    username,
    loading,
    refresh,
    login,
    completeTOTP,
    initAdmin,
    logout,
  }
})
