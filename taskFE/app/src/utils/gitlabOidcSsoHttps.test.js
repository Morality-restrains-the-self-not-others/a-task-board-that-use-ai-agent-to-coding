import { describe, expect, it } from 'vitest'
import {
  gitlabOidcSsoBlockReason,
  gitlabOidcSsoHttpWarning,
} from './gitlabOidcSsoHttps.js'

describe('gitlabOidcSsoHttps', () => {
  it('does not block public HTTP GitLab base URL', () => {
    expect(gitlabOidcSsoBlockReason('http://115.29.110.74')).toBe('')
  })

  it('warns on public HTTP and not on https or loopback http', () => {
    const warn = gitlabOidcSsoHttpWarning('http://115.29.110.74')
    expect(warn).toContain('HTTP')
    expect(warn).toContain('115.29.110.74')
    expect(gitlabOidcSsoHttpWarning('https://gitlab.daydaymoney.com')).toBe('')
    expect(gitlabOidcSsoHttpWarning('http://127.0.0.1:8929')).toBe('')
    expect(gitlabOidcSsoHttpWarning('http://localhost:8012')).toBe('')
  })

  it('allows https and loopback http', () => {
    expect(gitlabOidcSsoBlockReason('https://gitlab.daydaymoney.com')).toBe('')
    expect(gitlabOidcSsoBlockReason('http://127.0.0.1:8929')).toBe('')
    expect(gitlabOidcSsoBlockReason('http://localhost:8012')).toBe('')
  })

  it('does not block empty URL (parent may still be loading Path A)', () => {
    expect(gitlabOidcSsoBlockReason('')).toBe('')
    expect(gitlabOidcSsoBlockReason('   ')).toBe('')
  })

  it('blocks invalid and non-http(s) URLs', () => {
    expect(gitlabOidcSsoBlockReason('not-a-url')).toContain('http://')
    expect(gitlabOidcSsoBlockReason('ftp://gitlab.daydaymoney.com')).toContain('http://')
  })
})
