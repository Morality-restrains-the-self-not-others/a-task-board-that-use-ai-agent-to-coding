// @ts-check
import { afterEach, describe, expect, it, vi } from 'vitest'
import { sanitizeOidcResumeNext, resolveOidcResumeTarget, isOidcResumeSameSiteHost, _test } from '../utils/oidcResumeUrl.js'

describe('oidcResumeUrl', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.unstubAllGlobals()
  })

  it('accepts relative OIDC authorize path', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'http://127.0.0.1:18081')
    const path = '/api/oidc/authorize?client_id=gitlab&response_type=code'
    expect(sanitizeOidcResumeNext(path)).toBe(path)
  })

  it('accepts gateway absolute OIDC authorize URL', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'http://127.0.0.1:18081')
    const url = 'http://127.0.0.1:18081/api/oidc/authorize?client_id=gitlab'
    expect(sanitizeOidcResumeNext(url)).toBe(url)
  })

  it('rejects external open redirect', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'http://127.0.0.1:18081')
    expect(sanitizeOidcResumeNext('http://evil.example/api/oidc/authorize?x=1')).toBeNull()
  })

  it('accepts production base domain URL when baked base matches', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'https://example.com')
    const url =
      'https://example.com/api/oidc/authorize?client_id=gitlab-git-service&response_type=code'
    expect(sanitizeOidcResumeNext(url)).toBe(url)
  })

  it('derives base domain from www location when build-time env is empty', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', '')
    vi.stubEnv('VITE_API_BASE_URL', '')
    vi.stubGlobal('window', {
      location: { protocol: 'https:', hostname: 'www.daydaymoney.com' },
      config: {},
    })
    const url =
      'https://example.com/api/oidc/authorize?client_id=gitlab-git-service&response_type=code'
    expect(sanitizeOidcResumeNext(url)).toBe(url)
    expect(_test.gatewayPublicBase()).toBe('https://example.com')
  })

  it('does not treat empty VITE_API_BASE_URL as base when www can derive base host', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', '')
    vi.stubEnv('VITE_API_BASE_URL', '')
    vi.stubGlobal('window', {
      location: { protocol: 'https:', hostname: 'www.daydaymoney.com' },
      config: { TASK_GATEWAY_PUBLIC_BASE: '' },
    })
    expect(resolveOidcResumeTarget('/api/oidc/authorize?client_id=gitlab')).toBe(
      'https://example.com/api/oidc/authorize?client_id=gitlab',
    )
  })

  it('resolves relative target using explicit VITE_TASK_GATEWAY_PUBLIC_BASE', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'http://127.0.0.1:18081')
    vi.stubEnv('VITE_API_BASE_URL', '')
    expect(resolveOidcResumeTarget('/api/oidc/authorize?client_id=gitlab')).toBe(
      'http://127.0.0.1:18081/api/oidc/authorize?client_id=gitlab',
    )
  })

  it('accepts relative tenant-scoped OIDC authorize path', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'http://127.0.0.1:18081')
    const path = '/api/oidc/877397588196749312/authorize?client_id=gitlab-tenant-877397588196749312'
    expect(sanitizeOidcResumeNext(path)).toBe(path)
  })

  it('accepts absolute tenant-scoped OIDC authorize URL', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'https://api.daydaymoney.com')
    const url =
      'https://api.daydaymoney.com/api/oidc/877397588196749312/authorize?client_id=gitlab-tenant-877397588196749312'
    expect(sanitizeOidcResumeNext(url)).toBe(url)
  })

  it('rejects tenant path traversal as authorize resume', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'http://127.0.0.1:18081')
    expect(sanitizeOidcResumeNext('/api/oidc/../authorize?client_id=x')).toBeNull()
  })

  it('accepts www tenant authorize next when gateway base is apex', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'https://daydaymoney.com')
    const url =
      'https://www.daydaymoney.com/api/oidc/877397588196749312/authorize?client_id=gitlab-tenant-877397588196749312&redirect_uri=http%3A%2F%2F115.29.110.74%2Fusers%2Fauth%2Fopenid_connect%2Fcallback'
    expect(sanitizeOidcResumeNext(url)).toBe(url)
  })

  it('accepts api tenant authorize next when gateway base is apex', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'https://daydaymoney.com')
    const url =
      'https://api.daydaymoney.com/api/oidc/877397588196749312/authorize?client_id=gitlab-tenant-877397588196749312'
    expect(sanitizeOidcResumeNext(url)).toBe(url)
  })

  it('rejects sibling subdomain that is not www or api', () => {
    vi.stubEnv('VITE_TASK_GATEWAY_PUBLIC_BASE', 'https://daydaymoney.com')
    expect(
      sanitizeOidcResumeNext('https://evil.daydaymoney.com/api/oidc/authorize?client_id=x'),
    ).toBeNull()
    expect(isOidcResumeSameSiteHost('www.daydaymoney.com', 'daydaymoney.com')).toBe(true)
    expect(isOidcResumeSameSiteHost('api.daydaymoney.com', 'daydaymoney.com')).toBe(true)
    expect(isOidcResumeSameSiteHost('evil.daydaymoney.com', 'daydaymoney.com')).toBe(false)
  })
})
