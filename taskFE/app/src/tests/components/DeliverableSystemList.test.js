// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] DeliverableSystemList.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))
  vi.mock('../../utils/cookieUtils.js', () => ({
    getCookie: vi.fn(() => 'test-cookie'),
  }))
  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: '850256677331562496' } }),
    useRouter: () => ({ push: vi.fn() }),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { default: DeliverableSystemList } = await import('../../views/DeliverableSystemList.vue')

  const okJSON = (data) => ({ ok: true, json: async () => data, _errorData: null })

  describe('DeliverableSystemList fetchDeliverableSystems', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      // 默认交付物体系端点
      apiFetch.mockImplementation((url) => {
        if (url.includes('/default-deliverable-system/')) {
          return Promise.resolve(okJSON({ status: 'success', default_deliverable_system_id: '1' }))
        }
        // 公司交付物体系列表：后端已含 is_system=1 系统级体系（company_id=? OR is_system=1）
        return Promise.resolve(okJSON([
          { id: '1', name: '公司体系', description: '', is_system: false, is_default: false, level_names: ['价值流'] },
          { id: 'ds_sys', name: '系统体系', description: '', is_system: true, is_default: false, level_names: ['活动'] },
        ]))
      })
    })

    it('仅请求带 tenant_id 的交付物体系与默认体系，不再发冗余 400 的 /api/projects/deliverable-systems/', async () => {
      const wrapper = mount(DeliverableSystemList, {
        global: { stubs: { 'router-link': true } },
      })
      await flushPromises()

      const called = apiFetch.mock.calls.map((c) => c[0])
      expect(called).toContain('/api/projects/deliverable-systems/tenant_id/850256677331562496')
      expect(called).toContain('/api/projects/default-deliverable-system/tenant_id/850256677331562496')
      // 冗余无 tenant 的系统级列表端点不得再被调用（OPT-20260808-005）
      expect(called.some((u) => u === '/api/projects/deliverable-systems/')).toBe(false)
      expect(called.filter((u) => u.startsWith('/api/projects/deliverable-systems/')).length).toBe(1)
    })

    it('合并列表包含公司级与系统级体系（is_system 取自后端，非强制覆盖）', async () => {
      const wrapper = mount(DeliverableSystemList, {
        global: { stubs: { 'router-link': true } },
      })
      await flushPromises()

      const systems = wrapper.vm.deliverableSystems
      expect(systems.length).toBe(2)
      const sys = systems.find((s) => s.id === 'ds_sys')
      expect(sys.is_system).toBe(true)
      const company = systems.find((s) => s.id === '1')
      expect(company.is_system).toBe(false)
    })
  })
}
