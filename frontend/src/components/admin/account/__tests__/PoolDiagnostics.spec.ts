import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia } from 'pinia'
import AccountTestModal from '../AccountTestModal.vue'
import IQTestModal from '../IQTestModal.vue'
import type { PoolDiagnosticContext } from '@/api/poolDashboard'

const { models, reasoning, adminModels } = vi.hoisted(() => ({ models: vi.fn(), reasoning: vi.fn(), adminModels: vi.fn() }))
vi.mock('@/api/poolDashboard', async original => ({
  ...await original<typeof import('@/api/poolDashboard')>(),
  getPoolDiagnosticModels: models,
  getPoolDiagnosticReasoning: reasoning
}))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { getAvailableModels: adminModels } } }))
vi.mock('vue-i18n', async original => ({
  ...await original<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key })
}))

const context: PoolDiagnosticContext = { groupId: 12, account: { id: 42, name: 'Pool source', platform: 'openai', type: 'oauth', status: 'active' } }
const stubs = { BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' }, Select: true, TextArea: true, Input: true, Icon: true }
function streamResponse() {
  const bytes = new TextEncoder().encode('data: {"type":"content","text":"42"}\n\ndata: {"type":"test_complete","success":true}\n\n')
  let done = false
  return { ok: true, body: { getReader: () => ({ read: async () => {
    if (done) return { done: true }
    done = true
    return { done: false, value: bytes }
  } }) } } as Response
}
beforeEach(() => {
  vi.clearAllMocks()
  models.mockResolvedValue([{ id: 'gpt-6-astra', type: 'model', display_name: 'gpt-6-astra', created_at: '' }])
  reasoning.mockResolvedValue({ supported_reasoning_levels: ['low', 'medium'], default_reasoning_level: 'medium' })
  localStorage.setItem('auth_token', 'user-token')
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(streamResponse()))
})
afterEach(() => {
  localStorage.clear()
  vi.unstubAllGlobals()
})

describe('ordinary user diagnostics reuse existing modals', () => {
  it('uses the scoped connection API and omits admin-only modes and headers', async () => {
    const wrapper = mount(AccountTestModal, { props: { show: false, poolContext: context }, global: { stubs, plugins: [createPinia()] } })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(models).toHaveBeenCalledWith(context)
    expect(adminModels).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('admin.accounts.openai.testMode')
    await (wrapper.vm as any).startTest()
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toMatch(/\/user\/pool-dashboard\/groups\/12\/accounts\/42\/test$/)
    expect(JSON.parse(options!.body as string)).toEqual({ model_id: 'gpt-6-astra' })
    expect(Object.keys(options!.headers!)).toEqual(['Authorization', 'Content-Type'])
    wrapper.unmount()
  })

  it('uses fixed IQ questions, a single run and scoped model/reasoning APIs', async () => {
    const wrapper = mount(IQTestModal, { props: { show: false, poolContext: context }, global: { stubs, plugins: [createPinia()] } })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.find('[data-testid="mode-probe"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="mode-modeltrace"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="model-input-toggle"]').exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'TextArea' }).attributes('disabled')).toBeDefined()
    expect(reasoning).toHaveBeenCalledWith(context, 'gpt-6-astra')
    ;(wrapper.vm as any).parallelCount = 8
    await (wrapper.vm as any).startTest()
    expect(fetch).toHaveBeenCalledTimes(1)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toMatch(/\/user\/pool-dashboard\/groups\/12\/accounts\/42\/pelican-test$/)
    expect(JSON.parse(options!.body as string)).toEqual({ model_id: 'gpt-6-astra', question_kind: 'candy', reasoning_effort: 'medium' })
    expect(localStorage.getItem('sub2api-pelican-test:42')).toBeNull()
    wrapper.unmount()
  })

  it('surfaces a safe backend rate-limit message in the reused connection modal', async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: false, status: 429, json: async () => ({ message: '测试请求较多，请稍后再试' }) } as Response)
    const wrapper = mount(AccountTestModal, { props: { show: false, poolContext: context }, global: { stubs, plugins: [createPinia()] } })
    await wrapper.setProps({ show: true })
    await flushPromises()
    await (wrapper.vm as any).startTest()
    expect(wrapper.text()).toContain('测试请求较多，请稍后再试')
    wrapper.unmount()
  })
})
