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
export type ColorPresetId = 'theme' | 'indigo' | 'teal' | 'violet' | 'rose' | 'amber' | 'graphite'
export type FontPresetId = 'system' | 'humanist' | 'serif' | 'mono'
export type RadiusPresetId = 'sharp' | 'soft' | 'round'
export type DensityPresetId = 'compact' | 'comfortable' | 'spacious'
export type SidebarStyleId = 'default' | 'floating' | 'compact'
export type LayoutPresetId = 'default' | 'wide' | 'centered'
export type ContentWidthPresetId = 'standard' | 'wide' | 'full'

export const APP_THEME_STORAGE_KEY = 'app-theme'
export const COLOR_MODE_STORAGE_KEY = 'theme'
export const COLOR_PRESET_STORAGE_KEY = 'color-preset'
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
  source: string
  preview: string
}

// The list mirrors VibeHub's design topic page one-for-one. Every entry has a
// matching stylesheet imported by style.css; keeping the source URL here makes
// the mapping auditable instead of hiding it in component markup.
const appThemeDefinitions: readonly AppThemeDefinition[] = [
  { id: 'minimalism', labelKey: 'nav.themes.minimalism', stylesheet: 'minimalism.css', source: 'https://vibe-hub.org/style-minimal', preview: 'linear-gradient(135deg, #14b8a6 0 48%, #f8fafc 48% 100%)' },
  { id: 'apple', labelKey: 'nav.themes.apple', stylesheet: 'apple.css', source: 'https://vibe-hub.org/style-apple', preview: 'linear-gradient(135deg, #007aff 0 48%, #f5f5f7 48% 100%)' },
  { id: 'notion', labelKey: 'nav.themes.notion', stylesheet: 'notion.css', source: 'https://vibe-hub.org/style-notion', preview: 'linear-gradient(135deg, #2383e2 0 48%, #f7f6f3 48% 100%)' },
  { id: 'bento', labelKey: 'nav.themes.bento', stylesheet: 'bento.css', source: 'https://vibe-hub.org/style-bento', preview: 'linear-gradient(135deg, #6366f1 0 48%, #eef2ff 48% 100%)' },
  { id: 'glass', labelKey: 'nav.themes.glass', stylesheet: 'glass.css', source: 'https://vibe-hub.org/style-glass', preview: 'linear-gradient(135deg, #0ea5e9 0 48%, #dbeafe 48% 100%)' },
  { id: 'neoBrutalism', labelKey: 'nav.themes.neoBrutalism', stylesheet: 'neo-brutalism.css', source: 'https://vibe-hub.org/style-brutalism', preview: 'linear-gradient(135deg, #facc15 0 48%, #ff3ea5 48% 100%)' },
  { id: 'swiss', labelKey: 'nav.themes.swiss', stylesheet: 'swiss.css', source: 'https://vibe-hub.org/style-swiss', preview: 'linear-gradient(135deg, #e11d48 0 48%, #f8fafc 48% 100%)' },
  { id: 'editorial', labelKey: 'nav.themes.editorial', stylesheet: 'editorial.css', source: 'https://vibe-hub.org/style-editorial', preview: 'linear-gradient(135deg, #7c2d12 0 48%, #fff7ed 48% 100%)' },
  { id: 'skeuomorphism', labelKey: 'nav.themes.skeuomorphism', stylesheet: 'skeuomorphism.css', source: 'https://vibe-hub.org/style-skeuomorphism', preview: 'linear-gradient(135deg, #4b5563 0 48%, #e5e7eb 48% 100%)' },
  { id: 'flat', labelKey: 'nav.themes.flat', stylesheet: 'flat.css', source: 'https://vibe-hub.org/style-flat', preview: 'linear-gradient(135deg, #2563eb 0 48%, #f1f5f9 48% 100%)' },
  { id: 'material', labelKey: 'nav.themes.material', stylesheet: 'material.css', source: 'https://vibe-hub.org/style-material', preview: 'linear-gradient(135deg, #6750a4 0 48%, #fef7ff 48% 100%)' },
  { id: 'neumorphism', labelKey: 'nav.themes.neumorphism', stylesheet: 'neumorphism.css', source: 'https://vibe-hub.org/style-neumorphism', preview: 'linear-gradient(135deg, #64748b 0 48%, #e2e8f0 48% 100%)' },
  { id: 'saas', labelKey: 'nav.themes.saas', stylesheet: 'saas.css', source: 'https://vibe-hub.org/style-saas', preview: 'linear-gradient(135deg, #4f46e5 0 48%, #eef2ff 48% 100%)' },
  { id: 'enterprise', labelKey: 'nav.themes.enterprise', stylesheet: 'enterprise.css', source: 'https://vibe-hub.org/style-enterprise', preview: 'linear-gradient(135deg, #1d4ed8 0 48%, #eff6ff 48% 100%)' },
  { id: 'commerce', labelKey: 'nav.themes.commerce', stylesheet: 'commerce.css', source: 'https://vibe-hub.org/style-commerce', preview: 'linear-gradient(135deg, #c2410c 0 48%, #fff7ed 48% 100%)' },
  { id: 'darkTech', labelKey: 'nav.themes.darkTech', stylesheet: 'dark-tech.css', source: 'https://vibe-hub.org/style-dark-tech', preview: 'linear-gradient(135deg, #22d3ee 0 48%, #0f172a 48% 100%)' },
  { id: 'playful', labelKey: 'nav.themes.playful', stylesheet: 'playful.css', source: 'https://vibe-hub.org/style-playful', preview: 'linear-gradient(135deg, #ec4899 0 48%, #fef3c7 48% 100%)' },
  { id: 'organic', labelKey: 'nav.themes.organic', stylesheet: 'organic.css', source: 'https://vibe-hub.org/style-organic', preview: 'linear-gradient(135deg, #4d7c0f 0 48%, #f7fee7 48% 100%)' },
  { id: 'y2k', labelKey: 'nav.themes.y2k', stylesheet: 'y2k.css', source: 'https://vibe-hub.org/style-y2k', preview: 'linear-gradient(135deg, #8b5cf6 0 48%, #e0f2fe 48% 100%)' },
  { id: 'memphis', labelKey: 'nav.themes.memphis', stylesheet: 'memphis.css', source: 'https://vibe-hub.org/style-memphis', preview: 'linear-gradient(135deg, #f43f5e 0 48%, #fef08a 48% 100%)' },
  { id: 'terminal', labelKey: 'nav.themes.terminal', stylesheet: 'terminal.css', source: 'https://vibe-hub.org/style-terminal', preview: 'linear-gradient(135deg, #22c55e 0 48%, #052e16 48% 100%)' },
  { id: 'wabiSabi', labelKey: 'nav.themes.wabiSabi', stylesheet: 'wabi-sabi.css', source: 'https://vibe-hub.org/style-wabisabi', preview: 'linear-gradient(135deg, #9a7b4f 0 48%, #f4eee3 48% 100%)' },
  { id: 'bauhaus', labelKey: 'nav.themes.bauhaus', stylesheet: 'bauhaus.css', source: 'https://vibe-hub.org/style-bauhaus', preview: 'linear-gradient(135deg, #dc2626 0 48%, #fef3c7 48% 100%)' },
  { id: 'artDeco', labelKey: 'nav.themes.artDeco', stylesheet: 'art-deco.css', source: 'https://vibe-hub.org/style-art-deco', preview: 'linear-gradient(135deg, #b08d57 0 48%, #171717 48% 100%)' }
]

