<template>
  <article class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800" data-test="pool-account">
    <div class="mb-5 flex items-center justify-between gap-3">
      <div class="min-w-0">
        <h3 class="font-semibold text-gray-900 dark:text-gray-100">{{ t('poolDashboard.account') }} {{ account.label }}</h3>
        <p class="mt-1 text-xs uppercase tracking-wide text-gray-500">{{ account.platform }}</p>
      </div>
      <span class="shrink-0 rounded-full px-2 py-1 text-xs" :class="account.available ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'">
        {{ t(account.available ? 'poolDashboard.available' : 'poolDashboard.unavailable') }}
      </span>
    </div>
    <div class="space-y-4">
      <div v-for="window in account.windows" :key="window.window" class="pool-usage">
        <div class="mb-2 flex items-center justify-between gap-2 text-xs text-gray-500">
          <span>{{ window.label }}</span>
          <span>{{ t('poolDashboard.remaining') }}</span>
        </div>
        <UsageProgressBar v-if="quotaIsCurrent(window, now)" :label="t('poolDashboard.remaining')" :utilization="100 - (window.used_percent ?? 0)" :resets-at="window.resets_at" remaining-capacity label-width="auto" color="emerald" />
        <p v-else class="rounded-lg bg-gray-50 px-3 py-2 text-sm text-gray-400 dark:bg-dark-900">{{ t(window.status === 'stale' || window.used_percent != null ? 'poolDashboard.stale' : 'poolDashboard.unknown') }}</p>
      </div>
      <div v-if="account.balance != null">
        <p class="text-xs text-gray-500">{{ t('poolDashboard.balance') }}</p>
        <p class="mt-2 text-2xl font-semibold tabular-nums text-gray-900 dark:text-gray-100">{{ balanceIsCurrent(account, now) ? `${account.balance.toLocaleString(undefined, { maximumFractionDigits: 4 })} ${account.balance_unit}` : t('poolDashboard.stale') }}</p>
      </div>
      <p v-if="!account.windows.length && account.balance == null" class="rounded-lg bg-gray-50 p-4 text-sm text-gray-400 dark:bg-dark-900">{{ t('poolDashboard.unknown') }}</p>
    </div>
    <p class="mt-5 border-t border-gray-100 pt-3 text-xs text-gray-400 dark:border-dark-700">{{ t('poolDashboard.observed') }} {{ account.updated_at ? formatDateTime(account.updated_at) : '—' }}</p>
  </article>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import UsageProgressBar from './UsageProgressBar.vue'
import { balanceIsCurrent, quotaIsCurrent, type PoolQuotaAccount } from '@/api/poolDashboard'
import { formatDateTime } from '@/utils/format'
defineProps<{ account: PoolQuotaAccount; now: number }>()
const { t } = useI18n()
</script>
<style scoped>
.pool-usage :deep(.w-8) { flex: 1; width: auto; height: 0.5rem; }
.pool-usage :deep(.text-\[10px\]) { font-size: 0.75rem; }
.pool-usage :deep(.flex.items-center.gap-1) { flex-wrap: wrap; gap: 0.5rem; }
</style>
