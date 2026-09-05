// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentsPanel.noGlobalImageRuntimeHeader.test.js requires vitest runtime')
} else {
  /**
   * F-031 T10 / F-030 T13：评论列表顶端不得再出现全局「镜像运行状态」header。
   * 启动日志 / SSE / 镜像关联只留在对应评论「执行细节」内。
   */
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')
  const { default: TaskDetailCommentsPanel } = await import('./TaskDetailCommentsPanel.vue')

  const here = dirname(fileURLToPath(import.meta.url))

  function panelProps(overrides = {}) {
    return {
      newComment: '',
      'onUpdate:newComment': () => {},
      showContainerCloneProgressBanner: false,
      containerCloneProgressEntries: [],
      repoCloneFieldId: () => 'repo',
      shortCloneRepoLabel: (u) => u,
      cloneProgressRowHasSubPhases: () => false,
      cloneProgressRecvPct: () => 0,
      cloneProgressUnpackPct: () => 0,
      cloneProgressBarWidthTransitionClass: '',
      cloneProgressRowLogIsPlaceholder: () => false,
      cloneProgressRowLogDisplayText: () => '',
      displayComments: [{ id: 'c1', content: 'hello' }],
      aiStreamBusy: false,
      aiStreamBuffer: '',
      hasZtreeAgentSteps: false,
      agentStepCount: 0,
      tenantId: 't1',
      workspaceId: 'ws1',
      taskId: 'task_1',
      commentComposerChips: [],
      ...overrides,
    }
  }

  const childStubs = {
    TaskDetailConversationFeed: { template: '<div data-testid="stub-feed" />' },
    TaskDetailApplyPatchBar: { template: '<div data-testid="stub-patch" />' },
    TaskDetailCommentComposer: { template: '<div data-testid="stub-composer" />' },
  }

  describe('TaskDetailCommentsPanel 无全局镜像运行 header', () => {
    it('即使传入残留 imageRuntimeEntries 也不渲染 image-runtime-entry', () => {
      const wrapper = mount(TaskDetailCommentsPanel, {
        props: panelProps({
          imageRuntimeEntries: [
            {
              imageId: '875589715594604544',
              imageLabel: 'trae-agent',
              commentId: 'c1',
              commentLabel: '【自动运行】 写一个 hello world程序…',
              _hasBinding: true,
              _statusLogs: ['[18:44:33] 容器调度排队中'],
              _statusProgress: 0,
              _serverStatus: 'success',
              _isServerRunning: true,
            },
          ],
          sseLive: true,
        }),
        global: { stubs: childStubs },
      })
      expect(wrapper.find('[data-testid="image-runtime-entry"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="image-runtime-entries-section"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('镜像运行状态')
      expect(wrapper.get('#comments-container').exists()).toBe(true)
      expect(wrapper.text()).toContain('评论')
    })

    it('源码不再保留评论区全局镜像运行管道', () => {
      const panel = readFileSync(join(here, 'TaskDetailCommentsPanel.vue'), 'utf8')
      const section = readFileSync(join(here, 'TaskDetailCommentsSection.vue'), 'utf8')
      const detail = readFileSync(join(here, '../../views/TaskDetail.vue'), 'utf8')
      const bindings = readFileSync(join(here, '../../views/taskDetailSectionBindings.js'), 'utf8')
      const helpers = readFileSync(
        join(here, '../../composables/taskDetail/taskDetailCommentsSectionHelpers.js'),
        'utf8',
      )

      for (const [name, src] of [
        ['panel', panel],
        ['section', section],
        ['detail', detail],
        ['bindings', bindings],
        ['helpers', helpers],
      ]) {
        expect(src, name).not.toMatch(/image-runtime-entry/)
        expect(src, name).not.toMatch(/image-runtime-entries-section/)
        expect(src, name).not.toMatch(/buildImageRuntimeEntries/)
        expect(src, name).not.toMatch(/imageRuntimeEntries/)
      }
    })
  })
}
