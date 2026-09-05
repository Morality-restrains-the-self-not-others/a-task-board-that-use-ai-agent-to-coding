import { describe, expect, it } from 'vitest'
import {
  accessFilterChipLabel,
  filterTodosByAccess,
  normalizeAccessFilter,
  parseAccessFilterSubjects,
  resolveAccessPersonLabel,
  resolveCompanyMemberIdsFromGroupMembers,
  toggleAccessFilter,
} from './workPanelAccessFilter.js'

const todos = [
  { id: 't1', owner: '10', assignees: ['20'] },
  { id: 't2', owner: '30', assignees: [] },
  { id: 't3', owner: null, assignees: [40] },
  { id: 't4', owner: '50', assignees: ['60', '70'] },
]

describe('workPanelAccessFilter', () => {
  it('T1: null filter returns original list', () => {
    expect(filterTodosByAccess(todos, null)).toEqual(todos)
    expect(filterTodosByAccess(todos, undefined)).toEqual(todos)
    expect(filterTodosByAccess(todos, { kind: 'person' })).toEqual(todos)
  })

  it('T2: person filter matches owner', () => {
    const f = { kind: 'person', id: '10', label: 'Alice', memberIds: ['10'] }
    expect(filterTodosByAccess(todos, f).map((t) => t.id)).toEqual(['t1'])
  })

  it('T3: person filter matches assignee only', () => {
    const f = { kind: 'person', id: '40', memberIds: ['40'] }
    expect(filterTodosByAccess(todos, f).map((t) => t.id)).toEqual(['t3'])
  })

  it('T4: person filter no match yields empty', () => {
    const f = { kind: 'person', id: '999', memberIds: ['999'] }
    expect(filterTodosByAccess(todos, f)).toEqual([])
  })

  it('T5: group filter uses memberIds intersection', () => {
    const f = { kind: 'group', id: 'g1', label: 'Dev', memberIds: ['30', '70'] }
    expect(filterTodosByAccess(todos, f).map((t) => t.id)).toEqual(['t2', 't4'])
  })

  it('T6: empty memberIds yields empty list', () => {
    const f = { kind: 'group', id: 'g-empty', memberIds: [] }
    expect(filterTodosByAccess(todos, f)).toEqual([])
  })

  it('T7: numeric/string id coercion still matches', () => {
    const f = { kind: 'person', id: 40, memberIds: [40] }
    expect(filterTodosByAccess(todos, f).map((t) => t.id)).toEqual(['t3'])
    expect(normalizeAccessFilter({ kind: 'person', id: 1, memberIds: [2] })).toEqual({
      kind: 'person',
      id: '1',
      label: '',
      memberIds: ['2'],
    })
    expect(accessFilterChipLabel({ kind: 'person', id: '1', label: 'Bob' })).toBe('过滤：Bob')
    expect(accessFilterChipLabel({ kind: 'group', id: 'g', label: 'Ops' })).toBe('过滤：小组 Ops')
    expect(toggleAccessFilter(null, { kind: 'person', id: '1', memberIds: ['1'] })?.id).toBe('1')
    expect(
      toggleAccessFilter(
        { kind: 'person', id: '1', memberIds: ['1'] },
        { kind: 'person', id: '1', memberIds: ['1'] },
      ),
    ).toBeNull()
  })

  it('parseAccessFilterSubjects maps company_member_id and groups', () => {
    const perms = [
      { user_info: { id: 'u1', company_member_id: 'cm1', username: 'Alice' } },
      { user_info: { id: 'u2', company_member_id: null, username: 'Bob' } },
      { group_info: { id: 'g1', name: 'Core' } },
    ]
    const collabs = [{ id: 'cm2', user: 'u2' }]
    const { people, groups } = parseAccessFilterSubjects(perms, collabs)
    expect(people).toEqual([
      { id: 'cm1', label: 'Alice', userId: 'u1' },
      { id: 'cm2', label: 'Bob', userId: 'u2' },
    ])
    expect(groups).toEqual([{ id: 'g1', label: 'Core' }])
  })

  it('parseAccessFilterSubjects prefers member_name over username / id prefix', () => {
    const perms = [
      {
        user_info: {
          id: '850256677331562496',
          company_member_id: 'cm-zhao',
          username: '85025667',
          member_name: '赵六·工作面板',
        },
      },
      {
        user_info: {
          id: 'u-only-prefix',
          company_member_id: 'cm-from-collab',
          username: 'u-only-p',
          member_name: '',
        },
      },
    ]
    const collabs = [
      {
        id: 'cm-from-collab',
        user: 'u-only-prefix',
        username: 'u-only-p',
        member_name: '钱七·协作人',
      },
    ]
    const { people } = parseAccessFilterSubjects(perms, collabs)
    expect(people).toEqual([
      { id: 'cm-zhao', label: '赵六·工作面板', userId: '850256677331562496' },
      { id: 'cm-from-collab', label: '钱七·协作人', userId: 'u-only-prefix' },
    ])
    expect(
      resolveAccessPersonLabel(
        { username: '85025667', member_name: '赵六·工作面板' },
        'cm-zhao',
      ),
    ).toBe('赵六·工作面板')
    expect(
      resolveAccessPersonLabel(
        { member_name: '', username: '85025667' },
        'cm-x',
        new Map([['cm-x', { member_name: '孙八', username: 'sun8' }]]),
      ),
    ).toBe('孙八')
  })

  it('resolveCompanyMemberIdsFromGroupMembers maps user→member', () => {
    expect(
      resolveCompanyMemberIdsFromGroupMembers(
        [{ user: 'u1' }, { user: 'u9' }, { user: 'u1' }],
        [
          { id: 'cm1', user: 'u1' },
          { id: 'cm2', user: 'u2' },
        ],
      ),
    ).toEqual(['cm1'])
  })
})
