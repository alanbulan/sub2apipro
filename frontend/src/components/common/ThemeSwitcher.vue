<template>
  <div ref="triggerRef" class="theme-switcher" :class="{ 'theme-switcher-sidebar': props.variant === 'sidebar' }">
    <button
      type="button"
      class="theme-trigger"
      :class="[
        props.variant === 'sidebar' ? 'theme-trigger-sidebar' : 'theme-trigger-floating',
        { 'theme-trigger-collapsed': props.collapsed }
      ]"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      aria-controls="appearance-drawer"
      :title="t('nav.appearance')"
      @click.stop="toggleDrawer"
    >
      <Icon name="sparkles" size="sm" aria-hidden="true" />
      <span v-if="props.variant === 'sidebar'" class="theme-trigger-label">
        {{ t('nav.appearance') }}
      </span>
    </button>
  </div>

  <Teleport to="body">
    <Transition name="theme-drawer">
      <div v-if="isOpen" class="theme-drawer-layer" @click.self="closeDrawer">
        <aside
          id="appearance-drawer"
          ref="drawerRef"
          class="theme-drawer-panel"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="drawerTitleId"
          @click.stop
        >
          <header class="theme-drawer-header">
            <div class="theme-drawer-heading">
              <span class="theme-drawer-icon"><Icon name="sparkles" size="md" aria-hidden="true" /></span>
              <div>
                <h2 :id="drawerTitleId" class="theme-drawer-title">{{ t('nav.appearance') }}</h2>
                <p class="theme-drawer-subtitle">{{ t('nav.appearanceHint') }}</p>
              </div>
            </div>
            <button
              type="button"
              class="theme-drawer-close"
              :aria-label="t('common.close')"
              @click="closeDrawer"
            >
              <Icon name="x" size="md" aria-hidden="true" />
            </button>
          </header>

          <div class="theme-drawer-body">
            <section class="theme-setting-section">
              <div class="theme-setting-heading">
                <h3>{{ t('nav.colorMode') }}</h3>
              </div>
              <div class="theme-mode-grid">
                <button
                  v-for="option in modeOptions"
                  :key="option.value"
                  type="button"
                  class="theme-choice"
                  :class="{ 'theme-choice-active': colorMode === option.value }"
                  @click="setColorMode(option.value)"
                >
                  <Icon :name="option.icon" size="sm" aria-hidden="true" />
                  <span>{{ t(option.labelKey) }}</span>
                </button>
              </div>
            </section>

            <section class="theme-setting-section theme-setting-section-themes">
              <div class="theme-setting-heading">
                <div>
                  <h3>{{ t('nav.theme') }}</h3>
                  <p>{{ t('nav.themeCount', { count: appThemes.length }) }}</p>
                </div>
              </div>
              <div class="theme-card-grid">
                <button
                  v-for="theme in appThemes"
                  :key="theme.id"
                  type="button"
                  class="theme-card"
                  :class="{ 'theme-card-active': theme.id === activeTheme }"
                  :aria-pressed="theme.id === activeTheme"
                  @click="setAppTheme(theme.id)"
                >
                  <ThemePreview :theme-id="theme.id" />
                  <span class="theme-card-footer">
                    <span class="theme-card-name">{{ t(theme.labelKey) }}</span>
                    <Icon v-if="theme.id === activeTheme" name="checkCircle" size="sm" class="theme-card-check" aria-hidden="true" />
                  </span>
                </button>
              </div>
            </section>

            <section class="theme-setting-section theme-setting-grid-section">
              <div class="theme-setting-control">
                <div class="theme-setting-heading"><h3>{{ t('nav.fontPreset') }}</h3></div>
                <Select
                  class="theme-select"
                  :model-value="fontPreset"
                  :options="selectOptions.font"
                  :aria-label="t('nav.fontPreset')"
                  @update:model-value="handleFontPresetChange"
                />
              </div>
              <div class="theme-setting-control">
                <div class="theme-setting-heading"><h3>{{ t('nav.radiusPreset') }}</h3></div>
                <Select
                  class="theme-select"
                  :model-value="radiusPreset"
                  :options="selectOptions.radius"
                  :aria-label="t('nav.radiusPreset')"
                  @update:model-value="handleRadiusPresetChange"
                />
              </div>
              <div class="theme-setting-control">
                <div class="theme-setting-heading"><h3>{{ t('nav.density') }}</h3></div>
                <Select
                  class="theme-select"
                  :model-value="density"
                  :options="selectOptions.density"
                  :aria-label="t('nav.density')"
                  @update:model-value="handleDensityChange"
                />
              </div>
              <div class="theme-setting-control">
                <div class="theme-setting-heading"><h3>{{ t('nav.sidebarStyle') }}</h3></div>
                <Select
                  class="theme-select"
                  :model-value="sidebarStyle"
                  :options="selectOptions.sidebar"
                  :aria-label="t('nav.sidebarStyle')"
                  @update:model-value="handleSidebarStyleChange"
                />
              </div>
              <div class="theme-setting-control">
                <div class="theme-setting-heading"><h3>{{ t('nav.layout') }}</h3></div>
                <Select
                  class="theme-select"
                  :model-value="layout"
                  :options="selectOptions.layout"
                  :aria-label="t('nav.layout')"
                  @update:model-value="handleLayoutChange"
                />
              </div>
              <div class="theme-setting-control">
                <div class="theme-setting-heading"><h3>{{ t('nav.contentWidth') }}</h3></div>
                <Select
                  class="theme-select"
                  :model-value="contentWidth"
                  :options="selectOptions.contentWidth"
                  :aria-label="t('nav.contentWidth')"
                  @update:model-value="handleContentWidthChange"
                />
              </div>
            </section>

            <button type="button" class="theme-reset-button" @click="resetAppearance">
              <Icon name="refresh" size="sm" aria-hidden="true" />
              {{ t('nav.resetAppearance') }}
            </button>
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import ThemePreview from '@/components/common/ThemePreview.vue'
import {
  appThemes,
  setAppTheme,
  setColorMode,
  setContentWidth,
  setDensity,
  setFontPreset,
  setLayout,
  setRadiusPreset,
  setSidebarStyle,
  useAppTheme,
  type ColorMode,
  type ContentWidthPresetId,
  type DensityPresetId,
  type FontPresetId,
  type LayoutPresetId,
  type RadiusPresetId,
  type SidebarStyleId
} from '@/composables/useAppTheme'

