import { describe, expect, it } from 'vitest'
import { buildTaskRepoRows } from './taskRepoRows.js'

describe('buildTaskRepoRows', () => {
  it('returns rows only for projects linked on the task', () => {
    expect(buildTaskRepoRows(
      [{ project_id: 'proj-1', repo_index: 0, base_branch: 'main' }],
      [
        {
          id: 'proj-1',
          name: 'Alpha',
          git_repos: ['https://github.com/acme/repo.git', 'https://github.com/acme/other.git'],
        },
        {
          id: 'proj-2',
          name: 'Beta',
          git_repos: ['https://github.com/acme/unlinked.git'],
        },
      ],
    )).toEqual([
      {
        url: 'https://github.com/acme/repo.git',
        projectName: 'Alpha',
        projectId: 'proj-1',
      },
      {
        url: 'https://github.com/acme/other.git',
        projectName: 'Alpha',
        projectId: 'proj-1',
      },
    ])
  })

  it('returns empty when task has no linked projects', () => {
    expect(buildTaskRepoRows(
      [],
      [{ id: 'proj-1', name: 'Alpha', git_repos: ['https://github.com/acme/repo.git'] }],
    )).toEqual([])
  })

  it('falls back to task API repo URL when workspace catalog is empty', () => {
    expect(buildTaskRepoRows(
      [{
        project_id: 'proj-1',
        repo_index: 0,
        base_branch: 'master',
        stored_repo_address: 'https://github.com/ruandao/helloworld',
        project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
      }],
      [],
    )).toEqual([
      {
        url: 'https://github.com/test-ruandao/helloworld.git',
        projectName: 'proj-1',
        projectId: 'proj-1',
      },
    ])
  })
})
