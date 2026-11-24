import { apiClient } from '../client'

export interface FeishuKeyConfig {
  version: number
  group_ids: number[]
  initial_balance: number
  quota: number
  expires_in_days: number
  concurrency: number
  rate_limit_5h: number
  rate_limit_1d: number
  rate_limit_7d: number
  allow_contact_email: boolean
  name_suffix: string
  subject: string
  introduction: string
  api_url: string
}
export interface FeishuKeySettings { config: FeishuKeyConfig; smtp_configured: boolean }
export interface FeishuKeyRecipient {
  user_id: number; name: string; email: string; status: string; reason?: string
  key_id?: number; reuse: boolean; grant: number; balance: number
  group_names: string[]; quota: number; expires_at?: string
}
export interface FeishuKeyJob {
  id: string; status: 'preview' | 'running' | 'completed' | 'partial'
  rows: FeishuKeyRecipient[]; ready: number; sent: number; failed: number; skipped: number; uncertain: number
  total_grant: number; created_at: number; expires_at: number
}
const base = '/admin/users/feishu-key-delivery'
export async function getFeishuKeyConfig(): Promise<FeishuKeySettings> { return (await apiClient.get<FeishuKeySettings>(`${base}/config`)).data }
export async function saveFeishuKeyConfig(config: FeishuKeyConfig): Promise<FeishuKeySettings> { return (await apiClient.put<FeishuKeySettings>(`${base}/config`, config)).data }
export async function previewFeishuKeys(user_ids: number[], all: boolean): Promise<FeishuKeyJob> { return (await apiClient.post<FeishuKeyJob>(`${base}/preview`, { user_ids, all }, { timeout: 125000 })).data }
export async function currentFeishuKeyJob(): Promise<FeishuKeyJob | null> { return (await apiClient.get<FeishuKeyJob | null>(base)).data }
export async function startFeishuKeys(preview_id: string): Promise<FeishuKeyJob> { return (await apiClient.post<FeishuKeyJob>(base, { preview_id })).data }
export async function retryFeishuKeys(job_id: string, include_uncertain: boolean): Promise<FeishuKeyJob> { return (await apiClient.post<FeishuKeyJob>(`${base}/retry`, { job_id, include_uncertain })).data }
