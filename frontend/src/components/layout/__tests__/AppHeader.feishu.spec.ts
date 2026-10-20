import { enableAutoUnmount, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import AppHeader from '../AppHeader.vue'
import { useAuthStore } from '@/stores/auth'
import zh from '@/i18n/locales/zh'
import type { User } from '@/types'

enableAutoUnmount(afterEach)

const internalEmail = `${'a'.repeat(64)}@feishu-connect.invalid`

function createUser(overrides: Partial<User> = {}): User {
  return {
    id: 1,
    username: '测试员工',
    email: internalEmail,
    email_bound: false,
    feishu_bound: true,
    role: 'user',
    status: 'active',
    balance: 0,
    concurrency: 1,
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

async function mountHeader(overrides: Partial<User> = {}) {
  useAuthStore().user = createUser(overrides)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: ['/dashboard', '/profile', '/keys'].map(path => ({ path, component: { template: '<div />' } })),
  })
  await router.push('/dashboard')
  await router.isReady()
  const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh } })
  return mount(AppHeader, {
    global: {
      plugins: [router, i18n],
      stubs: {
        AnnouncementBell: true,
        LocaleSwitcher: true,
        ThemeSwitcher: true,
        SubscriptionProgressMini: true,
        Icon: true,
      },
    },
  })
}

describe('AppHeader account identity', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    setActivePinia(createPinia())
  })

  it('shows the Feishu name and enterprise label without exposing the internal email', async () => {
    const wrapper = await mountHeader()
    const trigger = wrapper.get('button[aria-controls="header-user-menu"]')
    expect(trigger.text()).toContain('测试员工')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    await trigger.trigger('click')
    const menu = wrapper.get('#header-user-menu')
    expect(menu.text()).toContain('测试员工')
    expect(menu.text()).toContain('飞书企业账号')
    expect(wrapper.html()).not.toContain(internalEmail)
    expect(wrapper.html()).not.toContain('feishu-connect.invalid')
    expect(useAuthStore().user?.email).toBe(internalEmail)
    await trigger.trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('#header-user-menu').exists()).toBe(false)
  })

  it('uses a friendly fallback when a Feishu name is empty', async () => {
    const wrapper = await mountHeader({ username: '   ' })
    await wrapper.get('button[aria-controls="header-user-menu"]').trigger('click')
    expect(wrapper.text()).toContain('飞书用户')
    expect(wrapper.html()).not.toContain('a'.repeat(64))
    expect(wrapper.html()).not.toContain('feishu-connect.invalid')
  })

  it('updates an open menu when the user binds a real email and changes their name', async () => {
    const wrapper = await mountHeader()
    await wrapper.get('button[aria-controls="header-user-menu"]').trigger('click')
    const email = `${'long-name-'.repeat(6)}@example.com`
    const username = '长姓名测试'.repeat(15)
    useAuthStore().user = createUser({ email, username, email_bound: true })
    await nextTick()
    expect(wrapper.get('#header-user-menu').text()).toContain(email)
    expect(wrapper.get('#header-user-menu').text()).not.toContain('飞书企业账号')
    expect(wrapper.find(`[title="${email}"]`).exists()).toBe(true)
    expect(wrapper.find(`[title="${username}"]`).exists()).toBe(true)
    expect(wrapper.html()).not.toContain('feishu-connect.invalid')
  })
})
