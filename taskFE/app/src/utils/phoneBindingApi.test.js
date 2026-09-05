import { describe, expect, it } from 'vitest'
import {
  PROFILE_BIND_PHONE_PATH,
  PROFILE_REPLACE_PHONE_PATH,
  PHONE_AMBIGUOUS_ERROR_CODE,
  bindPhoneRequestBody,
  isPhoneBindLimitError,
  isPhoneTakenBindError,
  shouldNavigateAfterPhoneBind,
} from './phoneBindingApi.js'

describe('phoneBindingApi', () => {
  it('资料页绑定/换绑走 profile 子路径（非 bind_phone 别名）', () => {
    expect(PROFILE_BIND_PHONE_PATH).toBe('/api/accounts/users/profile/bind-phone/')
    expect(PROFILE_REPLACE_PHONE_PATH).toBe('/api/accounts/users/profile/replace-phone/')
  })

  it('假成功 {ok:true} 不得触发绑定后回跳', () => {
    expect(shouldNavigateAfterPhoneBind(true, { ok: true })).toBe(false)
    expect(shouldNavigateAfterPhoneBind(true, {})).toBe(false)
    expect(shouldNavigateAfterPhoneBind(false, { bound: true })).toBe(false)
  })

  it('仅 bound:true 才回跳', () => {
    expect(shouldNavigateAfterPhoneBind(true, { bound: true })).toBe(true)
  })

  it('识别占用 409 可转移', () => {
    expect(isPhoneTakenBindError({ code: 'phone_taken', reclaim_available: true })).toBe(true)
    expect(isPhoneTakenBindError({ error: '该手机号已绑定其他账号' })).toBe(false)
    expect(isPhoneTakenBindError(null)).toBe(false)
  })

  it('绑定上限不是转移占用', () => {
    expect(isPhoneBindLimitError({ code: 'phone_bind_limit', reclaim_available: false })).toBe(true)
    expect(isPhoneTakenBindError({ code: 'phone_bind_limit', reclaim_available: false })).toBe(false)
    expect(isPhoneTakenBindError({ code: 'phone_bind_limit', reclaim_available: true })).toBe(false)
  })

  it('登录消歧错误码稳定', () => {
    expect(PHONE_AMBIGUOUS_ERROR_CODE).toBe('phone_ambiguous')
  })

  it('reclaim 请求体只在确认转移时带 reclaim', () => {
    expect(bindPhoneRequestBody({ phone: '+8613800138000', code: '123456' })).toEqual({
      phone: '+8613800138000',
      code: '123456',
    })
    expect(bindPhoneRequestBody({ phone: '+8613800138000', code: '123456', reclaim: true })).toEqual({
      phone: '+8613800138000',
      code: '123456',
      reclaim: true,
    })
  })
})