const props = withDefaults(defineProps<{ variant?: 'floating' | 'sidebar'; collapsed?: boolean }>(), {
  variant: 'floating',
  collapsed: false
})

const { t } = useI18n()
const {
  activeTheme,
  colorMode,
  contentWidth,
  density,
  fontPreset,
  layout,
  radiusPreset,
  sidebarStyle
} = useAppTheme()

const modeOptions: readonly { value: ColorMode; labelKey: string; icon: 'globe' | 'sun' | 'moon' }[] = [
  { value: 'system', labelKey: 'nav.systemMode', icon: 'globe' },
  { value: 'light', labelKey: 'nav.lightMode', icon: 'sun' },
  { value: 'dark', labelKey: 'nav.darkMode', icon: 'moon' }
]
const fontOptions: readonly { value: FontPresetId; labelKey: string }[] = [
  { value: 'system', labelKey: 'nav.fonts.system' },
  { value: 'humanist', labelKey: 'nav.fonts.humanist' },
  { value: 'serif', labelKey: 'nav.fonts.serif' },
  { value: 'mono', labelKey: 'nav.fonts.mono' }
]
const radiusOptions: readonly { value: RadiusPresetId; labelKey: string }[] = [
  { value: 'sharp', labelKey: 'nav.radii.sharp' },
  { value: 'soft', labelKey: 'nav.radii.soft' },
  { value: 'round', labelKey: 'nav.radii.round' }
]
const densityOptions: readonly { value: DensityPresetId; labelKey: string }[] = [
  { value: 'compact', labelKey: 'nav.densities.compact' },
  { value: 'comfortable', labelKey: 'nav.densities.comfortable' },
  { value: 'spacious', labelKey: 'nav.densities.spacious' }
]
const sidebarOptions: readonly { value: SidebarStyleId; labelKey: string }[] = [
  { value: 'default', labelKey: 'nav.sidebarStyles.default' },
  { value: 'floating', labelKey: 'nav.sidebarStyles.floating' },
  { value: 'compact', labelKey: 'nav.sidebarStyles.compact' }
]
const layoutOptions: readonly { value: LayoutPresetId; labelKey: string }[] = [
  { value: 'default', labelKey: 'nav.layouts.default' },
  { value: 'wide', labelKey: 'nav.layouts.wide' },
  { value: 'centered', labelKey: 'nav.layouts.centered' }
]
const contentWidthOptions: readonly { value: ContentWidthPresetId; labelKey: string }[] = [
  { value: 'standard', labelKey: 'nav.contentWidths.standard' },
  { value: 'wide', labelKey: 'nav.contentWidths.wide' },
  { value: 'full', labelKey: 'nav.contentWidths.full' }
]

const selectOptions = computed(() => ({
  font: fontOptions.map((option): SelectOption => ({ value: option.value, label: t(option.labelKey) })),
  radius: radiusOptions.map((option): SelectOption => ({ value: option.value, label: t(option.labelKey) })),
  density: densityOptions.map((option): SelectOption => ({ value: option.value, label: t(option.labelKey) })),
  sidebar: sidebarOptions.map((option): SelectOption => ({ value: option.value, label: t(option.labelKey) })),
  layout: layoutOptions.map((option): SelectOption => ({ value: option.value, label: t(option.labelKey) })),
  contentWidth: contentWidthOptions.map((option): SelectOption => ({ value: option.value, label: t(option.labelKey) }))
}))

