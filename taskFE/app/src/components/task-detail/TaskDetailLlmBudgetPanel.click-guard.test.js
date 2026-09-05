// @vitest-environment jsdom
// OPT-20260819-038: LLM 预算覆盖/上调（资金路径）接入 createClickGuard + Idempotency-Key。
// 回归断言：保存覆盖 PATCH 与临时上调 POST 均携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailLlmBudgetPanel.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))
  vi.mock('../../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-llm-test' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Panel } = await import('./TaskDetailLlmBudgetPanel.vue')

  function jsonOk(body = {}) {
    return {
      ok: true,
      status: 200,
      json: async () => body,
      traceId: '',
      headers: { get: () => 'application/json' },
    }
  }

  function setupApi() {
    apiFetchMock.mockReset()
    apiFetchMock.mockImplementation((url, options) => {
      const method = options?.method || 'GET'
      if (String(url).includes('/feature-params/tenant_id/')) {
        return Promise.resolve(jsonOk({ data: { llm_budget_enabled: true } }))
      }
      if (method === 'GET' && String(url).includes('/model-budgets/')) {
        return Promise.resolve(jsonOk({
          items: [
            { provider: 'deepseek', base_url: 'https://api.deepseek.com', model_name: 'deepseek-chat', spent_amount: '1.00', effective_budget_limit: '10', budget_limit_source: 'inherit' },
          ],
        }))
      }
      return Promise.resolve(jsonOk({}))
    })
  }

  const props = {
    tenantId: '123',
    workspaceId: 'ws-1',
    taskId: 'task-1',
    canEdit: true,
    canRaise: true,
  }

  describe('TaskDetailLlmBudgetPanel 预算写操作 clickGuard 接线', () => {
    beforeEach(() => {
      setupApi()
      vi.stubGlobal('confirm', vi.fn(() => true))
    })

    it('保存覆盖 PATCH 携带 Idempotency-Key', async () => {
      const wrapper = mount(Panel, { props })
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().includes('保存覆盖'))
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const patches = apiFetchMock.mock.calls.filter(([, o]) => o?.method === 'PATCH')
      expect(patches).toHaveLength(1)
      const [, options] = patches[0]
      expect(options.headers['Idempotency-Key']).toBe('ik-llm-test')
    })

    it('临时上调 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Panel, { props })
      await flushPromises()

      const raiseBtn = wrapper.findAll('button').find((b) => b.text().includes('临时上调'))
      expect(raiseBtn).toBeTruthy()
      await raiseBtn.trigger('click')
      await flushPromises()

      const posts = apiFetchMock.mock.calls.filter(([, o]) => o?.method === 'POST')
      expect(posts).toHaveLength(1)
      const [, options] = posts[0]
      expect(options.headers['Idempotency-Key']).toBe('ik-llm-test')
    })
  })
}
