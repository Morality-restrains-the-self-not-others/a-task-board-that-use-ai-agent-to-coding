// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  canSeeMenuKey,
  canSeeSection,
  menuKeysToPermissionCodes,
  permissionCodesToMenuKeys,
  accessRoleDisplayName,
  isSystemTenantRole,
} from './tenantConsoleNav.js'

describe('tenantConsoleNav', () => {
  it('billing:view 可见 billing 段，不可见 people', () => {
    const has = (c) => c === 'billing:view'
    expect(canSeeMenuKey('billing.orders', has)).toBe(true)
    expect(canSeeSection('billing', has)).toBe(true)
    expect(canSeeMenuKey('people.invite', has)).toBe(false)
    expect(canSeeSection('people', has)).toBe(false)
  })

  it('member:manage 可见访问管理与邀请人', () => {
    const has = (c) => c === 'member:manage'
    expect(canSeeMenuKey('people.access', has)).toBe(true)
    expect(canSeeMenuKey('people.invite', has)).toBe(true)
    expect(canSeeMenuKey('people.groups', has)).toBe(false)
  })

  it('group-members:manage 可见管理分组', () => {
    expect(canSeeMenuKey('people.groups', (c) => c === 'group-members:manage')).toBe(true)
  })

  it('勾选菜单展开为权限码并集', () => {
    const codes = menuKeysToPermissionCodes(['nav.projects', 'billing.overview', 'people.invite'])
    expect(codes).toEqual(['billing:view', 'member:manage', 'project:view'].sort())
  })

  it('权限码反推菜单勾选', () => {
    const keys = permissionCodesToMenuKeys(['project:view', 'member:manage'])
    expect(keys).toContain('nav.projects')
    expect(keys).toContain('people.access')
    expect(keys).not.toContain('billing.overview')
  })

  it('系统角色识别与展示名', () => {
    expect(isSystemTenantRole('tenant_admin')).toBe(true)
    expect(isSystemTenantRole('custom_x')).toBe(false)
    expect(accessRoleDisplayName('member', 'Alice')).toBe('访问·Alice')
    expect(accessRoleDisplayName('group', '研发')).toBe('访问·组·研发')
  })

  it('page:* 优先于粗码决定侧栏可见', () => {
    const has = (c) => c === 'page:billing.overview'
    expect(canSeeMenuKey('billing.overview', has)).toBe(true)
    expect(canSeeMenuKey('nav.projects', has)).toBe(false)
  })

  it('nav.feedback 可见于 page 或 feedback:view', () => {
    expect(canSeeMenuKey('nav.feedback', (c) => c === 'page:nav.feedback')).toBe(true)
    expect(canSeeMenuKey('nav.feedback', (c) => c === 'feedback:view')).toBe(true)
    expect(canSeeMenuKey('nav.feedback', (c) => c === 'billing:view')).toBe(false)
    expect(canSeeSection('feedback', (c) => c === 'feedback:view')).toBe(true)
  })

  it('groupKeysToMenuKeys 从 region 推导 page', async () => {
    const { groupKeysToMenuKeys } = await import('./tenantConsoleNav.js')
    expect(groupKeysToMenuKeys(['nav.projects.main', 'people.access.save_actions'])).toEqual(
      expect.arrayContaining(['nav.projects', 'people.access']),
    )
  })

  it('normalizeResourceGroupKey 剥离 page:/region: 前缀', async () => {
    const { normalizeResourceGroupKey } = await import('./tenantConsoleNav.js')
    expect(normalizeResourceGroupKey('page:settings.cloud')).toBe('settings.cloud')
    expect(normalizeResourceGroupKey('region:settings.cloud.main')).toBe('settings.cloud.main')
    expect(normalizeResourceGroupKey('settings.cloud')).toBe('settings.cloud')
  })

  it('buildTenantResourceHref：整页无 hash，区域带 #rg= 深链', async () => {
    const { buildTenantResourceHref } = await import('./tenantConsoleNav.js')
    expect(buildTenantResourceHref('874', { groupKey: 'settings.cloud' })).toBe(
      '/tenant/874/settings/cloud-platform/',
    )
    expect(
      buildTenantResourceHref('874', {
        groupKey: 'settings.cloud.main',
        routePrefix: '/settings/cloud-platform/',
      }),
    ).toBe('/tenant/874/settings/cloud-platform/#rg=settings.cloud.main')
    expect(buildTenantResourceHref('874', { groupKey: 'page:people.access' })).toBe(
      '/tenant/874/people/access/',
    )
    expect(buildTenantResourceHref('', { groupKey: 'settings.cloud' })).toBe('')
    expect(buildTenantResourceHref('874', { groupKey: 'unknown.page' })).toBe('')
  })
})
