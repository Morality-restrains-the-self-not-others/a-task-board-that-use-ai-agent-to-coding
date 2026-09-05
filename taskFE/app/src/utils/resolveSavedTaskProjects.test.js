import { describe, expect, it } from 'vitest'
import { resolveSavedTaskProjects } from './resolveSavedTaskProjects.js'

describe('resolveSavedTaskProjects', () => {
  it('keeps GET projects when they include project_id', () => {
    const getProjects = [{
      project_id: 'proj_1',
      stored_repo_address: 'https://github.com/ruandao/helloworld',
    }]
    expect(resolveSavedTaskProjects(
      { id: 'task-1', projects: getProjects },
      [{ project_id: 'proj_1', repo_index: 0 }],
    )).toEqual(getProjects)
  })

  it('falls back to send payload when GET has no linked rows', () => {
    const send = [{ project_id: 'proj_1', repo_index: 0 }]
    expect(resolveSavedTaskProjects({ id: 'task-1', title: 'demo' }, send)).toEqual(send)
    expect(resolveSavedTaskProjects({ id: 'task-1', projects: [] }, send)).toEqual(send)
  })
})
