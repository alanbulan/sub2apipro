import { buildApiUrl } from './client'
import { getAuthItem, AUTH_TOKEN_KEY } from '@/utils/authStorage'
import { ADMIN_UI_REQUEST_HEADER } from './adminUIRequest'

export interface IntelligenceEvent { type: string; text?: string; error?: string; success?: boolean }

// SSE frames can cross arbitrary network chunks. Cap input before retaining it
// and require an explicit successful completion instead of treating EOF as success.
export async function streamIntelligence(path: string, body: object, signal: AbortSignal, onEvent: (event: IntelligenceEvent) => void, admin = true) {
  const headers: Record<string, string> = { Authorization: `Bearer ${getAuthItem(AUTH_TOKEN_KEY) || ''}`, 'Content-Type': 'application/json' }
  if (admin) headers[ADMIN_UI_REQUEST_HEADER] = '1'
  const response = await fetch(buildApiUrl(path), { method: 'POST', credentials: 'include', headers, body: JSON.stringify(body), signal })
  if (!response.ok) {
    const error = await response.json().catch(() => null)
    throw new Error(typeof error?.message === 'string' ? error.message : `HTTP ${response.status}`)
  }
  const reader = response.body?.getReader()
  if (!reader) throw new Error('Empty test stream')
  const decoder = new TextDecoder()
  let buffer = '', bytes = 0, completed = false
  const consume = (line: string) => {
    if (!line.startsWith('data:')) return
    const raw = line.slice(5).trim()
    if (!raw || raw === '[DONE]') return
    const event = JSON.parse(raw) as IntelligenceEvent
    if (event.type === 'error') throw new Error(event.error || 'Test failed')
    if (event.type === 'test_complete') { if (!event.success) throw new Error('Test failed'); completed = true }
    onEvent(event)
  }
  try {
    while (true) {
      const chunk = await reader.read()
      if (chunk.done) break
      bytes += chunk.value.byteLength
      if (bytes > 2 * 1024 * 1024) throw new Error('Test output exceeded the size limit')
      buffer += decoder.decode(chunk.value, { stream: true })
      const lines = buffer.split(/\r?\n/)
      buffer = lines.pop() || ''
      lines.forEach(consume)
    }
    buffer += decoder.decode()
    if (buffer.trim()) consume(buffer)
    if (!completed) throw new Error('Test stream ended before completion')
  } finally { try { await reader.cancel() } catch { /* The stream may already be closed. */ } }
}
