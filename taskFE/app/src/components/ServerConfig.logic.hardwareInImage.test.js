// @vitest-environment jsdom
/**
 * 评论区直接渲染环境与硬件卡（方案 C，不再 Teleport）。
 */
if (!process.env.VITEST) {
  console.log('[skip] ServerConfig.logic.hardwareInImage.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { createRouter, createMemoryHistory } = await import('vue-router')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const { mockApiFetch } = vi.hoisted(() => ({
    mockApiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: mockApiFetch,
  }))

  vi.mock('../composables/useServerSectionBodyExpanded.js', () => ({
    useServerSectionBodyExpanded: () => ({
      serverSectionBodyExpanded: { value: true },
      expandServerSectionBody: vi.fn(),
      toggleServerSectionBody: vi.fn(),
    }),
  }))

  vi.mock('../utils/serverRuntimeStatusDetails.js', () => ({
    buildServerRuntimeStatusDetails: vi.fn(() => []),
    resolveRuntimeUptimeSource: vi.fn(() => null),
  }))

  const { default: ServerConfigLogic } = await import('./ServerConfig.logic.vue')

  const FP_SLOT = 'comment-composer-feature-params-slot'

  function ensureSlot(testId) {
    let slot = document.querySelector(`[data-testid="${testId}"]`)
    if (!slot) {
      slot = document.createElement('div')
      slot.setAttribute('data-testid', testId)
      document.body.appendChild(slot)
    }
    return slot
  }

  async function mountLogic() {
    ensureSlot(FP_SLOT)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/',
          component: { template: '<div/>' },
        },
      ],
    })
    await router.push('/tenant/t1/workspace/w1/task-detail/task_1/')
    await router.isReady()

    mockApiFetch.mockImplementation(async (url) => {
      if (String(url).includes('installed-images')) {
        return { ok: true, json: async () => ({ results: [] }) }
      }
      return { ok: true, json: async () => ({}) }
    })

    return mount(ServerConfigLogic, {
      props: {
        task: { id: 'task_1' },
        updateServerStatus: vi.fn(),
      },
      attachTo: document.body,
      global: {
        plugins: [router],
        stubs: {
          ServerConfigImageSectionHints: { template: '<div data-stub="hints" />' },
          ServerConfigFeatureParamsBlock: { template: '<div data-stub="feature-params" />' },
          ServerConfigSectionTabs: {
            template: '<div data-testid="server-section-tabs-stub" />',
            props: ['activeServerSection', 'isRelayToTraeEnabled', 'bodyExpanded'],
          },
          ServerConfigServerStartHistoryPanel: { template: '<div />' },
          ServerConfigRelayDirectPanel: { template: '<div />' },
          ServerConfigServerContentSection: { template: '<div />' },
        },
      },
    })
  }

  describe('ServerConfig.logic 硬件迁出 / 智能体配置就地挂载', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      ensureSlot(FP_SLOT).innerHTML = ''
    })

    afterEach(() => {
      document.querySelectorAll(`[data-testid="${FP_SLOT}"]`).forEach((el) => el.remove())
    })

    it('logic 不再渲染环境与硬件卡，硬件不在 server-section-body', async () => {
      const wrapper = await mountLogic()
      await flushPromises()
      await nextTick()

      expect(wrapper.find('[data-testid="server-image-config-card"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="server-hardware-config-panel"]').exists()).toBe(false)
      const sectionBody = wrapper.find('[data-testid="server-section-body"]')
      expect(sectionBody.exists()).toBe(true)
      expect(sectionBody.find('[data-testid="server-hardware-config-panel"]').exists()).toBe(false)

      wrapper.unmount()
    })

    it('logic 不再 Teleport 智能体资源配置到 feature-params 槽', async () => {
      const wrapper = await mountLogic()
      await flushPromises()
      await nextTick()

      expect(wrapper.find('[data-stub="feature-params"]').exists()).toBe(false)
      const fpSlot = document.querySelector(`[data-testid="${FP_SLOT}"]`)
      if (fpSlot) {
        expect(fpSlot.querySelector('[data-stub="feature-params"]')).toBeFalsy()
      }

      wrapper.unmount()
    })

    it('源码：硬件卡在 composer 直挂；logic 仅 Teleport 智能体配置；无硬件 Tab', () => {
      const root = dirname(fileURLToPath(import.meta.url))
      const logicSrc = readFileSync(join(root, 'ServerConfig.logic.vue'), 'utf8')
      const tabsSrc = readFileSync(join(root, 'ServerConfigSectionTabs.vue'), 'utf8')
      const composerSrc = readFileSync(
        join(root, 'task-detail', 'TaskDetailCommentComposer.vue'),
        'utf8',
      )
      const cardSrc = readFileSync(
        join(root, 'task-detail', 'CommentComposerHardwareCard.vue'),
        'utf8',
      )

      expect(logicSrc).not.toMatch(/comment-composer-env-hardware-slot/)
      expect(logicSrc).not.toContain('ServerConfigHardwarePanel')
      expect(logicSrc).not.toContain('<Teleport')
      expect(logicSrc).not.toContain('comment-composer-feature-params-slot')
      expect(logicSrc).not.toContain('ServerConfigFeatureParamsBlock')
      expect(logicSrc).not.toMatch(/v-show="activeServerSection === 'hardware'"/)
      expect(tabsSrc).not.toContain('server-config-hardware-tab')
      expect(tabsSrc).not.toContain('服务器硬件配置')
      expect(composerSrc).toContain('CommentComposerHardwareCard')
      expect(composerSrc).toContain('ServerConfigFeatureParamsBlock')
      const fpIdx = composerSrc.indexOf('comment-composer-feature-params-slot')
      const blockIdx = composerSrc.indexOf('<ServerConfigFeatureParamsBlock')
      const hwIdx = composerSrc.indexOf('<CommentComposerHardwareCard')
      expect(fpIdx).toBeGreaterThan(-1)
      expect(blockIdx).toBeGreaterThan(fpIdx)
      expect(hwIdx).toBeGreaterThan(blockIdx)
      expect(composerSrc).not.toContain('comment-composer-env-hardware-slot')
      expect(cardSrc).toContain('data-testid="server-image-config-card"')
      expect(cardSrc).toContain('ServerConfigHardwarePanel')
      expect(cardSrc).not.toContain('<Teleport')
    })
  })
}
