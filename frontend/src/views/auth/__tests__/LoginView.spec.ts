import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '@/views/auth/LoginView.vue'

const { getPublicSettingsMock, loginMock, pushMock } = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  loginMock: vi.fn(),
  pushMock: vi.fn()
}))

const publicSettings = {
  registration_enabled: true,
  turnstile_enabled: false,
  turnstile_site_key: '',
  tencent_captcha_enabled: false,
  tencent_captcha_app_id: '',
  aliyun_captcha_enabled: false,
  aliyun_captcha_scene_id: '',
  aliyun_captcha_prefix: '',
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  backend_mode_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  password_reset_enabled: false,
  passkey_enabled: false,
  login_agreement_enabled: false,
  login_agreement_documents: []
}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: pushMock,
    currentRoute: { value: { query: {} } }
  })
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login: (...args: unknown[]) => loginMock(...args),
    loginWithPasskey: vi.fn(),
    login2FA: vi.fn(),
    setLoginPersistence: vi.fn()
  }),
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  })
}))

vi.mock('@/api/auth', () => ({
  buildOAuthLoginStartURL: vi.fn(),
  getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
  isTotp2FARequired: vi.fn(() => false),
  isWeChatWebOAuthEnabled: vi.fn(() => false),
  startOAuthLogin: vi.fn()
}))

function mountLogin() {
  return mount(LoginView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        DingTalkOAuthSection: true,
        EmailOAuthButtons: true,
        Icon: true,
        LinuxDoOAuthSection: true,
        LoginAgreementPrompt: true,
        OidcOAuthSection: true,
        RouterLink: { template: '<a><slot /></a>' },
        TotpLoginModal: true,
        TurnstileWidget: true,
        WechatOAuthSection: true,
        transition: false
      }
    }
  })
}

describe('LoginView registration entry', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    loginMock.mockReset()
    pushMock.mockReset()
    localStorage.clear()
    sessionStorage.clear()
    getPublicSettingsMock.mockResolvedValue(publicSettings)
    loginMock.mockResolvedValue({})
  })

  it('shows the registration entry when registration is enabled', async () => {
    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).toContain('auth.signUp')
  })

  it('hides the registration entry when registration is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      registration_enabled: false
    })

    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).not.toContain('auth.signUp')
  })

  it('shows remember me unchecked by default', async () => {
    const wrapper = mountLogin()
    await flushPromises()

    const remember = wrapper.get('#remember-me')
    expect(wrapper.text()).toContain('auth.rememberMe')
    expect((remember.element as HTMLInputElement).checked).toBe(false)
  })

  it('passes remember me to login and remembers the account without web-storing the password', async () => {
    const wrapper = mountLogin()
    await flushPromises()

    await wrapper.get('#email').setValue('remember@example.com')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('#remember-me').setValue(true)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(loginMock).toHaveBeenCalledWith(
      expect.objectContaining({
        email: 'remember@example.com',
        password: 'secret-123'
      }),
      true
    )
    expect(localStorage.getItem('sub2api_remember_me')).toBe('true')
    expect(localStorage.getItem('sub2api_remembered_email')).toBe('remember@example.com')
    expect(Object.values(localStorage)).not.toContain('secret-123')
  })

  it('restores the remembered account and checkbox', async () => {
    localStorage.setItem('sub2api_remember_me', 'true')
    localStorage.setItem('sub2api_remembered_email', 'saved@example.com')

    const wrapper = mountLogin()
    await flushPromises()

    expect((wrapper.get('#remember-me').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('#email').element as HTMLInputElement).value).toBe('saved@example.com')
  })
})
