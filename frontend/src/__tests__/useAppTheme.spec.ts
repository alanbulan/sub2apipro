import { beforeEach, describe, expect, it, vi } from 'vitest'
import { appThemes, initAppTheme, setAppTheme, useAppTheme } from '@/composables/useAppTheme'

describe('app theme runtime', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.className = ''
  })

  it('initializes the default theme and system color mode', () => {
    window.matchMedia = vi.fn().mockReturnValue({ matches: true }) as never
    initAppTheme()

    expect(document.documentElement.dataset.appTheme).toBe('minimalism')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('persists a selected global theme without changing color mode', () => {
    window.matchMedia = vi.fn().mockReturnValue({ matches: false }) as never
    initAppTheme()
    setAppTheme('neoBrutalism')

    expect(localStorage.getItem('app-theme')).toBe('neoBrutalism')
    expect(useAppTheme().activeTheme.value).toBe('neoBrutalism')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('keeps one local stylesheet for every design topic', () => {
    expect(appThemes).toHaveLength(24)
    expect(new Set(appThemes.map((theme) => theme.id)).size).toBe(24)
    expect(new Set(appThemes.map((theme) => theme.stylesheet)).size).toBe(24)
    expect(appThemes.every((theme) => theme.stylesheet.endsWith('.css'))).toBe(true)
  })
})
