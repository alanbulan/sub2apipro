import { computed, ref } from 'vue'

export type AppThemeId =
  | 'minimalism'
  | 'apple'
  | 'notion'
  | 'bento'
  | 'glass'
  | 'neoBrutalism'
  | 'swiss'
  | 'editorial'
  | 'skeuomorphism'
  | 'flat'
  | 'material'
  | 'neumorphism'
  | 'saas'
  | 'enterprise'
  | 'commerce'
  | 'darkTech'
  | 'playful'
  | 'organic'
  | 'y2k'
  | 'memphis'
  | 'terminal'
  | 'wabiSabi'
  | 'bauhaus'
  | 'artDeco'

export type ColorMode = 'system' | 'light' | 'dark'
export type FontPresetId = 'system' | 'humanist' | 'serif' | 'mono'
export type RadiusPresetId = 'sharp' | 'soft' | 'round'
export type DensityPresetId = 'compact' | 'comfortable' | 'spacious'
export type SidebarStyleId = 'default' | 'floating' | 'compact'
export type LayoutPresetId = 'default' | 'wide' | 'centered'
export type ContentWidthPresetId = 'standard' | 'wide' | 'full'

export const APP_THEME_STORAGE_KEY = 'app-theme'
export const COLOR_MODE_STORAGE_KEY = 'theme'
export const FONT_PRESET_STORAGE_KEY = 'font-preset'
export const RADIUS_PRESET_STORAGE_KEY = 'radius-preset'
export const DENSITY_STORAGE_KEY = 'density'
export const SIDEBAR_STYLE_STORAGE_KEY = 'sidebar-style'
export const LAYOUT_STORAGE_KEY = 'layout'
export const CONTENT_WIDTH_STORAGE_KEY = 'content-width'

export type AppThemeDefinition = {
  id: AppThemeId
  labelKey: string
  stylesheet: string
}

// Every design topic has its own stylesheet. Keep this list in the same order
// as the design drawer so the 24 local theme files remain easy to audit.
const appThemeDefinitions: readonly AppThemeDefinition[] = [
  { id: 'minimalism', labelKey: 'nav.themes.minimalism', stylesheet: 'minimalism.css' },
  { id: 'apple', labelKey: 'nav.themes.apple', stylesheet: 'apple.css' },
  { id: 'notion', labelKey: 'nav.themes.notion', stylesheet: 'notion.css' },
  { id: 'bento', labelKey: 'nav.themes.bento', stylesheet: 'bento.css' },
  { id: 'glass', labelKey: 'nav.themes.glass', stylesheet: 'glass.css' },
  { id: 'neoBrutalism', labelKey: 'nav.themes.neoBrutalism', stylesheet: 'neo-brutalism.css' },
  { id: 'swiss', labelKey: 'nav.themes.swiss', stylesheet: 'swiss.css' },
  { id: 'editorial', labelKey: 'nav.themes.editorial', stylesheet: 'editorial.css' },
  { id: 'skeuomorphism', labelKey: 'nav.themes.skeuomorphism', stylesheet: 'skeuomorphism.css' },
  { id: 'flat', labelKey: 'nav.themes.flat', stylesheet: 'flat.css' },
  { id: 'material', labelKey: 'nav.themes.material', stylesheet: 'material.css' },
  { id: 'neumorphism', labelKey: 'nav.themes.neumorphism', stylesheet: 'neumorphism.css' },
  { id: 'saas', labelKey: 'nav.themes.saas', stylesheet: 'saas.css' },
  { id: 'enterprise', labelKey: 'nav.themes.enterprise', stylesheet: 'enterprise.css' },
  { id: 'commerce', labelKey: 'nav.themes.commerce', stylesheet: 'commerce.css' },
  { id: 'darkTech', labelKey: 'nav.themes.darkTech', stylesheet: 'dark-tech.css' },
  { id: 'playful', labelKey: 'nav.themes.playful', stylesheet: 'playful.css' },
  { id: 'organic', labelKey: 'nav.themes.organic', stylesheet: 'organic.css' },
  { id: 'y2k', labelKey: 'nav.themes.y2k', stylesheet: 'y2k.css' },
  { id: 'memphis', labelKey: 'nav.themes.memphis', stylesheet: 'memphis.css' },
  { id: 'terminal', labelKey: 'nav.themes.terminal', stylesheet: 'terminal.css' },
  { id: 'wabiSabi', labelKey: 'nav.themes.wabiSabi', stylesheet: 'wabi-sabi.css' },
  { id: 'bauhaus', labelKey: 'nav.themes.bauhaus', stylesheet: 'bauhaus.css' },
  { id: 'artDeco', labelKey: 'nav.themes.artDeco', stylesheet: 'art-deco.css' }
]

export const appThemes = appThemeDefinitions

const activeTheme = ref<AppThemeId>('minimalism')
const colorMode = ref<ColorMode>('system')
const isDark = ref(false)
const fontPreset = ref<FontPresetId>('system')
const radiusPreset = ref<RadiusPresetId>('soft')
const density = ref<DensityPresetId>('comfortable')
const sidebarStyle = ref<SidebarStyleId>('default')
const layout = ref<LayoutPresetId>('default')
const contentWidth = ref<ContentWidthPresetId>('standard')

let systemColorScheme: MediaQueryList | null = null

function isAppThemeId(value: unknown): value is AppThemeId {
  return appThemes.some((theme) => theme.id === value)
}

function isColorMode(value: unknown): value is ColorMode {
  return value === 'system' || value === 'light' || value === 'dark'
}

function isOneOf<T extends string>(values: readonly T[], value: unknown): value is T {
  return values.includes(value as T)
}

function prefersDarkMode() {
  return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches
}

