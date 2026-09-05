// @vitest-environment jsdom
/**
 * 镜像市场：同一镜像组不同版本叠列 —— 一组一张卡，卡内为版本列表。
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] ImageMarket.group-versions.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

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

  const makeImage = (overrides) => ({
    id: '1',
    name: 'trae-agent',
    description: '一个更精简的镜像版本',
    version: 'x86_64-latest',
    target_architectures: ['x86_64'],
    vendor: { id: '10', company_name: 'Acme' },
    auto_run_steps_md: '',
    auto_run_steps_extract_status: '',
    runtime_environments: [],
    updated_at: '2026-08-24T04:46:06Z',
    ...overrides,
  })

  const mountWith = async ({ devImages }) => {
    apiFetchMock.mockReset()
    apiFetchMock.mockImplementation(async (url) => {
      const u = String(url)
      if (u.includes('vendor-status')) {
        return jsonResponse({ is_vendor: true, status: 'qualified', has_email: true, vendor: null })
      }
      if (u.includes('dev-catalog')) {
        return jsonResponse(devImages)
      }
      return jsonResponse([])
    })
    const wrapper = mount(ImageMarket, {
      global: { stubs: { 'runtime-deps-block': true } },
    })
    await flushPromises()
    return wrapper
  }

  const groupCards = (wrapper) => wrapper.findAll('ul.mt-6.space-y-4 > li')
  const installButtons = (card) => card.findAll('button').filter((b) => b.text().includes('安装'))

  describe('ImageMarket.vue 同组不同版本叠列', () => {
    it('同组两版本合并为一张卡，卡内版本列表展示两个版本与各自安装按钮', async () => {
      const wrapper = await mountWith({
        devImages: [
          makeImage({ id: '1', version: 'x86_64-latest', image_group: { id: 'g1', name: 'trae-agent' } }),
          makeImage({ id: '2', version: 'private_x86_64-latest', image_group: { id: 'g1', name: 'trae-agent' } }),
          makeImage({ id: '3', name: 'other-image', version: '1.0', image_group: { id: 'g2', name: 'other-image' } }),
        ],
      })

      const cards = groupCards(wrapper)
      expect(cards.length).toBe(2)
      const first = cards[0]
      expect(first.text()).toContain('trae-agent')
      expect(first.text()).toContain('x86_64-latest')
      expect(first.text()).toContain('private_x86_64-latest')
      expect(installButtons(first).length).toBe(2)
      expect(cards[1].text()).toContain('other-image')
      expect(installButtons(cards[1]).length).toBe(1)
    })

    it('无 image_group 字段时按 name 分组', async () => {
      const wrapper = await mountWith({
        devImages: [makeImage({ id: '1', version: 'v1' }), makeImage({ id: '2', version: 'v2' })],
      })
      const cards = groupCards(wrapper)
      expect(cards.length).toBe(1)
      expect(cards[0].text()).toContain('v1')
      expect(cards[0].text()).toContain('v2')
    })

    it('搜索命中单个版本时，组内仅保留命中版本行', async () => {
      const wrapper = await mountWith({
        devImages: [
          makeImage({ id: '1', version: 'v-public', image_group: { id: 'g1', name: 'trae-agent' } }),
          makeImage({ id: '2', version: 'v-private', image_group: { id: 'g1', name: 'trae-agent' } }),
        ],
      })
      const search = wrapper.find('input[type="search"]')
      await search.setValue('v-private')
      await flushPromises()
      const cards = groupCards(wrapper)
      expect(cards.length).toBe(1)
      expect(cards[0].text()).toContain('v-private')
      expect(cards[0].text()).not.toContain('v-public')
    })
  })
}
