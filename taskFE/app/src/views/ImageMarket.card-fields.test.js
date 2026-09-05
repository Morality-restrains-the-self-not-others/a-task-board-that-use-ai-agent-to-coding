// @vitest-environment jsdom
/**
 * 镜像市场卡片：展示更新时间；自动运行为空时不占位。
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] ImageMarket.card-fields.test.js requires vitest runtime')
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
  })

  describe('ImageMarket.vue 卡片字段', () => {
    it('开发中镜像展示更新时间，空自动运行不出现「暂无自动运行说明」', async () => {
      apiFetchMock.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('vendor-status')) {
          return jsonResponse({ is_vendor: true, status: 'qualified', has_email: true, vendor: null })
        }
        if (u.includes('dev-catalog')) {
          return jsonResponse([
            {
              id: '31',
              name: 'empty-auto-run',
              description: 'no steps',
              version: '1.0',
              icon_url: '/api/public/image-groups/20/icon?h=abc',
              target_architectures: ['x86_64'],
              vendor: { id: '10', company_name: 'Acme' },
              auto_run_steps_md: '',
              auto_run_steps_extract_status: '',
              updated_at: '2026-08-20T12:00:00Z',
              runtime_environments: [],
            },
            {
              id: '32',
              name: 'with-auto-run',
              description: 'has steps',
              version: '2.0',
              target_architectures: ['x86_64'],
              vendor: { id: '10', company_name: 'Acme' },
              auto_run_steps_md: '# steps',
              auto_run_steps_extract_status: 'ok',
              updated_at: '2026-08-19T08:00:00Z',
              runtime_environments: [],
            },
          ])
        }
        if (u.includes('/catalog/')) {
          return jsonResponse([
            {
              id: '41',
              name: 'published-empty',
              description: 'pub',
              version: '3.0',
              target_architectures: ['x86_64'],
              vendor: { id: '10', company_name: 'Acme' },
              auto_run_steps_md: '',
              auto_run_steps_extract_status: 'not_found',
              updated_at: '2026-08-18T00:00:00Z',
              runtime_environments: [],
            },
          ])
        }
        return jsonResponse([])
      })
      const wrapper = mount(ImageMarket, {
        global: { stubs: { 'runtime-deps-block': true } },
      })
      await flushPromises()

      const updated = wrapper.findAll('[data-testid="image-updated-at"]')
      expect(updated.length).toBe(3)
      updated.forEach((el) => {
        expect(el.text()).toMatch(/更新时间/)
        expect(el.text()).not.toMatch(/—\s*$/)
        expect(el.text()).not.toContain('Invalid Date')
      })

      const cards = wrapper.findAll('ul.mt-6.space-y-4 li')
      expect(cards[0].text()).toContain('empty-auto-run')
      expect(cards[0].find('[data-testid="auto-run-steps-preview"]').exists()).toBe(false)
      expect(cards[0].text()).not.toContain('暂无自动运行说明')
      expect(cards[1].find('[data-testid="auto-run-steps-toggle"]').exists()).toBe(true)
      expect(cards[0].find('[data-testid="image-group-icon"]').attributes('src')).toBe(
        '/api/public/image-groups/20/icon?h=abc',
      )
      expect(cards[0].find('[data-testid="image-group-icon-placeholder"]').exists()).toBe(false)
      expect(cards[1].find('[data-testid="image-group-icon"]').exists()).toBe(false)
      expect(cards[1].find('[data-testid="image-group-icon-placeholder"]').exists()).toBe(true)
    })

    it('已安装镜像卡片展示图标：自身 icon_url 优先，否则用目录回退', async () => {
      apiFetchMock.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('vendor-status')) {
          return jsonResponse({ is_vendor: true, status: 'qualified', has_email: true, vendor: null })
        }
        if (u.includes('dev-catalog')) {
          return jsonResponse([])
        }
        if (u.includes('/catalog/')) {
          return jsonResponse([
            {
              id: '7319654012760069',
              name: 'trae-agent',
              description: 'pub',
              version: 'x86_64-latest',
              icon_url: '/api/ai-provider/public-image-groups/20/icon?h=abc',
              target_architectures: ['x86_64'],
              vendor: { id: '10', company_name: 'Acme' },
              auto_run_steps_md: '',
              auto_run_steps_extract_status: '',
              updated_at: '2026-08-27T12:21:00Z',
              runtime_environments: [],
            },
          ])
        }
        if (u.includes('installed-images')) {
          return jsonResponse([
            {
              id: 'inst-1',
              name: 'trae-agent',
              version: 'x86_64-latest',
              vendor_name: '',
              external_image_id: '7319654012760069',
              updated_at: '2026-08-27T12:21:00Z',
              runtime_environments: [],
            },
            {
              id: 'inst-2',
              name: 'own-icon',
              version: '1.0',
              icon_url: '/snapshotted-icon',
              external_image_id: 'missing-in-catalog',
              updated_at: '2026-08-27T12:21:00Z',
              runtime_environments: [],
            },
            {
              id: 'inst-3',
              name: 'no-icon',
              version: '1.0',
              updated_at: '2026-08-27T12:21:00Z',
              runtime_environments: [],
            },
          ])
        }
        return jsonResponse([])
      })
      const wrapper = mount(ImageMarket, {
        global: { stubs: { 'runtime-deps-block': true } },
      })
      await flushPromises()

      const installed = wrapper.find('[data-testid="installed-image-list"]')
      expect(installed.exists()).toBe(true)
      const cards = installed.findAll('li')
      expect(cards.length).toBe(3)
      expect(cards[0].text()).toContain('trae-agent')
      expect(cards[0].find('[data-testid="image-group-icon"]').attributes('src')).toBe(
        '/api/ai-provider/public-image-groups/20/icon?h=abc',
      )
      expect(cards[1].find('[data-testid="image-group-icon"]').attributes('src')).toBe('/snapshotted-icon')
      expect(cards[2].find('[data-testid="image-group-icon"]').exists()).toBe(false)
      expect(cards[2].find('[data-testid="image-group-icon-placeholder"]').exists()).toBe(true)
    })
  })
}
