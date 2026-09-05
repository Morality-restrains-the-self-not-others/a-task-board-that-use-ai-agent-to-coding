import { describe, it, expect } from 'vitest'
import {
  FEATURE_PARAMS_SUB_TOKEN_UI_ENABLED,
  resolveUseSubToken,
  applySubTokenUiPolicy,
} from './featureParamsSubTokenUi.js'

describe('featureParamsSubTokenUi', () => {
  it('派生子 Key 开关当前关闭', () => {
    expect(FEATURE_PARAMS_SUB_TOKEN_UI_ENABLED).toBe(false)
  })

  it('关闭时即使入参为 true 也不启用派生子 Key', () => {
    expect(resolveUseSubToken(true)).toBe(false)
    expect(resolveUseSubToken(false)).toBe(false)
  })

  it('关闭时把 provider 上的派生子 Key 与预算一并清掉', () => {
    const provider = { use_sub_token: true, budget_enabled: true }
    applySubTokenUiPolicy(provider)
    expect(provider.use_sub_token).toBe(false)
    expect(provider.budget_enabled).toBe(false)
  })
})
