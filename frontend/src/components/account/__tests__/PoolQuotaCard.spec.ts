import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PoolQuotaCard from '../PoolQuotaCard.vue'
import type { PoolQuotaAccount } from '@/api/poolDashboard'
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t:(key:string) => key })
}))
describe('user quota card', () => {
  it('reuses the quota progress component with remaining capacity and ignores private fields', () => {
    const now = Date.now()
    const account = { id:1,label:'#1',platform:'openai',available:true,balance:null,balance_unit:'',balance_fresh_until:null,updated_at:null,
      windows:[{ window:'5h',label:'5h',status:'ok',used_percent:25,observed_at:new Date(now-1000).toISOString(),resets_at:new Date(now+60000).toISOString() }],
      name:'private@example.com',credentials:{ access_token:'private-token' }
    } as PoolQuotaAccount
    const wrapper = mount(PoolQuotaCard, { props:{account,now}, global:{ stubs:{ UsageProgressBar:true } } })
    expect(wrapper.findComponent({name:'UsageProgressBar'}).props('utilization')).toBe(75)
    expect(wrapper.text()).not.toContain('private@example.com')
    expect(wrapper.html()).not.toContain('private-token')
    wrapper.unmount()
  })
})
