/** 个人资料页手机绑定/换绑表单逻辑（从面板拆出以满足行数门禁）。 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { safeResponseJson } from '../../utils/safeResponseJson.js'
import { humanizeRequestErrorMessage } from '../../utils/requestErrorDisplay.js'
import { maybeNavigateToPhoneVerifyRedirect } from '../../utils/phoneBindingDeepLink.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../../utils/clickGuard.js'
import {
  PROFILE_BIND_PHONE_PATH,
  PROFILE_REPLACE_PHONE_PATH,
  bindPhoneRequestBody,
  isPhoneTakenBindError,
  isPhoneBindLimitError,
  shouldNavigateAfterPhoneBind,
} from '../../utils/phoneBindingApi.js'
import { useAllowedCountryCodes } from './useAllowedCountryCodes.js'
import {
  buildPhoneForApi,
  isValidE164Loose,
  maxNationalDigitsForPrefix,
  parsePasteToPrefixAndNational,
} from '../../utils/phoneInput.js'


export function useUserProfilePhoneBindingPanel(emit) {
const sendReplaceSmsGuard = createClickGuard()
const submitReplaceGuard = createClickGuard()
const sendBindSmsGuard = createClickGuard()
const submitBindGuard = createClickGuard()

const { filteredCountryOptions, defaultCountryPrefix, syncPrefixWithAllowedOptions, fetchAllowedCodes, loading: countryLoading, error: countryError } = useAllowedCountryCodes()
// OPT-20260806-027: 默认前缀与后端白名单联动（白名单不含 +86 时取首个选项）
const replacePhonePrefix = ref(defaultCountryPrefix.value)
syncPrefixWithAllowedOptions(replacePhonePrefix)
const replacePhoneNational = ref('')
const replaceSmsCode = ref('')
const replaceSmsCountdown = ref(0)
let replaceSmsTimer = null
const isSendingReplaceSms = ref(false)
const isReplacingPhone = ref(false)
const replaceInlineError = ref('')
const replaceInlineErrorTraceId = ref('')
const replaceInlineMessage = ref('')
const replaceReclaimAvailable = ref(false)

const replacePhoneForApi = computed(() =>
  buildPhoneForApi(replacePhonePrefix.value, replacePhoneNational.value),
)
const isReplacePhoneValid = computed(() => isValidE164Loose(replacePhoneForApi.value))
const replaceNationalMaxLen = computed(() => maxNationalDigitsForPrefix(replacePhonePrefix.value))
const replaceNationalPlaceholder = computed(() =>
  replacePhonePrefix.value === '+86' || replacePhonePrefix.value === '86'
    ? '1×××××××××'
    : '本地号码',
)

function onReplacePhoneNationalInput(e) {
  const maxL = maxNationalDigitsForPrefix(replacePhonePrefix.value)
  replacePhoneNational.value = String(e?.target?.value ?? replacePhoneNational.value ?? '')
    .replace(/\D/g, '')
    .slice(0, maxL)
}

function onReplacePhoneNationalPaste(e) {
  const text = e.clipboardData?.getData('text') || ''
  const parsed = parsePasteToPrefixAndNational(text)
  if (parsed && parsed.national) {
    e.preventDefault()
    replacePhonePrefix.value = parsed.prefix
    const maxL = maxNationalDigitsForPrefix(parsed.prefix)
    replacePhoneNational.value = parsed.national.slice(0, maxL)
  }
}

function startReplaceSmsCountdown() {
  replaceSmsCountdown.value = 60
  if (replaceSmsTimer) clearInterval(replaceSmsTimer)
  replaceSmsTimer = setInterval(() => {
    replaceSmsCountdown.value -= 1
    if (replaceSmsCountdown.value <= 0) {
      clearInterval(replaceSmsTimer)
      replaceSmsTimer = null
    }
  }, 1000)
}

const sendReplaceSms = async () => {
  replaceInlineError.value = ''
  replaceInlineErrorTraceId.value = ''
  replaceInlineMessage.value = ''
  replaceReclaimAvailable.value = false
  if (!isReplacePhoneValid.value) {
    replaceInlineError.value = '请输入有效手机号（含国家/地区码）'
    return
  }
  await sendReplaceSmsGuard.run(async ({ idempotencyKey }) => {
    isSendingReplaceSms.value = true
    try {
      const response = await apiFetch('/api/accounts/users/send_verification_code/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ phone: replacePhoneForApi.value }),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        const raw = data?.phone?.[0] || data?.detail || data?.error || data?.message
        const msg = typeof raw === 'string' ? raw : (raw ? String(raw) : '') || '发送失败'
        replaceInlineErrorTraceId.value = traceId || ''
        replaceInlineError.value = humanizeRequestErrorMessage(msg)
        return
      }
      replaceInlineMessage.value = '验证码已发送'
      startReplaceSmsCountdown()
    } catch (error) {
      console.error('发送更换手机号验证码失败:', error)
      replaceInlineErrorTraceId.value = ''
      replaceInlineError.value = humanizeRequestErrorMessage(error.message || '发送失败')
    } finally {
      isSendingReplaceSms.value = false
    }
  })
}

const submitReplacePhone = async (reclaim = false) => {
  replaceInlineError.value = ''
  replaceInlineErrorTraceId.value = ''
  replaceInlineMessage.value = ''
  replaceReclaimAvailable.value = false
  emit('error', '')
  emit('message', '')
  if (!isReplacePhoneValid.value) {
    replaceInlineError.value = '请输入有效手机号'
    return
  }
  if (replaceSmsCode.value.trim().length !== 6) {
    replaceInlineError.value = '请输入 6 位验证码'
    return
  }
  await submitReplaceGuard.run(async ({ idempotencyKey }) => {
    isReplacingPhone.value = true
    try {
      const response = await apiFetch(PROFILE_REPLACE_PHONE_PATH, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify(bindPhoneRequestBody({
          phone: replacePhoneForApi.value,
          code: replaceSmsCode.value.trim(),
          reclaim,
        })),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        const errText = typeof data?.detail === 'string' ? data.detail : (typeof data?.error === 'string' ? data.error : '')
        replaceInlineErrorTraceId.value = traceId || ''
        replaceInlineError.value = humanizeRequestErrorMessage(errText || '更换失败')
        replaceReclaimAvailable.value = isPhoneTakenBindError(data) && !isPhoneBindLimitError(data)
        return
      }
      if (!shouldNavigateAfterPhoneBind(response.ok, data)) {
        replaceInlineError.value = '绑定未确认，请刷新后重试'
        return
      }
      emit('profile-updated', data)
      if (maybeNavigateToPhoneVerifyRedirect()) {
        return
      }
      replaceSmsCode.value = ''
      replacePhoneNational.value = ''
      emit('message', '已成功更换绑定手机号')
    } catch (error) {
      console.error('更换手机号失败:', error)
      replaceInlineErrorTraceId.value = ''
      replaceInlineError.value = humanizeRequestErrorMessage(error.message || '更换失败')
    } finally {
      isReplacingPhone.value = false
    }
  })
}

// ---- 首次绑定手机号 ----
// OPT-20260806-027: 默认前缀与后端白名单联动（白名单不含 +86 时取首个选项）
const bindPhonePrefix = ref(defaultCountryPrefix.value)
syncPrefixWithAllowedOptions(bindPhonePrefix)
const bindPhoneNational = ref('')
const bindSmsCode = ref('')
const bindSmsCountdown = ref(0)
let bindSmsTimer = null
const isSendingBindSms = ref(false)
const isBindingPhone = ref(false)
const bindInlineError = ref('')
const bindInlineErrorTraceId = ref('')
const bindInlineMessage = ref('')
const bindReclaimAvailable = ref(false)

const bindPhoneForApi = computed(() =>
  buildPhoneForApi(bindPhonePrefix.value, bindPhoneNational.value),
)
const isBindPhoneValid = computed(() => isValidE164Loose(bindPhoneForApi.value))
const bindNationalMaxLen = computed(() => maxNationalDigitsForPrefix(bindPhonePrefix.value))
const bindNationalPlaceholder = computed(() =>
  bindPhonePrefix.value === '+86' || bindPhonePrefix.value === '86'
    ? '1×××××××××'
    : '本地号码',
)

function onBindPhoneNationalInput(e) {
  const maxL = maxNationalDigitsForPrefix(bindPhonePrefix.value)
  bindPhoneNational.value = String(e?.target?.value ?? bindPhoneNational.value ?? '')
    .replace(/\D/g, '')
    .slice(0, maxL)
}

function onBindPhoneNationalPaste(e) {
  const text = e.clipboardData?.getData('text') || ''
  const parsed = parsePasteToPrefixAndNational(text)
  if (parsed && parsed.national) {
    e.preventDefault()
    bindPhonePrefix.value = parsed.prefix
    const maxL = maxNationalDigitsForPrefix(parsed.prefix)
    bindPhoneNational.value = parsed.national.slice(0, maxL)
  }
}

function startBindSmsCountdown() {
  bindSmsCountdown.value = 60
  if (bindSmsTimer) clearInterval(bindSmsTimer)
  bindSmsTimer = setInterval(() => {
    bindSmsCountdown.value -= 1
    if (bindSmsCountdown.value <= 0) {
      clearInterval(bindSmsTimer)
      bindSmsTimer = null
    }
  }, 1000)
}

const sendBindSms = async () => {
  bindInlineError.value = ''
  bindInlineErrorTraceId.value = ''
  bindInlineMessage.value = ''
  bindReclaimAvailable.value = false
  if (!isBindPhoneValid.value) {
    bindInlineError.value = '请输入有效手机号（含国家/地区码）'
    return
  }
  await sendBindSmsGuard.run(async ({ idempotencyKey }) => {
    isSendingBindSms.value = true
    try {
      const response = await apiFetch('/api/accounts/users/send_verification_code/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ phone: bindPhoneForApi.value }),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        const raw = data?.phone?.[0] || data?.detail || data?.error || data?.message
        const msg = typeof raw === 'string' ? raw : (raw ? String(raw) : '') || '发送失败'
        bindInlineErrorTraceId.value = traceId || ''
        bindInlineError.value = humanizeRequestErrorMessage(msg)
        return
      }
      bindInlineMessage.value = '验证码已发送'
      startBindSmsCountdown()
    } catch (error) {
      console.error('发送绑定手机号验证码失败:', error)
      bindInlineErrorTraceId.value = ''
      bindInlineError.value = humanizeRequestErrorMessage(error.message || '发送失败')
    } finally {
      isSendingBindSms.value = false
    }
  })
}

const submitBindPhone = async (reclaim = false) => {
  bindInlineError.value = ''
  bindInlineErrorTraceId.value = ''
  bindInlineMessage.value = ''
  bindReclaimAvailable.value = false
  emit('error', '')
  emit('message', '')
  if (!isBindPhoneValid.value) {
    bindInlineError.value = '请输入有效手机号'
    return
  }
  if (bindSmsCode.value.trim().length !== 6) {
    bindInlineError.value = '请输入 6 位验证码'
    return
  }
  await submitBindGuard.run(async ({ idempotencyKey }) => {
    isBindingPhone.value = true
    try {
      const response = await apiFetch(PROFILE_BIND_PHONE_PATH, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify(bindPhoneRequestBody({
          phone: bindPhoneForApi.value,
          code: bindSmsCode.value.trim(),
          reclaim,
        })),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        const errText = typeof data?.detail === 'string' ? data.detail : (typeof data?.error === 'string' ? data.error : '')
        bindInlineErrorTraceId.value = traceId || ''
        bindInlineError.value = humanizeRequestErrorMessage(errText || '绑定失败')
        bindReclaimAvailable.value = isPhoneTakenBindError(data) && !isPhoneBindLimitError(data)
        return
      }
      if (!shouldNavigateAfterPhoneBind(response.ok, data)) {
        bindInlineError.value = '绑定未确认，请刷新后重试'
        return
      }
      emit('profile-updated', data)
      if (maybeNavigateToPhoneVerifyRedirect()) {
        return
      }
      bindSmsCode.value = ''
      bindPhoneNational.value = ''
      emit('message', '已成功绑定手机号')
    } catch (error) {
      console.error('绑定手机号失败:', error)
      bindInlineErrorTraceId.value = ''
      bindInlineError.value = humanizeRequestErrorMessage(error.message || '绑定失败')
    } finally {
      isBindingPhone.value = false
    }
  })
}

onMounted(() => {
  fetchAllowedCodes()
})

onUnmounted(() => {
  if (replaceSmsTimer) {
    clearInterval(replaceSmsTimer)
    replaceSmsTimer = null
  }
  if (bindSmsTimer) {
    clearInterval(bindSmsTimer)
    bindSmsTimer = null
  }
})

  return {
    filteredCountryOptions,
    countryLoading,
    countryError,
    replacePhonePrefix,
    replacePhoneNational,
    replaceNationalMaxLen,
    replaceNationalPlaceholder,
    onReplacePhoneNationalInput,
    onReplacePhoneNationalPaste,
    isSendingReplaceSms,
    replaceSmsCountdown,
    isReplacePhoneValid,
    sendReplaceSms,
    replaceSmsCode,
    isReplacingPhone,
    submitReplacePhone,
    replaceInlineMessage,
    replaceInlineError,
    replaceInlineErrorTraceId,
    replaceReclaimAvailable,
    bindPhonePrefix,
    bindPhoneNational,
    bindNationalMaxLen,
    bindNationalPlaceholder,
    onBindPhoneNationalInput,
    onBindPhoneNationalPaste,
    isSendingBindSms,
    bindSmsCountdown,
    isBindPhoneValid,
    sendBindSms,
    bindSmsCode,
    isBindingPhone,
    submitBindPhone,
    bindInlineMessage,
    bindInlineError,
    bindInlineErrorTraceId,
    bindReclaimAvailable,
  }
}
