// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { overallCloneProgressPct } from '../../utils/commentCloneProgressFromLogs.js'
import { useCommentSectionCloneProgressRows } from './useCommentSectionCloneProgressRows.js'

const PARENT = 'https://git.example/ram-work.git'
const NESTED = 'https://git.example/task2app.git'

function makeProjects(autoCloneNestedRepos) {
  return [{
    project: {
      auto_clone_nested_repos: autoCloneNestedRepos,
      git_repos: [PARENT],
      git_repo_entries: [
        { url: PARENT },
        { url: NESTED, parent_repo_url: PARENT, clone_alias: 'task2app' },
      ],
    },
  }]
}

function makeHook(autoCloneNestedRepos) {
  const props = {
    displayComments: [],
    statusLogs: [],
    cloneProgressByCommentId: {
      C1: {
        [PARENT]: { progress: 100, message: '完成 ram-work', repoUrl: PARENT },
        [NESTED]: { progress: 0, message: 'task2app', repoUrl: NESTED },
      },
    },
    containerCloneProgressEntries: [],
    taskProjectsWithDetails: makeProjects(autoCloneNestedRepos),
    taskRepoRows: [],
    bootstrapCloneLogText: '',
    layerCloneLogText: '',
  }
  return useCommentSectionCloneProgressRows(props, {
    bindingStatusFor: () => '',
    buildPerBindingServerStatusProps: () => ({ statusLogs: [] }),
  })
}

describe('useCommentSectionCloneProgressRows', () => {
  it('T38: auto_clone off omits nested catalog rows so overall stays 100 not ~3%', () => {
    const { commentCloneProgressRows } = makeHook(false)
    const rows = commentCloneProgressRows('C1')
    expect(rows.map((r) => r.label || r.key)).toEqual(['ram-work'])
    expect(overallCloneProgressPct(rows)).toBe(100)
  })

  it('keeps nested rows when auto clone nested is on', () => {
    const { commentCloneProgressRows } = makeHook(true)
    const rows = commentCloneProgressRows('C1')
    expect(rows).toHaveLength(2)
    expect(rows.some((r) => String(r.repoUrl || r.key).includes('task2app'))).toBe(true)
  })
})
