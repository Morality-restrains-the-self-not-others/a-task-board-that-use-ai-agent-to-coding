// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { useAlreadyLoggedInPrompt } from './useAlreadyLoggedInPrompt.js'

const loggedInCurrentUser = {
  isAuthenticated: true,
  isSuperuser: false,
  userId: 'u1',
  username: 'alice',
  companies: [{ id: 't1', name: '租户一' }],
  current_company: { id: 't1' },
}

describe('useAlreadyLoggedInPrompt (OPT-20260810-044)', () => {
  it('未登录（window.currentUser 缺失 + /me/ 401）→ 不弹窗、不跳转', async () => {
    const modalConfirm = vi.fn()
    const navigate = vi.fn()
    const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt({
      getCurrentUser: () => undefined,
      fetchMe: async () => null,
      modalConfirm,
      navigate,
      getSearch: () => '',
    })
    expect(await maybePromptAlreadyLoggedIn()).toBe(false)
    expect(modalConfirm).not.toHaveBeenCalled()
    expect(navigate).not.toHaveBeenCalled()
  })

  it('window.currentUser 已登录 → 确认后跳转工作面板', async () => {
    const modalConfirm = vi.fn(async () => undefined)
    const navigate = vi.fn()
    const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt({
      getCurrentUser: () => loggedInCurrentUser,
      fetchMe: vi.fn(),
      modalConfirm,
      navigate,
      getSearch: () => '',
    })
    expect(await maybePromptAlreadyLoggedIn()).toBe(true)
    expect(modalConfirm).toHaveBeenCalledTimes(1)
    expect(modalConfirm.mock.calls[0][0]).toContain('工作面板')
    expect(navigate).toHaveBeenCalledWith('/tenant/t1/work-panel/')
  })

  it('/me/ 载荷（无 isAuthenticated，有 userId）→ 同样判定已登录', async () => {
    const modalConfirm = vi.fn(async () => undefined)
    const navigate = vi.fn()
    const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt({
      getCurrentUser: () => undefined,
      fetchMe: async () => ({ userId: 'u1', companies: [{ id: 't2' }] }),
      modalConfirm,
      navigate,
      getSearch: () => '',
    })
    expect(await maybePromptAlreadyLoggedIn()).toBe(true)
    expect(navigate).toHaveBeenCalledWith('/tenant/t2/work-panel/')
  })

  it('用户取消 → 留在登录页不跳转', async () => {
    const modalConfirm = vi.fn(async () => {
      throw new Error('cancel')
    })
    const navigate = vi.fn()
    const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt({
      getCurrentUser: () => loggedInCurrentUser,
      modalConfirm,
      navigate,
      getSearch: () => '',
    })
    expect(await maybePromptAlreadyLoggedIn()).toBe(false)
    expect(modalConfirm).toHaveBeenCalledTimes(1)
    expect(navigate).not.toHaveBeenCalled()
  })

  it('携带 next → 确认后跳转 next（非工作面板）', async () => {
    const modalConfirm = vi.fn(async () => undefined)
    const navigate = vi.fn()
    const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt({
      getCurrentUser: () => loggedInCurrentUser,
      modalConfirm,
      navigate,
      getSearch: () => '?next=/projects/',
    })
    expect(await maybePromptAlreadyLoggedIn()).toBe(true)
    expect(modalConfirm.mock.calls[0][0]).toContain('/projects/')
    expect(navigate).toHaveBeenCalledWith('/projects/')
  })

  it('进行中的微信回调参数 → 跳过弹窗（不打扰回调流程）', async () => {
    const modalConfirm = vi.fn()
    const navigate = vi.fn()
    const fetchMe = vi.fn()
    const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt({
      getCurrentUser: () => loggedInCurrentUser,
      fetchMe,
      modalConfirm,
      navigate,
      getSearch: () => '?wechat_token=abc',
    })
    expect(await maybePromptAlreadyLoggedIn()).toBe(false)
    expect(modalConfirm).not.toHaveBeenCalled()
    expect(navigate).not.toHaveBeenCalled()
  })

  it('进行中的 OIDC 回跳（code+state）→ 跳过弹窗', async () => {
    const modalConfirm = vi.fn()
    const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt({
      getCurrentUser: () => loggedInCurrentUser,
      modalConfirm,
      getSearch: () => '?code=c&state=s',
    })
    expect(await maybePromptAlreadyLoggedIn()).toBe(false)
    expect(modalConfirm).not.toHaveBeenCalled()
  })

  it('/me/ 请求异常 → 不弹窗不跳转（fail-closed）', async () => {
    const modalConfirm = vi.fn()
    const navigate = vi.fn()
    const { maybePromptAlreadyLoggedIn } = useAlreadyLoggedInPrompt({
      getCurrentUser: () => undefined,
      fetchMe: async () => {
        throw new Error('network')
      },
      modalConfirm,
      navigate,
      getSearch: () => '',
    })
    expect(await maybePromptAlreadyLoggedIn()).toBe(false)
    expect(modalConfirm).not.toHaveBeenCalled()
    expect(navigate).not.toHaveBeenCalled()
  })
})
