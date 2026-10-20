import { apiClient } from './client'

export interface PoolQuotaWindow {
  window: string
  label: string
  used_percent: number | null
  status: 'ok' | 'unknown' | 'stale' | 'unsupported' | 'failed'
  observed_at?: string | null
  resets_at?: string | null
}
export interface PoolQuotaAccount {
  id: number
  label: string
  platform: string
  available: boolean
  windows: PoolQuotaWindow[]
  balance: number | null
  balance_unit: string
  balance_fresh_until: string | null
  updated_at: string | null
}
export interface PoolQuotaGroup {
  id: number
  name: string
  platform: string
  accounts: PoolQuotaAccount[]
}
export interface UserPoolDashboard { generated_at: string; groups: PoolQuotaGroup[] }
export async function getPoolDashboard(signal?: AbortSignal): Promise<UserPoolDashboard> {
  return (await apiClient.get<UserPoolDashboard>('/user/pool-dashboard', { signal, timeout: 20000 })).data
}

export function quotaIsCurrent(window: PoolQuotaWindow, now: number): boolean {
  const observed = Date.parse(window.observed_at || '')
  return window.status === 'ok' && window.used_percent != null && Number.isFinite(window.used_percent)
    && window.used_percent >= 0 && window.used_percent <= 100
    && observed <= now && now - observed <= 15 * 60000 && Date.parse(window.resets_at || '') > now
}
export function balanceIsCurrent(account: PoolQuotaAccount, now: number): boolean {
  return account.balance != null && Number.isFinite(account.balance) && Date.parse(account.balance_fresh_until || '') > now
}
