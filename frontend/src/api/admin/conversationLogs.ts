import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface ConversationLog {
  id: number
  request_id: string
  session_id: string
  user_id: number
  user_email: string
  api_key_id: number
  api_key_name: string
  group_id?: number | null
  protocol: string
  endpoint: string
  model: string
  stream: boolean
  status: 'pending' | 'completed' | 'failed' | 'incomplete'
  status_code: number
  content_type: string
  request_body?: string
  response_body?: string
  request_bytes: number
  response_bytes: number
  request_truncated: boolean
  response_truncated: boolean
  duration_ms: number
  created_at: string
  completed_at?: string | null
}

export interface ConversationLogQuery {
  page?: number
  page_size?: number
  user_id?: number
  api_key_id?: number
  session_id?: string
  request_id?: string
  model?: string
  protocol?: string
  status?: string
  start_time?: string
  end_time?: string
}

export async function listConversationLogs(params: ConversationLogQuery): Promise<PaginatedResponse<ConversationLog>> {
  const { data } = await apiClient.get<PaginatedResponse<ConversationLog>>('/admin/conversation-logs', { params })
  return data
}

export async function getConversationLog(id: number): Promise<ConversationLog> {
  const { data } = await apiClient.get<ConversationLog>(`/admin/conversation-logs/${id}`)
  return data
}

export const adminConversationLogsAPI = { list: listConversationLogs, get: getConversationLog }

export default adminConversationLogsAPI
