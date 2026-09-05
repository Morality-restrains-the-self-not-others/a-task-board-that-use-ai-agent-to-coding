// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetail.smoke.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    catalogCompanyId: '',
    routeMock: {
      params: { tenant: 't1', workspaceId: 'w1', taskId: '42' },
      query: {},
    },
    routerMock: {
      push: vi.fn(),
      replace: vi.fn(),
    },
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../utils/config.js', () => ({
    getApiUrl: (path) => path,
  }))

  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: () => '',
  }))

  vi.mock('../utils/httpUrlHost.js', () => ({
    isLoopbackHttpUrl: () => false,
  }))

  vi.mock('../utils/layerZtreeNodes.js', () => ({
    buildZTreeNodesSerialFromLayers: () => [],
    LAYER_TREE_NODE_PREFIX: 'layer_',
    normalizeLayerGitDirty: () => null,
  }))

  vi.mock('../utils/taskDetailContainerCloneProgress.js', () => ({
    gitCloneRefMatchKey: () => '',
    shortCloneRepoLabel: () => '',
    parseBootstrapCloneLogSections: () => ({ preamble: '', sections: [] }),
    mergeCloneProgressSubPhases: () => ({ recv: null, unpack: null }),
    cloneProgressRowHasSubPhases: () => false,
    cloneProgressRecvPct: () => 0,
    cloneProgressUnpackPct: () => 0,
  }))

  vi.mock('../utils/taskDetailBranchAndRepoUtils.js', () => ({
    extractCompanyNickname: () => '',
    repoCloneFieldId: () => '',
    sanitizeBranchSegment: (s) => s,
    hasChineseCharacter: () => false,
    buildWorkBranchName: () => '',
    buildMergeTargetBranchName: () => '',
    resolveRepoCloneIdentityFromMap: () => '',
    githubRepoBindingsBySlug: () => ({}),
    githubRepoSlugFromUrl: () => '',
    validateTaskRepoSavedAssociations: () => ({ ok: true, message: '' }),
    normalizeMemberPk: (v) => v,
    inferWorkBranchPreset: () => 'custom',
    inferMergeTargetPreset: () => 'custom',
    resolveLayerGraphPushTargetBranch: () => '',
    resolveLayerGraphMergeTargetBranch: () => '',
    resolveDefaultCompanyGitIdentityId: () => '',
    collectLinkedRepoBranchTargets: () => [],
    intersectBranchNameLists: () => [],
  }))

  vi.mock('../utils/resolveLayerGraphCommandPayload.js', () => ({
    resolveLayerGraphCommandPayload: () => ({ error: null, repo_layer_id: '', parent_job_id: '' }),
  }))

  vi.mock('../utils/taskDetailExecLogFormatters.js', () => ({
    agentStepCardTitle: () => '',
    agentStepModelBadge: () => '',
    agentStepUsageBadge: () => '',
    agentStepCardSubtitle: () => '',
    agentStepToolResultDisplay: () => [],
    agentStepJsonPretty: () => '',
    agentStepCopyKey: () => '',
    agentStepBodyMode: () => 'text',
    agentStepRichHtml: () => '',
    agentStepMarkdownSource: () => '',
    agentStepPlainBodyForPre: () => '',
  }))

  vi.mock('../composables/useProjectGitOAuthCatalog.js', () => ({
    useProjectGitOAuthCatalog: (_apiFetch, getCompanyId) => {
      hoistedMocks.catalogCompanyId =
        typeof getCompanyId === 'function' ? String(getCompanyId() || '') : ''
      return {
        gitOAuthCatalogVersion: { value: 0 },
        bootstrapGitOAuthCatalog: async () => {},
      }
    },
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => hoistedMocks.routerMock,
  }))

  // jsdom doesn't have EventSource
  class MockEventSource {
    constructor() {
      this.readyState = 2
      this.onmessage = null
      this.onerror = null
      this.onclose = null
    }
    addEventListener() {}
    close() {}
    static CONNECTING = 0
    static OPEN = 1
    static CLOSED = 2
  }
  global.EventSource = MockEventSource

  // Stub all child components to avoid deep rendering
  const stubComponents = {
    TaskDetailPageHeader: { template: '<div class="stub-page-header"></div>', props: ['hasTask', 'isEditing', 'isForking', 'isSaving', 'backToWorkPanelRoute', 'forkTask', 'tenantId', 'taskProjectsWithDetails'] },
    TaskDetailEditErrorBanner: { template: '<div class="stub-edit-error"></div>', props: ['message'] },
    TaskDetailTaskIdentityPanel: { template: '<div class="stub-task-identity"></div>', props: ['task', 'isEditing', 'editingTask', 'collaboratorNameById', 'collaboratorOptions', 'forkSourceTaskRoute', 'progressStatusOptions', 'isProgressStatusesLoading', 'isUpdatingProgressStatus', 'progressStatusError', 'progressStatusErrorTraceId', 'resolvedDeliverableCategoryErrorTraceId'] },
    TaskDetailBranchStrategyPanel: { template: '<div class="stub-branch-strategy"></div>', props: ['task', 'taskId', 'isEditing', 'editingTask', 'companyUserName', 'isTaskTitleTranslating', 'taskTitleTranslationError', 'taskTitleTranslationErrorTraceId', 'commonMergeTargetBranches', 'commonMergeTargetBranchesLoading', 'commonMergeTargetBranchesError', 'commonMergeTargetBranchesErrorTraceId'] },
    TaskDetailContainerHttpUnreachableBanner: { template: '<div class="stub-container-banner"></div>', props: ['visible'] },
    TaskDetailServerStartStatusPanel: { template: '<div class="stub-server-status"></div>', props: ['serverStatus', 'isServerRunning', 'isServerStarting', 'sseLive', 'sseReconnecting', 'sseReconnectAttempts', 'statusProgress', 'statusMessage', 'statusLogs'] },
    TaskDetailLinkedProjectsPanel: { template: '<div class="stub-linked-projects"></div>', props: ['tenantId', 'workspaceId', 'taskId', 'isEditing', 'taskProjectsWithDetails', 'editingTask', 'containerEndpointRegistered', 'repoCloneIdentitySaveError', 'repoCloneIdentitySaveErrorTraceId', 'containerHeartbeatStatus', 'recloneLoadingByUrl', 'repoRecloneGlobalLoading', 'recloneStatusByUrl', 'repoCloneIdentityByUrl', 'layerGitIdentityOptions', 'repoCloneIdentitySaving', 'layerGitIdentityLoading', 'cloneProgressEntryByRepoMatchKey', 'gitCloneRefMatchKey', 'repoCloneFieldId', 'cloneProgressRowHasSubPhases', 'cloneProgressRecvPct', 'cloneProgressUnpackPct', 'cloneProgressBarWidthTransitionClass', 'workspaceProjects', 'workspaceProjectsLoading', 'getProjectRepos', 'getRepoBranchValue', 'getRepoBranches', 'isLoadingRepoBranches', 'selectedLayerGraphFileTreeLayerId', 'layerRepoGitIdentityLoading', 'layerRepoGitIdentityFetchError', 'layerRepoGitIdentityFetchErrorTraceId', 'perRepoGitIdentitySyncing', 'perRepoGitIdentitySyncError', 'perRepoGitIdentitySyncErrorTraceId', 'staleRepoSyncLoading', 'staleRepoSyncError', 'staleRepoSyncErrorTraceId', 'containerGitIdentityLineForRepoUrl'] },
    TaskDetailCommentsPanel: { template: '<div class="stub-comments"><slot name="execution-details" :comment="{ id: \'c1\' }" :is-active="true" :execution-mode="\'wait_previous\'" /></div>', props: ['newComment', 'showContainerCloneProgressBanner', 'containerCloneProgressEntries', 'repoCloneFieldId', 'shortCloneRepoLabel', 'cloneProgressRowHasSubPhases', 'cloneProgressRecvPct', 'cloneProgressUnpackPct', 'cloneProgressBarWidthTransitionClass', 'cloneProgressRowLogIsPlaceholder', 'cloneProgressRowLogDisplayText', 'displayComments', 'collaboratorNameById', 'collaboratorAvatarById', 'commentsFeedErrors', 'commentsHasMore', 'commentsLoadingMore', 'aiStreamBusy', 'aiStreamBuffer', 'activeContainerAgentId', 'activeExecutionCommentId', 'hasZtreeAgentSteps', 'agentStepCount', 'tenantId', 'workspaceId', 'taskId', 'commentComposerChips'] },
    ServerConfig: { template: '<div class="stub-server-config"></div>', props: ['task', 'workspaceId', 'sseConnection', 'updateServerStatus', 'statusMessage', 'statusProgress', 'statusLogs', 'serverStatus', 'isServerRunning', 'isServerStarting', 'serverUrl', 'containerVscodeUrl', 'containerHeartbeatStatus'] },
    TaskDetailTaskLayerAssociationPanel: { template: '<div class="stub-layer-association"></div>', props: ['layerGraphRefreshing', 'layerGraphZNodes', 'layerGraphMetaLine', 'layerGraphBusyActionKey', 'selectedLayerGraphNode', 'selectedZTreeLayerChangesPanel', 'selectedLayerGraphFileTreeLayerId', 'layerChangesRefreshBusy', 'layerChangesRefreshEnabled', 'layerChangesRefreshError', 'layerChangesRefreshErrorTraceId', 'layerChangesGitStagedActionsBlocked', 'layerChangesGitCommitIdentityBlocked', 'containerPageUrl', 'containerPageLinkPendingReveal', 'containerEndpointRegistered', 'containerHttpUnreachable', 'displayContainerVscodeUrl', 'projectFileTreeRefreshNonce', 'layerGraphModelSelectDisabled', 'layerGraphModelOptions', 'layerGraphDefaultModel', 'layerGraphEditRunTargetJobId', 'layerGraphModelLoadError', 'layerGraphCmdError', 'layerGraphCmdErrorTraceId', 'layerGraphCmdSending', 'containerActionsBlocked', 'layerExecLogLoading', 'layerExecLogCopyable', 'layerExecLogClearable', 'layerExecLogCopyFeedback', 'layerExecLogTopError', 'zTreeLogTargets', 'layerCloneLogFetchError', 'layerCloneLogText', 'layerLiveOutputDisplay', 'layerJobLogFetchError', 'layerJobExecutionPayload', 'layerJobCommandHead', 'layerJobOutputDisplay', 'layerAgentStepCopyFeedbackKey', 'layerAgentStepCards', 'layerGraphCommandKind', 'layerGraphSelectedModel', 'layerGraphAutoIterationCount', 'layerGraphCommandText'] },
  }

  describe('TaskDetail smoke', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({}),
      })
    })

    it('mounts without errors when task prop is provided', { timeout: 15000 }, async () => {
      const { default: TaskDetail } = await import('./TaskDetail.vue')

      const wrapper = mount(TaskDetail, {
        props: {
          task: {
            id: '42',
            title: 'Test Task',
            description: '',
            priority: 2,
            comments: [],
            ai_comments: [],
            projects: [],
            branch_strategy: {},
            parameters: {},
          },
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: '42',
        },
        global: {
          stubs: stubComponents,
        },
      })

      expect(wrapper.find('.embed-container').exists()).toBe(true)
      expect(wrapper.find('.content-card').exists()).toBe(true)
      expect(hoistedMocks.catalogCompanyId).toBe('t1')
    })

    it('mounts without task prop (fetches from API)', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          id: '42',
          title: 'Fetched Task',
          description: '',
          priority: 2,
          comments: [],
          ai_comments: [],
          projects: [],
          branch_strategy: {},
          parameters: {},
        }),
      })

      const { default: TaskDetail } = await import('./TaskDetail.vue')

      const wrapper = mount(TaskDetail, {
        props: {
          tenantId: 't1',
          workspaceId: 'w1',
          taskId: '42',
        },
        global: {
          stubs: stubComponents,
        },
      })

      expect(wrapper.find('.embed-container').exists()).toBe(true)
    })
  })
}
