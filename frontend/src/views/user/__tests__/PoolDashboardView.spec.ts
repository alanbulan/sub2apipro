import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
import PoolDashboardView from '../PoolDashboardView.vue'
import AccountTestModal from '@/components/admin/account/AccountTestModal.vue'
import IQTestModal from '@/components/admin/account/IQTestModal.vue'

const { getPool } = vi.hoisted(() => ({ getPool: vi.fn() }))
vi.mock('@/api/poolDashboard', async (original) => ({
  ...await original<typeof import('@/api/poolDashboard')>(),
  getPoolDashboard: getPool
}))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key })
}))

const account = { id: 7, name: 'Public source', type: 'oauth', platform: 'openai', available: true, windows: [], balance: null, balance_unit: '', balance_fresh_until: null, updated_at: null }
let wrapper: ReturnType<typeof mount> | undefined
function render() {
  wrapper = mount(PoolDashboardView, {
    attachTo: document.body,
    global: { stubs: {
      AppLayout: { template: '<div><slot/></div>' },
      AccountTestModal: true, IQTestModal: true,
      RouterLink: { template: '<a><slot/></a>' }
    } }
  })
  return wrapper
}
beforeEach(() => localStorage.removeItem('user-account-pool-refresh'))
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  localStorage.removeItem('user-account-pool-refresh')
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe('user account pool', () => {
  it('opens the reused modals for the clicked account and its source group from the single action row', async () => {
    getPool.mockResolvedValue({ generated_at: new Date().toISOString(), groups: [
      { id: 9, name: 'Source group', platform: 'openai', accounts: [account] }
    ] })
    const view = render()
    await flushPromises()
    for (const button of view.findAll('[data-test="test-connection"]')) {
      await button.trigger('click')
      const modal = view.getComponent(AccountTestModal)
      expect(modal.props('show')).toBe(true)
      expect(modal.props('poolContext')).toEqual({ groupId: 9, account: {
        id: 7, name: 'Public source', type: 'oauth', platform: 'openai', status: 'active'
      } })
      modal.vm.$emit('close')
      await flushPromises()
      expect(modal.props('show')).toBe(false)
    }
    for (const button of view.findAll('[data-test="test-intelligence"]')) {
      await button.trigger('click')
      expect(view.getComponent(IQTestModal).props('show')).toBe(true)
      expect(view.getComponent(IQTestModal).props('poolContext')?.groupId).toBe(9)
    }
  })
  it('deduplicates totals and filters groups through the shared dropdown inside the fullscreen container', async () => {
    getPool.mockResolvedValue({ generated_at: new Date().toISOString(), groups: [
      { id: 1, name: 'First source', platform: 'openai', accounts: [account] },
      { id: 2, name: 'Second source', platform: 'openai', accounts: [account] }
    ] })
    const view = render()
    await flushPromises()
    expect(view.findAll('[data-test="pool-account"]')).toHaveLength(2)
    expect(view.get('[data-test="metric-accounts"] .stat-value').text()).toBe('1')
    await view.get('[data-test="group-filter"] button').trigger('click')
    const dropdown = view.get('[data-test="pool-screen"] [role="listbox"]')
    expect(dropdown.exists()).toBe(true)
    const second = dropdown.findAll('[role="option"]').find(option => option.text() === 'Second source')!
    await second.trigger('click')
    expect(view.find('[data-test="pool-group-1"]').exists()).toBe(false)
    expect(view.find('[data-test="pool-group-2"]').exists()).toBe(true)
    expect(view.get('[data-test="group-filter"] button').text()).toContain('Second source')
  })

  it('shows read failures and retries instead of presenting an empty healthy pool', async () => {
    getPool.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ generated_at: new Date().toISOString(), groups: [] })
    const view = render()
    await flushPromises()
    expect(view.find('[role="alert"]').exists()).toBe(true)
    expect(view.find('[data-test="metric-accounts"]').exists()).toBe(false)
    await view.get('[data-test="refresh-pool"]').trigger('click')
    await flushPromises()
    expect(view.find('[role="alert"]').exists()).toBe(false)
    expect(view.text()).toContain('poolDashboard.empty')
  })

  it('uses the shared refresh controls, pauses hidden pages and remembers when refresh is disabled', async () => {
    vi.useFakeTimers()
    let hidden = false
    vi.spyOn(document, 'hidden', 'get').mockImplementation(() => hidden)
    getPool.mockResolvedValue({ generated_at: new Date().toISOString(), groups: [] })
    const view = render()
    await flushPromises()
    const refresh = view.getComponent(AutoRefreshButton)
    expect(refresh.props('countdown')).toBe(30)
    refresh.vm.$emit('update:interval', 60)
    await flushPromises()
    await vi.advanceTimersByTimeAsync(59000)
    expect(getPool).toHaveBeenCalledTimes(1)
    hidden = true
    await vi.advanceTimersByTimeAsync(60000)
    expect(getPool).toHaveBeenCalledTimes(1)
    hidden = false
    await vi.advanceTimersByTimeAsync(1000)
    expect(getPool).toHaveBeenCalledTimes(2)
    refresh.vm.$emit('update:enabled', false)
    await flushPromises()
    await vi.advanceTimersByTimeAsync(120000)
    expect(getPool).toHaveBeenCalledTimes(2)
    expect(JSON.parse(localStorage.getItem('user-account-pool-refresh')!)).toEqual({ enabled: false, interval_seconds: 60 })
    await view.get('[data-test="refresh-pool"]').trigger('click')
    await flushPromises()
    expect(getPool).toHaveBeenCalledTimes(3)
  })

  it('paginates accounts and resets a removed filter when the pool changes', async () => {
    getPool.mockResolvedValueOnce({ generated_at: new Date().toISOString(), groups: [
      { id: 1, name: 'First source', platform: 'openai', accounts: [account] },
      { id: 2, name: 'Second source', platform: 'openai', accounts: Array.from({ length: 25 }, (_, i) => ({ ...account, id: i + 100, name: `Source ${i + 100}` })) }
    ] }).mockResolvedValueOnce({ generated_at: new Date().toISOString(), groups: [
      { id: 1, name: 'First source', platform: 'openai', accounts: [account] }
    ] })
    const view = render()
    await flushPromises()
    const secondGroup = view.get('[data-test="pool-group-2"]')
    expect(secondGroup.findAll('[data-test="pool-account"]')).toHaveLength(24)
    await secondGroup.get('button[aria-label="pagination.next"]').trigger('click')
    expect(secondGroup.findAll('[data-test="pool-account"]')).toHaveLength(1)
    expect(secondGroup.text()).toContain('Source 124')
    await view.get('[data-test="group-filter"] button').trigger('click')
    await view.get('[role="listbox"]').findAll('[role="option"]')[2].trigger('click')
    await view.get('[data-test="refresh-pool"]').trigger('click')
    await flushPromises()
    expect(view.find('[data-test="pool-group-2"]').exists()).toBe(false)
    expect(view.find('[data-test="pool-group-1"]').exists()).toBe(true)
    expect(view.get('[data-test="group-filter"] button').text()).toContain('poolDashboard.allGroups')
  })
})
