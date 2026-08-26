import { computed, ref } from 'vue'

export type AppThemeId = 'minimalism' | 'neoBrutalism' | 'apple' | 'notion' | 'wabiSabi'

export const APP_THEME_STORAGE_KEY = 'app-theme'
export const COLOR_MODE_STORAGE_KEY = 'theme'
type AppThemeDefinition = {
  id: AppThemeId
  labelKey: string
  stylesheet: string
}

const appThemeDefinitions: readonly AppThemeDefinition[] = [
  {
    id: 'minimalism',
    labelKey: 'nav.themes.minimalism',
    stylesheet: 'minimalism.css'
  },
  {
    id: 'neoBrutalism',
    labelKey: 'nav.themes.neoBrutalism',
    stylesheet: 'neo-brutalism.css'
  },
  {
    id: 'apple',
    labelKey: 'nav.themes.apple',
    stylesheet: 'apple.css'
  },
  {
    id: 'notion',
    labelKey: 'nav.themes.notion',
    stylesheet: 'notion.css'
  },
  {
    id: 'wabiSabi',
    labelKey: 'nav.themes.wabiSabi',
    stylesheet: 'wabi-sabi.css'
  }
]

export const appThemes = appThemeDefinitions

const activeTheme = ref<AppThemeId>('minimalism')
const isDark = ref(false)

function isAppThemeId(value: unknown): value is AppThemeId {
  return appThemes.some((theme) => theme.id === value)
}

function syncDocument() {
  const root = document.documentElement
  root.dataset.appTheme = activeTheme.value
  root.classList.toggle('dark', isDark.value)
}

export function initAppTheme() {
  const savedTheme = localStorage.getItem(APP_THEME_STORAGE_KEY)
  activeTheme.value = isAppThemeId(savedTheme) ? savedTheme : 'minimalism'

  const savedMode = localStorage.getItem(COLOR_MODE_STORAGE_KEY)
  isDark.value =
    savedMode === 'dark' ||
    (!savedMode && window.matchMedia('(prefers-color-scheme: dark)').matches)
  syncDocument()
}

export function setAppTheme(theme: AppThemeId) {
  activeTheme.value = theme
  localStorage.setItem(APP_THEME_STORAGE_KEY, theme)
  syncDocument()
}

export function setColorMode(dark: boolean) {
  isDark.value = dark
  localStorage.setItem(COLOR_MODE_STORAGE_KEY, dark ? 'dark' : 'light')
  syncDocument()
}

export function toggleColorMode() {
  setColorMode(!isDark.value)
}

export function useAppTheme() {
  return {
    activeTheme: computed(() => activeTheme.value),
    isDark: computed(() => isDark.value),
    setAppTheme,
    setColorMode,
    toggleColorMode
  }
}
