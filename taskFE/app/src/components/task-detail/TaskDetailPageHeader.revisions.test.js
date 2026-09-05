// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailPageHeader.revisions.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../../utils/cookieUtils.js', () => ({
    getCookie: () => 'user-1',
  }))

  const TaskDetailPageHeader = (await import('./TaskDetailPageHeader.vue')).default

  describe('TaskDetailPageHeader revisions', () => {
    it('renders 历史版本 when task id is present and does not GET until opened', () => {
      hoistedMocks.apiFetchMock.mockReset()
      const wrapper = mount(TaskDetailPageHeader, {
        props: {
          hasTask: true,
          isEditing: false,
          isForking: false,
          isSaving: false,
          backToWorkPanelRoute: '/work-panel',
          forkTask: vi.fn(async () => true),
          tenantId: 't1',
          workspaceId: 'w1',
          sourceTask: { id: 'task-1', feature_params_source: 'company' },
          taskProjectsWithDetails: [],
        },
        global: {
          stubs: {
            'router-link': { template: '<a><slot /></a>' },
            ForkAutoRunConfirmModal: { template: '<div />' },
          },
        },
      })
      expect(wrapper.find('[data-testid="entity-revision-open"]').exists()).toBe(true)
      expect(hoistedMocks.apiFetchMock.mock.calls.some((call) => String(call[0]).includes('/revisions/'))).toBe(false)
    })
  })
}
