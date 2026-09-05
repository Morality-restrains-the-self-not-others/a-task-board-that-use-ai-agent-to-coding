// @vitest-environment node
import { describe, it, expect } from 'vitest'
import {
  resolveAlreadyLoggedInDestination,
  WORK_PANEL_LABEL,
  SYSTEM_ADMIN_LABEL,
} from './alreadyLoggedInDestination.js'

describe('resolveAlreadyLoggedInDestination (OPT-20260810-044)', () => {
  it('同源业务 next 原样返回', () => {
    expect(resolveAlreadyLoggedInDestination('/tenant/123/work-panel/?workspace_id=ws1')).toEqual({
      href: '/tenant/123/work-panel/?workspace_id=ws1',
      label: '/tenant/123/work-panel/?workspace_id=ws1',
    })
  })

  it('next 数组取首元素', () => {
    expect(resolveAlreadyLoggedInDestination(['/projects/'])).toEqual({
      href: '/projects/',
      label: '/projects/',
    })
  })

  it('外站 next（// 或带协议）被拒绝，落入工作面板', () => {
    const userInfo = { companies: [{ id: 't1' }] }
    expect(resolveAlreadyLoggedInDestination('//evil.com/x', userInfo)).toEqual({
      href: '/tenant/t1/work-panel/',
      label: WORK_PANEL_LABEL,
    })
    expect(resolveAlreadyLoggedInDestination('https://evil.com/x', userInfo)).toEqual({
      href: '/tenant/t1/work-panel/',
      label: WORK_PANEL_LABEL,
    })
  })

  it('OIDC 回跳 next（authorize 路径）原样保留', () => {
    const oidcNext = '/api/oidc/authorize?client_id=c&redirect_uri=r'
    expect(resolveAlreadyLoggedInDestination(oidcNext, { companies: [{ id: 't1' }] })).toEqual({
      href: oidcNext,
      label: oidcNext,
    })
  })

  it('无 next 时优先 current_company.id 定位工作面板', () => {
    expect(
      resolveAlreadyLoggedInDestination(null, {
        current_company: { id: 'c1' },
        companies: [{ id: 'other' }],
      }),
    ).toEqual({ href: '/tenant/c1/work-panel/', label: WORK_PANEL_LABEL })
  })

  it('无 next 且无 current_company 时兜底 companies[0].id', () => {
    expect(resolveAlreadyLoggedInDestination(undefined, { companies: [{ id: 'c2' }] })).toEqual({
      href: '/tenant/c2/work-panel/',
      label: WORK_PANEL_LABEL,
    })
  })

  it('空 next / 无公司时兜底系统管理（与登录成功默认一致）', () => {
    expect(resolveAlreadyLoggedInDestination('', { companies: [] })).toEqual({
      href: '/system-admin/',
      label: SYSTEM_ADMIN_LABEL,
    })
    expect(resolveAlreadyLoggedInDestination(null, {})).toEqual({
      href: '/system-admin/',
      label: SYSTEM_ADMIN_LABEL,
    })
  })
})
