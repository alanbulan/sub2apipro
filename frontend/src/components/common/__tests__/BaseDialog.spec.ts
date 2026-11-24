import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BaseDialog from '../BaseDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

describe('BaseDialog', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    Reflect.deleteProperty(document, 'fullscreenElement')
    document.body.classList.remove('modal-open')
    vi.restoreAllMocks()
  })

  it('keeps the dialog in the active fullscreen element', async () => {
    const target = document.createElement('section')
    document.body.appendChild(target)
    Object.defineProperty(document, 'fullscreenElement', { configurable: true, value: target })
    const wrapper = mount(BaseDialog, {
      props: { show: false, title: 'Account test' },
      global: { stubs: { Icon: true } }
    })
    await wrapper.setProps({ show: true })
    expect(target.querySelector('[role="dialog"]')).not.toBeNull()
    wrapper.unmount()
  })

  it('resets body scroll position when reopened', async () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: false, title: 'Details' },
      slots: { default: '<div style="height: 2000px">content</div>' },
      global: { stubs: { Icon: true } }
    })

    await wrapper.setProps({ show: true })
    await nextTick()
    const body = document.body.querySelector<HTMLElement>('.modal-body')
    expect(body).not.toBeNull()
    body!.scrollTop = 480

    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await nextTick()

    expect(document.body.querySelector<HTMLElement>('.modal-body')?.scrollTop).toBe(0)
    wrapper.unmount()
  })
})
