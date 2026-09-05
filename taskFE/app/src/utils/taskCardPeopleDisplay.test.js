import { describe, expect, it } from 'vitest'
import {
  buildCollaboratorAvatarById,
  buildCollaboratorNameById,
  formatCollaboratorsDisplay,
  formatSingleMemberDisplay,
  resolveCommentAuthorAvatar,
  resolveCommentAuthorDisplayName,
  resolveMemberDisplayName,
} from './taskCardPeopleDisplay.js'

describe('taskCardPeopleDisplay', () => {
  const nameById = { m1: '张三', m2: '李四', m3: '王五' }

  it('buildCollaboratorNameById prefers member_name', () => {
    const map = buildCollaboratorNameById([
      { id: 'm1', member_name: '张三', username: 'zhang' },
      { id: 2, username: 'li' },
      { id: '', member_name: 'ignore' },
    ])
    expect(map.m1).toBe('张三')
    expect(map['2']).toBe('li')
    expect(map['']).toBeUndefined()
  })

  it('buildCollaboratorNameById indexes by user id for comment authors', () => {
    const map = buildCollaboratorNameById([
      { id: '850256677331562497', user: '850256676127797248', member_name: '公司创建者' },
    ])
    expect(map['850256677331562497']).toBe('公司创建者')
    expect(map['850256676127797248']).toBe('公司创建者')
  })

  it('resolveCommentAuthorDisplayName prefers collaborator nickname over raw user id', () => {
    const map = { '850256676127797248': '公司创建者' }
    expect(resolveCommentAuthorDisplayName(
      { id: '850256676127797248', username: '850256676127797248' },
      map,
      '未知',
    )).toBe('公司创建者')
    expect(resolveCommentAuthorDisplayName(
      { id: 'u-missing', username: 'alice' },
      map,
      '未知',
    )).toBe('alice')
    expect(resolveCommentAuthorDisplayName(null, map, '容器 Agent')).toBe('容器 Agent')
  })

  it('buildCollaboratorAvatarById indexes personal/company avatar by id and user', () => {
    const map = buildCollaboratorAvatarById([
      {
        id: 'm1',
        user: 'u1',
        member_avatar_url: '/media/company/a.png',
        avatar_url: '/media/user/b.png',
      },
      { id: 'm2', user_id: 'u2', avatar_url: '/media/user/c.png' },
      { id: 'm3', user: 'u3' },
    ])
    expect(map.m1).toBe('/media/company/a.png')
    expect(map.u1).toBe('/media/company/a.png')
    expect(map.m2).toBe('/media/user/c.png')
    expect(map.u2).toBe('/media/user/c.png')
    expect(map.m3).toBeUndefined()
  })

  it('resolveCommentAuthorAvatar prefers created_by.avatar_url then collaborator map', () => {
    const map = { u1: '/media/from-map.png' }
    expect(resolveCommentAuthorAvatar(
      { id: 'u1', avatar_url: '/media/direct.png' },
      map,
    )).toBe('/media/direct.png')
    expect(resolveCommentAuthorAvatar({ id: 'u1', username: 'x' }, map)).toBe('/media/from-map.png')
    expect(resolveCommentAuthorAvatar({ id: 'missing' }, map)).toBe('')
    expect(resolveCommentAuthorAvatar(null, map)).toBe('')
  })

  it('resolveMemberDisplayName falls back to ID: prefix', () => {
    expect(resolveMemberDisplayName('m9', nameById)).toBe('ID:m9')
    expect(resolveMemberDisplayName('m1', nameById)).toBe('张三')
    expect(resolveMemberDisplayName(null, nameById)).toBe('')
  })

  it('formatSingleMemberDisplay is singular for 操作员/负责人', () => {
    expect(formatSingleMemberDisplay('m1', nameById)).toBe('张三')
    expect(formatSingleMemberDisplay('', nameById)).toBe('未指派')
    expect(formatSingleMemberDisplay(null, nameById)).toBe('未指派')
  })

  it('formatCollaboratorsDisplay joins assignees with顿号', () => {
    expect(formatCollaboratorsDisplay(['m1', 'm2'], nameById)).toBe('张三、李四')
    expect(formatCollaboratorsDisplay([], nameById)).toBe('未指派')
    expect(formatCollaboratorsDisplay(null, nameById)).toBe('未指派')
  })

  it('formatCollaboratorsDisplay caps at 2 names +N', () => {
    expect(formatCollaboratorsDisplay(['m1', 'm2', 'm3'], nameById)).toBe('张三、李四 +1')
    expect(formatCollaboratorsDisplay(['m1', 'm2', 'm3', 'm9'], nameById)).toBe('张三、李四 +2')
  })
})
