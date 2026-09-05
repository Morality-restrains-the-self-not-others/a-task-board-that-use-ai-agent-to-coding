import { describe, it, expect, vi } from 'vitest'
import {
  collectTaskRepoAddressMismatches,
  syncTaskRepoAddressesFromProjects,
} from './taskRepoAddressMismatch.js'

describe('collectTaskRepoAddressMismatches', () => {
  it('returns empty when no mismatch', () => {
    expect(collectTaskRepoAddressMismatches([])).toEqual([])
    expect(
      collectTaskRepoAddressMismatches([
        { project_id: 'p1', repo_address_mismatch: false },
      ]),
    ).toEqual([])
  })

  it('collects mismatch rows with addresses', () => {
    expect(
      collectTaskRepoAddressMismatches([
        {
          project_id: 'p1',
          repo_address_mismatch: true,
          stored_repo_address: 'http://old/repo',
          project_repo_url: 'http://new/repo',
        },
      ]),
    ).toEqual([
      {
        projectId: 'p1',
        storedRepoAddress: 'http://old/repo',
        projectRepoUrl: 'http://new/repo',
      },
    ])
  })

  it('skips rows missing stored or current url', () => {
    expect(
      collectTaskRepoAddressMismatches([
        {
          project_id: 'p1',
          repo_address_mismatch: true,
          stored_repo_address: '',
          project_repo_url: 'http://new/repo',
        },
      ]),
    ).toEqual([])
  })
})

describe('syncTaskRepoAddressesFromProjects', () => {
  it('PATCHes todos with project payload', async () => {
    const apiFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 'task-1', projects: [] }),
    })
    await syncTaskRepoAddressesFromProjects({
      apiFetch,
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task-1',
      projects: [
        {
          project_id: 'p1',
          repo_index: 0,
          base_branch: 'main',
          target_branch: 'dev',
        },
      ],
    })
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/tasks/todos/tenant_id/t1/workspace_id/w1/task-1/',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({
          projects: [
            {
              project_id: 'p1',
              repo_index: 0,
              base_branch: 'main',
              target_branch: 'dev',
            },
          ],
        }),
      }),
    )
  })

  it('throws on failed response', async () => {
    const apiFetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ detail: '权限不足' }),
    })
    await expect(
      syncTaskRepoAddressesFromProjects({
        apiFetch,
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task-1',
        projects: [{ project_id: 'p1' }],
      }),
    ).rejects.toThrow('权限不足')
  })
})
