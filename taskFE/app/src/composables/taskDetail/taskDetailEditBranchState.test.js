import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import {
  createTaskDetailEditBranchState,
  installTaskDetailEditBranchWatchers,
} from './taskDetailEditBranchState.js'

describe('createTaskDetailEditBranchState', () => {
  it('builds custom merge target from editingTask fallback', () => {
    const editingTask = ref({ mergeTargetName: 'release/x', mergeTargetPreset: 'custom' })
    const state = createTaskDetailEditBranchState({
      effectiveTenantId: ref('1'),
      effectiveTaskId: ref('42'),
      localTask: ref({ title: 'T' }),
      editingTask,
      isEditing: ref(false),
      editError: ref(''),
      workspaceProjects: ref([]),
      getProjectRepos: () => [],
      inferWorkBranchPreset: () => 'custom',
      inferMergeTargetPreset: () => 'custom',
    })
    expect(state.buildMergeTargetBranchName('custom', '42')).toBe('release/x')
  })

  it('cancelEdit clears editing state', () => {
    const editingTask = ref({ title: 'x' })
    const isEditing = ref(true)
    const editError = ref('err')
    const state = createTaskDetailEditBranchState({
      effectiveTenantId: ref('1'),
      effectiveTaskId: ref('42'),
      localTask: ref(null),
      editingTask,
      isEditing,
      editError,
      workspaceProjects: ref([]),
      getProjectRepos: () => [],
      inferWorkBranchPreset: () => 'custom',
      inferMergeTargetPreset: () => 'custom',
    })
    state.cancelEdit()
    expect(isEditing.value).toBe(false)
    expect(editingTask.value).toBe(null)
    expect(editError.value).toBe('')
  })

  it('addProjectAssociation adds at most one linked project shell', () => {
    const editingTask = ref({ linkedProjects: [] })
    const state = createTaskDetailEditBranchState({
      effectiveTenantId: ref('1'),
      effectiveTaskId: ref('42'),
      localTask: ref(null),
      editingTask,
      isEditing: ref(true),
      editError: ref(''),
      workspaceProjects: ref([]),
      getProjectRepos: () => [],
      inferWorkBranchPreset: () => 'custom',
      inferMergeTargetPreset: () => 'custom',
    })
    state.addProjectAssociation()
    state.addProjectAssociation()
    expect(editingTask.value.linkedProjects).toHaveLength(1)
  })

  it('title watcher schedules work-branch update when preset is not custom', async () => {
    vi.useFakeTimers()
    const editingTask = ref({
      title: 'hello',
      workBranchPreset: 'feat',
      workBranchName: '',
    })
    const state = createTaskDetailEditBranchState({
      effectiveTenantId: ref('1'),
      effectiveTaskId: ref('99'),
      localTask: ref({ title: 'hello', created_at: '2026-01-01T00:00:00Z' }),
      editingTask,
      isEditing: ref(true),
      editError: ref(''),
      workspaceProjects: ref([]),
      getProjectRepos: () => [],
      inferWorkBranchPreset: () => 'feat',
      inferMergeTargetPreset: () => 'custom',
    })
    const spy = vi.spyOn(state, 'scheduleWorkBranchNameUpdate')
    installTaskDetailEditBranchWatchers(state, { editingTask })
    editingTask.value = { ...editingTask.value, title: 'hello-2' }
    await Promise.resolve()
    expect(spy).toHaveBeenCalled()
    vi.useRealTimers()
  })
})
