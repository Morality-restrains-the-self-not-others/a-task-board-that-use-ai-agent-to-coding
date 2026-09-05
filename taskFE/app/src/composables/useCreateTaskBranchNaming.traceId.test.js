if (!process.env.VITEST) {
  console.log('[skip] useCreateTaskBranchNaming.traceId.test.js requires vitest runtime')
} else {
const { describe, it, expect, vi, beforeEach } = await import('vitest')
const { ref, nextTick } = await import('vue')

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

vi.mock('../utils/workPanelApiUtils.js', () => ({
  parseJsonSafe: vi.fn(async (response) => {
    try {
      return await response.json()
    } catch {
      return null
    }
  }),
  warnOptionalApiFailure: vi.fn(),
}))

const { apiFetch } = await import('../utils/apiUtils.js')
const { useCreateTaskBranchNaming } = await import('./useCreateTaskBranchNaming.js')

describe('useCreateTaskBranchNaming data-traceId', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('stores response.traceId on translate-branch-title failure', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 405,
      traceId: 'tid-translate-405',
      json: async () => ({ error: 'method not allowed' }),
    })

    const editingTask = ref({
      title: '中文任务标题',
      workBranchPreset: 'feature',
      workBranchName: '',
      mergeTargetPreset: 'custom',
      mergeTargetName: '',
      projectSelections: [],
    })

    const {
      updateWorkBranchNameByPreset,
      taskTitleTranslationError,
      taskTitleTranslationErrorTraceId,
    } = useCreateTaskBranchNaming({
      editingTask: () => editingTask.value,
      companyUserName: () => 'tester',
      tenantId: () => '850256677331562496',
      show: () => true,
      getProjectBranches: () => [],
      isProjectBranchesLoading: () => false,
      getProjectBranchesError: () => '',
      branchMapKey: (projectId, repoIndex) => `${projectId}::${repoIndex}`,
    })

    await updateWorkBranchNameByPreset()
    await nextTick()
    expect(taskTitleTranslationError.value).toBe('method not allowed')
    expect(taskTitleTranslationErrorTraceId.value).toBe('tid-translate-405')
    expect(apiFetch).toHaveBeenCalled()
  })

  it('does not render fanyi JSON parse internals on translate-branch-title 502', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 502,
      traceId: '6b246acd-ef86-440a-96a2-cc4d313db932',
      json: async () => ({
        error: '任务标题翻译失败: fanyi_agent 响应无效: unexpected end of JSON input',
      }),
    })

    const editingTask = ref({
      title: '中文任务标题',
      workBranchPreset: 'feature',
      workBranchName: '',
      mergeTargetPreset: 'custom',
      mergeTargetName: '',
      projectSelections: [],
    })

    const {
      updateWorkBranchNameByPreset,
      taskTitleTranslationError,
      taskTitleTranslationErrorTraceId,
    } = useCreateTaskBranchNaming({
      editingTask: () => editingTask.value,
      companyUserName: () => 'tester',
      tenantId: () => '850256677331562496',
      show: () => true,
      getProjectBranches: () => [],
      isProjectBranchesLoading: () => false,
      getProjectBranchesError: () => '',
      branchMapKey: (projectId, repoIndex) => `${projectId}::${repoIndex}`,
    })

    await updateWorkBranchNameByPreset()
    await nextTick()
    expect(taskTitleTranslationError.value).not.toContain('unexpected end of JSON')
    expect(taskTitleTranslationError.value).not.toContain('fanyi_agent')
    expect(taskTitleTranslationError.value).toContain('本地规则')
    expect(taskTitleTranslationErrorTraceId.value).toBe('6b246acd-ef86-440a-96a2-cc4d313db932')
  })

  it('surfaces categorized why from translate-branch-title 502', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 502,
      traceId: 'tid-empty-content-why',
      json: async () => ({
        error: '任务标题自动翻译失败：翻译服务未返回可用译文',
      }),
    })

    const editingTask = ref({
      title: '中文任务标题',
      workBranchPreset: 'feature',
      workBranchName: '',
      mergeTargetPreset: 'custom',
      mergeTargetName: '',
      projectSelections: [],
    })

    const {
      updateWorkBranchNameByPreset,
      taskTitleTranslationError,
      taskTitleTranslationErrorTraceId,
    } = useCreateTaskBranchNaming({
      editingTask: () => editingTask.value,
      companyUserName: () => 'tester',
      tenantId: () => '850256677331562496',
      show: () => true,
      getProjectBranches: () => [],
      isProjectBranchesLoading: () => false,
      getProjectBranchesError: () => '',
      branchMapKey: (projectId, repoIndex) => `${projectId}::${repoIndex}`,
    })

    await updateWorkBranchNameByPreset()
    await nextTick()
    expect(taskTitleTranslationError.value).toContain('未返回可用译文')
    expect(taskTitleTranslationError.value).toContain('本地规则')
    expect(taskTitleTranslationErrorTraceId.value).toBe('tid-empty-content-why')
  })

  it('does not show a red error when an in-flight title translate is aborted', async () => {
    const abortErr = new Error('aborted')
    abortErr.name = 'AbortError'
    apiFetch.mockRejectedValue(abortErr)

    const editingTask = ref({
      title: '中文任务标题',
      workBranchPreset: 'feature',
      workBranchName: 'feature/keep-me',
      mergeTargetPreset: 'custom',
      mergeTargetName: '',
      projectSelections: [],
    })

    const {
      updateWorkBranchNameByPreset,
      taskTitleTranslationError,
    } = useCreateTaskBranchNaming({
      editingTask: () => editingTask.value,
      companyUserName: () => 'tester',
      tenantId: () => '850256677331562496',
      show: () => true,
      getProjectBranches: () => [],
      isProjectBranchesLoading: () => false,
      getProjectBranchesError: () => '',
      branchMapKey: (projectId, repoIndex) => `${projectId}::${repoIndex}`,
    })

    await updateWorkBranchNameByPreset()
    await nextTick()
    expect(taskTitleTranslationError.value).toBe('')
    expect(editingTask.value.workBranchName).toBe('feature/keep-me')
  })
})

}
