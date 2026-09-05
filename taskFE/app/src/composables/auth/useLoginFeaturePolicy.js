import { ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'

export const EMAIL_PASSWORD_METHOD = 'emailPassword'
export const PHONE_PASSWORD_METHOD = 'phonePassword'

const LOGIN_FEATURE_POLICY_API = '/api/public/system-feature-policy/'

// 2026-08-24：手机号+验证码登录已整体移除（后端 /api/auth/ 拒绝 phone+code），
// 手机方式仅剩「手机号+密码」。
export const isPhoneLoginMethod = (method) => method === PHONE_PASSWORD_METHOD

export function useLoginFeaturePolicy({ loginMethod, phoneFields }) {
  const allowPhoneLogin = ref(false)
  const allowWechatLogin = ref(false)

  const normalizeAllowPhoneLogin = (payload) => {
    const rawPolicy = payload?.data ?? payload
    return Boolean(rawPolicy?.enable_phone_login)
  }

  // 后端计算的 wechat_login_available（策略开启 && 微信应用已配置）优先；
  // 旧后端无该字段时回退 enable_wechat_login（策略开关）。
  const normalizeAllowWechatLogin = (payload) => {
    const rawPolicy = payload?.data ?? payload
    return Boolean(rawPolicy?.wechat_login_available ?? rawPolicy?.enable_wechat_login)
  }

  const enforcePhonePolicy = () => {
    if (allowPhoneLogin.value || !isPhoneLoginMethod(loginMethod.value)) {
      return
    }
    loginMethod.value = EMAIL_PASSWORD_METHOD
    localStorage.setItem('preferredLoginMethod', EMAIL_PASSWORD_METHOD)
  }

  const loadPublicFeaturePolicy = async () => {
    try {
      const response = await apiFetch(LOGIN_FEATURE_POLICY_API, {
        method: 'GET',
        headers: { Accept: 'application/json' },
      })
      const payload = await response.json().catch(() => ({}))
      allowPhoneLogin.value = response.ok ? normalizeAllowPhoneLogin(payload) : false
      allowWechatLogin.value = response.ok ? normalizeAllowWechatLogin(payload) : false
    } catch {
      allowPhoneLogin.value = false
      allowWechatLogin.value = false
    } finally {
      enforcePhonePolicy()
    }
  }

  const handleMethodChanged = (method) => {
    if (!allowPhoneLogin.value && isPhoneLoginMethod(method)) {
      loginMethod.value = EMAIL_PASSWORD_METHOD
      localStorage.setItem('preferredLoginMethod', EMAIL_PASSWORD_METHOD)
      return
    }
    localStorage.setItem('preferredLoginMethod', method)
    phoneFields.resetPhoneFields()
  }

  return {
    allowPhoneLogin,
    allowWechatLogin,
    loadPublicFeaturePolicy,
    enforcePhonePolicy,
    handleMethodChanged,
  }
}