export const appThemes = appThemeDefinitions

const activeTheme = ref<AppThemeId>('minimalism')
const colorMode = ref<ColorMode>('system')
const isDark = ref(false)
const colorPreset = ref<ColorPresetId>('theme')
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
  root.dataset.colorPreset = colorPreset.value
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

  const savedColorPreset = localStorage.getItem(COLOR_PRESET_STORAGE_KEY)
  colorPreset.value = isOneOf(['theme', 'indigo', 'teal', 'violet', 'rose', 'amber', 'graphite'] as const, savedColorPreset)
    ? savedColorPreset
    : 'theme'
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

export function setColorPreset(value: ColorPresetId) {
  setStoredOption(value, colorPreset, COLOR_PRESET_STORAGE_KEY)
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
    colorPreset: computed(() => colorPreset.value),
    fontPreset: computed(() => fontPreset.value),
    radiusPreset: computed(() => radiusPreset.value),
    density: computed(() => density.value),
    sidebarStyle: computed(() => sidebarStyle.value),
    layout: computed(() => layout.value),
    contentWidth: computed(() => contentWidth.value),
    setAppTheme,
    setColorMode,
    toggleColorMode,
    setColorPreset,
    setFontPreset,
    setRadiusPreset,
    setDensity,
    setSidebarStyle,
    setLayout,
    setContentWidth
  }
}
