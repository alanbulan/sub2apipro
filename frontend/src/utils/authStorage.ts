export const AUTH_TOKEN_KEY = 'auth_token'
export const AUTH_USER_KEY = 'auth_user'
export const REFRESH_TOKEN_KEY = 'refresh_token'
export const TOKEN_EXPIRES_AT_KEY = 'token_expires_at'

const AUTH_STORAGE_MODE_KEY = 'auth_storage_mode'
const AUTH_STORAGE_KEYS = [
  AUTH_TOKEN_KEY,
  AUTH_USER_KEY,
  REFRESH_TOKEN_KEY,
  TOKEN_EXPIRES_AT_KEY
] as const

export type AuthStorageMode = 'persistent' | 'session'

function storageForMode(mode: AuthStorageMode): Storage {
  return mode === 'persistent' ? localStorage : sessionStorage
}

export function getAuthStorageMode(): AuthStorageMode {
  if (sessionStorage.getItem(AUTH_STORAGE_MODE_KEY) === 'session') {
    return 'session'
  }
  if (localStorage.getItem(AUTH_STORAGE_MODE_KEY) === 'persistent') {
    return 'persistent'
  }

  // Existing installations stored sessions in localStorage without a mode marker.
  if (sessionStorage.getItem(AUTH_TOKEN_KEY) || sessionStorage.getItem(REFRESH_TOKEN_KEY)) {
    return 'session'
  }
  return 'persistent'
}

export function getAuthStorage(): Storage {
  return storageForMode(getAuthStorageMode())
}

export function setAuthStorageMode(rememberMe: boolean): Storage {
  const mode: AuthStorageMode = rememberMe ? 'persistent' : 'session'
  const target = storageForMode(mode)
  const other = storageForMode(mode === 'persistent' ? 'session' : 'persistent')

  target.setItem(AUTH_STORAGE_MODE_KEY, mode)
  other.removeItem(AUTH_STORAGE_MODE_KEY)
  for (const key of AUTH_STORAGE_KEYS) {
    target.removeItem(key)
    other.removeItem(key)
  }

  return target
}

export function getAuthItem(key: string): string | null {
  return getAuthStorage().getItem(key)
}

export function setAuthItem(key: string, value: string): void {
  getAuthStorage().setItem(key, value)
}

export function clearAuthStorage(): void {
  for (const storage of [localStorage, sessionStorage]) {
    for (const key of AUTH_STORAGE_KEYS) {
      storage.removeItem(key)
    }
    storage.removeItem(AUTH_STORAGE_MODE_KEY)
  }
}
