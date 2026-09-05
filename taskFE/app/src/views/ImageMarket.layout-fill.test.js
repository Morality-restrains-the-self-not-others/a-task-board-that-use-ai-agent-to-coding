// @vitest-environment jsdom
/**
 * 镜像市场布局：内容区 flex 列撑满视窗，已安装镜像卡片 flex-1 贴底（WorkPanel 同款 fillRemaining 模式）。
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] ImageMarket.layout-fill.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  vi.mock('../utils/apiUtils', () => ({
    apiFetch: apiFetchMock,
    extractErrorMessage: vi.fn(() => ''),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: '877397588196749312' }, query: {} }),
    useRouter: () => ({ replace: vi.fn() }),
  }))

  const { default: ImageMarket } = await import('./ImageMarket.vue')

  const jsonResponse = (body) => ({
    ok: true,
    json: async () => body,
    text: async () => JSON.stringify(body),
  })

  beforeEach(() => {
    apiFetchMock.mockReset()
    apiFetchMock.mockImplementation(async (url) => {
      const u = String(url)
      if (u.includes('vendor-status')) {
        return jsonResponse({ is_vendor: true, status: 'qualified', has_email: true, vendor: null })
      }
      // 目录/已安装/开发中均返回空，避免阻塞挂载
      return jsonResponse([])
    })
  })

  describe('ImageMarket.vue 布局撑满', () => {
    it('内容区根元素为 flex 列容器（配合壳层 flex-1 撑满视窗高度）', async () => {
      const wrapper = mount(ImageMarket)
      await flushPromises()
      const root = wrapper.element
      expect(root.classList.contains('flex')).toBe(true)
      expect(root.classList.contains('flex-col')).toBe(true)
      expect(root.classList.contains('min-h-0')).toBe(true)
    })

    it('「已安装镜像」卡片为 flex-1，内容短于视窗时贴底撑满剩余高度', async () => {
      const wrapper = mount(ImageMarket)
      await flushPromises()
      const installedCard = wrapper.find('.mt-6.rounded-xl')
      expect(installedCard.classes()).toContain('flex-1')
      // 壳层 fallthrough 类也应保留（App.vue router-view 下发的 flex-1 min-h-0 min-w-0 w-full）
      expect(installedCard.element.parentElement.classList.contains('max-w-7xl')).toBe(true)
    })
  })
}
