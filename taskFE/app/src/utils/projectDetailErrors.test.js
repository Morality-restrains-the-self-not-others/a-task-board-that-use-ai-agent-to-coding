import { describe, expect, it } from 'vitest'
import { projectDetailFetchErrorMessage } from './projectDetailErrors.js'

describe('projectDetailFetchErrorMessage', () => {
  it('maps 404 and project not found to a deleted-project message', () => {
    expect(projectDetailFetchErrorMessage(404, 'project not found')).toBe('项目不存在或已删除')
    expect(projectDetailFetchErrorMessage(404, '')).toBe('项目不存在或已删除')
    expect(projectDetailFetchErrorMessage(200, 'project not found')).toBe('项目不存在或已删除')
  })

  it('keeps other details and falls back when empty', () => {
    expect(projectDetailFetchErrorMessage(500, 'db down')).toBe('db down')
    expect(projectDetailFetchErrorMessage(500, '')).toBe('获取项目详情失败')
  })
})
