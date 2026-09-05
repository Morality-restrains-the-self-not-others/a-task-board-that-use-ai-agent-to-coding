import { describe, expect, it } from 'vitest'
import { resolveRuntimeStatusOnAuthError } from './serverRuntimeAuthAlign.js'

describe('resolveRuntimeStatusOnAuthError', () => {
  it('aligns to Running when start status is running and auth missing', () => {
    const r = resolveRuntimeStatusOnAuthError({
      isServerRunning: true,
      errMsg: '未找到云平台授权信息',
    })
    expect(r).toEqual({
      runtimeStatus: 'Running',
      message:
        '未找到云平台授权信息（启动状态显示已运行；请重新配置云平台授权以刷新实例详情）',
    })
  })

  it('returns null when server is not running', () => {
    expect(
      resolveRuntimeStatusOnAuthError({
        isServerRunning: false,
        errMsg: '未找到云平台授权信息',
      }),
    ).toBeNull()
  })

  it('returns null for unrelated errors', () => {
    expect(
      resolveRuntimeStatusOnAuthError({
        isServerRunning: true,
        errMsg: '查询服务器运行状态失败',
      }),
    ).toBeNull()
  })
})
