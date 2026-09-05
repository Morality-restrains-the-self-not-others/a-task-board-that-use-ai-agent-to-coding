// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsGitlabOidcSso.test.js requires vitest runtime')
} else {
  const { nextTick } = await import('vue')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount } = await import('@vue/test-utils')

  const { apiFetchMock, writeText } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    writeText: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      load: vi.fn(async () => {}),
      hasRegionOperate: () => true,
      hasRegion: () => true,
    }),
  }))

  const { default: WorkspaceSettingsGitlabOidcSso } = await import('./WorkspaceSettingsGitlabOidcSso.vue')

  function jsonResp(body, { ok = true, status = 200, traceId = '' } = {}) {
    return {
      ok,
      status,
      traceId,
      json: async () => body,
    }
  }

  beforeEach(() => {
    apiFetchMock.mockReset()
    writeText.mockReset()
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    apiFetchMock.mockResolvedValue(jsonResp({
      configured: false,
      client_id: 'gitlab-tenant-123',
      issuer: 'https://www.daydaymoney.com',
      omniauth_snippet: 'authorization_endpoint: "https://api.daydaymoney.com/api/oidc/123/authorize"\ndiscovery: false',
    }))
  })

  describe('WorkspaceSettingsGitlabOidcSso', () => {
    it('loads GET and shows client_id without secret', async () => {
      const wrapper = mount(WorkspaceSettingsGitlabOidcSso, {
        props: { tenantId: '123', pathABaseUrl: 'https://gitlab.daydaymoney.com' },
      })
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-client-id"]').text()).toContain('gitlab-tenant-123')
      })
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-secret-once"]').exists()).toBe(false)
      expect(String(apiFetchMock.mock.calls[0][0])).toBe('/api/tenant/123/gitlab-oidc-sso/')
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-snippet"]').text()).toContain('/api/oidc/123/authorize')
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-snippet"]').text()).toContain('discovery: false')
    })

    it('explains where beginners must paste client_id issuer and secret', async () => {
      const wrapper = mount(WorkspaceSettingsGitlabOidcSso, {
        props: { tenantId: '123', pathABaseUrl: 'https://gitlab.daydaymoney.com' },
      })
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-where"]').exists()).toBe(true)
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-client-id-hint"]').exists()).toBe(true)
      })
      const where = wrapper.find('[data-testid="gitlab-oidc-sso-where"]').text()
      expect(where).toContain('/etc/gitlab/gitlab.rb')
      expect(where).toContain('gitlab-ctl reconfigure')
      expect(where).toContain('identifier')
      expect(where).toContain('Daydaymoney SSO')
      expect(where).toContain('不要填在本网站')
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-client-id-hint"]').text()).toContain('identifier')
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-issuer-hint"]').text()).toContain('issuer')
    })

    it('enable PUT sends Idempotency-Key and shows one-time secret', async () => {
      apiFetchMock
        .mockResolvedValueOnce(jsonResp({
          configured: false,
          client_id: 'gitlab-tenant-123',
          issuer: 'https://www.daydaymoney.com',
          omniauth_snippet: 'snippet',
        }))
        .mockResolvedValueOnce(jsonResp({
          configured: true,
          client_id: 'gitlab-tenant-123',
          client_secret: 'once-secret',
          issuer: 'https://www.daydaymoney.com',
          omniauth_snippet: 'identifier: "gitlab-tenant-123",\n        redirect_uri: "https://gitlab.daydaymoney.com/users/auth/openid_connect/callback"',
        }))
      const wrapper = mount(WorkspaceSettingsGitlabOidcSso, {
        props: { tenantId: '123', pathABaseUrl: 'https://gitlab.daydaymoney.com' },
      })
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-enable"]').exists()).toBe(true)
      })
      await wrapper.find('[data-testid="gitlab-oidc-sso-enable"]').trigger('click')
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-secret-once"]').text()).toContain('once-secret')
      })
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-secret-hint"]').text()).toContain('secret')
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-secret-hint"]').text()).toContain('仅显示一次')
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-snippet"]').text()).toContain('secret: "once-secret"')
      const putCall = apiFetchMock.mock.calls.find((c) => c[1] && c[1].method === 'PUT')
      expect(putCall).toBeTruthy()
      expect(putCall[1].headers['Idempotency-Key']).toBeTruthy()
      expect(JSON.parse(putCall[1].body).base_url).toBe('https://gitlab.daydaymoney.com')
    })

    it('shows HTTP warning and still PUTs for public HTTP GitLab', async () => {
      apiFetchMock
        .mockResolvedValueOnce(jsonResp({
          configured: false,
          client_id: 'gitlab-tenant-123',
          issuer: 'https://www.daydaymoney.com',
          omniauth_snippet: 'snippet',
        }))
        .mockResolvedValueOnce(jsonResp({
          configured: true,
          client_id: 'gitlab-tenant-123',
          client_secret: 'once-secret',
          issuer: 'https://www.daydaymoney.com',
          omniauth_snippet: 'snippet',
        }))
      const wrapper = mount(WorkspaceSettingsGitlabOidcSso, {
        props: { tenantId: '123', pathABaseUrl: 'http://115.29.110.74' },
      })
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-enable"]').exists()).toBe(true)
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-http-warning"]').exists()).toBe(true)
      })
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-http-warning"]').text()).toContain('HTTP')
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-http-warning"]').text()).toContain('115.29.110.74')
      const enable = wrapper.find('[data-testid="gitlab-oidc-sso-enable"]')
      expect(enable.attributes('disabled')).toBeUndefined()
      await enable.trigger('click')
      await vi.waitFor(() => {
        expect(apiFetchMock.mock.calls.some((c) => c[1] && c[1].method === 'PUT')).toBe(true)
      })
      const putCall = apiFetchMock.mock.calls.find((c) => c[1] && c[1].method === 'PUT')
      expect(JSON.parse(putCall[1].body).base_url).toBe('http://115.29.110.74')
    })

    it('copies one-time client_secret via clipboard without write API (OPT-20260826-014)', async () => {
      apiFetchMock
        .mockResolvedValueOnce(jsonResp({
          configured: false,
          client_id: 'gitlab-tenant-123',
          issuer: 'https://www.daydaymoney.com',
          omniauth_snippet: 'snippet',
        }))
        .mockResolvedValueOnce(jsonResp({
          configured: true,
          client_id: 'gitlab-tenant-123',
          client_secret: 'once-secret',
          issuer: 'https://www.daydaymoney.com',
          omniauth_snippet: 'snippet',
        }))
      const wrapper = mount(WorkspaceSettingsGitlabOidcSso, {
        props: { tenantId: '123', pathABaseUrl: 'https://gitlab.daydaymoney.com' },
      })
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-enable"]').exists()).toBe(true)
      })
      await wrapper.find('[data-testid="gitlab-oidc-sso-enable"]').trigger('click')
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-secret-once"]').text()).toContain('once-secret')
      })
      await wrapper.find('[data-testid="gitlab-oidc-sso-secret-copy"]').trigger('click')
      await vi.waitFor(() => {
        expect(writeText).toHaveBeenCalledWith('once-secret')
      })
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-copy-feedback"]').text()).toContain('已复制')
      // 复制是只读剪贴板操作：除 enable PUT 外不得发出任何写 API。
      const writes = apiFetchMock.mock.calls.filter((c) => c[1] && ['POST', 'PUT', 'PATCH', 'DELETE'].includes(c[1].method))
      expect(writes).toHaveLength(1)
    })

    it('copies gitlab.rb snippet via clipboard without write API (OPT-20260826-014)', async () => {
      apiFetchMock.mockResolvedValue(jsonResp({
        configured: true,
        client_id: 'gitlab-tenant-123',
        issuer: 'https://www.daydaymoney.com',
        omniauth_snippet: 'identifier: "gitlab-tenant-123",\n        redirect_uri: "https://gitlab.daydaymoney.com/users/auth/openid_connect/callback"',
      }))
      const wrapper = mount(WorkspaceSettingsGitlabOidcSso, {
        props: { tenantId: '123', pathABaseUrl: 'https://gitlab.daydaymoney.com' },
      })
      await vi.waitFor(() => {
        expect(wrapper.find('[data-testid="gitlab-oidc-sso-snippet-copy"]').exists()).toBe(true)
      })
      const snippetText = wrapper.find('[data-testid="gitlab-oidc-sso-snippet"]').text()
      await wrapper.find('[data-testid="gitlab-oidc-sso-snippet-copy"]').trigger('click')
      await vi.waitFor(() => {
        expect(writeText).toHaveBeenCalledWith(snippetText)
      })
      expect(wrapper.find('[data-testid="gitlab-oidc-sso-copy-feedback"]').text()).toContain('已复制')
      // 复制是只读剪贴板操作：GET 场景下不应发出任何写 API。
      const writes = apiFetchMock.mock.calls.filter((c) => c[1] && ['POST', 'PUT', 'PATCH', 'DELETE'].includes(c[1].method))
      expect(writes).toHaveLength(0)
    })
  })
  void nextTick
}
