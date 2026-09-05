// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailEditBranchState.abort.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { ref } = await import('vue')

vi.mock('./taskDetailFetchFns.js', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    fetchTranslatedTaskTitleSegment: vi.fn(),
  }
})

const { fetchTranslatedTaskTitleSegment } = await import('./taskDetailFetchFns.js')
const { createTaskDetailEditBranchState } = await import('./taskDetailEditBranchState.js')

describe('createTaskDetailEditBranchState abort', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not rewrite work branch when title translate is aborted', async () => {
    const abortErr = new Error('aborted')
    abortErr.name = 'AbortError'
    fetchTranslatedTaskTitleSegment.mockRejectedValue(abortErr)
    const editingTask = ref({
      title: '中文标题',
      workBranchPreset: 'feature',
      workBranchName: 'feature/keep-me',
    })
    const state = createTaskDetailEditBranchState({
      effectiveTenantId: ref('1'),
      effectiveTaskId: ref('42'),
      localTask: ref({ title: '中文标题' }),
      editingTask,
      isEditing: ref(true),
      editError: ref(''),
      workspaceProjects: ref([]),
      getProjectRepos: () => [],
      inferWorkBranchPreset: () => 'feature',
      inferMergeTargetPreset: () => 'custom',
    })
    await state.updateWorkBranchNameByPreset()
    expect(state.taskTitleTranslationError.value).toBe('')
    expect(editingTask.value.workBranchName).toBe('feature/keep-me')
  })
})
}
