<template>
  <div data-alias="cmp-register-form" class="bg-gradient-to-br from-primary/5 to-background min-h-screen flex flex-col">
    <!-- 注册表单 -->
    <main class="flex-grow flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div class="max-w-md w-full space-y-8 bg-white p-10 rounded-2xl shadow-xl">
        <div class="text-center">
          <div class="inline-block w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mb-6">
            <svg class="w-8 h-8 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z"></path>
            </svg>
          </div>
          <h2 class="text-[clamp(1.5rem,3vw,2.5rem)] font-bold text-text mb-2">创建新账户</h2>
          <p class="text-text-light">选择注册方式，开启您的SaaS管理之旅</p>
        </div>
        <form class="space-y-6" @submit.prevent="handleSubmit">
          <!-- 注册方式选择 -->
          <RegisterMethodSelector
            v-model="currentRegisterType"
            :allow-email="allowEmailRegister"
            @method-changed="handleMethodChanged"
          />

          <!-- 邮箱邀请状态提示 -->
          <div v-if="inviteTokenChecking" class="p-3 rounded-lg bg-blue-50 border border-blue-200 text-sm text-blue-700">
            正在验证邀请链接...
          </div>
          <div v-else-if="inviteTokenError" class="p-3 rounded-lg bg-red-50 border border-red-200 text-sm text-red-700" :data-traceId="inviteTokenErrorTraceId || undefined">
            {{ inviteTokenError }}
          </div>
          <div v-else-if="inviteTokenValid && inviteEmail" class="p-3 rounded-lg bg-green-50 border border-green-200 text-sm text-green-700">
            邀请验证成功！注册邮箱：<strong>{{ inviteEmail }}</strong>
          </div>
          <div v-else-if="currentRegisterType === 'email' && !inviteTokenValid" class="p-3 rounded-lg bg-amber-50 border border-amber-200 text-sm text-amber-700">
            邮箱注册需要管理员邀请。如需注册，请使用手机号，或联系管理员发送邮箱邀请。
          </div>

          <!-- 手机号注册表单 -->
          <PhoneRegister
            v-if="currentRegisterType === 'phone'"
            v-model="formData"
            v-model:errors="errors"
            :trace-id="submitErrorTraceId"
            @send-code="handleSendCode"
          />

          <!-- 邮箱注册表单 -->
          <EmailRegister
            v-else-if="allowEmailRegister"
            v-model="formData"
            v-model:errors="errors"
            :trace-id="submitErrorTraceId"
          />

          <!-- 通过推荐分享链接（?accessCode=）进入时，分享码即邀请凭证，不再要求邀请码 -->
          <div
            v-if="invitePolicyEnabled && accessCodePresent"
            data-testid="register-access-code-notice"
            class="p-3 rounded-lg bg-green-50 border border-green-200 text-sm text-green-700"
          >
            您已通过推荐邀请链接访问，无需填写邀请码
          </div>
          <RegistrationInviteCodeField
            v-else
            v-model="inviteCode"
            :enabled="invitePolicyEnabled"
            input-id="register-invite-code"
            hint="开启邀请注册时必填"
            :error-message="inviteError"
            :error-trace-id="inviteErrorTraceId"
          />
          
          <RegisterPasswordField v-model="formData.password" :error="errors.password" />
          
          <div class="rounded-lg border border-gray-200 bg-gray-50/50 px-4 py-3 space-y-2">
            <p v-if="privacyLoadError" class="text-sm text-amber-800">{{ privacyLoadError }}</p>
            <label class="flex items-start gap-2 cursor-pointer select-none">
              <input
                v-model="acceptPrivacy"
                data-testid="register-privacy-accept"
                type="checkbox"
                class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <span class="text-sm text-text">
                我已阅读并同意
                <button
                  type="button"
                  class="text-primary font-medium hover:underline"
                  @click="showPrivacyReader = true"
                >
                  《隐私政策》
                </button>
                <span v-if="currentPrivacyPolicy" class="text-text-light">（{{ currentPrivacyPolicy.version }}）</span>
              </span>
            </label>
            <label class="flex items-start gap-2 cursor-pointer select-none">
              <input
                v-model="acceptLicense"
                data-testid="register-license-accept"
                type="checkbox"
                class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <span class="text-sm text-text">
                我已阅读并同意
                <button
                  type="button"
                  class="text-primary font-medium hover:underline"
                  @click="showLicenseReader = true"
                >
                  《软件许可及服务协议》
                </button>
                <span v-if="currentLicenseAgreement" class="text-text-light">（{{ currentLicenseAgreement.version }}）</span>
              </span>
            </label>
          </div>

          <div
            v-if="submitError"
            data-testid="register-submit-error"
            class="p-3 rounded-lg bg-red-50 border border-red-200 text-sm text-red-700"
            role="alert"
            :data-traceId="submitErrorTraceId || undefined"
          >
            {{ submitError }}
          </div>
          <div>
            <button
              type="submit"
              data-testid="register-submit"
              class="btn-primary w-full py-3 text-base"
              :disabled="isSubmitting || !canSubmitRegister"
              :aria-busy="isSubmitting"
            >
              {{ isSubmitting ? '注册中...' : '注册' }}
            </button>
          </div>
        </form>
        
        <div class="text-center">
          <p class="text-sm text-text-light">
            已有账户？ <a href="/auth/login/" class="font-medium text-primary hover:text-primary-dark transition-colors duration-300">立即登录</a>
          </p>
        </div>
      </div>
    </main>

    <PageFooter />

    <div
      v-if="showPrivacyReader"
      class="fixed inset-0 z-[10040] flex items-end sm:items-center justify-center bg-black/50 p-0 sm:p-4"
      @click.self="showPrivacyReader = false"
    >
      <div class="w-full sm:max-w-2xl max-h-[85vh] flex flex-col rounded-t-2xl sm:rounded-2xl bg-white shadow-2xl">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <h2 class="text-base font-semibold text-gray-900">
            {{ currentPrivacyPolicy?.title || '隐私政策' }}
          </h2>
          <button type="button" class="text-sm text-primary" @click="showPrivacyReader = false">关闭</button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3 text-sm text-gray-800 whitespace-pre-wrap">
          <p v-if="!currentPrivacyPolicy" class="text-gray-500">正在加载或暂无内容…</p>
          <template v-else>{{ currentPrivacyPolicy.content }}</template>
        </div>
      </div>
    </div>

    <div
      v-if="showLicenseReader"
      class="fixed inset-0 z-[10040] flex items-end sm:items-center justify-center bg-black/50 p-0 sm:p-4"
      @click.self="showLicenseReader = false"
    >
      <div class="w-full sm:max-w-2xl max-h-[85vh] flex flex-col rounded-t-2xl sm:rounded-2xl bg-white shadow-2xl">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <h2 class="text-base font-semibold text-gray-900">
            {{ currentLicenseAgreement?.title || '软件许可及服务协议' }}
          </h2>
          <button type="button" class="text-sm text-primary" @click="showLicenseReader = false">关闭</button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3 text-sm text-gray-800 whitespace-pre-wrap">
          <p v-if="!currentLicenseAgreement" class="text-gray-500">正在加载或暂无内容…</p>
          <template v-else>{{ currentLicenseAgreement.content }}</template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js';
