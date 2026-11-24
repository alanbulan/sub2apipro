import { describe, expect, it } from 'vitest'
import { balanceIsCurrent, quotaIsCurrent, type PoolQuotaAccount, type PoolQuotaWindow } from '../poolDashboard'
describe('pool snapshot freshness', () => {
  it('expires elapsed windows and stale samples without assuming a reset', () => {
    const now = Date.now()
    const window: PoolQuotaWindow = { window:'5h',label:'5h',status:'ok',used_percent:0,observed_at:new Date(now-60000).toISOString(),resets_at:new Date(now+60000).toISOString() }
    expect(quotaIsCurrent(window,now)).toBe(true)
    expect(quotaIsCurrent(window,now+120000)).toBe(false)
    expect(quotaIsCurrent({ ...window, observed_at:new Date(now-16*60000).toISOString() },now)).toBe(false)
    expect(quotaIsCurrent({ ...window, used_percent:NaN },now)).toBe(false)
  })
  it('does not show expired balances', () => {
    const now = Date.now()
    const account = { balance:0,balance_fresh_until:new Date(now+1000).toISOString() } as PoolQuotaAccount
    expect(balanceIsCurrent(account,now)).toBe(true)
    expect(balanceIsCurrent(account,now+2000)).toBe(false)
  })
})
