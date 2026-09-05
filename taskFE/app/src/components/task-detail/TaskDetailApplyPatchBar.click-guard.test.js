// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailApplyPatchBar.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  // OPT-20260819-038: 实验性 Apply patch 面板写请求同样须防连点双发 POST。
  import.meta.env.VITE_ENABLE_APPLY_PATCH = 'true'

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  vi.mock('../../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-patch' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Component } = await import('./TaskDetailApplyPatchBar.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('TaskDetailApplyPatchBar 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async () => jsonOk({ ok: true }))
    })

    it('Apply patch POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, {
        props: {
          tenantId: 'tenant-1',
          workspaceId: 'ws-1',
          taskId: 'task-1',
        },
      })
      await flushPromises()

      const textarea = wrapper.find('textarea')
      await textarea.setValue('--- a/foo\n+++ b/foo\n@@ -1 +1 @@\n-x\n+y\n')
      await wrapper.find('button').trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/cloud/container-apply-patch/tenant_id/tenant-1/workspace_id/ws-1/task_id/task-1')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-patch')
      expect(postCall[1].body).toContain('+++ b/foo')
    })
  })
}
