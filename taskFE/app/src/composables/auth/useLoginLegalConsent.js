import { ref, computed } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'

export function useLoginLegalConsent() {
  const currentPrivacyPolicy = ref(null)
  const privacyLoadError = ref('')
  const acceptPrivacy = ref(false)
  const showPrivacyReader = ref(false)
  const currentLicenseAgreement = ref(null)
  const licenseLoadError = ref('')
  const acceptLicense = ref(false)
  const showLicenseReader = ref(false)

  const acceptAll = computed({
    get: () => acceptPrivacy.value && acceptLicense.value,
    set: (val) => {
      acceptPrivacy.value = val
      acceptLicense.value = val
    },
  })

  const loadCurrentPrivacy = async () => {
    try {
      const r = await apiFetch('/api/privacy-policy/public/current/', {
        method: 'GET',
        headers: { Accept: 'application/json' },
      })
      if (r.ok) {
        currentPrivacyPolicy.value = await r.json()
        privacyLoadError.value = ''
      } else {
        currentPrivacyPolicy.value = null
        privacyLoadError.value = '系统尚未发布隐私条款，请联系管理员后再登录。'
      }
    } catch {
      currentPrivacyPolicy.value = null
      privacyLoadError.value = '无法加载隐私条款，请刷新重试。'
    }
  }

  const loadCurrentLicenseAgreement = async () => {
    try {
      const r = await apiFetch('/api/license-agreement/public/current/', {
        method: 'GET',
        headers: { Accept: 'application/json' },
      })
      if (r.ok) {
        currentLicenseAgreement.value = await r.json()
        licenseLoadError.value = ''
      } else {
        currentLicenseAgreement.value = null
        licenseLoadError.value = '系统尚未发布服务协议，请联系管理员后再登录。'
      }
    } catch {
      currentLicenseAgreement.value = null
      licenseLoadError.value = '无法加载服务协议，请刷新重试。'
    }
  }

  return {
    currentPrivacyPolicy,
    privacyLoadError,
    acceptPrivacy,
    showPrivacyReader,
    currentLicenseAgreement,
    licenseLoadError,
    acceptLicense,
    showLicenseReader,
    acceptAll,
    loadCurrentPrivacy,
    loadCurrentLicenseAgreement,
  }
}
