// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminFeedbackLinks.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoisted.apiFetchMock,
  }))
  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))

  describe('SystemAdminFeedbackLinks', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      hoisted.apiFetchMock.mockImplementation(async (url, opts = {}) => {
        if (String(url).includes('feedback-resource-kinds')) {
          return { ok: true, json: async () => ({ results: [{ kind: 'task_post', display_name: '帖', unit: '帖' }] }) }
        }
        if (opts.method === 'POST' || opts.method === 'PUT') {
          return { ok: true, json: async () => ({ id: '1', name: 'g' }) }
        }
        return { ok: true, json: async () => ({ results: [] }) }
      })
    })

    it('F5 保存只发一次且带 Idempotency-Key', async () => {
      const { default: Comp } = await import('./SystemAdminFeedbackLinks.vue')
      const w = mount(Comp)
      await flushPromises()
      await w.find('[data-testid=feedback-group-name]').setValue('社区')
      await w.find('[data-testid=feedback-add-link]').trigger('click')
      const title = w.find('[data-testid=feedback-link-title-0]')
      const url = w.find('[data-testid=feedback-link-url-0]')
      if (title.exists()) await title.setValue('问卷')
      if (url.exists()) await url.setValue('https://example.com/q')
      await w.find('[data-testid=feedback-save]').trigger('click')
      await w.find('[data-testid=feedback-save]').trigger('click')
      await flushPromises()
      const posts = hoisted.apiFetchMock.mock.calls.filter((c) => c[1]?.method === 'POST')
      expect(posts.length).toBe(1)
      expect(posts[0][1].headers['Idempotency-Key']).toBeTruthy()
    })
  })
}
