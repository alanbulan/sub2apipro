import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { runModelTrace, type ModelTraceResult } from '../modelTrace'
import { AUTH_TOKEN_KEY, setAuthItem, setAuthStorageMode } from '@/utils/authStorage'

const result = {
  requested_model: 'public-model', reasoning_effort: '', status: 'insufficient',
  bank: { source_commit: 'fixture', sha256: 'fixture', built_at: '', algorithm: 'fixture', models: [] },
  samples: [], disagreement: false, created_at: '2026-10-18T12:00:00Z'
} satisfies ModelTraceResult

function stream(chunks: string[]) {
  const encoder = new TextEncoder()
  const reader = {
    read: vi.fn(async () => chunks.length ? { value: encoder.encode(chunks.shift()!), done: false } : { value: undefined, done: true }),
    cancel: vi.fn(async () => undefined), releaseLock: vi.fn()
  }
  return { response: { ok: true, body: { getReader: () => reader } } as unknown as Response, reader }
}

beforeEach(() => { localStorage.clear(); sessionStorage.clear(); localStorage.setItem('auth_token', 'mock-admin-token') })
afterEach(() => { vi.unstubAllGlobals(); localStorage.clear(); sessionStorage.clear() })

describe('ModelTrace stream', () => {
  it('handles split events and keepalives and requires a completed round', async () => {
    const { response, reader } = stream([': keepalive\n\nda', `ta: ${JSON.stringify({ type: 'modeltrace_complete', data: result })}\n\n`])
    const fetchMock = vi.fn().mockResolvedValue(response)
    vi.stubGlobal('fetch', fetchMock)
    const event = vi.fn()
    await expect(runModelTrace(42, 'public-model', '', new AbortController().signal, event)).resolves.toEqual(result)
    expect(event).toHaveBeenCalledTimes(1)
    expect(reader.cancel).toHaveBeenCalled()
    expect(reader.releaseLock).toHaveBeenCalled()
    const [url, request] = fetchMock.mock.calls[0]!
    expect(url).toContain('/admin/accounts/42/modeltrace-test')
    expect(request.headers.Authorization).toBe('Bearer mock-admin-token')
    expect(JSON.parse(request.body)).toEqual({ model_id: 'public-model', reasoning_effort: '' })
    expect(request.body).not.toContain('mock-admin-token')
  })

  it.each([false, true])('uses selected auth storage with rememberMe=%s', async (rememberMe) => {
    setAuthStorageMode(rememberMe)
    setAuthItem(AUTH_TOKEN_KEY, 'selected-admin-token')
    const otherStorage = rememberMe ? sessionStorage : localStorage
    otherStorage.setItem(AUTH_TOKEN_KEY, 'stale-other-token')
    const { response } = stream([`data: ${JSON.stringify({ type: 'modeltrace_complete', data: result })}\n\n`])
    const fetchMock = vi.fn().mockResolvedValue(response)
    vi.stubGlobal('fetch', fetchMock)
    await runModelTrace(42, 'public-model', '', new AbortController().signal, vi.fn())
    expect(fetchMock.mock.calls[0]![1].headers.Authorization).toBe('Bearer selected-admin-token')
  })

  it('rejects early EOF and does not turn a keepalive into a result', async () => {
    const { response, reader } = stream([': keepalive\n\n'])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
    await expect(runModelTrace(42, 'gpt-6-astra', '', new AbortController().signal, vi.fn())).rejects.toThrow('INCOMPLETE_STREAM')
    expect(reader.cancel).toHaveBeenCalled()
  })

  it('does not expose error response bodies', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 409, text: () => 'sensitive provider body' }))
    await expect(runModelTrace(42, 'gpt-6-astra', '', new AbortController().signal, vi.fn())).rejects.toThrow('HTTP 409')
  })

  it('rejects completion after cancellation', async () => {
    const active = new AbortController()
    const { response } = stream([`data: ${JSON.stringify({ type: 'modeltrace_complete', data: result })}\n\n`])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
    await expect(runModelTrace(42, 'gpt-6-astra', '', active.signal, () => active.abort())).rejects.toMatchObject({ name: 'AbortError' })
  })
})
