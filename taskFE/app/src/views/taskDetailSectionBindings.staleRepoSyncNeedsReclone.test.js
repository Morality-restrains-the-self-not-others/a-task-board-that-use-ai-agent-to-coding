// @vitest-environment node
/**
 * Production task-detail TypeError: runtimeSectionProps read
 * staleRepoSyncNeedsReclone.value when useTaskDetail omitted the ref.
 * That aborted v-bind and left 关联项目 on 暂无关联项目 despite GET projects.
 */
if (!process.env.VITEST) {
  console.log('[skip] taskDetailSectionBindings.staleRepoSyncNeedsReclone.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { ref } = await import('vue')
  const { useTaskDetailRuntimeSectionBindings } = await import('./taskDetailSectionBindings.js')

  const FN_KEYS = new Set([
    'updateServerStatus',
    'pauseContainerHeartbeatForRelayStop',
    'resumeContainerHeartbeatForRelayStart',
    'appendContainerHeartbeatLogLines',
    'fetchTaskDetail',
    'getProjectRepos',
    'getRepoBranchValue',
    'getRepoBranches',
    'getRepoBranchError',
    'isLoadingRepoBranches',
    'gitCloneRefMatchKey',
    'repoCloneFieldId',
    'cloneProgressRowHasSubPhases',
    'cloneProgressRecvPct',
    'cloneProgressUnpackPct',
    'containerGitIdentityLineForRepoUrl',
    'onRepoOAuthReadiness',
  ])

  function makeRuntimeCtx(overrides) {
    return new Proxy(overrides, {
      get(target, prop) {
        if (prop in target) return target[prop]
        if (FN_KEYS.has(prop)) return () => undefined
        if (prop === 'runtimeSectionRef' || prop === 'serverConfigRef' || prop === 'linkedProjectsPanelRef') {
          return ref(null)
        }
        if (prop === 'taskProjectsWithDetails' || prop === 'workspaceProjects' || prop === 'displayComments') {
          return { value: [] }
        }
        if (prop === 'localTask' || prop === 'editingTask') return { value: {} }
        if (prop === 'recloneLoadingByUrl' || prop === 'recloneStatusByUrl' || prop === 'repoCloneIdentityByUrl') {
          return { value: {} }
        }
        return { value: '' }
      },
    })
  }

  describe('runtimeSectionProps staleRepoSyncNeedsReclone', () => {
    it('throws the production TypeError when the ref is missing', () => {
      const { runtimeSectionProps } = useTaskDetailRuntimeSectionBindings(
        makeRuntimeCtx({ staleRepoSyncNeedsReclone: undefined }),
      )
      expect(() => runtimeSectionProps.value).toThrow(/reading 'value'/)
    })

    it('passes linked-project rows through when the ref exists', () => {
      const rows = [{ id: 'proj_881195029417193472', project_name: 'helloworld' }]
      const { runtimeSectionProps } = useTaskDetailRuntimeSectionBindings(
        makeRuntimeCtx({
          staleRepoSyncNeedsReclone: { value: false },
          taskProjectsWithDetails: { value: rows },
          localTask: {
            value: {
              id: 'task_881388002226499584',
              projects: [{ project_id: 'proj_881195029417193472' }],
            },
          },
        }),
      )
      expect(runtimeSectionProps.value.taskProjectsWithDetails).toEqual(rows)
      expect(runtimeSectionProps.value.staleRepoSyncNeedsReclone).toBe(false)
      expect(runtimeSectionProps.value.task.projects).toHaveLength(1)
    })
  })
}
