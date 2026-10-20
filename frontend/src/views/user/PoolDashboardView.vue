<template>
  <AppLayout>
    <div ref="screen" class="pool-screen space-y-6 rounded-2xl bg-gray-50 p-1 dark:bg-dark-900">
      <header class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('poolDashboard.title') }}</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-500">{{ t('poolDashboard.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <label class="flex items-center gap-2 text-sm text-gray-500"><input v-model="autoRefresh" type="checkbox" class="rounded" />{{ t('poolDashboard.autoRefresh') }}</label>
          <button class="btn btn-secondary" :disabled="loading" data-test="refresh-pool" @click="load">{{ t('common.refresh') }}</button>
          <button v-if="fullscreenSupported" class="btn btn-secondary" @click="toggleFullscreen">{{ t(fullscreen ? 'poolDashboard.exitFullscreen' : 'poolDashboard.fullscreen') }}</button>
        </div>
      </header>
      <div v-if="error" class="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-900/20 dark:text-red-300" role="alert">{{ error }}</div>
      <div v-if="!data && loading" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-busy="true"><div v-for="n in 4" :key="n" class="h-28 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-700" /></div>
      <template v-if="data">
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div v-for="metric in metrics" :key="metric.key" class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
            <p class="text-sm text-gray-500">{{ t(`poolDashboard.${metric.key}`) }}</p>
            <p class="mt-3 text-3xl font-semibold tabular-nums text-gray-900 dark:text-white" :data-test="`metric-${metric.key}`">{{ metric.value }}</p>
          </div>
        </div>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <select v-model="groupFilter" class="input max-w-xs" :aria-label="t('poolDashboard.groups')" data-test="group-filter">
            <option value="all">{{ t('poolDashboard.allGroups') }}</option>
            <option v-for="group in data.groups" :key="group.id" :value="String(group.id)">{{ group.name }}</option>
          </select>
          <p class="text-xs text-gray-400">{{ t('poolDashboard.snapshot') }} {{ formatDateTime(data.generated_at) }}</p>
        </div>
        <div v-if="!data.groups.length" class="rounded-2xl border border-dashed border-gray-300 p-12 text-center dark:border-dark-600">
          <p class="text-gray-500">{{ t('poolDashboard.empty') }}</p>
          <RouterLink to="/keys" class="btn btn-primary mt-5">{{ t('nav.apiKeys') }}</RouterLink>
        </div>
        <section v-for="group in visibleGroups" :key="group.id" class="space-y-4" :data-test="`pool-group-${group.id}`">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex min-w-0 items-center gap-3"><span class="h-7 w-1 rounded-full bg-primary-500" /><h2 class="min-w-0 break-words text-lg font-semibold text-gray-900 dark:text-white">{{ group.name }}</h2><span class="rounded-full bg-gray-200 px-2 py-1 text-xs uppercase text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ group.platform }}</span></div>
            <p class="text-sm text-gray-500">{{ t('poolDashboard.groupCount', { count: group.accounts.length, available: group.accounts.filter(a => a.available).length }) }}</p>
          </div>
          <div v-if="group.accounts.length" class="grid items-start gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
            <PoolQuotaCard v-for="account in pageAccounts(group)" :key="account.id" :account="account" :now="now" />
          </div>
          <p v-else class="rounded-xl bg-white p-6 text-sm text-gray-400 dark:bg-dark-800">{{ t('poolDashboard.emptyGroup') }}</p>
          <div v-if="group.accounts.length > pageSize" class="flex items-center justify-end gap-3 text-sm">
            <button class="btn btn-secondary" :disabled="page(group.id) <= 1" @click="pages[group.id] = page(group.id) - 1">{{ t('poolDashboard.previous') }}</button>
            <span>{{ page(group.id) }} / {{ Math.ceil(group.accounts.length / pageSize) }}</span>
            <button class="btn btn-secondary" :disabled="page(group.id) * pageSize >= group.accounts.length" @click="pages[group.id] = page(group.id) + 1">{{ t('common.next') }}</button>
          </div>
        </section>
        <p class="text-xs leading-6 text-gray-400">{{ t('poolDashboard.quotaHint') }}</p>
      </template>
    </div>
  </AppLayout>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import PoolQuotaCard from '@/components/account/PoolQuotaCard.vue'
import { balanceIsCurrent, getPoolDashboard, quotaIsCurrent, type PoolQuotaGroup, type UserPoolDashboard } from '@/api/poolDashboard'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
const { t } = useI18n()
const data = ref<UserPoolDashboard | null>(null)
const loading = ref(false)
const error = ref('')
const groupFilter = ref('all')
const autoRefresh = ref(true)
const now = ref(Date.now())
const pages = reactive<Record<number, number>>({})
const pageSize = 24
const screen = ref<HTMLElement | null>(null)
const fullscreen = ref(false)
const fullscreenSupported = typeof document !== 'undefined' && !!document.fullscreenEnabled
const visibleGroups = computed(() => (data.value?.groups || []).filter(g => groupFilter.value === 'all' || String(g.id) === groupFilter.value))
const uniqueAccounts = computed(() => [...new Map((data.value?.groups || []).flatMap(g => g.accounts).map(a => [a.id, a])).values()])
const metrics = computed(() => [
  { key:'groups',value:data.value?.groups.length || 0 },
  { key:'accounts',value:uniqueAccounts.value.length },
  { key:'available',value:uniqueAccounts.value.filter(a => a.available).length },
  { key:'knownQuota',value:uniqueAccounts.value.filter(a => balanceIsCurrent(a,now.value) || a.windows.some(w => quotaIsCurrent(w,now.value))).length }
])
const page = (id: number) => pages[id] || 1
function pageAccounts(group: PoolQuotaGroup) {
  const current = Math.min(page(group.id), Math.max(1,Math.ceil(group.accounts.length/pageSize)))
  return group.accounts.slice((current-1)*pageSize,current*pageSize)
}
let controller: AbortController | undefined
let timer: ReturnType<typeof setInterval> | undefined
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
    error.value = ''; now.value = Date.now()
  } catch (reason) {
    if (!disposed) error.value = extractApiErrorMessage(reason,t('poolDashboard.failed'))
  } finally { loading.value = false }
}
function onFullscreenChange() { fullscreen.value = document.fullscreenElement === screen.value }
async function toggleFullscreen() {
  try { if (fullscreen.value) await document.exitFullscreen(); else await screen.value?.requestFullscreen() }
  catch { error.value = t('poolDashboard.fullscreenFailed') }
}
onMounted(() => {
  void load()
  timer = setInterval(() => { now.value = Date.now(); if (autoRefresh.value && !document.hidden) void load() },30000)
  document.addEventListener('fullscreenchange',onFullscreenChange)
})
onUnmounted(() => { disposed = true; controller?.abort(); clearInterval(timer); document.removeEventListener('fullscreenchange',onFullscreenChange) })
</script>
<style scoped>
.pool-screen:fullscreen { overflow: auto; padding: 2rem; }
</style>
