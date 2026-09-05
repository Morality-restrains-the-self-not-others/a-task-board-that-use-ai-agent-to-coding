// @vitest-environment jsdom
// OPT-20260819-038: 云平台授权/凭据写操作（资源路径）接入 createClickGuard + Idempotency-Key。
// 回归断言：删除授权 / 删除 OAuth / 凭据校验 / 激活切换 的 POST/PUT/DELETE 均携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsCloudPlatform.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))
  // 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，断言写请求带 Idempotency-Key。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-cp-test' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('../utils/modalService.js', () => ({
    default: {
      alert: vi.fn().mockResolvedValue(undefined),
      confirm: vi.fn().mockResolvedValue(true),
    },
  }))
  vi.mock('../composables/useTenantPageAccess.js', () => ({
    useTenantPageAccess: () => ({ accessAllowed: true }),
  }))

  const { default: View } = await import('./WorkspaceSettingsCloudPlatform.vue')

  function jsonOk(body = []) {
    return {
      ok: true,
      status: 200,
      json: async () => body,
      traceId: '',
      headers: { get: () => 'application/json' },
    }
  }

  // 子行组件 stub：发出对应事件，驱动父组件 handler。
  const CloudPlatformRowStub = {
    name: 'CloudPlatformAuthorizationRow',
    props: ['authorization'],
    emits: ['edit', 'delete', 'toggle-active', 'verify-credentials'],
    template: '<div class="cp-row-stub" />',
  }
  const OAuthTokenRowStub = {
    name: 'OAuthTokenRow',
    props: ['token'],
    emits: ['delete', 'toggle-active'],
    template: '<div class="oauth-row-stub" />',
  }
  const AddAuthModalStub = { name: 'AddCloudPlatformAuthorizationModal', template: '<div />' }
  const EditAuthModalStub = { name: 'EditCloudPlatformAuthorizationModal', template: '<div />' }

  const stubs = {
    CloudPlatformAuthorizationRow: CloudPlatformRowStub,
    OAuthTokenRow: OAuthTokenRowStub,
    AddCloudPlatformAuthorizationModal: AddAuthModalStub,
    EditCloudPlatformAuthorizationModal: EditAuthModalStub,
  }

  async function mountView() {
    window.history.pushState({}, '', '/tenant/123/settings/cloud')
    const wrapper = mount(View, {
      global: { stubs },
    })
    await flushPromises()
    return wrapper
  }

  describe('WorkspaceSettingsCloudPlatform 云平台授权写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
      // 初始加载返回一条授权 + 一条 OAuth Token，使行组件渲染可触发事件。
      apiFetchMock.mockImplementation((url, options) => {
        const method = options?.method || 'GET'
        if (method === 'GET' && String(url).includes('/cloud-platform-authorizations/tenant_id/')) {
          return Promise.resolve(jsonOk([
            { id: 'auth-1', platform_type: 'aliyun', secret_id: 's', secret_key: 'k', is_active: true, authorization_type: 'access_key' },
          ]))
        }
        if (method === 'GET' && String(url).includes('/oauth-tokens/tenant_id/')) {
          return Promise.resolve(jsonOk([{ id: 'tok-1', platform_type: 'aliyun' }]))
        }
        return Promise.resolve(jsonOk({}))
      })
    })

    it('删除云平台授权 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = await mountView()
      const row = wrapper.findComponent(CloudPlatformRowStub)
      row.vm.$emit('delete', 'auth-1', 'aliyun')
      await flushPromises()

      const deletes = apiFetchMock.mock.calls.filter(([, o]) => o.method === 'DELETE')
      expect(deletes).toHaveLength(1)
      const [url, options] = deletes[0]
      expect(url).toContain('/cloud-platform-authorizations/auth-1/')
      expect(options.headers['Idempotency-Key']).toBe('ik-cp-test')
    })

    it('删除 OAuth Token DELETE 携带 Idempotency-Key', async () => {
      const wrapper = await mountView()
      const row = wrapper.findComponent(OAuthTokenRowStub)
      row.vm.$emit('delete', 'tok-1', 'aliyun')
      await flushPromises()

      const deletes = apiFetchMock.mock.calls.filter(([, o]) => o.method === 'DELETE')
      expect(deletes).toHaveLength(1)
      const [url, options] = deletes[0]
      expect(url).toContain('/oauth-tokens/tok-1/')
      expect(options.headers['Idempotency-Key']).toBe('ik-cp-test')
    })

    it('凭据校验 POST 携带 Idempotency-Key', async () => {
      apiFetchMock.mockImplementation((url, options) => {
        const method = options?.method || 'GET'
        if (method === 'GET' && String(url).includes('/cloud-platform-authorizations/tenant_id/')) {
          return Promise.resolve(jsonOk([{ id: 'auth-2', platform_type: 'aliyun', authorization_type: 'access_key' }]))
        }
        if (method === 'GET' && String(url).includes('/oauth-tokens/tenant_id/')) {
          return Promise.resolve(jsonOk([]))
        }
        return Promise.resolve(jsonOk({ success: true, caller_identity: { user_id: 'u1' } }))
      })
      const wrapper = await mountView()
      const row = wrapper.findComponent(CloudPlatformRowStub)
      row.vm.$emit('verify-credentials', {
        id: 'auth-2',
        authorization_type: 'access_key',
        secret_id: 's',
        secret_key: 'k',
        is_active: true,
      })
      await flushPromises()

      const calls = apiFetchMock.mock.calls.filter(([url]) => String(url).includes('/verify-credentials/'))
      expect(calls).toHaveLength(1)
      const [url, options] = calls[0]
      expect(options.method).toBe('POST')
      expect(url).toContain('/verify-credentials/')
      expect(options.headers['Idempotency-Key']).toBe('ik-cp-test')
    })

    it('激活状态切换 POST 携带 Idempotency-Key', async () => {
      const wrapper = await mountView()
      const row = wrapper.findComponent(CloudPlatformRowStub)
      row.vm.$emit('toggle-active', 'auth-3', false)
      await flushPromises()

      const posts = apiFetchMock.mock.calls.filter(([url]) => String(url).includes('/toggle-active/'))
      expect(posts).toHaveLength(1)
      const [, options] = posts[0]
      expect(options.method).toBe('POST')
      expect(options.headers['Idempotency-Key']).toBe('ik-cp-test')
    })
  })
}
