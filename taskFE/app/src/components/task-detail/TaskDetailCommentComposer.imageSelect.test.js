// @vitest-environment jsdom
/**
 * 评论区镜像选择：$镜像 提及同步 selectedImageId。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentComposer.imageSelect.test.js requires vitest runtime')
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
  vi.mock('vue-router', () => ({
    useRoute: () => ({ path: '/t', query: {}, hash: '' }),
    useRouter: () => ({ replace: () => {} }),
  }))
  vi.mock('../../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: async () => 'u1',
  }))
  vi.mock('../../utils/githubAppReturnStorage.js', () => ({
    createGithubAppReturnKey: () => 'a'.repeat(32),
    setGithubAppReturnTarget: () => {},
  }))

  const { default: TaskDetailCommentComposer } = await import('./TaskDetailCommentComposer.vue')

  describe('TaskDetailCommentComposer 镜像选择并入评论区', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      mockApiFetch.mockImplementation(async (url) => {
        if (String(url).includes('workspaces')) {
          return {
            ok: true,
            json: async () => ({ container_image_at_mode_enabled: true }),
          }
        }
        if (String(url).includes('installed-images')) {
          return {
            ok: true,
            json: async () => ([
              { id: 'img-a', name: 'trae-agent', version: 'x86_64-latest' },
              { id: 'img-b', name: 'other', version: '1' },
            ]),
          }
        }
        return { ok: true, json: async () => ({}) }
      })
    })

    it('无 $镜像 不渲染硬件卡；feature-params 槽常驻隐藏；mention 后出现镜像说明与硬件卡', async () => {
      const { bridgedSelectedImageId } = await import('../../composables/taskDetail/taskDetailImageSelectionBridge.js')
      bridgedSelectedImageId.value = 'img-a'

      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: 'hello',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'w1',
          showDependencyPicker: false,
        },
        global: {
          stubs: {
            CommentImageMentionEditor: {
              template: '<div data-testid="comment-image-mention-editor" />',
              props: ['modelValue', 'installedImages', 'placeholder'],
              emits: ['mention-change', 'keydown', 'update:modelValue'],
            },
            CommentExecutionDependencyPicker: true,
            ServerConfigImageSectionHints: true,
            CommentComposerHardwareCard: {
              template: '<div data-testid="comment-composer-env-hardware-slot" />',
            },
          },
        },
      })

      await flushPromises()
      await nextTick()

      expect(wrapper.find('#detail-image').exists()).toBe(false)
      expect(wrapper.find('[data-testid="comment-composer-env-hardware-slot"]').exists()).toBe(false)
      const runRow = wrapper.find('[data-testid="comment-composer-run-config-row"]')
      expect(runRow.exists()).toBe(true)
      expect(String(runRow.attributes('style') || '')).toMatch(/display:\s*none/)
      const fpSlot = wrapper.find('[data-testid="comment-composer-feature-params-slot"]')
      expect(fpSlot.exists()).toBe(true)
      expect(wrapper.get('[data-testid="task-detail-comment-submit"]').text()).toContain('提交评论')

      wrapper.vm.onMentionChange({ id: 'img-b', name: 'other' })
      await nextTick()
      expect(wrapper.find('[data-testid="comment-composer-repo-identity"]').exists()).toBe(false)

      await wrapper.setProps({
        taskProjectsWithDetails: [
          { project: { git_repos: ['https://github.com/acme/demo.git'] } },
        ],
      })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('[data-testid="comment-composer-repo-identity"]').exists()).toBe(true)
      await nextTick()
      expect(bridgedSelectedImageId.value).toBe('img-b')
      expect(wrapper.find('[data-testid="comment-composer-mentioned-image"]').text()).toMatch(/other/)
      const hwSlot = wrapper.find('[data-testid="comment-composer-env-hardware-slot"]')
      expect(hwSlot.exists()).toBe(true)
      expect(String(runRow.attributes('style') || '')).not.toMatch(/display:\s*none/)
      const imageSelect = wrapper.find('[data-testid="comment-composer-image-select"]')
      expect(imageSelect.find('[data-testid="comment-composer-env-hardware-slot"]').exists()).toBe(false)
      expect(imageSelect.element.parentElement).toBe(fpSlot.element.parentElement)
      const following = Node.DOCUMENT_POSITION_FOLLOWING
      expect(imageSelect.element.compareDocumentPosition(fpSlot.element) & following).toBe(following)
      expect(fpSlot.element.compareDocumentPosition(hwSlot.element) & following).toBe(following)
      expect(wrapper.get('[data-testid="task-detail-comment-submit"]').text()).toContain('提交并运行')

      wrapper.unmount()
    })

    it('$镜像 后说明展示已安装镜像的 name:version', async () => {
      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: 'hello',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'w1',
          showDependencyPicker: false,
        },
        global: {
          stubs: {
            CommentImageMentionEditor: { template: '<div />' },
            CommentExecutionDependencyPicker: true,
            ServerConfigImageSectionHints: true,
            CommentComposerHardwareCard: true,
          },
        },
      })

      await flushPromises()
      await nextTick()
      wrapper.vm.onMentionChange({ id: 'img-a', name: 'trae-agent' })
      await nextTick()
      expect(wrapper.get('[data-testid="comment-composer-mentioned-image"]').text().trim())
        .toBe('将使用镜像 trae-agent:x86_64-latest 运行')

      wrapper.vm.onMentionChange({ id: 'img-b', name: 'other' })
      await nextTick()
      expect(wrapper.get('[data-testid="comment-composer-mentioned-image"]').text().trim())
        .toBe('将使用镜像 other:1 运行')

      wrapper.unmount()
    })

    it('镜像无 version 时说明仅展示名称', async () => {
      mockApiFetch.mockImplementation(async (url) => {
        if (String(url).includes('workspaces')) {
          return {
            ok: true,
            json: async () => ({ container_image_at_mode_enabled: true }),
          }
        }
        if (String(url).includes('installed-images')) {
          return {
            ok: true,
            json: async () => ([{ id: 'img-c', name: 'plain' }]),
          }
        }
        return { ok: true, json: async () => ({}) }
      })

      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: 'hello',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'w1',
          showDependencyPicker: false,
        },
        global: {
          stubs: {
            CommentImageMentionEditor: { template: '<div />' },
            CommentExecutionDependencyPicker: true,
            ServerConfigImageSectionHints: true,
            CommentComposerHardwareCard: true,
          },
        },
      })

      await flushPromises()
      await nextTick()
      wrapper.vm.onMentionChange({ id: 'img-c', name: 'plain' })
      await nextTick()
      expect(wrapper.get('[data-testid="comment-composer-mentioned-image"]').text().trim())
        .toBe('将使用镜像 plain 运行')

      wrapper.unmount()
    })

    it('镜像无 version 但有 tag 时说明使用 tag 作为版本', async () => {
      mockApiFetch.mockImplementation(async (url) => {
        if (String(url).includes('workspaces')) {
          return {
            ok: true,
            json: async () => ({ container_image_at_mode_enabled: true }),
          }
        }
        if (String(url).includes('installed-images')) {
          return {
            ok: true,
            json: async () => ([{ id: 'img-d', name: 'tagged', tag: 'v9.9' }]),
          }
        }
        return { ok: true, json: async () => ({}) }
      })

      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: 'hello',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'w1',
          showDependencyPicker: false,
        },
        global: {
          stubs: {
            CommentImageMentionEditor: { template: '<div />' },
            CommentExecutionDependencyPicker: true,
            ServerConfigImageSectionHints: true,
            CommentComposerHardwareCard: true,
          },
        },
      })

      await flushPromises()
      await nextTick()
      wrapper.vm.onMentionChange({ id: 'img-d', name: 'tagged' })
      await nextTick()
      expect(wrapper.get('[data-testid="comment-composer-mentioned-image"]').text().trim())
        .toBe('将使用镜像 tagged:v9.9 运行')

      wrapper.unmount()
    })

    it('源码：ServerConfig.logic 镜像卡不再含 detail-image；Composer 无常驻镜像下拉', () => {
      const root = dirname(fileURLToPath(import.meta.url))
      const componentsDir = join(root, '..')
      const logicSrc = readFileSync(join(componentsDir, 'ServerConfig.logic.vue'), 'utf8')
      const composerSrc = readFileSync(join(root, 'TaskDetailCommentComposer.vue'), 'utf8')
      expect(logicSrc).not.toMatch(/id="detail-image"/)
      expect(composerSrc).not.toMatch(/id="detail-image"/)
      expect(composerSrc).toContain('data-testid="comment-composer-image-select"')
      expect(composerSrc).toContain('composerSelectedImageId')
      expect(composerSrc).toContain('onMentionChange')
      expect(composerSrc).toContain('showRunConfig')
      let legacyExists = true
      try {
        readFileSync(join(componentsDir, 'ServerConfig.vue'), 'utf8')
      } catch {
        legacyExists = false
      }
      expect(legacyExists).toBe(false)
    })

    it('源码：$镜像 后 composer 直接挂载硬件卡（非 Teleport），位于智能体资源配置槽之后', () => {
      const root = dirname(fileURLToPath(import.meta.url))
      const composerSrc = readFileSync(join(root, 'TaskDetailCommentComposer.vue'), 'utf8')
      const cardSrc = readFileSync(join(root, 'CommentComposerHardwareCard.vue'), 'utf8')
      expect(composerSrc).toContain('CommentComposerHardwareCard')
      const fpIdx = composerSrc.indexOf('comment-composer-feature-params-slot')
      const hwIdx = composerSrc.indexOf('<CommentComposerHardwareCard')
      const submitIdx = composerSrc.indexOf('task-detail-comment-submit')
      expect(fpIdx).toBeGreaterThan(-1)
      expect(hwIdx).toBeGreaterThan(fpIdx)
      expect(submitIdx).toBeGreaterThan(hwIdx)
      expect(composerSrc).not.toContain('comment-composer-env-hardware-slot')
      expect(composerSrc).toMatch(/v-if="showRunConfig"[\s\S]*CommentComposerHardwareCard/)
      expect(cardSrc).toContain('data-testid="comment-composer-env-hardware-slot"')
      expect(cardSrc).toContain('ServerConfigHardwarePanel')
      expect(cardSrc).toContain('data-testid="server-image-config-card"')
      expect(cardSrc).not.toContain('<Teleport')
      expect(cardSrc).not.toContain('HardwareConfigCommentBar')
      expect(cardSrc).not.toContain('hardware-config-comment-bar')
    })

    it('源码：评论输入后镜像说明在左、智能体配置在右同一行，再硬件卡、再提交', () => {
      const root = dirname(fileURLToPath(import.meta.url))
      const composerSrc = readFileSync(join(root, 'TaskDetailCommentComposer.vue'), 'utf8')
      const inputIdx = composerSrc.indexOf(':data-testid="inputTestId')
      const rowIdx = composerSrc.indexOf('comment-composer-run-config-row')
      const imageIdx = composerSrc.indexOf('comment-composer-image-select')
      const fpSlotIdx = composerSrc.indexOf('comment-composer-feature-params-slot')
      const hwIdx = composerSrc.indexOf('<CommentComposerHardwareCard')
      const submitIdx = composerSrc.indexOf('task-detail-comment-submit')
      expect(inputIdx).toBeGreaterThan(-1)
      expect(rowIdx).toBeGreaterThan(inputIdx)
      expect(imageIdx).toBeGreaterThan(rowIdx)
      expect(fpSlotIdx).toBeGreaterThan(imageIdx)
      expect(hwIdx).toBeGreaterThan(fpSlotIdx)
      expect(submitIdx).toBeGreaterThan(hwIdx)
      expect(composerSrc).toMatch(/v-show="showRunConfig"[\s\S]*comment-composer-run-config-row/)
      expect(composerSrc).toMatch(/flex items-start gap-4/)
    })

    it('评论区镜像 hints 收到非空 task 与 viewerTaskId（OPT-20260812-053）', async () => {
      let hintsProps = null
      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: 'hello',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'w1',
          showDependencyPicker: false,
          task: { id: 'task_1', container_image: 'img-a' },
          viewerTaskId: 'task_1',
        },
        global: {
          stubs: {
            CommentImageMentionEditor: { template: '<div />' },
            CommentExecutionDependencyPicker: true,
            CommentComposerHardwareCard: true,
            ServerConfigImageSectionHints: {
              template: '<div data-testid="hints-stub" />',
              props: ['installedImages', 'selectedImageId', 'task', 'runtimeStatus', 'viewerTaskId', 'tenantId', 'workspaceId'],
              mounted() { hintsProps = this.$props },
            },
          },
        },
      })

      await flushPromises()
      await nextTick()

      wrapper.vm.onMentionChange({ id: 'img-a', name: 'trae-agent' })
      await flushPromises()
      await nextTick()

      expect(hintsProps).not.toBeNull()
      expect(hintsProps.viewerTaskId).toBe('task_1')
      expect(hintsProps.task).toMatchObject({ id: 'task_1' })

      wrapper.unmount()
    })

    it('源码：Composer 将 task/viewerTaskId 透传给 ServerConfigImageSectionHints（OPT-20260812-053）', () => {
      const root = dirname(fileURLToPath(import.meta.url))
      const composerSrc = readFileSync(join(root, 'TaskDetailCommentComposer.vue'), 'utf8')
      expect(composerSrc).toMatch(/viewer-task-id="imageHintsTaskId"/)
      expect(composerSrc).toMatch(/:task="imageHintsTask"/)
      expect(composerSrc).toMatch(/const imageHintsTask = computed\(\(\) => props\.task \|\| null\)/)
      expect(composerSrc).toMatch(/const imageHintsTaskId = computed\(\(\) => String\(props\.viewerTaskId \|\| ''\)\)/)
    })
  })
}
