// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import {
  buildWechatOAuthUrl,
  navigateWechatOAuth,
  resolveWechatCallbackRedirect,
} from './wechatLoginFlow.js'

describe('buildWechatOAuthUrl', () => {
  it('默认 app=web（扫码）并携带净化后的 next', () => {
    expect(buildWechatOAuthUrl('web', '/projects/')).toBe(
      '/api/auth/wechat/login/?app=web&next=%2Fprojects%2F',
    )
  })

  it('inapp 应用（公众号授权）', () => {
    expect(buildWechatOAuthUrl('inapp', null)).toBe('/api/auth/wechat/login/?app=inapp')
  })

  it('丢弃开放重定向 / 非法 next（防开放重定向，与后端 sanitizeNextPath 对齐）', () => {
    expect(buildWechatOAuthUrl('web', 'https://evil.example/phish')).toBe(
      '/api/auth/wechat/login/?app=web',
    )
    expect(buildWechatOAuthUrl('web', '//evil.example')).toBe('/api/auth/wechat/login/?app=web')
    expect(buildWechatOAuthUrl('web', 'javascript:alert(1)')).toBe(
      '/api/auth/wechat/login/?app=web',
    )
  })

  it('空 next 不出现空参数', () => {
    expect(buildWechatOAuthUrl('web', '')).toBe('/api/auth/wechat/login/?app=web')
    expect(buildWechatOAuthUrl('web', undefined)).toBe('/api/auth/wechat/login/?app=web')
  })

  // OPT-20260806-043: next 指向 /onboarding 时置空，让后端按角色计算落点
  it('丢弃 next=/onboarding/（防误导性回跳，后端按角色计算落点）', () => {
    expect(buildWechatOAuthUrl('web', '/onboarding/')).toBe('/api/auth/wechat/login/?app=web')
    expect(buildWechatOAuthUrl('web', '/onboarding')).toBe('/api/auth/wechat/login/?app=web')
    expect(buildWechatOAuthUrl('web', '/onboarding/setup-company')).toBe(
      '/api/auth/wechat/login/?app=web',
    )
  })

  it('业务路径 next 不受影响（/projects/ 等照常携带）', () => {
    expect(buildWechatOAuthUrl('web', '/projects/')).toBe(
      '/api/auth/wechat/login/?app=web&next=%2Fprojects%2F',
    )
    expect(buildWechatOAuthUrl('web', '/tenant/88/work-panel/')).toBe(
      '/api/auth/wechat/login/?app=web&next=%2Ftenant%2F88%2Fwork-panel%2F',
    )
  })

  // OPT-20260820-036: sessionStorage 有推荐码时透传 accessCode（微信 OAuth 自动注册回填）
  it('sessionStorage 有推荐码时携带 accessCode', () => {
    sessionStorage.setItem('referral_access_code', 'DR2AKvP9J9')
    try {
      expect(buildWechatOAuthUrl('web', null)).toBe(
        '/api/auth/wechat/login/?app=web&accessCode=DR2AKvP9J9',
      )
      expect(buildWechatOAuthUrl('web', '/projects/')).toBe(
        '/api/auth/wechat/login/?app=web&next=%2Fprojects%2F&accessCode=DR2AKvP9J9',
      )
    } finally {
      sessionStorage.removeItem('referral_access_code')
    }
  })

  it('sessionStorage 无推荐码时不带 accessCode（行为不变）', () => {
    sessionStorage.removeItem('referral_access_code')
    expect(buildWechatOAuthUrl('web', '/projects/')).toBe(
      '/api/auth/wechat/login/?app=web&next=%2Fprojects%2F',
    )
  })
})

describe('resolveWechatCallbackRedirect', () => {
  // 回归：「微信扫码登录后跳转到首页」— 此前回调兜底 next||'/' 落到公开营销首页
  it('后端回调携带 next（用户原始回跳）时原样使用，绝不落到 "/"', () => {
    expect(resolveWechatCallbackRedirect('/tenant/88/work-panel/')).toBe('/tenant/88/work-panel/')
    expect(resolveWechatCallbackRedirect('/onboarding/')).toBe('/onboarding/')
    expect(resolveWechatCallbackRedirect('/system-admin/')).toBe('/system-admin/')
  })

  it('无 next 时回退默认（与正常登录一致），禁止公开首页 "/"', () => {
    expect(resolveWechatCallbackRedirect(null)).toBe('/system-admin/')
    expect(resolveWechatCallbackRedirect(undefined)).toBe('/system-admin/')
    expect(resolveWechatCallbackRedirect(null)).not.toBe('/')
  })

  it('拒绝开放重定向 next（后端已净化，前端双保险）', () => {
    expect(resolveWechatCallbackRedirect('//evil.example/phish')).toBe('/system-admin/')
    expect(resolveWechatCallbackRedirect('https://evil.example/x')).toBe('/system-admin/')
  })
})

describe('navigateWechatOAuth', () => {
  it('预检 302 / opaqueredirect 时执行 assign，不报 unavailable', async () => {
    const assigned = []
    const headers = new Headers({ 'X-Trace-Id': 'abc12345-trace-ok' })
    const fetchImpl = async () =>
      ({ status: 302, type: 'opaqueredirect', headers })
    const result = await navigateWechatOAuth('/api/auth/wechat/login/?app=web', {
      fetchImpl,
      assign: (href) => assigned.push(href),
    })
    expect(result.status).toBe('ok')
    expect(assigned).toEqual(['/api/auth/wechat/login/?app=web'])
  })

  it('预检 502 时不跳转，返回 unavailable + traceId（防裸 APISIX 502）', async () => {
    const assigned = []
    const headers = new Headers({ 'X-Trace-Id': '6370502f2f293bffbced1737698e0fd4' })
    const fetchImpl = async () => ({ status: 502, type: 'basic', headers })
    const result = await navigateWechatOAuth('/api/auth/wechat/login/?app=web', {
      fetchImpl,
      assign: (href) => assigned.push(href),
    })
    expect(result.status).toBe('unavailable')
    expect(result.traceId).toBe('6370502f2f293bffbced1737698e0fd4')
    expect(result.message).toMatch(/暂时不可用/)
    expect(assigned).toEqual([])
  })

  it('网络失败时不跳转且带 fallback traceId', async () => {
    const assigned = []
    const fetchImpl = async () => {
      throw new TypeError('Failed to fetch')
    }
    const result = await navigateWechatOAuth('/api/auth/wechat/login/?app=web', {
      fetchImpl,
      assign: (href) => assigned.push(href),
    })
    expect(result.status).toBe('unavailable')
    expect(result.traceId).toMatch(/^fe-wechat-/)
    expect(assigned).toEqual([])
  })
})