function resolveDarkMode() {
  isDark.value = colorMode.value === 'dark' || (colorMode.value === 'system' && prefersDarkMode())
}

function syncDocument() {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.dataset.appTheme = activeTheme.value
  root.dataset.fontPreset = fontPreset.value
  root.dataset.radius = radiusPreset.value
  root.dataset.density = density.value
  root.dataset.sidebarStyle = sidebarStyle.value
  root.dataset.layout = layout.value
  root.dataset.contentWidth = contentWidth.value
  root.dataset.colorMode = colorMode.value
  root.classList.toggle('dark', isDark.value)
}

function syncSystemColorSchemeListener() {
  if (systemColorScheme) {
    systemColorScheme.removeEventListener?.('change', handleSystemColorSchemeChange)
  }
  systemColorScheme = typeof window !== 'undefined' ? window.matchMedia('(prefers-color-scheme: dark)') : null
  systemColorScheme?.addEventListener?.('change', handleSystemColorSchemeChange)
}

function handleSystemColorSchemeChange() {
  if (colorMode.value === 'system') {
    resolveDarkMode()
    syncDocument()
  }
}

export function initAppTheme() {
  const savedTheme = localStorage.getItem(APP_THEME_STORAGE_KEY)
  activeTheme.value = isAppThemeId(savedTheme) ? savedTheme : 'minimalism'

  const savedMode = localStorage.getItem(COLOR_MODE_STORAGE_KEY)
  // Older builds only wrote light/dark. Treat a missing value as system so a
  // first visit follows the OS preference while existing choices stay intact.
  colorMode.value = isColorMode(savedMode) ? savedMode : 'system'

  const savedFontPreset = localStorage.getItem(FONT_PRESET_STORAGE_KEY)
  fontPreset.value = isOneOf(['system', 'humanist', 'serif', 'mono'] as const, savedFontPreset) ? savedFontPreset : 'system'
  const savedRadiusPreset = localStorage.getItem(RADIUS_PRESET_STORAGE_KEY)
  radiusPreset.value = isOneOf(['sharp', 'soft', 'round'] as const, savedRadiusPreset) ? savedRadiusPreset : 'soft'
  const savedDensity = localStorage.getItem(DENSITY_STORAGE_KEY)
  density.value = isOneOf(['compact', 'comfortable', 'spacious'] as const, savedDensity) ? savedDensity : 'comfortable'
  const savedSidebarStyle = localStorage.getItem(SIDEBAR_STYLE_STORAGE_KEY)
  sidebarStyle.value = isOneOf(['default', 'floating', 'compact'] as const, savedSidebarStyle) ? savedSidebarStyle : 'default'
  const savedLayout = localStorage.getItem(LAYOUT_STORAGE_KEY)
  layout.value = isOneOf(['default', 'wide', 'centered'] as const, savedLayout) ? savedLayout : 'default'
  const savedContentWidth = localStorage.getItem(CONTENT_WIDTH_STORAGE_KEY)
  contentWidth.value = isOneOf(['standard', 'wide', 'full'] as const, savedContentWidth) ? savedContentWidth : 'standard'

  syncSystemColorSchemeListener()
  resolveDarkMode()
  syncDocument()
}

export function setAppTheme(theme: AppThemeId) {
  activeTheme.value = theme
  localStorage.setItem(APP_THEME_STORAGE_KEY, theme)
  syncDocument()
}

export function setColorMode(mode: ColorMode | boolean) {
  // Boolean support keeps the small public API backwards compatible with the
  // previous theme toggle while the drawer can use the explicit three modes.
  colorMode.value = typeof mode === 'boolean' ? (mode ? 'dark' : 'light') : mode
  localStorage.setItem(COLOR_MODE_STORAGE_KEY, colorMode.value)
  resolveDarkMode()
  syncDocument()
}

export function toggleColorMode() {
  setColorMode(isDark.value ? 'light' : 'dark')
}

function setStoredOption<T extends string>(value: T, state: { value: T }, storageKey: string, dataSync = true) {
  state.value = value
  localStorage.setItem(storageKey, value)
  if (dataSync) syncDocument()
}

export function setFontPreset(value: FontPresetId) {
  setStoredOption(value, fontPreset, FONT_PRESET_STORAGE_KEY)
}

export function setRadiusPreset(value: RadiusPresetId) {
  setStoredOption(value, radiusPreset, RADIUS_PRESET_STORAGE_KEY)
}

export function setDensity(value: DensityPresetId) {
  setStoredOption(value, density, DENSITY_STORAGE_KEY)
}

export function setSidebarStyle(value: SidebarStyleId) {
  setStoredOption(value, sidebarStyle, SIDEBAR_STYLE_STORAGE_KEY)
}

export function setLayout(value: LayoutPresetId) {
  setStoredOption(value, layout, LAYOUT_STORAGE_KEY)
}

export function setContentWidth(value: ContentWidthPresetId) {
  setStoredOption(value, contentWidth, CONTENT_WIDTH_STORAGE_KEY)
}

export function useAppTheme() {
  return {
    activeTheme: computed(() => activeTheme.value),
    colorMode: computed(() => colorMode.value),
    isDark: computed(() => isDark.value),
    fontPreset: computed(() => fontPreset.value),
    radiusPreset: computed(() => radiusPreset.value),
    density: computed(() => density.value),
    sidebarStyle: computed(() => sidebarStyle.value),
    layout: computed(() => layout.value),
    contentWidth: computed(() => contentWidth.value),
    setAppTheme,
    setColorMode,
    toggleColorMode,
    setFontPreset,
    setRadiusPreset,
    setDensity,
    setSidebarStyle,
    setLayout,
    setContentWidth
  }
}
