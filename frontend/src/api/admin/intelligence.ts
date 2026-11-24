import { apiClient } from '../client'

export type OpenAICodexStateVerdict = 'healthy' | 'degraded' | 'inconclusive'

export interface OpenAICodexStateProbeResult {
  account_id: number
  model: string
  verdict: OpenAICodexStateVerdict
  reason: string
  failure?: string
  detail?: string
  mint_status: number
  continue_status: number
  minted: boolean
  new_ticket: boolean
  ticket_length: number
  continue_ticket_length: number
  reported_model?: string
  latency_ms: number
  started_at: string
  finished_at: string
}

/**
 * Two-shot Codex turn-state probe: mint a ticket, then continue with it. A new
 * ticket on the continuation means the account is degraded. Each shot may take
 * up to 45s upstream, so the default client timeout is too short.
 */
export async function probeOpenAICodexState(
  id: number,
  modelId?: string,
  options?: { signal?: AbortSignal }
): Promise<OpenAICodexStateProbeResult> {
  const { data } = await apiClient.post<OpenAICodexStateProbeResult>(
    `/admin/accounts/${id}/probe-state`,
    { model_id: modelId?.trim() || undefined },
    { timeout: 120_000, signal: options?.signal }
  )
  return data
}


export interface AccountTestReasoning {
  supported_reasoning_levels: string[]
  default_reasoning_level: string
}

export async function getModelReasoning(id: number, modelId: string): Promise<AccountTestReasoning> {
  const { data } = await apiClient.get<AccountTestReasoning>(`/admin/accounts/${id}/models/reasoning`, {
    params: { model_id: modelId }
  })
  return data
}
