// @vitest-environment jsdom
// 任务详情「添加评论」自动执行卡内嵌：加入/离开队列 + 状态 chip + 「自动调度安排」入口。
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailQueuedScheduleToggle.test.js requires vitest runtime')
} else {
const { afterEach, describe, expect, it, vi } = await import('vitest')
const { flushPromises, mount } = await import('@vue/test-utils')

vi.mock('../../utils/modalService.js', () => ({
  default: { confirm: vi.fn() },
}))

const { default: TaskDetailQueuedScheduleToggle } = await import('./TaskDetailQueuedScheduleToggle.vue')
const { default: modalService } = await import('../../utils/modalService.js')

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

function jsonResp(body, { ok = true, status = 200, traceId = null } = {}) {
  return {
    ok,
    status,
    headers: { get: (k) => (String(k).toLowerCase() === 'x-trace-id' ? traceId : null) },
    json: async () => body,
  }
}

function stubApiFetch({
  enabled = true,
  patchBody = { id: 't1', queued_auto_run: true },
  patchOk = true,
  patchTrace = null,
  getOk = true,
  getTrace = null,
} = {}) {
  const apiFetch = vi.fn(async (url, opts) => {
    const u = String(url)
    const method = opts?.method || 'GET'
    if (u.includes('/queue-schedule/') && method === 'GET') {
      if (!getOk) {
        return jsonResp(
          { error: 'qs fail', trace_id: getTrace },
          { ok: false, status: 500, traceId: getTrace },
        )
      }
      return jsonResp({
        schedule_rhythm: enabled ? { enabled: true } : { enabled: false },
      })
    }
    if (method === 'PATCH') {
      return jsonResp(patchBody, {
        ok: patchOk,
        status: patchOk ? 200 : 500,
        traceId: patchTrace,
      })
    }
    return jsonResp({})
  })
  vi.stubGlobal('apiFetch', apiFetch)
  return apiFetch
}

function mountToggle(task) {
  return mount(TaskDetailQueuedScheduleToggle, {
    props: {
      task,
      tenantId: 'ten1',
      workspaceId: 'ws1',
    },
    global: {
      stubs: {
        'router-link': {
          props: ['to'],
          template: '<a :href="typeof to === \'string\' ? to : \'\'"><slot /></a>',
        },
      },
    },
    attachTo: document.body,
  })
}

function patchCalls(apiFetch) {
  return apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'PATCH')
}

describe('TaskDetailQueuedScheduleToggle', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
    modalService.confirm.mockReset()
  })

  it('未入队：展示「加入自动执行队列」与页面入口链接，无节奏表单', () => {
    const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: false })
    expect(wrapper.find('[data-testid="queued-auto-run-join"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="queued-auto-run-leave"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="schedule-rhythm-enabled"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="task-queued-schedule-open"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="task-queued-schedule-modal"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('已入队：展示「离开队列」与状态 chip', () => {
    const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: true, queued_auto_run_status: 'queued' })
    expect(wrapper.find('[data-testid="queued-auto-run-join"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="queued-auto-run-leave"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="queued-auto-run-status-chip"]').text()).toContain('排队中')
    wrapper.unmount()
  })

  it('入口链接指向自动调度安排页面并携带 workspace_id', () => {
    const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: false })
    const link = wrapper.find('[data-testid="queued-schedule-page-link"]')
    expect(link.attributes('href')).toBe('/tenant/ten1/queue-schedule/?workspace_id=ws1')
    expect(link.text()).toContain('自动调度安排')
    wrapper.unmount()
  })

  it('工作空间已启用自动调度：GET 后 PATCH queued_auto_run=true 并上抛 updated', async () => {
    const apiFetch = stubApiFetch({ enabled: true })
    const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: false })
    await wrapper.find('[data-testid="queued-auto-run-join"]').trigger('click')
    await flushPromises()
    const patches = patchCalls(apiFetch)
    expect(apiFetch.mock.calls[0][0]).toBe('/api/tenant/ten1/workspace/ws1/queue-schedule/')
    expect(apiFetch.mock.calls[0][1].method).toBe('GET')
    expect(patches).toHaveLength(1)
    expect(patches[0][0]).toBe('/api/tenant/ten1/workspace/ws1/todos/t1/')
    expect(JSON.parse(patches[0][1].body).queued_auto_run).toBe(true)
    expect(wrapper.emitted('updated')).toBeTruthy()
    expect(modalService.confirm).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('工作空间未启用自动调度且取消：不 PATCH', async () => {
    const apiFetch = stubApiFetch({ enabled: false })
    modalService.confirm.mockRejectedValue(false)
    const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: false })
    await wrapper.find('[data-testid="queued-auto-run-join"]').trigger('click')
    await flushPromises()
    expect(modalService.confirm).toHaveBeenCalled()
    expect(String(modalService.confirm.mock.calls[0][0])).toContain('尚未启用自动调度')
    expect(patchCalls(apiFetch)).toHaveLength(0)
    expect(wrapper.emitted('updated')).toBeFalsy()
    wrapper.unmount()
  })

  it('工作空间未启用且确认前往设置：location.assign 指向 queue-schedule', async () => {
    const apiFetch = stubApiFetch({ enabled: false })
    modalService.confirm.mockResolvedValue(true)
    const assign = vi.fn()
    vi.stubGlobal('location', { assign, href: 'http://localhost/task' })
    const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: false })
    await wrapper.find('[data-testid="queued-auto-run-join"]').trigger('click')
    await flushPromises()
    expect(patchCalls(apiFetch)).toHaveLength(0)
    expect(assign).toHaveBeenCalledWith('/tenant/ten1/queue-schedule/?workspace_id=ws1')
    wrapper.unmount()
  })

  it('GET 调度失败 → 错误块挂 data-traceId 且不 PATCH', async () => {
    const apiFetch = stubApiFetch({ getOk: false, getTrace: 'tr-qs-err' })
    const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: false })
    await wrapper.find('[data-testid="queued-auto-run-join"]').trigger('click')
    await flushPromises()
    expect(patchCalls(apiFetch)).toHaveLength(0)
    const errEl = wrapper.find('[data-testid="queued-auto-run-save-error"]')
    expect(errEl.exists()).toBe(true)
    const lowerName = 'data-traceid'
    for (const attr of errEl.element.attributes) {
      if (attr.name.toLowerCase() === lowerName) {
        expect(attr.value).toBe('tr-qs-err')
        wrapper.unmount()
        return
      }
    }
    wrapper.unmount()
    expect.fail('data-traceId attribute missing')
  })

  it('PATCH 失败 → 错误块挂 data-traceId', async () => {
    const apiFetch = stubApiFetch({
      enabled: true,
      patchOk: false,
      patchBody: { trace_id: 'tr-err' },
      patchTrace: 'tr-err',
    })
    const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: false })
    await wrapper.find('[data-testid="queued-auto-run-join"]').trigger('click')
    await flushPromises()
    const errEl = wrapper.find('[data-testid="queued-auto-run-save-error"]')
    expect(errEl.exists()).toBe(true)
    const lowerName = 'data-traceid'
    for (const attr of errEl.element.attributes) {
      if (attr.name.toLowerCase() === lowerName) {
        expect(attr.value).toBe('tr-err')
        wrapper.unmount()
        return
      }
    }
    wrapper.unmount()
    expect.fail('data-traceId attribute missing')
  })
})
}
