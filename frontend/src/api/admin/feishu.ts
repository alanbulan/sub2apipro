import { apiClient } from '../client'

export interface FeishuSettings {
  enabled: boolean
  app_id: string
  app_secret_configured: boolean
  tenant_key: string
  tenant_name: string
  redirect_url: string
}
export interface FeishuSetup {
  id?: string
  verification_url?: string
  expires_at?: number
  interval: number
  status: 'pending' | 'submitted'
  settings?: FeishuSettings
}
const base = '/admin/settings/feishu'
export async function getFeishuSettings(): Promise<FeishuSettings> {
  return (await apiClient.get<FeishuSettings>(base)).data
}
export async function setFeishuEnabled(enabled: boolean): Promise<FeishuSettings> {
  return (await apiClient.put<FeishuSettings>(base, { enabled })).data
}
export async function enableFeishuEmailSync(): Promise<FeishuSettings> {
  return (await apiClient.post<FeishuSettings>(`${base}/email-sync`, undefined, { timeout: 70000 })).data
}
export async function startFeishuSetup(): Promise<FeishuSetup> {
  return (await apiClient.post<FeishuSetup>(`${base}/setup/start`, { site_origin: window.location.origin })).data
}
export async function pollFeishuSetup(id: string): Promise<FeishuSetup> {
  return (await apiClient.post<FeishuSetup>(`${base}/setup/${encodeURIComponent(id)}/poll`, undefined, { timeout: 100000 })).data
}
