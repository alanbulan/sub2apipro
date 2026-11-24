import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { auth } = vi.hoisted(() => ({
  auth: {
    checkAuth: vi.fn(),
    isAuthenticated: true,
    isAdmin: false,
    isSimpleMode: false,
    hasPendingAuthSession: false,
  },
}))

vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ siteName: 'BeaconChip', backendModeEnabled: false, cachedPublicSettings: null }) }))
vi.mock('@/stores/adminSettings', () => ({ useAdminSettingsStore: () => ({ customMenuItems: [] }) }))
vi.mock('@/stores/adminCompliance', () => ({ useAdminComplianceStore: () => ({}) }))
vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({ startNavigation: vi.fn(), endNavigation: vi.fn() }),
}))
vi.mock('@/composables/useRoutePrefetch', () => ({ useRoutePrefetch: () => ({ triggerPrefetch: vi.fn() }) }))
vi.mock('@/views/auth/LoginView.vue', () => ({ default: { template: '<div>Login</div>' } }))
vi.mock('@/views/user/PoolDashboardView.vue', () => ({ default: { template: '<div>Account pool</div>' } }))
vi.mock('@/router/title', () => ({ resolveRouteDocumentTitle: () => 'Account Pool - BeaconChip' }))

beforeEach(() => {
  auth.isAuthenticated = true
  auth.isAdmin = false
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
})
afterEach(() => { vi.restoreAllMocks() })

describe('account pool navigation', () => {
  it.each(['/pool-dashboard', '/account-pool'])('lets an ordinary user open %s through the real router', async path => {
    const { default: router } = await import('@/router')
    await router.push(path)
    expect(router.currentRoute.value.name).toBe('UserAccountPool')
    expect(router.currentRoute.value.meta.requiresAuth).toBe(true)
    expect(router.currentRoute.value.meta.requiresAdmin).toBe(false)
  })

  it('keeps the old address as an alias of the menu destination', async () => {
    const { default: router } = await import('@/router')
    const canonical = router.resolve('/pool-dashboard')
    const legacy = router.resolve('/account-pool')
    expect(canonical.matched).toHaveLength(1)
    expect(legacy.matched[0]?.aliasOf).toBe(canonical.matched[0])
  })

  it('requires login and preserves a direct link including its query', async () => {
    auth.isAuthenticated = false
    const { default: router } = await import('@/router')
    const destination = '/pool-dashboard?group_id=7'
    await router.push(destination)
    expect(router.currentRoute.value.path).toBe('/login')
    expect(router.currentRoute.value.query.redirect).toBe(destination)
  })
})
