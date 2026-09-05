// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TenantConsoleFeedbackNav.unit.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: hoisted.apiFetchMock,
  }))

  describe('TenantConsoleFeedbackNav', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
    })

    it('F1 按组名分段且 href 与 API 一致', async () => {
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          groups: [
            { id: 'g1', name: '社区', links: [{ id: 'l1', title: '问卷', url: 'https://example.com/q' }] },
            { id: 'g2', name: '支持', links: [{ id: 'l2', title: '工单', url: 'https://example.com/t' }] },
          ],
        }),
      })
      const { default: Comp } = await import('./TenantConsoleFeedbackNav.vue')
      const w = mount(Comp, { props: { tenantId: '881', collapsed: false } })
      await flushPromises()
      w.find('[data-testid=tenant-feedback-toggle]').trigger('click')
      await flushPromises()
      const names = w.findAll('[data-testid=tenant-feedback-group-name]').map((n) => n.text())
      expect(names).toEqual(['社区', '支持'])
      const links = w.findAll('[data-testid=tenant-feedback-link]')
      expect(links[0].attributes('href')).toBe('https://example.com/q')
      expect(links[1].attributes('href')).toBe('https://example.com/t')
    })

    it('F2 空数组仍保留一级，子菜单空态', async () => {
      hoisted.apiFetchMock.mockResolvedValue({ ok: true, json: async () => ({ groups: [] }) })
      const { default: Comp } = await import('./TenantConsoleFeedbackNav.vue')
      const w = mount(Comp, { props: { tenantId: '881', collapsed: false } })
      await flushPromises()
      expect(w.find('[data-testid=tenant-feedback-toggle]').exists()).toBe(true)
      w.find('[data-testid=tenant-feedback-toggle]').trigger('click')
      await flushPromises()
      expect(w.find('[data-testid=tenant-feedback-empty]').text()).toContain('暂无链接')
    })

    it('F3 真实 a target=_blank 无 prevent', async () => {
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          groups: [{ id: 'g1', name: '社区', links: [{ id: 'l1', title: '问卷', url: 'https://example.com/q' }] }],
        }),
      })
      const { default: Comp } = await import('./TenantConsoleFeedbackNav.vue')
      const w = mount(Comp, { props: { tenantId: '881', collapsed: false } })
      await flushPromises()
      w.find('[data-testid=tenant-feedback-toggle]').trigger('click')
      await flushPromises()
      const a = w.find('[data-testid=tenant-feedback-link]')
      expect(a.element.tagName).toBe('A')
      expect(a.attributes('target')).toBe('_blank')
      expect(a.attributes('rel')).toContain('noopener')
    })

    it('F6 缩窄时点击发出 expand-sidebar', async () => {
      hoisted.apiFetchMock.mockResolvedValue({ ok: true, json: async () => ({ groups: [] }) })
      const { default: Comp } = await import('./TenantConsoleFeedbackNav.vue')
      const w = mount(Comp, { props: { tenantId: '881', collapsed: true } })
      await flushPromises()
      await w.find('[data-testid=tenant-feedback-toggle]').trigger('click')
      expect(w.emitted('expand-sidebar')).toBeTruthy()
    })
  })
}
