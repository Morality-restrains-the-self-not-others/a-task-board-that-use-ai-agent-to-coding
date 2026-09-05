// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { listOrphanCustomAccessRoles } from './saveSubjectResourceAccess.js'

describe('listOrphanCustomAccessRoles', () => {
  it('返回未被成员/小组引用的自定义角色，优先 访问·', () => {
    const orphans = listOrphanCustomAccessRoles({
      roles: [
        { id: '1', name: 'member', display_name: '成员', is_system: true },
        { id: '2', name: 'custom_bound', display_name: '访问·Bob', is_system: false },
        { id: '3', name: 'custom_orphan_access', display_name: '访问·Alice', is_system: false },
        { id: '4', name: 'custom_orphan_other', display_name: '临时角色', is_system: false },
      ],
      memberRoles: [{ member_id: 'm1', role_name: 'custom_bound' }],
      groupRoles: [],
    })
    expect(orphans.map((r) => r.name)).toEqual(['custom_orphan_access', 'custom_orphan_other'])
  })

  it('系统角色名不会被当成孤儿', () => {
    const orphans = listOrphanCustomAccessRoles({
      roles: [{ id: '1', name: 'tenant_admin', display_name: '管理员', is_system: false }],
      memberRoles: [],
      groupRoles: [],
    })
    expect(orphans).toEqual([])
  })
})