const isOpen = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const drawerRef = ref<HTMLElement | null>(null)
const drawerTitleId = `appearance-drawer-title-${Math.random().toString(36).slice(2, 9)}`
let previousBodyOverflow = ''

function toggleDrawer() {
  isOpen.value = !isOpen.value
}

function closeDrawer() {
  isOpen.value = false
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as Node
  if (!triggerRef.value?.contains(target) && !drawerRef.value?.contains(target)) {
    closeDrawer()
  }
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && isOpen.value) closeDrawer()
}

function handleFontPresetChange(value: SelectOption['value']) {
  if (typeof value === 'string') setFontPreset(value as FontPresetId)
}

function handleRadiusPresetChange(value: SelectOption['value']) {
  if (typeof value === 'string') setRadiusPreset(value as RadiusPresetId)
}

function handleDensityChange(value: SelectOption['value']) {
  if (typeof value === 'string') setDensity(value as DensityPresetId)
}

function handleSidebarStyleChange(value: SelectOption['value']) {
  if (typeof value === 'string') setSidebarStyle(value as SidebarStyleId)
}

function handleLayoutChange(value: SelectOption['value']) {
  if (typeof value === 'string') setLayout(value as LayoutPresetId)
}

function handleContentWidthChange(value: SelectOption['value']) {
  if (typeof value === 'string') setContentWidth(value as ContentWidthPresetId)
}

function resetAppearance() {
  setAppTheme('minimalism')
  setColorMode('system')
  setFontPreset('system')
  setRadiusPreset('soft')
  setDensity('comfortable')
  setSidebarStyle('default')
  setLayout('default')
  setContentWidth('standard')
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleKeydown)
  document.body.style.overflow = previousBodyOverflow
})

watch(isOpen, (open) => {
  if (open) {
    previousBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = previousBodyOverflow
  }
})
</script>

<style scoped>
.theme-switcher {
  position: relative;
}

.theme-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.65rem;
  min-height: 2.5rem;
  border: 1px solid transparent;
  border-radius: var(--control-radius, 0.75rem);
  color: var(--muted);
  background: transparent;
  transition: background-color 160ms ease, color 160ms ease, border-color 160ms ease;
}

.theme-trigger:hover,
.theme-trigger:focus-visible {
  color: var(--ink);
  background: color-mix(in srgb, var(--primary-500) 10%, var(--surface-raised));
  border-color: color-mix(in srgb, var(--primary-500) 22%, var(--border-color));
  outline: none;
}

.theme-trigger-floating {
  padding: 0 0.75rem;
}

.theme-trigger-sidebar {
  width: 100%;
  justify-content: flex-start;
  padding: 0.625rem 1.0625rem;
}

.theme-trigger-collapsed {
  justify-content: center;
  padding-left: 0.875rem;
  padding-right: 0.875rem;
}

.theme-trigger-label {
  overflow: hidden;
  max-width: 12rem;
  white-space: nowrap;
  transition: max-width 180ms ease, opacity 140ms ease;
}

.theme-trigger-collapsed .theme-trigger-label {
  max-width: 0;
  opacity: 0;
}

.theme-drawer-layer {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  justify-content: flex-end;
  background: color-mix(in srgb, var(--dark-base) 58%, transparent);
  backdrop-filter: blur(3px);
}

.theme-drawer-panel {
  display: flex;
  flex-direction: column;
  width: min(100%, 27rem);
  height: 100%;
  overflow: hidden;
  color: var(--ink);
  background: var(--surface-raised);
  border-left: 1px solid var(--border-color);
  box-shadow: -1.5rem 0 4rem color-mix(in srgb, var(--dark-base) 18%, transparent);
}

.theme-drawer-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 1.25rem 1.25rem 1rem;
  border-bottom: 1px solid var(--border-color);
}

.theme-drawer-heading {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
}

.theme-drawer-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  flex: 0 0 auto;
  border-radius: var(--control-radius, 0.75rem);
  color: var(--readable-primary, var(--ink));
  background: color-mix(in srgb, var(--primary-500) 13%, var(--surface-muted));
}

.dark .theme-drawer-icon {
  color: var(--readable-primary, var(--ink));
}

.theme-drawer-title {
  margin: 0;
  color: var(--ink);
  font-size: 1rem;
  font-weight: 700;
}

.theme-drawer-subtitle {
  margin: 0.2rem 0 0;
  color: var(--muted);
  font-size: 0.75rem;
}

