// @vitest-environment jsdom
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { resolvePostLoginRedirectUrl } from './resolvePostLoginRedirect.js'
import { POST_LOGIN_REDIRECT_STORAGE_KEY } from './authReturnUrl.js'

describe('resolvePostLoginRedirectUrl', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'https://example.com')
  })

  it('优先使用同站业务 next（支付页）', () => {
    const next = '/tenant/850256677331562496/billing/orders/create/'
    expect(resolvePostLoginRedirectUrl(next, {})).toBe(next)
  })

  it('拒绝开放重定向 next', () => {
    expect(resolvePostLoginRedirectUrl('//evil.example/phish', {})).toBe('/system-admin/')
    expect(resolvePostLoginRedirectUrl('https://evil.example/x', {})).toBe('/system-admin/')
  })

  it('OIDC authorize next 优先于业务 path 形态（绝对网关 URL）', () => {
    const oidc =
      'https://example.com/api/oidc/authorize?client_id=1&redirect_uri=https%3A%2F%2Fwww.daydaymoney.com%2F'
    const result = resolvePostLoginRedirectUrl(oidc, {
      redirect_url: '/should-not-use/',
    })
    expect(result).toBe(oidc)
  })

  it('无 next 时消费 postLoginRedirect storage', () => {
    localStorage.setItem(POST_LOGIN_REDIRECT_STORAGE_KEY, '/user/1/profile/')
    expect(resolvePostLoginRedirectUrl('', {})).toBe('/user/1/profile/')
    expect(localStorage.getItem(POST_LOGIN_REDIRECT_STORAGE_KEY)).toBeNull()
  })

  it('回退 data.redirect_url 与默认', () => {
    expect(resolvePostLoginRedirectUrl(null, { redirect_url: '/work-panel/' })).toBe('/work-panel/')
    expect(resolvePostLoginRedirectUrl(undefined, {})).toBe('/system-admin/')
  })

  it('add_account 时纠偏租户 next（AC7）', () => {
    const next = '/tenant/850256677331562496/billing/orders/create/'
    expect(
      resolvePostLoginRedirectUrl(next, {}, localStorage, {
        addAccountQuery: '1',
        newUserId: '99',
      }),
    ).toBe('/user/99/profile/')
  })

  it('不传 addAccountQuery 时租户 next 保持原样（登录不再续加槽位，OPT-20260807-015）', () => {
    const next = '/tenant/850256677331562496/billing/orders/create/'
    expect(resolvePostLoginRedirectUrl(next, {}, localStorage)).toBe(next)
  })

  it('GitLab 租户 SSO 的 www next 在 apex gateway 下仍回跳 authorize', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'https://daydaymoney.com')
    const next =
      'https://www.daydaymoney.com/api/oidc/877397588196749312/authorize?client_id=gitlab-tenant-877397588196749312'
    expect(resolvePostLoginRedirectUrl(next, { redirect_url: '/system-admin/' })).toBe(next)
  })
})
