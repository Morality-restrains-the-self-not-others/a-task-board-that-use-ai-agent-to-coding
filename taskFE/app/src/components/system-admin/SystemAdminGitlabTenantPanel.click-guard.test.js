// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGitlabTenantPanel.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言开通 POST 携带 Idempotency-Key 头。
  vi.mock('../../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Panel } = await import('./SystemAdminGitlabTenantPanel.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
      clone: async () => ({ json: async () => body }),
    }
  }

  describe('SystemAdminGitlabTenantPanel 开通实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('查询后点开通实施发起一次 POST 且携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') {
          return jsonOk({ ok: true })
        }
        return jsonOk({
          tenant_id: 't-1',
          region: 'tencent-sh-1',
          provisioning_status: 'pending_admin',
          disk_gb: 1,
          disk_used_gb: 0,
          traffic_prepaid_gb: 0,
          traffic_used_gb: 0,
        })
      })
      const wrapper = mount(Panel)
      await wrapper.find('input[placeholder="输入租户 ID 查询"]').setValue('t-1')
      await wrapper.find('input[placeholder="区域 slug（必填）"]').setValue('tencent-sh-1')
      await wrapper.find('button').trigger('click') // 查询
      await flushPromises()

      const provisionBtn = wrapper.findAll('button').find((b) => b.text().includes('开通实施'))
      expect(provisionBtn).toBeTruthy()
      await provisionBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toContain('/gitlab-resources/tenant_id/t-1/provision/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
      expect(postCall[1].body).toContain('"region":"tencent-sh-1"')
    })
  })
}
