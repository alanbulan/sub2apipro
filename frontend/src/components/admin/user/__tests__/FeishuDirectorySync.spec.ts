import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import FeishuDirectorySync from '../FeishuDirectorySync.vue'
const { getSync, startSync } = vi.hoisted(() => ({ getSync:vi.fn(), startSync:vi.fn() }))
vi.mock('@/api/admin/feishu', () => ({ getFeishuDirectorySync:getSync, startFeishuDirectorySync:startSync }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError:vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t:(key:string) => key }) }))
afterEach(() => { vi.useRealTimers(); vi.clearAllMocks() })
describe('Feishu directory task', () => {
  it('resumes a running task and refreshes users on completion', async () => {
    vi.useFakeTimers()
    getSync.mockResolvedValueOnce({ id:'task',status:'syncing',processed:1,total:2 })
      .mockResolvedValueOnce({ id:'task',status:'succeeded',processed:2,total:2 })
    const wrapper = mount(FeishuDirectorySync)
    await flushPromises()
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    await vi.advanceTimersByTimeAsync(2000); await flushPromises()
    expect(wrapper.emitted('synced')).toEqual([[]])
    expect(startSync).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
