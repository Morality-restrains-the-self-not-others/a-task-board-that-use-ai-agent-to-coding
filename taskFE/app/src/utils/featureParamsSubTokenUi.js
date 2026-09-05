/**
 * 环境变量设置页「启用派生子Key」开关。
 * 暂未开放：改 true 即可恢复勾选与自动勾选。
 */
export const FEATURE_PARAMS_SUB_TOKEN_UI_ENABLED = false

export function resolveUseSubToken(rawValue) {
  return FEATURE_PARAMS_SUB_TOKEN_UI_ENABLED && Boolean(rawValue)
}

export function applySubTokenUiPolicy(provider) {
  if (!provider || typeof provider !== 'object') return provider
  if (!FEATURE_PARAMS_SUB_TOKEN_UI_ENABLED) {
    provider.use_sub_token = false
    provider.budget_enabled = false
  }
  return provider
}
