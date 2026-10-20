import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PoolDashboardView from '../PoolDashboardView.vue'
const { getPool } = vi.hoisted(() => ({ getPool: vi.fn() }))
vi.mock('@/api/poolDashboard', async (original) => ({ ...await original<typeof import('@/api/poolDashboard')>(), getPoolDashboard:getPool }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t:(key:string) => key }) }))
afterEach(() => { vi.useRealTimers(); vi.clearAllMocks() })
const account = { id:7,label:'#7',platform:'openai',available:true,windows:[],balance:null,balance_unit:'',balance_fresh_until:null,updated_at:null }
function render() { return mount(PoolDashboardView, { global: { stubs: {
  AppLayout:{template:'<div><slot/></div>'}, RouterLink:{template:'<a><slot/></a>'}
} } }) }
describe('user account pool', () => {
  it('groups quota cards, deduplicates totals and filters groups', async () => {
    getPool.mockResolvedValue({ generated_at:new Date().toISOString(),groups:[
      { id:1,name:'First source',platform:'openai',accounts:[account] },
      { id:2,name:'Second source',platform:'openai',accounts:[account] }
    ] })
    const wrapper = render(); await flushPromises()
    expect(wrapper.findAll('[data-test="pool-account"]')).toHaveLength(2)
    expect(wrapper.get('[data-test="metric-accounts"]').text()).toBe('1')
    await wrapper.get('[data-test="group-filter"]').setValue('2')
    expect(wrapper.find('[data-test="pool-group-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="pool-group-2"]').exists()).toBe(true)
    wrapper.unmount()
  })
  it('shows read failures and retries instead of presenting an empty healthy pool', async () => {
    getPool.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ generated_at:new Date().toISOString(),groups:[] })
    const wrapper = render(); await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="metric-accounts"]').exists()).toBe(false)
    await wrapper.get('[data-test="refresh-pool"]').trigger('click'); await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('poolDashboard.empty')
    wrapper.unmount()
  })
})
