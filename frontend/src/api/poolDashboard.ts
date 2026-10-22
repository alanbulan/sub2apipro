import { apiClient } from './client'
import type { Account, ClaudeModel } from '@/types'

export type PoolDiagnosticAccount = Pick<Account, 'id' | 'name' | 'platform' | 'type' | 'status'> & Partial<Pick<Account, 'credentials' | 'extra'>>
export interface PoolDiagnosticContext { groupId: number; account: PoolDiagnosticAccount }
export function poolDiagnosticPath(context: PoolDiagnosticContext, operation: string): string {
  return `/user/pool-dashboard/groups/${context.groupId}/accounts/${context.account.id}/${operation}`
}
export async function getPoolDiagnosticModels(context: PoolDiagnosticContext): Promise<ClaudeModel[]> {
  return (await apiClient.get<ClaudeModel[]>(poolDiagnosticPath(context, 'models'))).data
}
export async function getPoolDiagnosticReasoning(context: PoolDiagnosticContext, modelId: string) {
  return (await apiClient.get<{ supported_reasoning_levels: string[]; default_reasoning_level: string }>(
    poolDiagnosticPath(context, 'models/reasoning'), { params: { model_id: modelId } }
  )).data
}
export async function poolTestError(response: Response): Promise<Error> {
  const body = await response.json().catch(() => null)
  return new Error(typeof body?.message === 'string' ? body.message : `HTTP ${response.status}`)
}

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
  name: string
  type: Account['type']
  platform: Account['platform']
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
