import { beforeEach, describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import {
  AUTH_TOKEN_KEY,
  AUTH_USER_KEY,
  clearAuthStorage,
  getAuthItem,
  setAuthItem,
  setAuthStorageMode
} from '@/utils/authStorage'

describe('Pro remember-me invariants', () => {
  beforeEach(() => {
    clearAuthStorage()
  })

  it('keeps non-remembered authentication in the current browser session', () => {
    setAuthStorageMode(false)
    setAuthItem(AUTH_TOKEN_KEY, 'session-token')
    setAuthItem(AUTH_USER_KEY, '{"id":1}')

    expect(getAuthItem(AUTH_TOKEN_KEY)).toBe('session-token')
    expect(sessionStorage.getItem(AUTH_TOKEN_KEY)).toBe('session-token')
    expect(localStorage.getItem(AUTH_TOKEN_KEY)).toBeNull()
  })

  it('keeps remembered authentication across browser sessions', () => {
    setAuthStorageMode(true)
    setAuthItem(AUTH_TOKEN_KEY, 'persistent-token')

    expect(getAuthItem(AUTH_TOKEN_KEY)).toBe('persistent-token')
    expect(localStorage.getItem(AUTH_TOKEN_KEY)).toBe('persistent-token')
    expect(sessionStorage.getItem(AUTH_TOKEN_KEY)).toBeNull()
  })

  it('keeps the login control and delegates password storage to the browser', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/auth/LoginView.vue'), 'utf8')

    expect(source).toContain('id="remember-me"')
    expect(source).toContain("t('auth.rememberMe')")
    expect(source).toContain('navigator.credentials.store')
    expect(source).not.toMatch(/localStorage\.setItem\([^\n]*password/i)
  })
})
