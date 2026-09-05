import { ref, computed } from 'vue'
import { useAllowedCountryCodes } from './useAllowedCountryCodes.js'
import {
  DEFAULT_PHONE_COUNTRY_PREFIX,
  buildPhoneForApi,
  isValidE164Loose,
  maxNationalDigitsForPrefix,
  parsePasteToPrefixAndNational,
} from '../../utils/phoneInput.js'

export function useLoginPhoneFields() {
  const { filteredCountryOptions, defaultCountryPrefix, syncPrefixWithAllowedOptions, fetchAllowedCodes, loading: countryLoading, error: countryError } = useAllowedCountryCodes()
  // OPT-20260806-027: 默认前缀与后端白名单联动（白名单不含 +86 时取首个选项）
  const loginPhonePrefix = ref(defaultCountryPrefix.value)
  syncPrefixWithAllowedOptions(loginPhonePrefix)
  // 2026-08-24：手机号+验证码登录移除，仅保留密码方式的手机号字段
  const phoneNationalPassword = ref('')

  const loginPhonePasswordForApi = computed(() =>
    buildPhoneForApi(loginPhonePrefix.value, phoneNationalPassword.value),
  )

  const maxNationalLen = computed(() => maxNationalDigitsForPrefix(loginPhonePrefix.value))

  const nationalPlaceholder = computed(() =>
    loginPhonePrefix.value === '+86' || loginPhonePrefix.value === '86'
      ? '1×××××××××'
      : '本地号码（不含区号）',
  )

  const isPhoneValidForPassword = computed(() => isValidE164Loose(loginPhonePasswordForApi.value))

  function onLoginPhoneNationalPwdInput(e) {
    const maxL = maxNationalDigitsForPrefix(loginPhonePrefix.value)
    phoneNationalPassword.value = String(e?.target?.value ?? phoneNationalPassword.value ?? '')
      .replace(/\D/g, '')
      .slice(0, maxL)
  }

  function onLoginPhoneNationalPwdPaste(e) {
    const text = e.clipboardData?.getData('text') || ''
    const parsed = parsePasteToPrefixAndNational(text)
    if (parsed && parsed.national) {
      e.preventDefault()
      loginPhonePrefix.value = parsed.prefix
      const maxL = maxNationalDigitsForPrefix(parsed.prefix)
      phoneNationalPassword.value = parsed.national.slice(0, maxL)
    }
  }

  function resetPhoneFields() {
    loginPhonePrefix.value = defaultCountryPrefix.value
    phoneNationalPassword.value = ''
  }

  return {
    filteredCountryOptions,
    fetchAllowedCodes,
    countryLoading,
    countryError,
    loginPhonePrefix,
    phoneNationalPassword,
    loginPhonePasswordForApi,
    maxNationalLen,
    nationalPlaceholder,
    isPhoneValidForPassword,
    onLoginPhoneNationalPwdInput,
    onLoginPhoneNationalPwdPaste,
    resetPhoneFields,
  }
}
