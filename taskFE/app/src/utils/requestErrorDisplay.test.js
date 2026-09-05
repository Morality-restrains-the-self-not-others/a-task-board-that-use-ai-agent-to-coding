// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  showRequestError,
  toastRequestError,
  isForwardAuthSessionExpired,
  redirectToLoginOnSessionExpired,
  humanizeRequestErrorMessage,
  isTaskDetailAccessCodeSharePage,
} from './requestErrorDisplay.js'
import modalService from './modalService.js'
import toastService from './toastService.js'

describe('requestErrorDisplay', () => {
  beforeEach(() => {
    modalService.close()
    toastService.hide()
  })

  it('showRequestError sets modalService.state.traceId from Error', async () => {
    const err = new Error('网络错误')
    err.traceId = 'modal-tid'
    const p = showRequestError(err)
    expect(modalService.state.show).toBe(true)
    expect(modalService.state.message).toBe('网络错误')
    expect(modalService.state.traceId).toBe('modal-tid')
    modalService.handleConfirm()
    await p
  })

  it('showRequestError does not put Chinese error message into data-traceId', async () => {
    const err = new Error('创建分组失败')
    const p = showRequestError(err.message, err)
    expect(modalService.state.message).toBe('创建分组失败')
    expect(modalService.state.traceId).toBe('')
    modalService.handleConfirm()
    await p
  })

  it('showRequestError keeps valid Response.traceId when message is Chinese', async () => {
    const err = new Error('创建分组失败')
    err.traceId = 'web-req-12345678'
    const p = showRequestError(err.message, err)
    expect(modalService.state.traceId).toBe('web-req-12345678')
    modalService.handleConfirm()
    await p
  })

  it('humanizeRequestErrorMessage 将动态导入 chunk 失败与裸 Failed to fetch 区分', () => {
    expect(
      humanizeRequestErrorMessage(
        'Failed to fetch dynamically imported module: https://www.daydaymoney.com/static/assets/gitOauthPushPrecheck-C15U_Qut.js',
      ),
    ).toBe('页面已更新，请刷新后重试')
  })

  it('humanizeRequestErrorMessage 将原始英文网络错误中文化（OPT-20260811-070）', () => {
    expect(humanizeRequestErrorMessage('Failed to fetch')).toBe('网络错误，请稍后重试')
    expect(humanizeRequestErrorMessage('NetworkError when attempting to fetch resource.')).toBe('网络错误，请稍后重试')
    expect(humanizeRequestErrorMessage('Load failed')).toBe('网络错误，请稍后重试')
    expect(humanizeRequestErrorMessage('The Internet connection appears to be offline.')).toBe('网络错误，请稍后重试')
  })

  it('humanizeRequestErrorMessage 将 git oauth not connected 中文化', () => {
    expect(humanizeRequestErrorMessage('git oauth not connected')).toBe(
      '尚未绑定 Git 网站 OAuth，请先在账号中心完成授权后再试',
    )
    expect(humanizeRequestErrorMessage('Git OAuth not connected')).toBe(
      '尚未绑定 Git 网站 OAuth，请先在账号中心完成授权后再试',
    )
  })

  it('humanizeRequestErrorMessage 将 gitlab refresh http 400 转为重新绑定引导', () => {
    expect(humanizeRequestErrorMessage('gitlab refresh http 400')).toBe(
      'Git OAuth 授权已失效，请重新绑定后再试',
    )
    expect(humanizeRequestErrorMessage('无法获取 Git 访问令牌：gitlab refresh http 400')).toBe(
      'Git OAuth 授权已失效，请重新绑定后再试',
    )
  })

  it('humanizeRequestErrorMessage 保留业务中文错误并处理空值', () => {
    expect(humanizeRequestErrorMessage('创建分组失败')).toBe('创建分组失败')
    expect(humanizeRequestErrorMessage('HTTP 403 权限不足')).toBe('HTTP 403 权限不足')
    expect(humanizeRequestErrorMessage(undefined)).toBe('请求失败，请稍后重试')
    expect(humanizeRequestErrorMessage('')).toBe('请求失败，请稍后重试')
  })

  it('showRequestError 对裸 Failed to fetch 中文化', async () => {
    const err = new Error('Failed to fetch')
    const p = showRequestError(err)
    expect(modalService.state.message).toBe('网络错误，请稍后重试')
    modalService.handleConfirm()
    await p
  })

  it('isForwardAuthSessionExpired matches the canonical forward-auth 401 detail', () => {
    expect(isForwardAuthSessionExpired({ detail: '无法解析登录凭据，请重新登录' })).toBe(true)
    expect(isForwardAuthSessionExpired({ detail: '登录状态无效或已失效，请重新登录' })).toBe(false)
    expect(isForwardAuthSessionExpired({ detail: '账号不可用或已归档，请联系管理员' })).toBe(false)
    expect(isForwardAuthSessionExpired({})).toBe(false)
    expect(isForwardAuthSessionExpired(null)).toBe(false)
    expect(isForwardAuthSessionExpired(undefined, { _errorData: { detail: '无法解析登录凭据，请重新登录' } })).toBe(true)
  })

  it('toastRequestError sets toastService.state.traceId', () => {
    vi.useFakeTimers()
    const err = new Error('fail')
    err.traceId = 'toast-tid'
    toastRequestError(err)
    expect(toastService.state.show).toBe(true)
    expect(toastService.state.type).toBe('error')
    expect(toastService.state.traceId).toBe('toast-tid')
    vi.runAllTimers()
    vi.useRealTimers()
  })

  it('redirectToLoginOnSessionExpired carries the next param', () => {
    vi.stubGlobal('location', { pathname: '/system-admin/', search: '', href: '' })
    redirectToLoginOnSessionExpired('/system-admin/')
    expect(window.location.href).toBe('/auth/login/?next=%2Fsystem-admin%2F')
  })
})

