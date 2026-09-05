// @ts-check
import { describe, expect, it } from 'vitest'
import {
  deriveGatewayPublicBaseFromLocation,
  isIpOrLocalHostname,
  resolveSharedCookieDomain,
} from './ssoCookieDomain.js'

describe('ssoCookieDomain', () => {
  it('detects ip and localhost hosts', () => {
    expect(isIpOrLocalHostname('127.0.0.1')).toBe(true)
    expect(isIpOrLocalHostname('127.0.0.1')).toBe(true)
    expect(isIpOrLocalHostname('localhost')).toBe(true)
    expect(isIpOrLocalHostname('www.daydaymoney.com')).toBe(false)
  })

  it('uses configured cookie domain when provided', () => {
    expect(resolveSharedCookieDomain('www.daydaymoney.com', 'example.com')).toBe(
      '.example.com',
    )
    expect(resolveSharedCookieDomain('www.daydaymoney.com', '.example.com')).toBe(
      '.example.com',
    )
  })

  it('derives parent domain for www/api hosts', () => {
    expect(resolveSharedCookieDomain('www.daydaymoney.com', '')).toBe('.example.com')
    expect(resolveSharedCookieDomain('api.daydaymoney.com', '')).toBe('.example.com')
  })

  it('keeps host-only cookies on ip/localhost', () => {
    expect(resolveSharedCookieDomain('127.0.0.1', '')).toBe('')
    expect(resolveSharedCookieDomain('localhost', '')).toBe('')
    // 配置了共享域也不能覆盖 localhost/IP，否则 cookie 会被静默丢弃
    expect(resolveSharedCookieDomain('localhost', 'example.com')).toBe('')
    expect(resolveSharedCookieDomain('127.0.0.1', '.example.com')).toBe('')
  })

  it('ignores configured domain when hostname is outside that domain', () => {
    expect(resolveSharedCookieDomain('example.com', 'example.com')).toBe('')
  })

  it('derives base public origin from www location (no api. prefix)', () => {
    expect(
      deriveGatewayPublicBaseFromLocation({
        protocol: 'https:',
        hostname: 'www.daydaymoney.com',
      }),
    ).toBe('https://example.com')
  })

  it('does not invent gateway base for ip hosts', () => {
    expect(
      deriveGatewayPublicBaseFromLocation({
        protocol: 'http:',
        hostname: '127.0.0.1',
      }),
    ).toBe('')
  })
})
