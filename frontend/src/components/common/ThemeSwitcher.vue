<template>
  <div ref="dropdownRef" class="relative" :class="{ 'w-full': variant === 'sidebar' }">
    <button
      type="button"
      :aria-expanded="isOpen"
      aria-haspopup="true"
      class="flex items-center rounded-lg transition-colors"
      :class="variant === 'sidebar' ? 'sidebar-link mb-2 w-full' : 'h-10 px-2.5 text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white'"
      :title="t('nav.appearance')"
      @click="toggleDropdown"
    >
      <svg class="h-5 w-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
        <circle cx="12" cy="12" r="8" />
        <path d="M12 4v16M4 12h16" />
        <circle cx="8.5" cy="8.5" r="2.35" fill="currentColor" stroke="none" />
        <circle cx="15.5" cy="8.5" r="2.35" fill="currentColor" fill-opacity=".72" stroke="none" />
        <circle cx="8.5" cy="15.5" r="2.35" fill="currentColor" fill-opacity=".55" stroke="none" />
        <circle cx="15.5" cy="15.5" r="2.35" fill="currentColor" fill-opacity=".85" stroke="none" />
      </svg>
      <span v-if="variant === 'sidebar'" class="sidebar-label" :class="{ 'sidebar-label-collapsed': collapsed }">
        {{ t('nav.appearance') }}
      </span>
    </button>

    <transition name="theme-dropdown">
      <div
        v-if="isOpen"
        class="absolute z-50 mt-1 w-56 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-lg dark:border-dark-700 dark:bg-dark-800"
        :class="variant === 'sidebar' ? 'left-full bottom-0 ml-2' : 'right-0'"
      >
        <div class="px-3 pb-1 pt-3 text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-dark-500">
          {{ t('nav.theme') }}
        </div>
        <button
          v-for="theme in appThemes"
          :key="theme.id"
          type="button"
          class="flex w-full items-center gap-3 px-3 py-2 text-left text-sm text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700"
          :class="{ 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300': theme.id === activeTheme }"
          @click="selectTheme(theme.id)"
        >
          <span
            class="h-5 w-5 shrink-0 rounded-full border border-current/20"
            :data-theme-preview="theme.id"
          />
          <span class="flex-1 truncate">{{ t(theme.labelKey) }}</span>
          <Icon v-if="theme.id === activeTheme" name="check" size="sm" class="text-primary-500" />
        </button>

        <div class="mt-1 border-t border-gray-100 p-2 dark:border-dark-700">
          <button
            type="button"
            class="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-sm text-gray-600 transition-colors hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700"
            @click="toggleColorMode(); closeDropdown()"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
            {{ isDark ? t('nav.lightMode') : t('nav.darkMode') }}
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { appThemes, setAppTheme, toggleColorMode, useAppTheme, type AppThemeId } from '@/composables/useAppTheme'

withDefaults(defineProps<{ variant?: 'floating' | 'sidebar'; collapsed?: boolean }>(), {
  variant: 'floating',
  collapsed: false
})

const { t } = useI18n()
const { activeTheme, isDark } = useAppTheme()
const isOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)

function toggleDropdown() {
  isOpen.value = !isOpen.value
}

function closeDropdown() {
  isOpen.value = false
}

function selectTheme(theme: AppThemeId) {
  setAppTheme(theme)
  closeDropdown()
}

function handleClickOutside(event: MouseEvent) {
  if (!dropdownRef.value?.contains(event.target as Node)) {
    closeDropdown()
  }
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))
</script>

<style scoped>
.theme-dropdown-enter-active,
.theme-dropdown-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.theme-dropdown-enter-from,
.theme-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

[data-theme-preview='minimalism'] {
  background: conic-gradient(from 180deg, var(--primary-500), var(--surface), var(--dark-base), var(--primary-500));
}

[data-theme-preview='neoBrutalism'] {
  background: conic-gradient(from 180deg, var(--primary-400), var(--accent-base), var(--dark-700), var(--primary-400));
}

[data-theme-preview='apple'] {
  background: conic-gradient(from 180deg, var(--primary-500), var(--surface-muted), var(--dark-base), var(--primary-500));
}

[data-theme-preview='notion'] {
  background: conic-gradient(from 180deg, var(--primary-600), var(--surface-raised), var(--surface-muted), var(--primary-600));
}

[data-theme-preview='wabiSabi'] {
  background: conic-gradient(from 180deg, var(--accent-500), var(--surface), var(--dark-800), var(--accent-500));
}
</style>