.theme-drawer-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.25rem;
  height: 2.25rem;
  flex: 0 0 auto;
  border-radius: var(--control-radius, 0.75rem);
  color: var(--muted);
  transition: color 160ms ease, background-color 160ms ease;
}

.theme-drawer-close:hover,
.theme-drawer-close:focus-visible {
  color: var(--ink);
  background: var(--surface-muted);
  outline: none;
}

.theme-drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 1rem 1.25rem 1.5rem;
  overscroll-behavior: contain;
}

.theme-setting-section {
  padding: 1rem 0;
  border-bottom: 1px solid var(--border-color);
}

.theme-setting-section:first-child {
  padding-top: 0;
}

.theme-setting-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.65rem;
}

.theme-setting-heading h3 {
  margin: 0;
  color: var(--ink);
  font-size: 0.8rem;
  font-weight: 700;
}

.theme-setting-heading p {
  margin: 0.2rem 0 0;
  color: var(--muted);
  font-size: 0.7rem;
}

.theme-mode-grid {
  display: grid;
  gap: 0.5rem;
}

.theme-mode-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.theme-choice {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  min-width: 0;
  min-height: 2.35rem;
  padding: 0.5rem 0.4rem;
  border: 1px solid var(--border-color);
  border-radius: var(--control-radius, 0.75rem);
  color: var(--muted);
  background: var(--surface);
  font-size: 0.72rem;
  transition: border-color 160ms ease, background-color 160ms ease, color 160ms ease;
}

.theme-choice:hover,
.theme-choice-active {
  color: var(--readable-primary, var(--ink));
  border-color: var(--primary-500);
  background: color-mix(in srgb, var(--primary-500) 10%, var(--surface-raised));
}

.dark .theme-choice:hover,
.dark .theme-choice-active {
  color: var(--readable-primary, var(--ink));
}

.theme-card-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.65rem;
}

.theme-card {
  position: relative;
  min-width: 0;
  padding: 0.35rem;
  overflow: hidden;
  border: 1px solid var(--border-color);
  border-radius: var(--control-radius, 0.75rem);
  color: var(--ink);
  background: var(--surface);
  text-align: left;
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;
}

.theme-card:hover,
.theme-card:focus-visible {
  border-color: color-mix(in srgb, var(--primary-500) 70%, var(--border-color));
  box-shadow: 0 0.4rem 1.2rem color-mix(in srgb, var(--primary-500) 14%, transparent);
  outline: none;
  transform: translateY(-1px);
}

.theme-card-active {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary-500) 22%, transparent);
}

.theme-card-footer {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.5rem 0.2rem 0.1rem;
}

.theme-card-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.73rem;
  font-weight: 600;
}

.theme-card-check {
  flex: 0 0 auto;
  color: var(--readable-primary, var(--ink));
}

.theme-setting-grid-section {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.9rem 0.75rem;
}

.theme-setting-grid-section .theme-setting-heading {
  margin-bottom: 0.4rem;
}

.theme-select {
  width: 100%;
}

.theme-select :deep(.select-trigger) {
  min-height: 2.25rem;
  padding: 0.4rem 0.55rem;
  border-color: var(--border-color);
  border-radius: var(--control-radius, 0.75rem);
  color: var(--ink);
  background: var(--surface);
  font-size: 0.74rem;
}

.theme-reset-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  width: 100%;
  margin-top: 1rem;
  min-height: 2.35rem;
  border: 1px solid var(--border-color);
  border-radius: var(--control-radius, 0.75rem);
  color: var(--muted);
  background: var(--surface);
  font-size: 0.75rem;
  transition: color 160ms ease, border-color 160ms ease, background-color 160ms ease;
}

.theme-reset-button:hover,
.theme-reset-button:focus-visible {
  color: var(--ink);
  border-color: var(--primary-500);
  background: var(--surface-muted);
  outline: none;
}

.theme-drawer-enter-active,
.theme-drawer-leave-active {
  transition: opacity 220ms ease;
}

.theme-drawer-enter-active .theme-drawer-panel,
.theme-drawer-leave-active .theme-drawer-panel {
  transition: transform 220ms ease;
}

.theme-drawer-enter-from,
.theme-drawer-leave-to {
  opacity: 0;
}

.theme-drawer-enter-from .theme-drawer-panel,
.theme-drawer-leave-to .theme-drawer-panel {
  transform: translateX(100%);
}

@media (max-width: 480px) {
  .theme-drawer-panel {
    width: 100%;
  }

  .theme-drawer-header,
  .theme-drawer-body {
    padding-left: 1rem;
    padding-right: 1rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .theme-drawer-enter-active,
  .theme-drawer-leave-active,
  .theme-drawer-enter-active .theme-drawer-panel,
  .theme-drawer-leave-active .theme-drawer-panel {
    transition-duration: 1ms;
  }
}
</style>
