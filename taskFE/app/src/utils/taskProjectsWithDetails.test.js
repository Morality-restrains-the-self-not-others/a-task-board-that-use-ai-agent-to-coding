import { describe, expect, it } from 'vitest'
import { buildTaskProjectsWithDetails } from './taskProjectsWithDetails.js'

describe('buildTaskProjectsWithDetails', () => {
  const workspaceProjects = [
    {
      id: 'proj_881195029417193472',
      name: 'helloworld',
      git_repos: ['https://github.com/test-ruandao/helloworld.git'],
    },
  ]

  it('flags mismatch when task stored URL owner differs from current project git_repos', () => {
    const rows = buildTaskProjectsWithDetails(
      [
        {
          project_id: 'proj_881195029417193472',
          repo_index: 0,
          base_branch: 'master',
          stored_repo_address: 'https://github.com/ruandao/helloworld',
          repo_address_mismatch: false,
        },
      ],
      workspaceProjects,
    )
    expect(rows).toHaveLength(1)
    expect(rows[0].repo_address_mismatch).toBe(true)
    expect(rows[0].stored_repo_address).toBe('https://github.com/ruandao/helloworld')
    expect(rows[0].repo_branches['https://github.com/test-ruandao/helloworld.git']).toBe('master')
  })

  it('does not flag mismatch when only .git suffix differs', () => {
    const rows = buildTaskProjectsWithDetails(
      [
        {
          project_id: 'proj_881195029417193472',
          repo_index: 0,
          base_branch: 'master',
          stored_repo_address: 'https://github.com/test-ruandao/helloworld',
          repo_address_mismatch: false,
        },
      ],
      workspaceProjects,
    )
    expect(rows[0].repo_address_mismatch).toBe(false)
  })

  it('accepts camelCase projectId when API rows omit project_id', () => {
    const rows = buildTaskProjectsWithDetails(
      [
        {
          projectId: 'proj_881195029417193472',
          repo_index: 0,
          base_branch: 'master',
          project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
        },
      ],
      [],
    )
    expect(rows).toHaveLength(1)
    expect(rows[0].project_id).toBe('proj_881195029417193472')
    expect(rows[0].project.git_repos).toEqual(['https://github.com/test-ruandao/helloworld.git'])
    expect(rows[0].project_missing).toBe(true)
  })

  it('uses API repo URLs when workspace catalog is empty', () => {
    const rows = buildTaskProjectsWithDetails(
      [
        {
          project_id: 'proj_881195029417193472',
          repo_index: 0,
          base_branch: 'master',
          stored_repo_address: 'https://github.com/ruandao/helloworld',
          project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
          repo_address_mismatch: true,
        },
      ],
      [],
    )
    expect(rows).toHaveLength(1)
    expect(rows[0].project_id).toBe('proj_881195029417193472')
    expect(rows[0].project.git_repos).toEqual(['https://github.com/test-ruandao/helloworld.git'])
    expect(rows[0].repo_branches['https://github.com/test-ruandao/helloworld.git']).toBe('master')
    expect(rows[0].repo_address_mismatch).toBe(true)
  })

  it('keeps API mismatch flag even if stored equals current', () => {
    const rows = buildTaskProjectsWithDetails(
      [
        {
          project_id: 'proj_881195029417193472',
          repo_index: 0,
          base_branch: 'master',
          stored_repo_address: 'https://github.com/test-ruandao/helloworld.git',
          repo_address_mismatch: true,
        },
      ],
      workspaceProjects,
    )
    expect(rows[0].repo_address_mismatch).toBe(true)
  })

  it('marks catalog-missing project_id as project_missing and does not fake a name from the id', () => {
    const rows = buildTaskProjectsWithDetails(
      [
        {
          project_id: 'proj_880498883115905024',
          repo_index: 0,
          base_branch: 'main',
          project_repo_url: 'http://115.29.110.74/example-user/somanyad.git',
        },
      ],
      [],
    )
    expect(rows).toHaveLength(1)
    expect(rows[0].project_missing).toBe(true)
    expect(rows[0].project_name).toBe('')
    expect(rows[0].project.git_repos).toEqual(['http://115.29.110.74/example-user/somanyad.git'])
  })
})
