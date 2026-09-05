// @vitest-environment jsdom
// OPT-20260819-038 回归：项目内联编辑 保存标签/自动运行 PATCH 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetailInlineEditableFields.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
  }))
  vi.mock('../components/ProjectTagsInput.vue', () => ({
    default: { template: '<div />' },
  }))
  vi.mock('../components/ProjectDetailImageField.vue', () => ({
    default: { template: '<div />' },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PATCH 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-inline' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const baseProject = {
    id: 'proj_1',
    tags: ['alpha'],
    container_image_id: '42',
    container_image: 'ubuntu-dev',
    server_run_template: {
      platform: 'aliyun',
      region: 'cn-hangzhou',
      default_auto_run: false,
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.apiFetch.mockImplementation(async () => ({
      ok: true,
      json: async () => ({ id: 'proj_1', tags: ['alpha'] }),
      jsonPreservingSnowflakeIds: async () => ({ id: 'proj_1', tags: ['alpha'] }),
      text: async () => '{}',
    }))
  })

  const { default: ProjectDetailInlineEditableFields } = await import(
    './ProjectDetailInlineEditableFields.vue'
  )

  describe('ProjectDetailInlineEditableFields 写操作 clickGuard 接线', () => {
    it('保存标签 PATCH 携带 Idempotency-Key', async () => {
      const wrapper = mount(ProjectDetailInlineEditableFields, {
        props: { field: 'tags', project: baseProject, tenantId: 't1', projectId: 'proj_1' },
      })
      await flushPromises()

      await wrapper.find('[data-testid="project-tags-display"]').trigger('click')
      await flushPromises()
      await wrapper.find('[data-testid="project-tags-save"]').trigger('click')
      await flushPromises()

      const patchCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'PATCH')
      expect(patchCall).toBeTruthy()
      expect(patchCall[0]).toBe('/api/projects/proj_1/tenant_id/t1/')
      expect(patchCall[1].headers['Idempotency-Key']).toBe('ik-test-inline')
    })
  })
}
