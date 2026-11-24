import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import FeishuKeyDelivery from '../FeishuKeyDelivery.vue'

const api = vi.hoisted(() => ({ config: vi.fn(), current: vi.fn(), preview: vi.fn(), start: vi.fn(), retry: vi.fn(), error: vi.fn() }))
vi.mock('@/api/admin/feishuKeyDelivery', () => ({ getFeishuKeyConfig: api.config, currentFeishuKeyJob: api.current, previewFeishuKeys: api.preview, startFeishuKeys: api.start, retryFeishuKeys: api.retry }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: api.error }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const Dialog = defineComponent({ props: ['show'], template: '<section v-if="show"><slot/><slot name="footer"/></section>' })
const Settings = defineComponent({ props: ['show'], template: '<div v-if="show" data-test="settings-dialog" />' })
const makeJob = (status = 'preview') => ({ id: 'a'.repeat(40), status, rows: [{ user_id: 7, name: 'Employee', email: 'staff@example.com', status: 'ready', group_names: ['Pool'], grant: 10, balance: 0, reuse: false }], ready: 1, sent: 0, skipped: 0, failed: 0, uncertain: 0, total_grant: 10, expires_at: Date.now() / 1000 + 900 })
const wrappers: ReturnType<typeof mount>[] = []
function render(ids = [7]) {
  const wrapper = mount(FeishuKeyDelivery, { props: { selectedIds: ids }, global: { stubs: { BaseDialog: Dialog, FeishuKeyDeliverySettings: Settings, Icon: true, Select: true, Toggle: true } } })
  wrappers.push(wrapper)
  return wrapper
}
beforeEach(() => {
  vi.resetAllMocks()
  api.config.mockResolvedValue({ smtp_configured: true, config: { version: 1, group_ids: [1] } })
  api.current.mockResolvedValue(null)
  api.preview.mockResolvedValue(makeJob())
  api.start.mockResolvedValue(makeJob('running'))
})
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); vi.useRealTimers() })
describe('Feishu key delivery', () => {
  it('requires a server preview, then submits only its ID; a lost response retries the same preview', async () => {
    const wrapper = render()
    await flushPromises()
    await wrapper.get('[data-test="feishu-key-open"]').trigger('click'); await flushPromises()
    expect(wrapper.find('[data-test="feishu-key-send"]').exists()).toBe(false)
    expect(api.start).not.toHaveBeenCalled()
    await wrapper.get('[data-test="feishu-key-preview"]').trigger('click'); await flushPromises()
    expect(api.preview).toHaveBeenCalledWith([7], false)
    api.start.mockRejectedValueOnce(new Error('Response lost'))
    await wrapper.get('[data-test="feishu-key-send"]').trigger('click'); await flushPromises()
    expect(api.error).toHaveBeenCalled()
    await wrapper.get('[data-test="feishu-key-send"]').trigger('click'); await flushPromises()
    expect(api.start.mock.calls).toEqual([['a'.repeat(40)], ['a'.repeat(40)]])
  })
  it('uses all enterprise users when nothing is selected and clears previews when the selection changes', async () => {
    const wrapper = render([]); await flushPromises()
    await wrapper.get('[data-test="feishu-key-open"]').trigger('click'); await flushPromises()
    await wrapper.get('[data-test="feishu-key-preview"]').trigger('click'); await flushPromises()
    expect(api.preview).toHaveBeenCalledWith([], true)
    await wrapper.setProps({ selectedIds: [8] })
    expect(wrapper.find('[data-test="feishu-key-send"]').exists()).toBe(false)
    expect(api.start).not.toHaveBeenCalled()
  })
  it('blocks sending when SMTP is not configured and opens visible settings', async () => {
    api.config.mockResolvedValue({ smtp_configured: false, config: { group_ids: [1] } })
    const wrapper = render(); await flushPromises()
    await wrapper.get('[data-test="feishu-key-open"]').trigger('click'); await flushPromises()
    await wrapper.get('[data-test="feishu-key-preview"]').trigger('click'); await flushPromises()
    expect(wrapper.get('[data-test="feishu-key-send"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-test="feishu-key-settings"]').trigger('click')
    expect(wrapper.find('[data-test="settings-dialog"]').exists()).toBe(true)
    expect(api.start).not.toHaveBeenCalled()
  })
  it('resumes persisted progress and refreshes users on completion without sending again', async () => {
    vi.useFakeTimers()
    api.current.mockResolvedValueOnce(makeJob('running')).mockResolvedValue({ ...makeJob('completed'), ready: 0, sent: 1 })
    const wrapper = render(); await flushPromises()
    await vi.advanceTimersByTimeAsync(2000); await flushPromises()
    expect(wrapper.emitted('updated')).toEqual([[]])
    expect(api.start).not.toHaveBeenCalled()
  })
})
