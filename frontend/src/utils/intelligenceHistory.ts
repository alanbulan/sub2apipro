// Keep the existing administrator history key and legacy record shape.
// Ordinary-user diagnostics never call these functions.
export interface IntelligenceHistoryRun { model: string; kind: string; output: string; error: string; status: string }
export interface IntelligenceHistoryRecord { createdAt: string; modelId: string; questionKind: string; prompt: string; runs: IntelligenceHistoryRun[] }
const prefix = 'sub2api-pelican-test:'
const string = (value: unknown, limit: number) => typeof value === 'string' ? value.slice(0, limit) : ''
export function readIntelligenceHistory(accountId: number): IntelligenceHistoryRecord[] {
  try {
    const raw = localStorage.getItem(prefix + accountId)
    if (!raw || raw.length > 5 * 1024 * 1024) return []
    const records: unknown = JSON.parse(raw)
    if (!Array.isArray(records)) return []
    return records.slice(0, 8).filter(record => record && Array.isArray(record.runs)).map(record => ({
      createdAt: string(record.createdAt, 64), modelId: string(record.modelId, 200), questionKind: string(record.questionKind, 40) || 'pelican', prompt: string(record.prompt, 4096),
      runs: record.runs.slice(0, 8).map((run: Record<string, unknown>) => ({
        model: string(run.model ?? run.modelId ?? record.modelId, 200), kind: string(run.kind ?? run.questionKind ?? record.questionKind, 40) || 'pelican',
        output: string(run.output, 64 * 1024), error: string(run.error, 2048), status: run.status === 'success' ? 'success' : 'failed'
      }))
    }))
  } catch { return [] }
}
export function saveIntelligenceHistory(accountId: number, record: IntelligenceHistoryRecord): IntelligenceHistoryRecord[] {
  const records = [record, ...readIntelligenceHistory(accountId)].slice(0, 8)
  try { localStorage.setItem(prefix + accountId, JSON.stringify(records)) } catch { /* Current results remain visible when browser storage is full. */ }
  return readIntelligenceHistory(accountId)
}
