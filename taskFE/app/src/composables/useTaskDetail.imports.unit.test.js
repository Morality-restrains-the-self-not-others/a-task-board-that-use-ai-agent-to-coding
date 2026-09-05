import { describe, expect, it } from 'vitest'
import { readUseTaskDetailBundle } from './useTaskDetailBundle.js'

describe('useTaskDetail imports', () => {
  it('does not declare createTaskDetailProjectRepoState twice', () => {
    const src = readUseTaskDetailBundle()
    const matches = src.match(/createTaskDetailProjectRepoState/g) || []
    // one import binding + usages is fine; exactly one import from path
    const importBlocks = src.match(/import\s*\{[^}]*createTaskDetailProjectRepoState[^}]*\}\s*from\s*['"].*taskDetailProjectRepoState/g) || []
    expect(importBlocks.length).toBe(1)
    expect(matches.length).toBeGreaterThanOrEqual(1)
  })

  it('wires parent/fork workspace_seq from detail fetch, not only workspaceTodos', () => {
    const src = readUseTaskDetailBundle()
    expect(src).toContain('const parentTaskSeq = ref(0)')
    expect(src).toContain('const forkSourceSeq = ref(0)')
    expect(src).toContain('parentTaskSeq,')
    expect(src).toContain('forkSourceSeq,')
    expect(src).toContain("provide('taskDetailParentDeliverable'")
    expect(src).toMatch(/parentTaskSeq,\s*\n\s*isParentTaskTitleLoading/)
  })

  it('T35: bootstrap-done watch also snaps comment-level live clone map', () => {
    const src = readUseTaskDetailBundle()
    expect(src).toMatch(
      /watch\(\s*\[\s*containerBootstrapCloneLogFull,\s*containerCloneProgressByKey,\s*containerCloneProgressByCommentId\s*\]/,
    )
  })
})
