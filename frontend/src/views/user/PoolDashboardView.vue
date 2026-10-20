<template>
  <AppLayout>
    <div ref="screen" class="pool-screen space-y-6" data-test="pool-screen">
      <header class="flex items-start gap-4">
        <div class="stat-icon stat-icon-primary hidden shrink-0 sm:flex">
          <Icon name="server" size="lg" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('poolDashboard.title') }}</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-500 dark:text-dark-400">{{ t('poolDashboard.description') }}</p>
        </div>
      </header>

      <div v-if="error" class="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-900/20 dark:text-red-300" role="alert">
        <Icon name="exclamationCircle" class="mt-0.5 shrink-0" aria-hidden="true" />
        <p class="min-w-0 break-words">{{ error }}</p>
      </div>

      <div v-if="!data && loading" class="grid grid-cols-2 gap-4 xl:grid-cols-4" aria-busy="true" :aria-label="t('common.loading')">
        <Skeleton v-for="n in 4" :key="n" :height="112" />
      </div>
      <div v-if="data" class="pool-metrics grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
        <StatCard
          v-for="metric in metrics"
          :key="metric.key"
          :title="t(`poolDashboard.${metric.key}`)"
          :value="metric.value"
          :icon="metric.icon"
          :icon-variant="metric.variant"
          class="min-w-0 flex-col gap-3 p-4 sm:flex-row sm:gap-4 sm:p-5"
          :data-test="`metric-${metric.key}`"
        />
      </div>

      <div class="card relative z-10 flex flex-wrap items-center justify-between gap-4 p-4 sm:p-5">
        <div class="w-full min-w-0 sm:w-72">
          <Select
            v-model="groupFilter"
            :options="groupOptions"
            :disabled="!data?.groups.length"
            :aria-label="t('poolDashboard.groups')"
            :teleport-to="screen || 'body'"
            data-test="group-filter"
          />
        </div>
        <div class="pool-actions flex w-full flex-wrap items-center gap-2 sm:w-auto">
          <AutoRefreshButton
            :enabled="refreshEnabled"
            :interval-seconds="refreshInterval"
            :countdown="countdown"
            :intervals="intervals"
            data-test="auto-refresh"
            @update:enabled="setRefreshEnabled"
            @update:interval="setRefreshInterval"
          />
          <button type="button" class="btn btn-secondary gap-2" :disabled="loading" data-test="refresh-pool" @click="load">
            <Icon name="refresh" size="sm" :class="loading && 'animate-spin'" aria-hidden="true" />
            {{ t('common.refresh') }}
          </button>
          <button v-if="fullscreenSupported" type="button" class="btn btn-secondary gap-2" :aria-pressed="fullscreen" data-test="fullscreen-pool" @click="toggleFullscreen">
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
              <path v-if="fullscreen" stroke-linecap="round" stroke-linejoin="round" d="M9 3v6H3m12-6v6h6M9 21v-6H3m12 6v-6h6" />
              <path v-else stroke-linecap="round" stroke-linejoin="round" d="M9 3H3v6m12-6h6v6M3 15v6h6m6 0h6v-6" />
            </svg>
            {{ t(fullscreen ? 'poolDashboard.exitFullscreen' : 'poolDashboard.fullscreen') }}
          </button>
        </div>
      </div>

      <template v-if="data">
        <div class="flex flex-wrap items-center justify-between gap-2 text-xs leading-5 text-gray-500 dark:text-dark-400">
          <p>{{ t('poolDashboard.visibleGroups', { count: visibleGroups.length }) }}</p>
          <p class="flex items-center gap-1.5"><Icon name="clock" size="xs" aria-hidden="true" />{{ t('poolDashboard.snapshot') }} {{ formatDateTime(data.generated_at) }}</p>
        </div>
        <EmptyState
          v-if="!data.groups.length"
          class="card"
          :title="t('poolDashboard.emptyTitle')"
          :description="t('poolDashboard.empty')"
          :action-text="t('poolDashboard.manageKeys')"
          action-to="/keys"
          :action-icon="false"
        />
        <section v-for="group in visibleGroups" :key="group.id" class="card min-w-0" :aria-labelledby="`pool-group-title-${group.id}`" :data-test="`pool-group-${group.id}`">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 p-4 dark:border-dark-700 sm:px-5">
            <div class="flex min-w-0 flex-1 items-start gap-3">
              <div class="stat-icon stat-icon-primary !h-10 !w-10 shrink-0 !rounded-xl"><Icon name="grid" size="md" aria-hidden="true" /></div>
              <div class="min-w-0">
                <h2 :id="`pool-group-title-${group.id}`" class="break-words text-base font-semibold text-gray-900 dark:text-white">{{ group.name }}</h2>
                <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('poolDashboard.groupCount', { count: group.accounts.length, available: group.accounts.filter(a => a.available).length }) }}</p>
              </div>
            </div>
            <span class="max-w-full break-words rounded-lg px-2.5 py-1 text-xs font-medium" :class="platformBadgeLightClass(group.platform)">{{ platformLabel(group.platform) }}</span>
          </div>
          <div v-if="group.accounts.length" class="pool-account-grid grid gap-4 p-4 sm:p-5">
            <PoolQuotaCard v-for="account in pageAccounts(group)" :key="account.id" :account="account" :now="now" />
          </div>
          <EmptyState v-else :title="t('poolDashboard.emptyGroup')" />
          <Pagination
            v-if="group.accounts.length > pageSize"
            :total="group.accounts.length"
            :page="page(group.id)"
            :page-size="pageSize"
            :show-page-size-selector="false"
            class="pool-pagination rounded-b-2xl"
            @update:page="pages[group.id] = $event"
          />
        </section>
        <div class="flex items-start gap-2 px-1 text-xs leading-6 text-gray-500 dark:text-dark-400">
          <Icon name="infoCircle" size="sm" class="mt-1 shrink-0" aria-hidden="true" />
          <p>{{ t('poolDashboard.quotaHint') }}</p>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import PoolQuotaCard from '@/components/account/PoolQuotaCard.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import StatCard from '@/components/common/StatCard.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { balanceIsCurrent, getPoolDashboard, quotaIsCurrent, type PoolQuotaGroup, type UserPoolDashboard } from '@/api/poolDashboard'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import { platformBadgeLightClass, platformLabel } from '@/utils/platformColors'

