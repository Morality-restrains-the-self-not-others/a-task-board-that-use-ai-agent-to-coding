// @vitest-environment jsdom
/**
 * 评论区硬件卡：直接渲染环境与硬件面板，不含与卡体重叠的摘要条。
 */
if (!process.env.VITEST) {
  console.log('[skip] CommentComposerHardwareCard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const { mockApiFetch } = vi.hoisted(() => ({ mockApiFetch: vi.fn() }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: mockApiFetch,
  }))

  const { default: CommentComposerHardwareCard } = await import('./CommentComposerHardwareCard.vue')
  const {
    registerCommentHardwarePanelReader,
    readRegisteredCommentHardwarePanel,
  } = await import('../../composables/taskDetail/commentRunHardwareTemplate.js')

  describe('CommentComposerHardwareCard', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      registerCommentHardwarePanelReader(() => null)
      mockApiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
    })

    it('不含重叠摘要条，直接渲染环境与硬件面板并登记 reader', async () => {
      const initHardwareFromParent = vi.fn()
      const wrapper = mount(CommentComposerHardwareCard, {
        props: {
          projectServerRunTemplate: {
            cloud_platform_id: 'p1',
            region: 'cn-qingdao',
            instance_type: 'ecs.c6.large',
          },
          installedImages: [{ id: 'img-a', name: 'trae-agent' }],
          task: { id: 'task_1' },
          workspaceId: 'w1',
          selectedImageId: 'img-a',
        },
        global: {
          stubs: {
            ServerConfigHardwarePanel: {
              name: 'ServerConfigHardwarePanel',
              template: '<div data-testid="server-hardware-config-panel" />',
              methods: {
                openTemporaryConfig: vi.fn(),
                closeTemporaryConfig: vi.fn(),
                restoreProjectTemplateDefaults: vi.fn(),
                initHardwareFromParent,
                getHardwareConfigSource: () => 'project_template',
              },
            },
          },
        },
      })

      await flushPromises()
      await nextTick()

      expect(wrapper.find('[data-testid="hardware-config-comment-bar"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="comment-composer-hardware-run-hint"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="server-image-config-card"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="server-hardware-config-panel"]').exists()).toBe(true)
      expect(initHardwareFromParent).toHaveBeenCalled()
      expect(readRegisteredCommentHardwarePanel()).toBeTruthy()

      wrapper.unmount()
      expect(readRegisteredCommentHardwarePanel()).toBeNull()
    })

    it('临时硬件配不齐时提示无法运行，不把提示写成提交拦截', async () => {
      const wrapper = mount(CommentComposerHardwareCard, {
        props: {
          installedImages: [{ id: 'img-a', name: 'trae-agent' }],
          task: { id: 'task_1' },
          workspaceId: 'w1',
          selectedImageId: 'img-a',
        },
        global: {
          stubs: {
            ServerConfigHardwarePanel: {
              name: 'ServerConfigHardwarePanel',
              template: '<div data-testid="server-hardware-config-panel" />',
              methods: {
                initHardwareFromParent: vi.fn(),
                getHardwareConfigSource: () => 'temporary',
                buildRunTemplatePayload: () => ({ cloud_platform_id: 'plat-1' }),
              },
            },
          },
        },
      })

      await flushPromises()
      await nextTick()

      const hint = wrapper.get('[data-testid="comment-composer-hardware-run-hint"]')
      expect(hint.text()).toMatch(/无法启动运行/)
      expect(hint.text()).toMatch(/仍可发送评论/)
      expect(hint.text()).not.toMatch(/后再提交/)

      wrapper.unmount()
    })

    it('切换到不完整临时配置后出现无法运行提示', async () => {
      const { ref } = await import('vue')
      const source = ref('project_template')
      const wrapper = mount(CommentComposerHardwareCard, {
        props: {
          installedImages: [{ id: 'img-a', name: 'trae-agent' }],
          task: { id: 'task_1' },
          workspaceId: 'w1',
          selectedImageId: 'img-a',
        },
        global: {
          stubs: {
            ServerConfigHardwarePanel: {
              name: 'ServerConfigHardwarePanel',
              template: '<div data-testid="server-hardware-config-panel" />',
              setup(_props, { expose }) {
                expose({
                  hardwareConfigSource: source,
                  getHardwareConfigSource: () => source.value,
                  buildRunTemplatePayload: () => ({ cloud_platform_id: 'plat-1' }),
                  initHardwareFromParent: vi.fn(),
                })
                return {}
              },
            },
          },
        },
      })

      await flushPromises()
      await nextTick()
      expect(wrapper.find('[data-testid="comment-composer-hardware-run-hint"]').exists()).toBe(false)

      source.value = 'temporary'
      await nextTick()
      const hint = wrapper.get('[data-testid="comment-composer-hardware-run-hint"]')
      expect(hint.text()).toMatch(/无法启动运行/)
      expect(hint.text()).toMatch(/仍可发送评论/)

      wrapper.unmount()
    })

    it('源码不含 Teleport 与摘要条，含真源面板', () => {
      const root = dirname(fileURLToPath(import.meta.url))
      const src = readFileSync(join(root, 'CommentComposerHardwareCard.vue'), 'utf8')
      expect(src).not.toContain('<Teleport')
      expect(src).not.toContain('HardwareConfigCommentBar')
      expect(src).not.toContain('hardware-config-comment-bar')
      expect(src).toContain('ServerConfigHardwarePanel')
    })
  })
}
