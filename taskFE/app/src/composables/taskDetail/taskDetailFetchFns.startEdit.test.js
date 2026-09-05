// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import { startEdit } from './taskDetailFetchFns.js'

describe('startEdit linkedProjects', () => {
  const baseDeps = () => {
    const localTask = {
      value: {
        id: '860371538948571136',
        title: 'relay task',
        description: 'seed',
        priority: 2,
        owner: '858973721300316160',
        assignees: ['858973721300316160'],
        branch_strategy: {
          work_branch_name: 'feature/demo',
          merge_target_branch_name: 'develop',
        },
        projects: [],
      },
    }
    const editingTask = { value: null }
    const isEditing = { value: false }
    const editError = { value: '' }
    const getProjectRepos = vi.fn(() => [])
    const inferWorkBranchPreset = () => 'feature'
    const inferMergeTargetPreset = () => 'develop'
    return {
      localTask,
      workspaceProjects: { value: [] },
      editingTask,
      isEditing,
      editError,
      getProjectRepos,
      inferWorkBranchPreset,
      inferMergeTargetPreset,
    }
  }

  it('无关联项目时编辑态应保留一行空项目选择', () => {
    const deps = baseDeps()
    startEdit(deps)
    expect(deps.isEditing.value).toBe(true)
    expect(deps.editingTask.value.linkedProjects).toEqual([
      { project_id: '', repo_branches: {} },
    ])
  })

  it('已有关联项目时编辑态仅保留单项目行并回填分支', () => {
    const deps = baseDeps()
    deps.localTask.value.projects = [
      { project_id: '100', repo_index: 0, base_branch: 'main' },
      { project_id: '100', repo_index: 1, base_branch: 'develop' },
      { project_id: '200', repo_index: 0, base_branch: 'ignored' },
    ]
    deps.getProjectRepos.mockImplementation((pid) => {
      if (String(pid) === '100') {
        return ['https://git.example/a.git', 'https://git.example/b.git']
      }
      return []
    })
    startEdit(deps)
    expect(deps.editingTask.value.linkedProjects).toHaveLength(1)
    expect(deps.editingTask.value.linkedProjects[0]).toEqual({
      project_id: '100',
      repo_branches: {
        'https://git.example/a.git': 'main',
        'https://git.example/b.git': 'develop',
      },
    })
  })

  it('编辑态应解析 ${taskTitle} 与历史 __taskTitle_', () => {
    const deps = baseDeps()
    deps.localTask.value.title = 'Fix Title'
    deps.localTask.value.branch_strategy = {
      work_branch_name: 'feature/2026-07-12_u_daydaymoney${taskId}___taskTitle_',
      merge_target_branch_name: 'develop',
    }
    startEdit(deps)
    expect(deps.editingTask.value.workBranchName).toBe(
      'feature/2026-07-12_u_aidev860371538948571136_Fix_Title',
    )
    expect(deps.editingTask.value.workBranchName).not.toContain('taskTitle')
  })

  it('编辑态回填 deliverable_obj_id', () => {
    const deps = baseDeps()
    deps.localTask.value.deliverable_obj_id = 'cat-9'
    startEdit(deps)
    expect(deps.editingTask.value.deliverable_obj_id).toBe('cat-9')
  })
})
