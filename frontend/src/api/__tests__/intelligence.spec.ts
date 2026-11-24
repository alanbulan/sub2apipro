import { afterEach, describe, expect, it, vi } from 'vitest'
import { streamIntelligence } from '../intelligence'

function response(chunks: string[]) {
  const cancel = vi.fn().mockResolvedValue(undefined)
  const reader = { read: vi.fn(async () => chunks.length ? {done:false,value:new TextEncoder().encode(chunks.shift())} : {done:true}), cancel }
  return { response: {ok:true,body:{getReader:()=>reader}} as unknown as Response, cancel }
}
afterEach(()=>{vi.unstubAllGlobals();localStorage.clear();sessionStorage.clear()})
describe('intelligence stream boundaries',()=>{
  it('handles split frames and uses only scoped user headers',async()=>{
    const fixture=response(['data: {"type":"content","te','xt":"42"}\n\ndata: {"type":"test_complete","success":true}\n\n'])
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(fixture.response))
    sessionStorage.setItem('auth_storage_mode','session');sessionStorage.setItem('auth_token','session-only')
    const events: unknown[]=[]
    await streamIntelligence('/user/pool-dashboard/groups/1/accounts/2/pelican-test',{model_id:'model'},new AbortController().signal,event=>events.push(event),false)
    expect(events).toEqual([{type:'content',text:'42'},{type:'test_complete',success:true}])
    const options=vi.mocked(fetch).mock.calls[0][1]!
    expect(options.headers).toEqual({Authorization:'Bearer session-only','Content-Type':'application/json'})
    expect(fixture.cancel).toHaveBeenCalled()
  })
  it('does not treat partial output or an explicit error as success',async()=>{
    for(const chunks of [['data: {"type":"content","text":"partial"}\n\n'],['data: {"type":"error","error":"bounded safe failure"}\n\n']]){
      const fixture=response(chunks);vi.stubGlobal('fetch',vi.fn().mockResolvedValue(fixture.response))
      await expect(streamIntelligence('/admin/accounts/2/test-pelican',{},new AbortController().signal,()=>undefined)).rejects.toThrow()
      expect(fixture.cancel).toHaveBeenCalled()
    }
  })
})
