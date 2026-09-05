// @vitest-environment jsdom
// OPT-20260819-038: 加入/离开自动执行队列是写操作（PATCH），防连点双发。
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailQueuedScheduleToggle.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PATCH 携带 Idempotency-Key 头。
  vi.mock('../../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-queued' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Component } = await import('./TaskDetailQueuedScheduleToggle.vue')

  function mountToggle(task) {
    return mount(Component, {
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

  describe('TaskDetailQueuedScheduleToggle 写操作 clickGuard 接线', () => {
    let apiFetch
    beforeEach(() => {
      apiFetch = vi.fn(async (url, opts) => {
        const method = opts?.method || 'GET'
        if (String(url).includes('/queue-schedule/') && method === 'GET') {
          return {
            ok: true,
            headers: { get: () => null },
            json: async () => ({ schedule_rhythm: { enabled: true } }),
          }
        }
        return {
          ok: true,
          headers: { get: () => null },
          json: async () => ({ id: 't1', queued_auto_run: true }),
        }
      })
      vi.stubGlobal('apiFetch', apiFetch)
    })
    afterEach(() => {
      vi.unstubAllGlobals()
    })

    it('加入队列 PATCH 携带 Idempotency-Key', async () => {
      const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: false })
      await wrapper.find('[data-testid="queued-auto-run-join"]').trigger('click')
      await flushPromises()

      const patchCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PATCH')
      expect(patchCall).toBeTruthy()
      const [url, opts] = patchCall
      expect(url).toBe('/api/tenant/ten1/workspace/ws1/todos/t1/')
      expect(opts.method).toBe('PATCH')
      expect(opts.headers['Idempotency-Key']).toBe('ik-test-queued')
      expect(JSON.parse(opts.body).queued_auto_run).toBe(true)
      wrapper.unmount()
    })

    it('离开队列 PATCH 携带 Idempotency-Key', async () => {
      const wrapper = mountToggle({ id: 't1', parent_task: null, queued_auto_run: true, queued_auto_run_status: 'queued' })
      await wrapper.find('[data-testid="queued-auto-run-leave"]').trigger('click')
      await flushPromises()

      const [url, opts] = apiFetch.mock.calls[0]
      expect(url).toBe('/api/tenant/ten1/workspace/ws1/todos/t1/')
      expect(opts.method).toBe('PATCH')
      expect(opts.headers['Idempotency-Key']).toBe('ik-test-queued')
      expect(JSON.parse(opts.body).queued_auto_run).toBe(false)
      wrapper.unmount()
    })
  })
}