import { safeJson, safeResponseJson } from '../utils/safeResponseJson.js';
import { extractTraceId } from '../utils/traceId.js';
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { mapRegisterApiError } from '../utils/mapRegisterApiError.js'

/* @alias:cmp-register-form */
import { ref, computed, onMounted, watch } from 'vue'
import RegisterMethodSelector from '../components/auth/RegisterMethodSelector.vue'
import PhoneRegister from '../components/auth/PhoneRegister.vue'
import EmailRegister from '../components/auth/EmailRegister.vue'
import RegistrationInviteCodeField from '../components/RegistrationInviteCodeField.vue'
import RegisterPasswordField from '../components/auth/RegisterPasswordField.vue'
import PageFooter from './PageFooter.vue'
import toastService from '../utils/toastService'
import { readStoredInviteCode, writeStoredInviteCode } from '../utils/registrationInviteUtils.js'
import {
  fetchInvitePolicyEnabled,
  fetchCurrentPrivacyPolicy,
  fetchCurrentLicenseAgreement,
} from '../utils/registerLegalDocuments.js'
import {
  captureReferralAccessCodeFromSearch,
  readStoredReferralAccessCode,
} from '../utils/referralAccessCodeUtils.js'
import { useRegisterFeaturePolicy } from '../composables/auth/useRegisterFeaturePolicy.js'

const props = defineProps({
  user: {
    type: Object,
    default: () => ({
      isAuthenticated: false,
      isSuperuser: false,
      username: ''
    })
  }
})

const emit = defineEmits(['register-success', 'register-failure'])

