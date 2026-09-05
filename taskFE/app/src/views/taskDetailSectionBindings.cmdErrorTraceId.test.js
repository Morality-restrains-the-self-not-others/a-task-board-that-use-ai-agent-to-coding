// @vitest-environment node
/**
 * 评论区绑定必须把 layerGraphCmdErrorTraceId 交给 CommentsSection，
 * 否则命令失败红字无法挂 data-traceId，页面会只剩裸 HTTP 403。
 */
if (!process.env.VITEST) {
  console.log('[skip] taskDetailSectionBindings.cmdErrorTraceId.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')
  const { useTaskDetailCommentsSectionBindings } = await import('./taskDetailSectionBindings.js')

  const ARRAY_KEYS = new Set([
    'taskRepoRows',
    'containerCloneProgressEntries',
    'displayComments',
    'layerGraphZNodes',
    'layerGraphModelOptions',
    'layerAgentStepCards',
    'taskProjectsWithDetails',
    'commentComposerChips',
  ])
  const FN_KEYS = new Set([
    'repoCloneFieldId',
    'shortCloneRepoLabel',
    'cloneProgressRowHasSubPhases',
    'cloneProgressRecvPct',
    'cloneProgressUnpackPct',
    'cloneProgressRowLogIsPlaceholder',
    'cloneProgressRowLogDisplayText',
    'bindingStatusFor',
    'bindingCscIdFor',
    'bindingContainerNameFor',
    'commentHasLiveBinding',
    'commentOwnsSharedContainer',
    'bindingStartTraceIdFor',
    'buildPerBindingServerStatusProps',
    'markBindingReconnecting',
    'cancelWaitingPreviousBinding',
    'isCancelWaitingBusy',
  ])

  function makeCommentsCtx(overrides) {
    return new Proxy(overrides, {
      get(target, prop) {
        if (prop in target) return target[prop]
        if (FN_KEYS.has(prop)) return () => undefined
        if (ARRAY_KEYS.has(prop)) return { value: [] }
        if (prop === 'localTask') return { value: {} }
        if (prop === 'layerPanelStore') return { state: { value: {} } }
        if (prop === 'zTreeLogTargets') return { value: {} }
        return { value: '' }
      },
    })
  }

  describe('useTaskDetailCommentsSectionBindings cmd error traceId', () => {
    it('forwards page-level cmd error and traceId into commentsSectionProps', () => {
      const { commentsSectionProps } = useTaskDetailCommentsSectionBindings(
        makeCommentsCtx({
          layerGraphCmdError: { value: '没有权限执行该操作，或登录态/容器授权已失效。请刷新页面后重试。' },
          layerGraphCmdErrorTraceId: { value: 'tid-page-403' },
        }),
      )
      expect(commentsSectionProps.value.layerGraphCmdError).toContain('没有权限')
      expect(commentsSectionProps.value.layerGraphCmdError).not.toBe('HTTP 403')
      expect(commentsSectionProps.value.layerGraphCmdErrorTraceId).toBe('tid-page-403')
    })

    it('TaskDetail.vue still destructures and passes layerGraphCmdErrorTraceId', () => {
      const vueSrc = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'TaskDetail.vue'), 'utf8')
      expect(vueSrc).toContain('layerGraphCmdErrorTraceId')
      expect(vueSrc).toMatch(/useTaskDetailCommentsSectionBindings\(\{[\s\S]*layerGraphCmdErrorTraceId/)
    })
  })
}
