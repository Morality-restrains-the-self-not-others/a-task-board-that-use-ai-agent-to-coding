// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentsPanel.noEmptyCommentExecutionFallback.test.js requires vitest runtime')
} else {
  /**
   * 零评论时不得挂「执行细节」任务级 fallback（空 comment-id +「当前执行」误导）。
   * 执行细节仅挂在真实评论上；auto_run 跳过横幅由 TaskDetailAutoRunSkipBanner 独立展示。
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
      displayComments: [],
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
    TaskDetailConversationFeed: {
      template: '<p data-testid="stub-feed-empty">暂无评论</p>',
    },
    TaskDetailApplyPatchBar: { template: '<div data-testid="stub-patch" />' },
    TaskDetailCommentComposer: { template: '<div data-testid="stub-composer" />' },
  }

  describe('TaskDetailCommentsPanel 零评论无执行细节 fallback', () => {
    it('displayComments 为空时不渲染 comment-execution-details-fallback-wrap', () => {
      const wrapper = mount(TaskDetailCommentsPanel, {
        props: panelProps(),
        slots: {
          'execution-details-fallback':
            '<details data-testid="comment-execution-details" data-comment-id="" open>执行细节</details>',
        },
        global: { stubs: childStubs },
      })
      expect(wrapper.find('[data-testid="comment-execution-details-fallback-wrap"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="comment-execution-details"]').exists()).toBe(false)
      expect(wrapper.text()).toContain('暂无评论')
      expect(wrapper.text()).toContain('添加评论')
    })

    it('Panel/Section 源码不再挂空 comment-id 的执行细节 fallback', () => {
      const panel = readFileSync(join(here, 'TaskDetailCommentsPanel.vue'), 'utf8')
      const section = readFileSync(join(here, 'TaskDetailCommentsSection.vue'), 'utf8')
      const bindings = readFileSync(
        join(here, '../../views/taskDetailSectionBindings.js'),
        'utf8',
      )
      expect(panel).not.toMatch(/comment-execution-details-fallback-wrap/)
      expect(panel).not.toMatch(/execution-details-fallback/)
      expect(section).not.toMatch(/execution-details-fallback/)
      expect(section).not.toMatch(/comment-id=""/)
      expect(section).toMatch(/v-if="String\(comment\?\.id \|\| ''\)\.trim\(\)"/)
      // defineProps 不得再声明任务级启服/心跳（仅曾供 fallback；connectionStatusBind 本地映射键除外）
      expect(section).not.toMatch(/serverStatus:\s*\{\s*type:/)
      expect(section).not.toMatch(/statusLogs:\s*\{\s*type:/)
      expect(section).not.toMatch(/containerHeartbeatStatus:\s*\{\s*type:/)
      expect(section).not.toMatch(/runtimeStatus:\s*\{\s*type:/)
      expect(section).not.toMatch(/statusTraceId:\s*\{\s*type:/)
      // commentsSectionProps 不得再透传上述任务级键
      const commentsBlock = bindings.slice(
        bindings.indexOf('const commentsSectionProps'),
        bindings.indexOf('return { commentsSectionProps }'),
      )
      expect(commentsBlock).not.toMatch(/serverStatus:/)
      expect(commentsBlock).not.toMatch(/statusLogs:/)
      expect(commentsBlock).not.toMatch(/containerHeartbeatStatus:/)
      expect(commentsBlock).not.toMatch(/runtimeStatus:/)
      expect(commentsBlock).not.toMatch(/statusTraceId:/)
    })
  })
}