const { t } = useI18n()
const data = ref<UserPoolDashboard | null>(null)
const loading = ref(false)
const error = ref('')
const groupFilter = ref('all')
const now = ref(Date.now())
const pages = reactive<Record<number, number>>({})
const pageSize = 24
const screen = ref<HTMLElement | null>(null)
const fullscreen = ref(false)
const fullscreenSupported = typeof document !== 'undefined' && !!document.fullscreenEnabled
const metricIcons = {
  groups: () => h(Icon, { name: 'grid' }),
  accounts: () => h(Icon, { name: 'server' }),
  available: () => h(Icon, { name: 'checkCircle' }),
  knownQuota: () => h(Icon, { name: 'chart' })
}
const groupOptions = computed(() => [
  { value: 'all', label: t('poolDashboard.allGroups') },
  ...(data.value?.groups || []).map(group => ({ value: String(group.id), label: group.name }))
])
const visibleGroups = computed(() => (data.value?.groups || []).filter(g => groupFilter.value === 'all' || String(g.id) === groupFilter.value))
const uniqueAccounts = computed(() => [...new Map((data.value?.groups || []).flatMap(g => g.accounts).map(a => [a.id, a])).values()])
const metrics = computed(() => [
  { key: 'groups', value: data.value?.groups.length || 0, icon: metricIcons.groups, variant: 'primary' as const },
  { key: 'accounts', value: uniqueAccounts.value.length, icon: metricIcons.accounts, variant: 'primary' as const },
  { key: 'available', value: uniqueAccounts.value.filter(a => a.available).length, icon: metricIcons.available, variant: 'success' as const },
  { key: 'knownQuota', value: uniqueAccounts.value.filter(a => balanceIsCurrent(a, now.value) || a.windows.some(w => quotaIsCurrent(w, now.value))).length, icon: metricIcons.knownQuota, variant: 'success' as const }
])
const {
  enabled: refreshEnabled,
  intervalSeconds: refreshInterval,
  countdown,
  intervals,
  setEnabled: setRefreshEnabled,
  setInterval: setRefreshInterval,
  resetCountdown,
  start: startRefresh
} = useAutoRefresh({
  storageKey: 'user-account-pool-refresh',
  intervals: [30, 60, 120],
  defaultInterval: 30,
  defaultEnabled: true,
  onRefresh: load,
  shouldPause: () => document.hidden || loading.value
})

const page = (id: number) => pages[id] || 1
function pageAccounts(group: PoolQuotaGroup) {
  const current = Math.min(page(group.id), Math.max(1, Math.ceil(group.accounts.length / pageSize)))
  return group.accounts.slice((current - 1) * pageSize, current * pageSize)
}
let controller: AbortController | undefined
let clock: ReturnType<typeof setInterval> | undefined
let disposed = false

async function load() {
  if (loading.value) return
  loading.value = true
  controller = new AbortController()
  try {
    const result = await getPoolDashboard(controller.signal)
    if (disposed) return
    data.value = result
    for (const group of result.groups) pages[group.id] = Math.min(page(group.id), Math.max(1, Math.ceil(group.accounts.length / pageSize)))
    if (groupFilter.value !== 'all' && !result.groups.some(g => String(g.id) === groupFilter.value)) groupFilter.value = 'all'
    error.value = ''
    now.value = Date.now()
    resetCountdown()
  } catch (reason) {
    if (!disposed) error.value = extractApiErrorMessage(reason, t('poolDashboard.failed'))
  } finally {
    loading.value = false
  }
}
function onFullscreenChange() {
  fullscreen.value = document.fullscreenElement === screen.value
}
function updateClock() {
  now.value = Date.now()
}
async function toggleFullscreen() {
  try {
    if (fullscreen.value) await document.exitFullscreen()
    else await screen.value?.requestFullscreen()
  } catch {
    error.value = t('poolDashboard.fullscreenFailed')
  }
}
onMounted(() => {
  void load()
  resetCountdown()
  if (refreshEnabled.value) startRefresh()
  clock = setInterval(updateClock, 30000)
  document.addEventListener('visibilitychange', updateClock)
  document.addEventListener('fullscreenchange', onFullscreenChange)
})
onUnmounted(() => {
  disposed = true
  controller?.abort()
  clearInterval(clock)
  document.removeEventListener('visibilitychange', updateClock)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
})
</script>

<style scoped>
.pool-account-grid {
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 19rem), 1fr));
}
.pool-actions :deep(.app-theme-control) {
  min-height: 2.5rem;
  font-size: 0.875rem;
  padding: 0.5rem 0.75rem;
}
.pool-actions :deep(.app-theme-menu-surface) {
  left: 0;
  right: auto;
}
.pool-pagination :deep(.sm\:flex) {
  flex-wrap: wrap;
  gap: 0.75rem;
}
.pool-screen:fullscreen {
  overflow: auto;
  padding: clamp(1rem, 3vw, 2.5rem);
  background: var(--surface-muted);
}
</style>
