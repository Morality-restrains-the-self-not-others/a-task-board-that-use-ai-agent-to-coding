import { ref, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'

const REGISTER_FEATURE_POLICY_API = '/api/public/system-feature-policy/'

/**
 * 注册页读取系统策略，控制邮箱注册入口可用性。
 * 策略加载失败时安全默认：关闭邮箱注册。
 */
export function useRegisterFeaturePolicy({ currentRegisterType }) {
  const allowEmailRegister = ref(false) // 邮箱注册已全局关闭，仅支持手机号注册
  const policyLoaded = ref(false)

  const normalizeAllowEmailRegister = (payload) => {
    const rawPolicy = payload?.data ?? payload
    return rawPolicy?.enable_email_register !== false
  }

  const enforceEmailRegisterPolicy = () => {
    if (allowEmailRegister.value) {
      return
    }
    if (currentRegisterType.value === 'email') {
      currentRegisterType.value = 'phone'
    }
  }

  const loadPublicFeaturePolicy = async () => {
    try {
      const response = await apiFetch(REGISTER_FEATURE_POLICY_API, {
        method: 'GET',
        headers: { Accept: 'application/json' },
      })
      const payload = await response.json().catch(() => ({}))
      allowEmailRegister.value = response.ok
        ? normalizeAllowEmailRegister(payload)
        : false
    } catch {
      allowEmailRegister.value = false
    } finally {
      policyLoaded.value = true
      enforceEmailRegisterPolicy()
    }
  }

  watch(allowEmailRegister, () => {
    enforceEmailRegisterPolicy()
  })

  return {
    allowEmailRegister,
    policyLoaded,
    loadPublicFeaturePolicy,
    enforceEmailRegisterPolicy,
  }
}
