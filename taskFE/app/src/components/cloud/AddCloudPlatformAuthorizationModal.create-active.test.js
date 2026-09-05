// @vitest-environment jsdom
// 创建云平台授权时勾选「启用」必须把 is_active 写入 POST body（后端据此写入 active_methods）。
if (!process.env.VITEST) {
  console.log('[skip] AddCloudPlatformAuthorizationModal.create-active.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { reactive } = await import('vue')

  const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))
  vi.mock('../../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-create-active' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('../../utils/modalService.js', () => ({
    default: {
      alert: vi.fn().mockResolvedValue(undefined),
      confirm: vi.fn().mockResolvedValue(true),
    },
  }))
  vi.mock('../../composables/useTenantPageAccess.js', () => ({
    useTenantPageAccess: () => ({ accessAllowed: true }),
  }))

  const { default: Modal } = await import('./AddCloudPlatformAuthorizationModal.vue')
  const { default: View } = await import('../../views/WorkspaceSettingsCloudPlatform.vue')

  function jsonOk(body = [], extra = {}) {
    return {
      ok: true,
      status: extra.status || 200,
      json: async () => body,
      traceId: '',
      headers: { get: () => 'application/json' },
    }
  }

  const stubs = {
    CloudPlatformAuthorizationRow: { name: 'CloudPlatformAuthorizationRow', template: '<div />' },
    OAuthTokenRow: { name: 'OAuthTokenRow', template: '<div />' },
    EditCloudPlatformAuthorizationModal: { name: 'EditCloudPlatformAuthorizationModal', template: '<div />' },
    TenantPageAccessEmpty: { name: 'TenantPageAccessEmpty', template: '<div />' },
  }

  describe('AddCloudPlatformAuthorizationModal 启用勾选', () => {
    it('受 v-model 控制且无原生 checked 属性', async () => {
      const form = reactive({
        platform_type: 'aliyun',
        authorization_type: 'access_key',
        access_key: '',
        secret_key: '',
        remark: '',
        is_active: true,
      })
      const wrapper = mount(Modal, {
        props: {
          visible: true,
          form,
          cloudPlatforms: [{ value: 'aliyun', label: '阿里云' }],
          availableAuthTypes: [{ value: 'access_key', label: 'Access Key', disabled: false }],
          addingAuthorization: false,
        },
      })
      const checkbox = wrapper.find('#add-cloud-auth-is-active')
      expect(checkbox.exists()).toBe(true)
      expect(checkbox.element.checked).toBe(true)
      expect(checkbox.attributes('checked')).toBeUndefined()
      await checkbox.setValue(false)
      expect(form.is_active).toBe(false)
    })
  })

  describe('WorkspaceSettingsCloudPlatform 创建授权 is_active', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
      apiFetchMock.mockImplementation((url, options) => {
        const method = options?.method || 'GET'
        if (method === 'POST' && String(url).includes('/cloud-platform-authorizations/')) {
          return Promise.resolve(jsonOk({ id: 'cpa-new', status: 'created' }, { status: 201 }))
        }
        if (method === 'GET' && String(url).includes('/cloud-platform-authorizations/tenant_id/')) {
          return Promise.resolve(jsonOk([]))
        }
        if (method === 'GET' && String(url).includes('/oauth-tokens/tenant_id/')) {
          return Promise.resolve(jsonOk([]))
        }
        return Promise.resolve(jsonOk({}))
      })
    })

    async function openAndFill(wrapper, { isActive }) {
      const openBtn = wrapper.findAll('button').find((b) => b.text().includes('添加云平台授权'))
      expect(openBtn).toBeTruthy()
      await openBtn.trigger('click')
      await flushPromises()

      const modal = wrapper.findComponent({ name: 'AddCloudPlatformAuthorizationModal' })
      expect(modal.exists()).toBe(true)
      await modal.find('select').setValue('aliyun')
      await flushPromises()

      const checkbox = modal.find('#add-cloud-auth-is-active')
      expect(checkbox.exists()).toBe(true)
      const currentlyChecked = checkbox.element.checked
      if (currentlyChecked !== isActive) {
        await checkbox.setValue(isActive)
      }

      const textInputs = modal.findAll('input[type="text"], input[type="password"]')
      await textInputs[0].setValue('AKTEST')
      await textInputs[1].setValue('SKTEST')

      await modal.find('form').trigger('submit')
      await flushPromises()
    }

    it('勾选启用时 POST body 含 is_active true', async () => {
      window.history.pushState({}, '', '/tenant/123/settings/cloud-platform/')
      const wrapper = mount(View, { global: { stubs } })
      await flushPromises()
      await openAndFill(wrapper, { isActive: true })

      const createPosts = apiFetchMock.mock.calls.filter(
        ([url, o]) => o?.method === 'POST' && String(url).includes('/cloud-platform-authorizations/'),
      )
      expect(createPosts).toHaveLength(1)
      const body = JSON.parse(createPosts[0][1].body)
      expect(body.is_active).toBe(true)
      expect(body.secret_id).toBe('AKTEST')
    })

    it('取消勾选时 POST body 含 is_active false', async () => {
      window.history.pushState({}, '', '/tenant/123/settings/cloud-platform/')
      const wrapper = mount(View, { global: { stubs } })
      await flushPromises()
      await openAndFill(wrapper, { isActive: false })

      const createPosts = apiFetchMock.mock.calls.filter(
        ([url, o]) => o?.method === 'POST' && String(url).includes('/cloud-platform-authorizations/'),
      )
      expect(createPosts).toHaveLength(1)
      const body = JSON.parse(createPosts[0][1].body)
      expect(body.is_active).toBe(false)
    })
  })
}