const currentRegisterType = ref('phone') // 默认手机号注册
const { allowEmailRegister, loadPublicFeaturePolicy } = useRegisterFeaturePolicy({
  currentRegisterType,
})
// 邮箱邀请注册
const inviteToken = ref('')
const inviteEmail = ref('')
const inviteTokenValid = ref(false)
const inviteTokenChecking = ref(false)
const inviteTokenError = ref('')
const inviteTokenErrorTraceId = ref('')
const isSubmitting = ref(false)
const currentPrivacyPolicy = ref(null)
const privacyLoadError = ref('')
const acceptPrivacy = ref(false)
const showPrivacyReader = ref(false)
const currentLicenseAgreement = ref(null)
const licenseLoadError = ref('')
const acceptLicense = ref(false)
const showLicenseReader = ref(false)
const invitePolicyEnabled = ref(false)
const accessCodePresent = ref(false)
const inviteCode = ref(readStoredInviteCode())
const inviteError = ref('')
const inviteErrorTraceId = ref('')
const submitError = ref('')
const submitErrorTraceId = ref('')
const registerGuard = createClickGuard()

watch(inviteCode, (v) => {
  writeStoredInviteCode(v)
  inviteError.value = ''
  inviteErrorTraceId.value = ''
})

const canSubmitRegister = computed(
  () =>
    Boolean(acceptPrivacy.value && currentPrivacyPolicy.value && !privacyLoadError.value && acceptLicense.value && currentLicenseAgreement.value && !licenseLoadError.value)
)

const formData = ref({
  phone: '',
  email: '',
  code: '',
  password: ''
})

const errors = ref({
  phone: '',
  email: '',
  code: '',
  password: ''
})

const handleMethodChanged = () => {
  hideAllErrors()
}

// 处理发送验证码：通过 resolve 回调把发码结果回报给 PhoneRegister，
// 由子组件决定是否进入倒计时或展示内联错误（对齐登录页 UX，OPT-20260811-013）。
const handleSendCode = async ({ phone, resolve }) => {
  try {
    const response = await apiFetch('/api/accounts/users/send_verification_code/', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ phone })
    })

    const responseData = await safeJson(response, {})

    if (response.ok) {
      resolve?.({ ok: true })
    } else {
      const message = responseData.phone
        ? responseData.phone[0]
        : (responseData.detail || responseData.error || '发送验证码失败')
      resolve?.({
        ok: false,
        message,
        traceId: extractTraceId(response) || responseData.trace_id || responseData.traceId || '',
      })
    }
  } catch (error) {
    console.error('发送验证码失败:', error)
    resolve?.({ ok: false, message: '发送验证码失败，请稍后重试', traceId: extractTraceId(error) })
  }
}

const hideAllErrors = () => {
  Object.keys(errors.value).forEach(key => {
    errors.value[key] = ''
  })
  submitError.value = ''
  submitErrorTraceId.value = ''
}

const applyMappedRegisterError = (data, traceId) => {
  const mapped = mapRegisterApiError(data, {
    registerType: currentRegisterType.value,
    traceId,
  })
  inviteError.value = mapped.inviteError
  inviteErrorTraceId.value = mapped.inviteErrorTraceId
  errors.value = { ...errors.value, ...mapped.fieldErrors }
  submitError.value = mapped.submitError
  submitErrorTraceId.value = mapped.submitErrorTraceId
  if (mapped.submitError) {
    console.warn('[Register] 注册失败', {
      trace_id: mapped.submitErrorTraceId,
      message: mapped.submitError,
    })
  }
}

