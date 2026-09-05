// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetailInlineEditableFields.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  const { default: ProjectDetailInlineEditableFields } = await import(
    './ProjectDetailInlineEditableFields.vue'
  )

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

  const installedImages = [{ id: '42', name: 'ubuntu-dev' }, { id: '9', name: 'other' }]

  beforeEach(() => {
    hoistedMocks.apiFetchMock.mockReset()
  })

  describe('ProjectDetailInlineEditableFields', () => {
    it('clicks tags display to enter edit mode', async () => {
      const wrapper = mount(ProjectDetailInlineEditableFields, {
        props: {
          field: 'tags',
          project: baseProject,
          tenantId: 't1',
          projectId: 'proj_1',
        },
      })
      expect(wrapper.find('[data-testid="project-tags-display"]').exists()).toBe(true)
      await wrapper.find('[data-testid="project-tags-display"]').trigger('click')
      await nextTick()
      expect(wrapper.find('[data-testid="project-tags-edit"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="project-tags-input"]').exists()).toBe(true)
    })

    it('saves tags via PATCH and emits saved', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ ...baseProject, tags: ['alpha', 'beta'] }),
        jsonPreservingSnowflakeIds: async () => ({ ...baseProject, tags: ['alpha', 'beta'] }),
      })
      const wrapper = mount(ProjectDetailInlineEditableFields, {
        props: {
          field: 'tags',
          project: baseProject,
          tenantId: 't1',
          projectId: 'proj_1',
        },
      })
      await wrapper.find('[data-testid="project-tags-display"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="project-tags-save"]').trigger('click')
      await nextTick()
      await Promise.resolve()
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/projects/proj_1/tenant_id/t1/',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ tags: ['alpha'] }),
        }),
      )
      expect(wrapper.emitted('saved')?.length).toBe(1)
      expect(wrapper.find('[data-testid="project-tags-display"]').exists()).toBe(true)
    })

    it('cancel does not PATCH', async () => {
      const wrapper = mount(ProjectDetailInlineEditableFields, {
        props: {
          field: 'auto_run',
          project: baseProject,
          tenantId: 't1',
          projectId: 'proj_1',
        },
      })
      await wrapper.find('[data-testid="project-default-auto-run-display"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="project-default-auto-run-cancel"]').trigger('click')
      await nextTick()
      expect(hoistedMocks.apiFetchMock).not.toHaveBeenCalled()
      expect(wrapper.find('[data-testid="project-default-auto-run-display"]').exists()).toBe(true)
    })

    it('shows auto-run disabled hint when no image', async () => {
      const wrapper = mount(ProjectDetailInlineEditableFields, {
        props: {
          field: 'auto_run',
          project: {
            ...baseProject,
            container_image_id: '',
          },
          tenantId: 't1',
          projectId: 'proj_1',
        },
      })
      await wrapper.find('[data-testid="project-default-auto-run-display"]').trigger('click')
      await nextTick()
      expect(wrapper.find('[data-testid="project-auto-run-disabled-hint"]').text()).toContain(
        '请先选择已安装镜像',
      )
      expect(wrapper.find('[data-testid="project-default-auto-run"]').attributes('disabled')).toBeDefined()
    })

    it('shows 无法启动 and auto-disables when git auth gate blocks', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          ...baseProject,
          server_run_template: { ...baseProject.server_run_template, default_auto_run: false },
        }),
        jsonPreservingSnowflakeIds: async () => ({
          ...baseProject,
          server_run_template: { ...baseProject.server_run_template, default_auto_run: false },
        }),
      })
      const wrapper = mount(ProjectDetailInlineEditableFields, {
        props: {
          field: 'auto_run',
          project: {
            ...baseProject,
            server_run_template: {
              ...baseProject.server_run_template,
              default_auto_run: true,
            },
          },
          tenantId: 't1',
          projectId: 'proj_1',
          gitAuthGate: {
            blocked: true,
            code: 'AUTO_RUN_GIT_AUTH_ERROR',
            message: '存在 Git 仓库授权异常，自动运行无法启动；请先完成授权后再启用',
            displayLabel: '无法启动',
          },
        },
      })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('[data-testid="project-default-auto-run-label"]').text()).toBe('无法启动')
      expect(wrapper.find('[data-testid="project-auto-run-git-gate-hint"]').text()).toContain('授权异常')
      const authHint = wrapper.find('[data-testid="project-auto-run-git-gate-hint"]')
      expect(authHint.attributes('data-traceid') || authHint.attributes('data-traceId')).toBeUndefined()
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/projects/proj_1/tenant_id/t1/',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({
            server_run_template: {
              platform: 'aliyun',
              region: 'cn-hangzhou',
              default_auto_run: false,
            },
          }),
        }),
      )
      expect(wrapper.emitted('saved')?.length).toBe(1)

      await wrapper.find('[data-testid="project-default-auto-run-display"]').trigger('click')
      await nextTick()
      expect(wrapper.find('[data-testid="project-default-auto-run-edit"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="project-auto-run-disabled-hint"]').text()).toContain('授权异常')
      expect(wrapper.find('[data-testid="project-default-auto-run"]').attributes('disabled')).toBeDefined()
    })

    it('挂载 data-traceId on nested-repos gate hint when gate.traceId present', async () => {
      const nestedWrapper = mount(ProjectDetailInlineEditableFields, {
        props: {
          field: 'auto_run',
          project: {
            ...baseProject,
            server_run_template: {
              ...baseProject.server_run_template,
              default_auto_run: false,
            },
          },
          tenantId: 't1',
          projectId: 'proj_1',
          gitAuthGate: {
            blocked: true,
            code: 'AUTO_RUN_NESTED_REPOS_UNAVAILABLE',
            message: '无法获取子 Git 仓库列表，自动运行无法启动；请先解决授权或网络问题后重试',
            displayLabel: '无法启动',
            traceId: 'tid-auto-run-nested-1',
          },
        },
      })
      await flushPromises()
      await nextTick()
      const nestedHint = nestedWrapper.find('[data-testid="project-auto-run-git-gate-hint"]')
      expect(nestedHint.text()).toContain('无法获取子 Git 仓库列表')
      expect(nestedHint.attributes('data-traceid') || nestedHint.attributes('data-traceId')).toBe(
        'tid-auto-run-nested-1',
      )
      nestedWrapper.unmount()
    })

    it('saves image via PATCH with container_image_id', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ ...baseProject, container_image_id: '9', container_image: 'other' }),
        jsonPreservingSnowflakeIds: async () => ({
          ...baseProject,
          container_image_id: '9',
          container_image: 'other',
        }),
      })
      const wrapper = mount(ProjectDetailInlineEditableFields, {
        props: {
          field: 'image',
          project: baseProject,
          installedImages,
          tenantId: 't1',
          projectId: 'proj_1',
        },
      })
      await wrapper.find('[data-testid="project-container-image-display"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="project-container-image-select"]').setValue('9')
      await wrapper.find('[data-testid="project-container-image-save"]').trigger('click')
      await flushPromises()
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/projects/proj_1/tenant_id/t1/',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({
            container_image_id: '9',
            container_image: 'other',
            server_run_template: baseProject.server_run_template,
          }),
        }),
      )
      expect(wrapper.emitted('saved')?.length).toBe(1)
    })
  })
}
