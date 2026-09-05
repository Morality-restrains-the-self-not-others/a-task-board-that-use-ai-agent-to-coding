import { describe, expect, it, vi } from 'vitest'
import {
  assignableRoles,
  assignSubjectRoles,
  roleNamesForSubject,
} from './assignSubjectRoles.js'

describe('assignSubjectRoles helpers', () => {
  it('aggregates multi role names per subject', () => {
    const rows = [
      { member_id: 'm1', role_name: 'member' },
      { member_id: 'm1', role_name: 'custom_a' },
      { member_id: 'm2', role_name: 'member' },
    ]
    expect(roleNamesForSubject(rows, 'member', 'm1')).toEqual(['member', 'custom_a'])
  })

  it('filters assignable roles', () => {
    const list = assignableRoles([
      { name: 'tenant_admin', display_name: '管理员' },
      { name: 'group_admin', display_name: '组管' },
      { name: 'custom_x', display_name: '财务' },
    ])
    expect(list.map((r) => r.name)).toEqual(['tenant_admin', 'custom_x'])
  })

  it('PUT role_names replace-all', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({ role_names: ['a', 'b'] }),
    }))
    const result = await assignSubjectRoles({
      apiFetch,
      companyId: 'c1',
      subjectType: 'member',
      subjectId: 'm1',
      roleNames: ['a', 'b', 'a'],
    })
    expect(result.roleNames).toEqual(['a', 'b'])
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/tenant/member-role/company_id/c1/member_id/m1/',
      expect.objectContaining({ method: 'PUT' }),
    )
    const body = JSON.parse(apiFetch.mock.calls[0][1].body)
    expect(body.role_names).toEqual(['a', 'b'])
  })
})
