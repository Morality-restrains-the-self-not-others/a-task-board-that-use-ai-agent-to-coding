// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailRuntimeSection.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailRuntimeSection } = await import('./TaskDetailRuntimeSection.vue')

  const fn = () => {}

  function mountSection(overrides = {}) {
    return mount(TaskDetailRuntimeSection, {
      props: {
        task: {},
        workspaceId: 'w1',
        tenantId: 't1',
        taskId: 'task_881388002226499584',
        updateServerStatus: fn,
        pauseContainerHeartbeatForRelayStop: fn,
        resumeContainerHeartbeatForRelayStart: fn,
        appendContainerHeartbeatLogLines: fn,
        refreshTaskDetail: fn,
        gitCloneRefMatchKey: fn,
        repoCloneFieldId: fn,
        cloneProgressRowHasSubPhases: fn,
        cloneProgressRecvPct: fn,
        cloneProgressUnpackPct: fn,
        getProjectRepos: fn,
        getRepoBranchValue: fn,
        getRepoBranches: fn,
        getRepoBranchError: fn,
        isLoadingRepoBranches: fn,
        containerGitIdentityLineForRepoUrl: fn,
        ...overrides,
      },
      global: {
        stubs: {
          ServerConfig: {
            props: ['taskId', 'tenantId'],
            template:
              '<div data-testid="runtime-server-config" :data-task-id="taskId" :data-tenant-id="tenantId"></div>',
          },
          TaskDetailLinkedProjectsPanel: { template: '<div />' },
          TaskDetailContainerHttpUnreachableBanner: { template: '<div />' },
        },
      },
    })
  }

  describe('TaskDetailRuntimeSection 把路由任务 ID 传给 ServerConfig', () => {
    it('task 对象无 id 时仍把 task-id / tenant-id 传给 ServerConfig', () => {
      const wrapper = mountSection()
      const cfg = wrapper.get('[data-testid="runtime-server-config"]')
      expect(cfg.attributes('data-task-id')).toBe('task_881388002226499584')
      expect(cfg.attributes('data-tenant-id')).toBe('t1')
    })
  })
}