describe('handleForwardAuthSessionExpired（apiFetch 统一收口）', () => {
  let handle
  beforeEach(async () => {
    // resetModules + 动态导入：隔离模块级「已跳转」标记，保证测试顺序无关
    vi.resetModules()
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    const mod = await import('./requestErrorDisplay.js')
    handle = mod.handleForwardAuthSessionExpired
  })

  it('命中即跳转登录并携带当前页作为 next；重复命中只触发一次', () => {
    vi.stubGlobal('location', { pathname: '/system-admin/', search: '', href: '' })
    expect(handle()).toBe(true)
    expect(window.location.href).toBe('/auth/login/?next=%2Fsystem-admin%2F')
    // 并发多个 401：第二次命中不再重复改写 location
    expect(handle()).toBe(false)
    expect(window.location.href).toBe('/auth/login/?next=%2Fsystem-admin%2F')
  })

  it('已处于 /auth/* 公开页时不触发（防循环跳转）', () => {
    vi.stubGlobal('location', { pathname: '/auth/login/', search: '', href: '' })
    expect(handle()).toBe(false)
    expect(window.location.href).toBe('')
  })

  it('Git OAuth 回调落地页（/redirect/gitsite/*）不触发：换票会话无关，跳登录会中断授权', () => {
    vi.stubGlobal('location', {
      pathname: '/redirect/gitsite/github.com/oauth/callback/',
      search: '?code=abc&state=v1.eyJ2Ijox',
      href: '',
    })
    expect(handle()).toBe(false)
    expect(window.location.href).toBe('')
  })

  it('Git OAuth 回调落地页带错误参数也不触发（exchange_rejected 等由回调页自行处理）', () => {
    vi.stubGlobal('location', {
      pathname: '/redirect/gitsite/gitlab.daydaymoney.com/oauth/callback/',
      search: '?error=access_denied&state=v1.eyJ2Ijox',
      href: '',
    })
    expect(handle()).toBe(false)
    expect(window.location.href).toBe('')
  })

  it('公网 FAQ 页不触发（Navbar /me 401 不得挡住文档）', () => {
    vi.stubGlobal('location', { pathname: '/faq/', search: '', href: '' })
    expect(handle()).toBe(false)
    expect(window.location.href).toBe('')
  })

  it('公网首页与定价页不触发', () => {
    vi.stubGlobal('location', { pathname: '/', search: '', href: '' })
    expect(handle()).toBe(false)
    vi.stubGlobal('location', { pathname: '/pricing', search: '', href: '' })
    expect(handle()).toBe(false)
    expect(window.location.href).toBe('')
  })

  it('任务详情 accessCode 分享页不触发（访客绑定探测 401 不得踢登录）', () => {
    vi.stubGlobal('location', {
      pathname: '/tenant/877397588196749312/workspace/ws_-1/task-detail/task_878932440129761280/',
      search: '?accessCode=DR2AKvP9J9&gitlab=ok',
      href: '',
    })
    expect(handle()).toBe(false)
    expect(window.location.href).toBe('')
  })

  it('工作面板带 accessCode 仍跳登录（分享码不开放非 task-detail 页）', () => {
    vi.stubGlobal('location', {
      pathname: '/tenant/877397588196749312/work-panel/',
      search: '?accessCode=DR2AKvP9J9',
      href: '',
    })
    expect(handle()).toBe(true)
    expect(window.location.href).toContain('/auth/login/')
  })

  it('next 缺省时取当前 pathname+search', () => {
    vi.stubGlobal('location', { pathname: '/user/me/profile/', search: '?tab=1', href: '' })
    expect(handle()).toBe(true)
    expect(window.location.href).toBe('/auth/login/?next=%2Fuser%2Fme%2Fprofile%2F%3Ftab%3D1')
  })
})

describe('isTaskDetailAccessCodeSharePage', () => {
  it('task-detail + 非空 accessCode 为分享页', () => {
    expect(isTaskDetailAccessCodeSharePage(
      '/tenant/1/workspace/w/task-detail/t1/',
      '?accessCode=DR2AKvP9J9',
    )).toBe(true)
  })

  it('无 accessCode 或空值不是分享页', () => {
    expect(isTaskDetailAccessCodeSharePage('/tenant/1/workspace/w/task-detail/t1/', '')).toBe(false)
    expect(isTaskDetailAccessCodeSharePage('/tenant/1/workspace/w/task-detail/t1/', '?accessCode=')).toBe(false)
  })

  it('非 task-detail 即使有 accessCode 也不是分享页', () => {
    expect(isTaskDetailAccessCodeSharePage('/tenant/1/work-panel/', '?accessCode=DR2AKvP9J9')).toBe(false)
  })
})
