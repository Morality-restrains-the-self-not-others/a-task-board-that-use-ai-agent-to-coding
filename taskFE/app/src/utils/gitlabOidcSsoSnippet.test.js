import { describe, expect, it } from 'vitest'
import { insertOmniAuthSecret } from './gitlabOidcSsoSnippet.js'

const SNIPPET = `gitlab_rails['omniauth_providers'] = [
  {
    args: {
      issuer: "https://api.daydaymoney.com",
      client_options: {
        identifier: "gitlab-tenant-123",
        redirect_uri: "https://gitlab.daydaymoney.com/users/auth/openid_connect/callback"
      }
    }
  }
]
`

describe('insertOmniAuthSecret', () => {
  it('inserts secret on the line after identifier', () => {
    const out = insertOmniAuthSecret(SNIPPET, 'once-secret')
    expect(out).toContain('identifier: "gitlab-tenant-123",\n        secret: "once-secret",')
    expect(out).toContain('redirect_uri:')
  })

  it('leaves snippet unchanged when secret is empty', () => {
    expect(insertOmniAuthSecret(SNIPPET, '')).toBe(SNIPPET)
    expect(insertOmniAuthSecret(SNIPPET, '   ')).toBe(SNIPPET)
  })

  it('does not duplicate secret when snippet already has secret', () => {
    const withSecret = insertOmniAuthSecret(SNIPPET, 'once-secret')
    expect(insertOmniAuthSecret(withSecret, 'once-secret')).toBe(withSecret)
  })

  it('escapes quotes in secret for gitlab.rb', () => {
    const out = insertOmniAuthSecret(SNIPPET, 'a"b')
    expect(out).toContain('secret: "a\\"b",')
  })

  it('replaces the server placeholder with the real secret (OPT-20260826-013)', () => {
    const withPlaceholder = SNIPPET.replace(
      '        redirect_uri: "https://gitlab.daydaymoney.com/users/auth/openid_connect/callback"',
      "        secret: '<PASTE_CLIENT_SECRET>',\n        redirect_uri: \"https://gitlab.daydaymoney.com/users/auth/openid_connect/callback\"",
    )
    expect(withPlaceholder).toContain("secret: '<PASTE_CLIENT_SECRET>',")
    const out = insertOmniAuthSecret(withPlaceholder, 'once-secret')
    expect(out).toContain('secret: "once-secret",')
    expect(out).not.toContain("'<PASTE_CLIENT_SECRET>'")
    expect(out).toContain('redirect_uri:')
  })

  it('does not duplicate secret when placeholder is already replaced', () => {
    const withPlaceholder = SNIPPET.replace(
      '        redirect_uri: "https://gitlab.daydaymoney.com/users/auth/openid_connect/callback"',
      "        secret: '<PASTE_CLIENT_SECRET>',\n        redirect_uri: \"https://gitlab.daydaymoney.com/users/auth/openid_connect/callback\"",
    )
    const once = insertOmniAuthSecret(withPlaceholder, 'once-secret')
    expect(insertOmniAuthSecret(once, 'once-secret')).toBe(once)
  })
})
