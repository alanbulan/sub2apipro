import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BulkRechargeUserModal from '../BulkRechargeUserModal.vue'
const { batchRecharge } = vi.hoisted(() => ({ batchRecharge: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { users: { batchRecharge } } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('@/utils/authStorage', () => ({ AUTH_USER_KEY: 'auth_user', getAuthItem: () => '{"id":7}' }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
function render() {
  return mount(BulkRechargeUserModal, { props: { show: true, selectedIds: [2,1,1] }, global: { stubs: {
    BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' }
  } } })
}
describe('bulk recharge retry safety', () => {
  beforeEach(() => { localStorage.clear(); batchRecharge.mockReset(); vi.spyOn(window, 'confirm').mockReturnValue(true) })
  it('reuses the persisted operation after a lost response and a remount', async () => {
    batchRecharge.mockRejectedValueOnce(new Error('lost response'))
    const first = render()
    await first.get('#recharge-amount').setValue('5')
    await first.get('form').trigger('submit'); await flushPromises()
    const [body,key] = batchRecharge.mock.calls[0]!
    expect(body).toEqual({ user_ids:[1,2], amount:5, notes:'' })
    expect(first.emitted('success')).toBeUndefined()
    first.unmount()
    batchRecharge.mockResolvedValueOnce({ affected:2 })
    const second = render()
    expect(second.get('#recharge-amount').attributes('disabled')).toBeDefined()
    await second.get('form').trigger('submit'); await flushPromises()
    expect(batchRecharge).toHaveBeenLastCalledWith(body,key)
    expect(second.emitted('success')).toEqual([[2]])
    expect(localStorage.getItem('batch-recharge-pending:7')).toBeNull()
    second.unmount()
  })
})
