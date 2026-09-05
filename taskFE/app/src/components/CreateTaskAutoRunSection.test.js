// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskAutoRunSection.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  const { default: CreateTaskAutoRunSection } = await import('./CreateTaskAutoRunSection.vue')

  const configuredProject = {
    id: 'p1',
    name: 'Demo',
    server_run_template: {
      default_auto_run: true,
      platform: 'aliyun',
      region: 'cn-qingdao',
      selected_instance: 'ecs.e-c1m4.large',
    },
  }

  const makeTask = (overrides = {}) => ({
    auto_run: true,
    queued_auto_run: false,
    force_auto_run: false,
    container_image: { id: 'img-1' },
    projectSelections: [{ projectId: 'p1' }],
    ...overrides,
  })

  const jsonResponse = (body, status = 200, headers = {}) => ({
    ok: status >= 200 && status < 300,
    status,
    headers: { get: (k) => headers[k] || '' },
    json: async () => body,
  })

  describe('CreateTaskAutoRunSection queued auto run', () => {
    afterEach(() => {
      vi.clearAllMocks()
    })

    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockResolvedValue(
        jsonResponse({ schedule_rhythm: { enabled: true } }),
      )
    })

    it('T1 shows queue checkbox unchecked when workspace schedule is enabled', async () => {
      const editingTask = makeTask()
      const wrapper = mount(CreateTaskAutoRunSection, {
        props: {
          editingTask,
          projects: [configuredProject],
          installedImages: [{ id: 'img-1', name: 'img' }],
          tenantId: 't1',
          workspaceId: 'ws1',
          show: true,
        },
      })
      await flushPromises()
      const box = wrapper.find('[data-testid="task-queued-auto-run-checkbox"]')
      expect(box.exists()).toBe(true)
      expect(box.element.checked).toBe(false)
      expect(wrapper.find('[data-testid="task-queued-auto-run-hint"]').text()).toContain('不立即启服')
      wrapper.unmount()
    })

    it('T2 hides queue checkbox when workspace schedule is disabled', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(
        jsonResponse({ schedule_rhythm: { enabled: false } }),
      )
      const wrapper = mount(CreateTaskAutoRunSection, {
        props: {
          editingTask: makeTask(),
          projects: [configuredProject],
          installedImages: [{ id: 'img-1', name: 'img' }],
          tenantId: 't1',
          workspaceId: 'ws1',
          show: true,
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="task-queued-auto-run-checkbox"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('T5 hides queue checkbox when auto_run is turned off', async () => {
      const editingTask = makeTask({ auto_run: true })
      const wrapper = mount(CreateTaskAutoRunSection, {
        props: {
          editingTask,
          projects: [configuredProject],
          installedImages: [{ id: 'img-1', name: 'img' }],
          tenantId: 't1',
          workspaceId: 'ws1',
          show: true,
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="task-queued-auto-run-checkbox"]').exists()).toBe(true)
      editingTask.queued_auto_run = true
      await wrapper.find('[data-testid="task-auto-run-checkbox"]').setValue(false)
      await flushPromises()
      expect(editingTask.auto_run).toBe(false)
      expect(editingTask.queued_auto_run).toBe(false)
      expect(wrapper.find('[data-testid="task-queued-auto-run-checkbox"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('T6 GET failure shows error with data-traceId and hides checkbox', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(
        jsonResponse({ error: 'boom' }, 500, { 'X-Trace-Id': 'trace-q' }),
      )
      const wrapper = mount(CreateTaskAutoRunSection, {
        props: {
          editingTask: makeTask(),
          projects: [configuredProject],
          installedImages: [{ id: 'img-1', name: 'img' }],
          tenantId: 't1',
          workspaceId: 'ws1',
          show: true,
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="task-queued-auto-run-checkbox"]').exists()).toBe(false)
      const err = wrapper.find('[data-testid="task-queued-auto-run-schedule-error"]')
      expect(err.exists()).toBe(true)
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('trace-q')
      wrapper.unmount()
    })
  })
}
