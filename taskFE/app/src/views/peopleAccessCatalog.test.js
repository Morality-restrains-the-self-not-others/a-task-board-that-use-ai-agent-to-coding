// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  filterCatalogPages,
  flattenCatalogGroupKeys,
  hasPeopleAccessWrite,
  omitPeopleAccessSaveActionsRegion,
  resolveSubjectSelectedGroupKeys,
} from './peopleAccessCatalog.js'

const sampleCatalog = [
  {
    group_key: 'nav.projects',
    display_name: '项目列表',
    children: [
      { group_key: 'nav.projects.main', display_name: '主区域' },
    ],
  },
  {
    group_key: 'people.access',
    display_name: '访问管理',
    children: [
      { group_key: 'people.access.subject_list', display_name: '主体列表' },
      { group_key: 'people.access.region_matrix', display_name: '资源区域' },
      { group_key: 'people.access.save_actions', display_name: '勾选保存' },
    ],
  },
]

describe('peopleAccessCatalog', () => {
  it('flattenCatalogGroupKeys 展开 page + 全部 ui_region', () => {
    expect(flattenCatalogGroupKeys(sampleCatalog).sort()).toEqual(
      [
        'nav.projects',
        'nav.projects.main',
        'people.access',
        'people.access.region_matrix',
        'people.access.save_actions',
        'people.access.subject_list',
      ].sort(),
    )
  })

  it('tenant_admin 选中时应勾选目录全部 page/region（租户全权限展示）', () => {
    const keys = resolveSubjectSelectedGroupKeys({
      roleName: 'tenant_admin',
      role: { name: 'tenant_admin', is_system: true, permissions: ['project:view'] },
      boundResourceGroups: [],
      catalogPages: sampleCatalog,
    })
    expect(keys.sort()).toEqual(flattenCatalogGroupKeys(sampleCatalog).sort())
    // 不得只回落到粗码→菜单（否则 region 全未勾选）
    expect(keys).toContain('people.access.save_actions')
    expect(keys).toContain('nav.projects.main')
  })

  it('自定义角色优先使用资源组绑定', () => {
    const keys = resolveSubjectSelectedGroupKeys({
      roleName: '访问·Alice',
      role: { name: '访问·Alice', is_system: false, permissions: [] },
      boundResourceGroups: [{ group_key: 'people.access.save_actions', kind: 'ui_region' }],
      catalogPages: sampleCatalog,
    })
    expect(keys).toEqual(['people.access.save_actions'])
  })

  it('无绑定的普通角色回退粗码→菜单 page keys', () => {
    const keys = resolveSubjectSelectedGroupKeys({
      roleName: 'member',
      role: { name: 'member', is_system: true, permissions: ['project:view', 'member:manage'] },
      boundResourceGroups: [],
      catalogPages: sampleCatalog,
    })
    expect(keys).toContain('nav.projects')
    expect(keys).toContain('people.access')
    expect(keys).not.toContain('people.access.save_actions')
  })

  it('filterCatalogPages 按显示名过滤', () => {
    expect(filterCatalogPages(sampleCatalog, '访问').map((p) => p.group_key)).toEqual([
      'people.access',
    ])
  })

  it('filterCatalogPages 页面名命中时保留整页全部 children', () => {
    const out = filterCatalogPages(sampleCatalog, '访问管理')
    expect(out).toHaveLength(1)
    expect(out[0].group_key).toBe('people.access')
    expect(out[0].children).toHaveLength(2)
    expect(out[0].children.map((c) => c.group_key)).not.toContain('people.access.save_actions')
  })

  it('filterCatalogPages 页面名未命中但子区域命中时仅保留匹配 children', () => {
    const out = filterCatalogPages(sampleCatalog, '资源区域')
    expect(out).toHaveLength(1)
    expect(out[0].group_key).toBe('people.access')
    expect(out[0].children.map((c) => c.group_key)).toEqual(['people.access.region_matrix'])
  })

  it('filterCatalogPages 无命中时返回空数组', () => {
    expect(filterCatalogPages(sampleCatalog, '不存在的关键词')).toEqual([])
  })

  it('isPageFullySelected / togglePageGroupKeys 整页勾选联动 region', async () => {
    const { isPageFullySelected, togglePageGroupKeys } = await import('./peopleAccessCatalog.js')
    const page = sampleCatalog[1]
    expect(isPageFullySelected(['people.access'], page)).toBe(false)
    const all = togglePageGroupKeys([], page, true)
    expect(isPageFullySelected(all, page)).toBe(true)
    expect(togglePageGroupKeys(all, page, false)).toEqual([])
  })

  it('omitPeopleAccessSaveActionsRegion 去掉独立 save_actions 子区域', () => {
    const out = omitPeopleAccessSaveActionsRegion(sampleCatalog)
    const access = out.find((p) => p.group_key === 'people.access')
    expect(access.children.map((c) => c.group_key)).toEqual([
      'people.access.subject_list',
      'people.access.region_matrix',
    ])
  })

  it('hasPeopleAccessWrite：matrix/list operate 或存量 save_actions 或 member:manage', () => {
    const stub = (codes) => ({
      hasRegionOperate: (_cid, key) => codes.includes(`region:${key}:operate`),
      hasPerm: (_cid, code) => codes.includes(code),
    })
    expect(hasPeopleAccessWrite(stub(['region:people.access.subject_list:operate']), 't1')).toBe(true)
    expect(hasPeopleAccessWrite(stub(['region:people.access.region_matrix:operate']), 't1')).toBe(true)
    expect(hasPeopleAccessWrite(stub(['region:people.access.save_actions:operate']), 't1')).toBe(true)
    expect(hasPeopleAccessWrite(stub(['member:manage']), 't1')).toBe(true)
    expect(hasPeopleAccessWrite(stub(['region:people.access.subject_list:view']), 't1')).toBe(false)
  })
})
