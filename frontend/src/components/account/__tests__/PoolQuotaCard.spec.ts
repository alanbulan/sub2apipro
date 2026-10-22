import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PoolQuotaCard from '../PoolQuotaCard.vue'
import type { PoolQuotaAccount } from '@/api/poolDashboard'
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t:(key:string) => key })
}))
describe('user quota card', () => {
  it('renders two shared action rows and emits the same actions from either row', async () => {
    const account: PoolQuotaAccount = { id: 1, name: 'Source name', type: 'oauth', platform: 'openai', available: true,
      balance: null, balance_unit: '', balance_fresh_until: null, updated_at: null, windows: [] }
    const wrapper = mount(PoolQuotaCard, { props: { account, now: Date.now() } })
    expect(wrapper.get('h3').text()).toBe('Source name')
    expect(wrapper.findAll('[data-test="account-diagnostic-actions"]')).toHaveLength(2)
    for (const button of wrapper.findAll('[data-test="test-connection"]')) await button.trigger('click')
    for (const button of wrapper.findAll('[data-test="test-intelligence"]')) await button.trigger('click')
    expect(wrapper.emitted('test')).toHaveLength(2)
    expect(wrapper.emitted('intelligence')).toHaveLength(2)
    await wrapper.setProps({ account: { ...account, name: '' } })
    expect(wrapper.get('h3').text()).toBe('poolDashboard.unnamedAccount')
    expect(wrapper.text()).not.toContain('#1')
    wrapper.unmount()
  })
  it('reuses the quota progress component with remaining capacity and ignores private fields', () => {
    const now = Date.now()
    const account = { id:1,type:'oauth',name:'Production source',platform:'openai',available:true,balance:null,balance_unit:'',balance_fresh_until:null,updated_at:null,
      windows:[{ window:'5h',label:'5h',status:'ok',used_percent:25,observed_at:new Date(now-1000).toISOString(),resets_at:new Date(now+60000).toISOString() }],
      credentials:{ access_token:'private-token', email:'private@example.com' }
    } as PoolQuotaAccount
    const wrapper = mount(PoolQuotaCard, { props:{account,now}, global:{ stubs:{ UsageProgressBar:true } } })
    expect(wrapper.findComponent({name:'UsageProgressBar'}).props('utilization')).toBe(75)
    expect(wrapper.get('h3').text()).toBe('Production source')
    expect(wrapper.text()).not.toContain('private@example.com')
    expect(wrapper.html()).not.toContain('private-token')
    wrapper.unmount()
  })
})
