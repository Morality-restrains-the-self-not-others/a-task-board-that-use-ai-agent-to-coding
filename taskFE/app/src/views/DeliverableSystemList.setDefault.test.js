// @vitest-environment jsdom
// OPT-20260807-053 回归测试：设置默认交付物体系必须使用真实体系 id。
// 根因：迁移 e105bad/7bd031b 将系统交付物体系 URL 从 ${deliverableSystem.id}
// 误改为硬编码 ${1}；系统级体系 id 为 ds_xxx 格式（如 ds_default_global），
// 后端按 id=1 查询 → 404「交付物体系不存在」→ 页面弹窗报错且无 traceId。
if (!process.env.VITEST) {
  console.log('[skip] DeliverableSystemList.setDefault.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    showRequestError: vi.fn(),
    useRoute: () => ({ params: { tenant: '873472655125147648' }, query: {}, name: 'deliverable_systems', fullPath: '/', path: '/' }),
    useRouter: () => ({ push: vi.fn() }),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: (...args) => mocks.showRequestError(...args),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.useRoute(),
    useRouter: () => mocks.useRouter(),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => 'x' }))

  const SYSTEM_SYS = { id: 'ds_default_global', name: '系统默认', description: '', is_system: true, is_default: false, level_names: ['价值流', '活动'] }
  const COMPANY_SYS = { id: 'ds_company_a', name: '公司体系A', description: '', is_system: false, is_default: false, level_names: ['价值流', '活动'] }

  /** 模拟 onMounted 的列表加载：2 个并行 GET。
   * 后端公司列表已含系统级体系（company_id=? OR is_system=1），
   * 无独立系统级 GET（OPT-20260808-005 移除冗余无 tenant 的 400 调用）。 */
  function mockListLoad() {
    mocks.apiFetch.mockImplementation((url) => {
      if (url.includes('deliverable-systems/tenant_id/') && !url.includes('set-default')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([COMPANY_SYS, SYSTEM_SYS]) })
      }
      if (url.includes('default-deliverable-system')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success' }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    })
  }

  /** 模拟 set-default POST 响应 */
  function mockPostResponse(status, body) {
    mocks.apiFetch.mockImplementation((url, options) => {
      if (options?.method === 'POST') {
        const resp = {
          ok: status === 200,
          status,
          traceId: 'trace-uuid-20260807-abc',
          // 真实 apiFetch 会将非 ok 响应体附着为 _errorData；_safeParseSetDefaultResult 依赖它提取 message
          _errorData: body,
          json: () => Promise.resolve(body),
        }
        return Promise.resolve(resp)
      }
      // 列表加载回退
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  const { default: DeliverableSystemList } = await import('./DeliverableSystemList.vue')

describe('set-default URL 构建（根因修复：禁止硬编码 ${1}）', () => {
    it('系统交付物体系 set-default 使用真实 id（非 1）', async () => {
      mockListLoad()
      const wrapper = mount(DeliverableSystemList)
      await flushPromises()
      mockPostResponse(200, { status: 'success', systems: [] })

      await wrapper.vm.setAsDefault(SYSTEM_SYS)
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/projects/deliverable-systems/ds_default_global/set-default/tenant_id/873472655125147648/')
      expect(postCall[0]).not.toContain('deliverable-systems/1/set-default')
    })

    it('公司交付物体系 set-default 使用真实 id', async () => {
      mockListLoad()
      const wrapper = mount(DeliverableSystemList)
      await flushPromises()
      mockPostResponse(200, { status: 'success', systems: [] })

      await wrapper.vm.setAsDefault(COMPANY_SYS)
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall[0]).toBe('/api/projects/deliverable-systems/tenant_id/873472655125147648/ds_company_a/set-default')
    })

    it('select change（真实用户路径）对系统体系使用真实 id', async () => {
      mockListLoad()
      const wrapper = mount(DeliverableSystemList)
      await flushPromises()
      mockPostResponse(200, { status: 'success', systems: [] })

      await wrapper.find('select').setValue('ds_default_global')
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/projects/deliverable-systems/ds_default_global/set-default/tenant_id/873472655125147648/')
    })
  })

  describe('错误响应 traceId 传递（data-traceId 链路）', () => {
    it('404 时 showRequestError 携带 response.traceId', async () => {
      mockListLoad()
      const wrapper = mount(DeliverableSystemList)
      await flushPromises()
      // 模拟修复前的线上场景：后端 404 + 响应体 message
      mockPostResponse(404, { status: 'error', message: '交付物体系不存在' })

      await wrapper.vm.setAsDefault(SYSTEM_SYS)
      await flushPromises()

      expect(mocks.showRequestError).toHaveBeenCalledTimes(1)
      const [msg, source] = mocks.showRequestError.mock.calls[0]
      expect(msg).toContain('设置默认交付物体系失败: 交付物体系不存在')
      // trace 源必须是响应对象（带 traceId），而非错误文案
      expect(source.traceId).toBe('trace-uuid-20260807-abc')
    })
  })

  describe('OPT-20260808-004 回归：保留后端 is_default 标注（不强制覆盖）', () => {
    it('后端列表标注 is_default=true 且默认端点无 id 时，标注不被前端覆盖', async () => {
      // Red 场景：修复前 processedCompanyDeliverableSystems 强制 is_default:false，
      // 且 default-deliverable-system 无 id（override 分支不执行）→ 标注丢失。
      // 修复后保留后端标注 → is_default 必须为 true。
      mocks.apiFetch.mockImplementation((url) => {
        if (url.includes('deliverable-systems/tenant_id/') && !url.includes('set-default')) {
          return Promise.resolve({ ok: true, json: () => Promise.resolve([{ ...COMPANY_SYS, is_default: true }]) })
        }
        if (url.includes('default-deliverable-system')) {
          return Promise.resolve({ ok: true, json: () => Promise.resolve({ status: 'success' }) })
        }
        return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
      })
      const wrapper = mount(DeliverableSystemList)
      await flushPromises()

      const systems = wrapper.vm.deliverableSystems
      expect(systems.find(s => s.id === 'ds_company_a').is_default).toBe(true)
    })
  })

  describe('成功响应字段对齐（后端返回 systems 而非 company_/system_ 旧字段）', () => {
    it('result.systems 直接更新列表（含 is_default 标注）', async () => {
      mockListLoad()
      const wrapper = mount(DeliverableSystemList)
      await flushPromises()
      const updated = [
        { id: 'ds_company_a', name: '公司体系A', is_system: false, is_default: true },
        { id: 'ds_default_global', name: '系统默认', is_system: true, is_default: false },
      ]
      mockPostResponse(200, { status: 'success', message: '已设置为默认交付物体系', systems: updated })

      await wrapper.vm.setAsDefault(COMPANY_SYS)
      await flushPromises()

      const systems = wrapper.vm.deliverableSystems
      expect(systems.length).toBe(2)
      expect(systems.find(s => s.id === 'ds_company_a').is_default).toBe(true)
    })
  })
}