const handleSubmit = async () => {
  await registerGuard.run(async ({ idempotencyKey }) => {
    try {
      isSubmitting.value = true
      hideAllErrors()

      if (!acceptPrivacy.value || !currentPrivacyPolicy.value?.id) {
        toastService.error('请阅读并勾选同意隐私政策后再注册')
        return
      }

      if (!acceptLicense.value || !currentLicenseAgreement.value?.id) {
        toastService.error('请阅读并勾选同意服务协议后再注册')
        return
      }

      inviteError.value = ''
      inviteErrorTraceId.value = ''
      const normalizedInvite = String(inviteCode.value || '').trim().toUpperCase()
      if (invitePolicyEnabled.value && !normalizedInvite && !accessCodePresent.value) {
        inviteError.value = '请填写邀请码'
        return
      }

      const submitData = {
        password: formData.value.password,
        accepted_privacy_policy_id: String(currentPrivacyPolicy.value.id),
        accepted_license_agreement_id: String(currentLicenseAgreement.value.id)
      }
      if (normalizedInvite) {
        submitData.invite_code = normalizedInvite
      }
      const accessCode = readStoredReferralAccessCode()
      if (accessCode) {
        submitData.access_code = accessCode
      }

      let apiEndpoint = '/api/accounts/users/phone_register/'

      if (currentRegisterType.value === 'phone') {
        submitData.phone = formData.value.phone.trim()
        submitData.code = formData.value.code.trim()
      } else {
        if (!inviteToken.value) {
          toastService.error('邮箱注册需要管理员邀请，请使用手机号注册')
          return
        }
        submitData.email = formData.value.email.trim()
        submitData.invite_token = inviteToken.value
        apiEndpoint = '/api/accounts/users/email_register/'
      }

      const response = await apiFetch(apiEndpoint, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
        }, idempotencyKey),
        body: JSON.stringify(submitData)
      })

      const { data, traceId } = await safeResponseJson(response, {})

      if (response.ok) {
        if (currentRegisterType.value === 'email') {
          if (data.user_existed) {
            if (data.is_active) {
              toastService.info('该邮箱已注册并激活，请直接登录')
            } else {
              toastService.info('该邮箱已注册但未激活，已重新发送激活邮件，请查收')
            }
          } else {
            toastService.success('注册成功！即将跳转到登录页面')
          }
        } else {
          toastService.success('注册成功，即将跳转到登录页面')
        }

        setTimeout(() => {
          window.location.href = '/auth/login/'
        }, 2000)
        emit('register-success', data)
      } else {
        applyMappedRegisterError(data, traceId || extractTraceId(response) || '')
        emit('register-failure', data)
      }
    } catch (error) {
      console.error('注册失败:', error)
      submitError.value = '注册失败：网络错误，请稍后重试'
      submitErrorTraceId.value = extractTraceId(error) || ''
      toastService.error(submitError.value, { traceId: submitErrorTraceId.value })
      emit('register-failure', { error: error.message })
    } finally {
      isSubmitting.value = false
    }
  })
}

const checkInviteToken = async (token) => {
  if (!token) return
  inviteTokenChecking.value = true
  inviteToken.value = token
  inviteTokenErrorTraceId.value = ''
  try {
    const r = await apiFetch(`/api/public/email-invitation/${token}/`, {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    const { data: d, traceId } = await safeResponseJson(r, {})
    if (r.ok && d.valid) {
      inviteTokenValid.value = true
      inviteEmail.value = d.email || ''
      formData.value.email = d.email || ''
      allowEmailRegister.value = true
      currentRegisterType.value = 'email'
      inviteTokenErrorTraceId.value = ''
      // 清除 URL 中的 token，保持页面干净
      const newUrl = window.location.pathname + window.location.search.replace(/[?&]invite_token=[^&]*/g, '').replace(/\?$/, '')
      window.history.replaceState({}, document.title, newUrl)
    } else {
      inviteTokenError.value = d.error || d.detail || '邀请链接无效或已过期'
      inviteTokenErrorTraceId.value = traceId || extractTraceId(r) || ''
    }
  } catch (e) {
    const msg = e && e.message ? String(e.message) : ''
    if (msg.includes('Failed to fetch') || msg.includes('NetworkError')) {
      inviteTokenError.value = '网络连接失败，请检查网络后刷新重试'
    } else if (msg.includes('JSON') || msg.includes('Unexpected token')) {
      inviteTokenError.value = '服务响应异常，请稍后重试'
    } else {
      inviteTokenError.value = '无法验证邀请链接（' + (msg || '未知错误') + '）'
    }
    inviteTokenErrorTraceId.value = ''
  } finally {
    inviteTokenChecking.value = false
  }
}

    onMounted(() => {
      loadPublicFeaturePolicy()
      accessCodePresent.value = Boolean(captureReferralAccessCodeFromSearch(window.location.search))
    
      // 检测 URL 中的邀请 token
  const urlParams = new URLSearchParams(window.location.search)
  const token = urlParams.get('invite_token')
  if (token) {
    checkInviteToken(token)
  }

  ;(async () => {
    invitePolicyEnabled.value = await fetchInvitePolicyEnabled(apiFetch)
  })()
  ;(async () => {
    const { policy, error } = await fetchCurrentPrivacyPolicy(apiFetch, safeJson)
    currentPrivacyPolicy.value = policy
    privacyLoadError.value = error
  })()
  ;(async () => {
    const { agreement, error } = await fetchCurrentLicenseAgreement(apiFetch, safeJson)
    currentLicenseAgreement.value = agreement
    licenseLoadError.value = error
  })()
})
</script>

