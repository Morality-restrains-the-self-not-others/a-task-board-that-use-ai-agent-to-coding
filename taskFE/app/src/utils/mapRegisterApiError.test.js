import { describe, expect, it } from 'vitest'
import { mapRegisterApiError } from './mapRegisterApiError.js'

const TRACE = '07b8eec9-7eca-4e1c-8d57-b5cdb12ab02b'

describe('mapRegisterApiError — phone_register 通用业务错误须可见', () => {
  it('验证码无效或已过期不得只写入 email 字段（手机表单不渲染邮箱错误）', () => {
    const mapped = mapRegisterApiError(
      {
        error: '验证码无效或已过期',
        message: '验证码无效或已过期',
        status: 'error',
        trace_id: TRACE,
      },
      { registerType: 'phone', traceId: TRACE },
    )
    expect(mapped.submitError).toBe('验证码无效或已过期')
    expect(mapped.submitErrorTraceId).toBe(TRACE)
    expect(mapped.fieldErrors.email).toBe('')
    expect(mapped.fieldErrors.code).toBe('验证码无效或已过期')
    expect(mapped.inviteError).toBe('')
  })

  it('invite_code_invalid 走邀请码内联错误并带 traceId', () => {
    const mapped = mapRegisterApiError(
      { error: 'invite_code_invalid', trace_id: TRACE },
      { registerType: 'phone', traceId: TRACE },
    )
    expect(mapped.inviteError).toBe('邀请码无效')
    expect(mapped.inviteErrorTraceId).toBe(TRACE)
    expect(mapped.submitError).toBe('')
  })

  it('access_code_invalid 走邀请码内联错误（邀请链接无效）', () => {
    const mapped = mapRegisterApiError(
      { error: 'access_code_invalid', trace_id: TRACE },
      { registerType: 'phone', traceId: TRACE },
    )
    expect(mapped.inviteError).toBe('邀请链接无效，请联系分享者获取新的邀请链接')
    expect(mapped.inviteErrorTraceId).toBe(TRACE)
    expect(mapped.submitError).toBe('')
  })

  it('invite_gate_unavailable 走提交横幅错误（服务暂不可用）', () => {
    const mapped = mapRegisterApiError(
      { error: 'invite_gate_unavailable', trace_id: TRACE },
      { registerType: 'phone', traceId: TRACE },
    )
    expect(mapped.inviteError).toBe('邀请验证服务暂不可用，请稍后重试')
    expect(mapped.inviteErrorTraceId).toBe(TRACE)
    expect(mapped.submitError).toBe('')
  })

  it('字段级 code 错误映射到 fieldErrors.code', () => {
    const mapped = mapRegisterApiError(
      { code: ['验证码错误'] },
      { registerType: 'phone', traceId: TRACE },
    )
    expect(mapped.fieldErrors.code).toBe('验证码错误')
    expect(mapped.submitError).toBe('')
  })

  it('邮箱注册通用 error 也走 submitError，避免静默失败', () => {
    const mapped = mapRegisterApiError(
      { error: '该邮箱已被注册', message: '该邮箱已被注册', trace_id: TRACE },
      { registerType: 'email', traceId: TRACE },
    )
    expect(mapped.submitError).toBe('该邮箱已被注册')
    expect(mapped.submitErrorTraceId).toBe(TRACE)
  })

  it('已注册手机号 400 user_existed 展示请直接登录，不得当成注册成功', () => {
    const mapped = mapRegisterApiError(
      {
        error: '该手机号已被注册，请直接登录',
        user_existed: true,
        is_active: true,
        trace_id: TRACE,
      },
      { registerType: 'phone', traceId: TRACE },
    )
    expect(mapped.submitError).toBe('该手机号已被注册，请直接登录')
    expect(mapped.submitErrorTraceId).toBe(TRACE)
    expect(mapped.fieldErrors.email).toBe('')
  })
})
