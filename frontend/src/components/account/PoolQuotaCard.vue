<template>
  <article class="card flex min-w-0 flex-col p-4 sm:p-5" data-test="pool-account">
    <div class="mb-5 flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <h3 class="break-words font-semibold text-gray-900 dark:text-gray-100">{{ t('poolDashboard.account') }} {{ account.label }}</h3>
        <p class="mt-1 break-words text-xs text-gray-500 dark:text-dark-400">{{ platformLabel(account.platform) }}</p>
      </div>
      <StatusBadge
        :status="account.available ? 'success' : 'warning'"
        :label="t(account.available ? 'poolDashboard.available' : 'poolDashboard.unavailable')"
        class="shrink-0"
      />
    </div>
    <div class="min-w-0 flex-1 space-y-4">
      <div v-for="window in account.windows" :key="window.window" class="pool-usage min-w-0">
        <p class="mb-2 break-words text-sm font-medium leading-5 text-gray-700 dark:text-dark-200">{{ quotaLabel(window.label) }}</p>
        <UsageProgressBar
          v-if="quotaIsCurrent(window, now)"
          :label="t('poolDashboard.remaining')"
          :utilization="100 - (window.used_percent ?? 0)"
          :resets-at="window.resets_at"
          remaining-capacity
          label-width="auto"
          color="emerald"
        />
        <p v-else class="rounded-lg bg-gray-50 px-3 py-2 text-xs leading-5 text-gray-500 dark:bg-dark-900/60 dark:text-dark-400">{{ t(window.status === 'stale' || window.used_percent != null ? 'poolDashboard.stale' : 'poolDashboard.unknown') }}</p>
      </div>
      <div v-if="account.balance != null" class="min-w-0 rounded-xl bg-gray-50 p-3 dark:bg-dark-900/60">
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('poolDashboard.balance') }}</p>
        <p v-if="balanceIsCurrent(account, now)" class="mt-2 break-words text-2xl font-semibold tabular-nums text-gray-900 dark:text-gray-100">
          {{ account.balance.toLocaleString(undefined, { maximumFractionDigits: 4 }) }}
          <span class="text-sm font-normal text-gray-500 dark:text-dark-400">{{ account.balance_unit }}</span>
        </p>
        <p v-else class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('poolDashboard.stale') }}</p>
      </div>
      <div v-if="!account.windows.length && account.balance == null" class="flex min-h-20 items-center gap-2 rounded-xl bg-gray-50 p-3 text-sm text-gray-500 dark:bg-dark-900/60 dark:text-dark-400">
        <Icon name="clock" size="sm" class="shrink-0" aria-hidden="true" />
        <p class="min-w-0 leading-6">{{ t('poolDashboard.unknown') }}</p>
      </div>
    </div>
    <p class="mt-5 flex flex-wrap items-center gap-x-2 gap-y-1 border-t border-gray-100 pt-3 text-xs leading-5 text-gray-500 dark:border-dark-700 dark:text-dark-400">
      <span>{{ t('poolDashboard.observed') }}</span>
      <span>{{ account.updated_at ? formatDateTime(account.updated_at) : '—' }}</span>
    </p>
  </article>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import UsageProgressBar from './UsageProgressBar.vue'
import { balanceIsCurrent, quotaIsCurrent, type PoolQuotaAccount } from '@/api/poolDashboard'
import { formatDateTime } from '@/utils/format'
import { platformLabel } from '@/utils/platformColors'

defineProps<{ account: PoolQuotaAccount; now: number }>()
const { t } = useI18n()
function quotaLabel(label: string) {
  if (label === '5h') return t('poolDashboard.fiveHours')
  if (label === '7d') return t('poolDashboard.sevenDays')
  return label
}
</script>

<style scoped>
.pool-usage :deep(.w-8) { flex: 1; min-width: 2rem; width: auto; height: 0.5rem; }
.pool-usage :deep(.text-\[10px\]) { font-size: 0.75rem; }
.pool-usage :deep(.flex.items-center.gap-1) { flex-wrap: wrap; gap: 0.5rem; }
</style>
