// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminContainerImages.marketplace.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  vi.mock('../utils/apiUtils', () => ({
    apiFetch: apiFetchMock,
    extractErrorMessage: vi.fn(() => ''),
  }))
  vi.mock('../utils/toastService', () => ({
    default: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PATCH 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-marketplace' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('../composables/useSystemAdminContainerImages.js', () => ({
    useSystemAdminContainerImages: () => ({
      uploadModalVisible: { value: false },
      editModalVisible: { value: false },
      deleteModalVisible: { value: false },
      environmentModalVisible: { value: false },
      loadingEdit: { value: false },
      loadingEnvironment: { value: false },
      uploading: { value: false },
      editing: { value: false },
      deleting: { value: false },
      settingEnvironment: { value: false },
      uploadForm: { value: {} },
      editForm: { value: {} },
      environmentForm: { value: {} },
      containerImages: { value: [] },
      serverImages: { value: [] },
      cloudPlatforms: { value: [] },
      formatFileSize: (n) => String(n),
      formatDate: (d) => String(d),
      handleUploadImage: vi.fn(),
      openEditImageModal: vi.fn(),
      handleEditImage: vi.fn(),
      openDeleteImageModal: vi.fn(),
      handleDeleteImage: vi.fn(),
      openSetEnvironmentModal: vi.fn(),
      handleSetEnvironment: vi.fn(),
    }),
  }))
  vi.mock('../components/SystemAdminContainerImageUploadModal.vue', () => ({
    default: { name: 'UploadModal', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminContainerImageEditModal.vue', () => ({
    default: { name: 'EditModal', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminContainerImageDeleteModal.vue', () => ({
    default: { name: 'DeleteModal', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminContainerImageEnvironmentModal.vue', () => ({
    default: { name: 'EnvModal', template: '<div />' },
  }))
  vi.mock('../utils/config', () => ({
    getApiUrl: (path) => path,
  }))

  const { default: SystemAdminContainerImages } = await import('./SystemAdminContainerImages.vue')

  beforeEach(() => {
    apiFetchMock.mockReset()
    apiFetchMock.mockImplementation(async (url) => {
      if (String(url).includes('admin-marketplace-settings')) {
        return {
          ok: true,
          json: async () => ({ vendor_application_review_enabled: true }),
        }
      }
      return { ok: true, json: async () => ({}) }
    })
  })

  describe('SystemAdminContainerImages 镜像市场面板', () => {
    it('渲染 SSO 入口与厂商申请审核开关', async () => {
      const wrapper = mount(SystemAdminContainerImages)
      await flushPromises()
      const panel = wrapper.find('[data-testid="marketplace-admin-panel"]')
      expect(panel.exists()).toBe(true)
      expect(panel.text()).toContain('开启厂商申请审核')
      const sso = wrapper.find('a[href="/api/accounts/sso/ai-provider/admin/"]')
      expect(sso.exists()).toBe(true)
      expect(sso.text()).toContain('镜像市场管理（SSO）')
    })

    it('切换审核开关保存 PATCH 携带 Idempotency-Key', async () => {
      const wrapper = mount(SystemAdminContainerImages)
      await flushPromises()

      const toggle = wrapper.find('input[type="checkbox"]')
      expect(toggle.exists()).toBe(true)
      await toggle.setValue(false)
      await flushPromises()

      const patchCall = apiFetchMock.mock.calls.find(([, opts]) => opts?.method === 'PATCH')
      expect(patchCall).toBeTruthy()
      expect(patchCall[0]).toBe('/api/ai-provider/admin-marketplace-settings/')
      expect(patchCall[1].headers['Idempotency-Key']).toBe('ik-test-marketplace')
    })
  })
}
