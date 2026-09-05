// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { applyContainerGitCloneProgress } from './applyContainerGitCloneProgress.js'

function makeDeps(overrides = {}) {
  return {
    containerBootstrapCloneLogFull: ref(''),
    containerCloneProgressByKey: ref({}),
    containerCloneProgressByCommentId: ref({}),
    markContainerTransportOk: vi.fn(),
    onBootstrapCloneLogUpdate: vi.fn(),
    refreshLayerGraphFromServer: vi.fn(),
    bumpProjectFileTreeRefresh: vi.fn(),
    rescheduleContainerCloneProgressClearTimer: vi.fn(),
    CONTAINER_CLONE_PROGRESS_GLOBAL_KEY: '__global__',
    containerCloneProgressTimer: null,
    ...overrides,
  }
}

describe('applyContainerGitCloneProgress', () => {
  it('ignores non-clone statuses', () => {
    const deps = makeDeps()
    expect(applyContainerGitCloneProgress({ status: 'processing' }, deps)).toBe(false)
    expect(deps.containerCloneProgressByKey.value).toEqual({})
  })

  it('T8: mirrors per-repo progress onto comment_id map and task-level map', () => {
    const deps = makeDeps()
    const ok = applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 40,
      message: '【项目克隆】(1/1) alpha … 40%',
      repo_url: 'https://git.example/alpha.git',
    }, deps)
    expect(ok).toBe(true)
    expect(deps.markContainerTransportOk).toHaveBeenCalled()
    expect(deps.containerCloneProgressByKey.value['https://git.example/alpha.git'].progress).toBe(40)
    expect(deps.containerCloneProgressByCommentId.value.C1['https://git.example/alpha.git'].progress).toBe(40)
    expect(deps.rescheduleContainerCloneProgressClearTimer).toHaveBeenCalled()
  })

  it('does not copy C1 progress onto C2', () => {
    const deps = makeDeps()
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 40,
      message: 'a',
      repo_url: 'https://git.example/alpha.git',
    }, deps)
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C2',
      progress: 90,
      message: 'b',
      repo_url: 'https://git.example/beta.git',
    }, deps)
    expect(deps.containerCloneProgressByCommentId.value.C1['https://git.example/alpha.git'].progress).toBe(40)
    expect(deps.containerCloneProgressByCommentId.value.C2['https://git.example/beta.git'].progress).toBe(90)
    expect(deps.containerCloneProgressByCommentId.value.C2['https://git.example/alpha.git']).toBeUndefined()
  })

  it('all-done clears task-level map but keeps per-repo comment bar at 100', () => {
    const deps = makeDeps()
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 50,
      message: 'mid',
      repo_url: 'https://git.example/alpha.git',
    }, deps)
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 100,
      message: '【项目克隆】仓库克隆已完成',
    }, deps)
    expect(deps.containerCloneProgressByKey.value).toEqual({})
    expect(deps.containerCloneProgressByCommentId.value.C1['https://git.example/alpha.git'].progress).toBe(100)
    expect(deps.containerCloneProgressByCommentId.value.C1.__global__).toBeUndefined()
    expect(deps.refreshLayerGraphFromServer).toHaveBeenCalled()
  })

  it('T36: all-done then late 3% with repo_url must not drop comment bar from 100', () => {
    const deps = makeDeps()
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 100,
      message: '项目克隆 (1/1) 完成 ram-work',
      repo_url: 'https://git.example/ram-work.git',
    }, deps)
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 100,
      message: '【项目克隆】仓库克隆已完成',
    }, deps)
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 3,
      message: '【项目克隆】(1/1) ram-work … 3%',
      repo_url: 'https://git.example/ram-work.git',
      segment: { recv_progress: 3 },
    }, deps)
    const row = deps.containerCloneProgressByCommentId.value.C1['https://git.example/ram-work.git']
    expect(row.progress).toBe(100)
    expect(row.message).toContain('完成')
  })

  it('does not overlay global failure summary when per-repo rows exist', () => {
    const deps = makeDeps()
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 0,
      message: '【项目克隆】(1/1) 失败 ram-work: git exit 128',
      repo_url: 'https://git.example/ram-work.git',
    }, deps)
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 0,
      message: '【项目克隆】部分失败：失败 1/1 个仓库：ram-work（其余已就绪，引导继续）',
    }, deps)
    const commentMap = deps.containerCloneProgressByCommentId.value.C1
    expect(commentMap['https://git.example/ram-work.git'].message).toContain('git exit 128')
    expect(commentMap.__global__).toBeUndefined()
    expect(deps.containerCloneProgressByKey.value.__global__).toBeUndefined()
  })

  it('T32: later 9% SSE cannot overwrite per-repo 100% completion', () => {
    const deps = makeDeps()
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 9,
      message: '【项目克隆】(1/1) ram-work … 9%',
      repo_url: 'https://git.example/ram-work.git',
    }, deps)
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 100,
      message: '项目克隆 (1/1) 完成 ram-work',
      repo_url: 'https://git.example/ram-work.git',
      segment: { recv_progress: 100, unpack_progress: 100 },
    }, deps)
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      comment_id: 'C1',
      progress: 9,
      message: '【项目克隆】(1/1) ram-work … 9%',
      repo_url: 'https://git.example/ram-work.git',
      segment: { recv_progress: 9 },
    }, deps)
    const row = deps.containerCloneProgressByCommentId.value.C1['https://git.example/ram-work.git']
    expect(row.progress).toBe(100)
    expect(row.message).toContain('完成')
    expect(row.recvProgress).toBe(100)
    expect(row.unpackProgress).toBe(100)
    expect(deps.containerCloneProgressByKey.value['https://git.example/ram-work.git'].progress).toBe(100)
  })

  it('skips comment map when comment_id missing (task-level only)', () => {
    const deps = makeDeps()
    applyContainerGitCloneProgress({
      status: 'container_git_clone_progress',
      progress: 12,
      message: 'boot',
    }, deps)
    expect(deps.containerCloneProgressByKey.value.__global__.progress).toBe(12)
    expect(deps.containerCloneProgressByCommentId.value).toEqual({})
  })
})
