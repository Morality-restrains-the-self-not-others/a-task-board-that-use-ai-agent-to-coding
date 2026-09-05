<template>
  <div v-if="visible" class="space-y-3 mb-4 p-4 border border-amber-200 bg-amber-50 rounded-lg">
    <h3 class="font-medium text-gray-800">安全验证</h3>
    <p class="text-sm text-gray-600">{{ descriptionText }}</p>

    <div v-if="loading" class="text-sm text-gray-500">检查验证状态…</div>

    <!-- 已有绑定手机号：显示并发送验证码 -->
    <div v-else-if="hasPhone" class="space-y-3">
      <p class="text-sm text-gray-700">
        已绑定手机：<span class="font-medium">{{ phoneMasked || '—' }}</span>
      </p>
      <div class="flex flex-wrap gap-2 items-center">
        <button
          type="button"
          :disabled="sendingSms || smsCountdown > 0"
          class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
          @click="sendSms"
        >
          {{ smsCountdown > 0 ? `${smsCountdown} 秒后可重发` : (sendingSms ? '发送中…' : '发送验证码') }}
        </button>
      </div>
      <div class="flex flex-wrap gap-2 items-center">
        <input
          v-model="smsCode"
          type="text"
          inputmode="numeric"
          maxlength="6"
          autocomplete="one-time-code"
          placeholder="6 位短信验证码"
          class="w-40 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          @keyup.enter="verifyCode"
        >
        <button
          type="button"
          :disabled="verifying || smsCode.trim().length !== 6"
          class="px-4 py-2 rounded-lg bg-green-600 text-white text-sm hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          @click="verifyCode"
        >
          {{ verifying ? '验证中…' : '验证' }}
        </button>
      </div>
    </div>

    <!-- 未绑定手机号：先输入手机号再发验证码 -->
    <div v-else class="space-y-3">
      <p class="text-sm text-gray-600">请先绑定手机号以完成安全验证。</p>
      <div class="flex flex-wrap gap-2 items-center">
        <div class="flex rounded-lg border border-gray-300 focus-within:ring-2 focus-within:ring-primary relative bg-white">
          <div class="relative z-30 shrink-0 border-r border-gray-300 bg-gray-50" :title="countryError ? '区域限制暂不可用，显示全部地区' : ''">
            <select
              v-model="phonePrefix"
              class="block w-full min-w-[6.5rem] max-h-48 py-2 pl-2 pr-8 text-sm text-gray-800 bg-transparent border-0 appearance-none cursor-pointer"
              aria-label="国家或地区代码"
            >
              <option
                v-for="c in filteredCountryOptions"
                :key="c.code"
                :value="c.code"
              >
                {{ c.code }} {{ c.name }}
              </option>
            </select>
            <span v-if="countryLoading" class="absolute right-1 top-1/2 -translate-y-1/2 flex items-center" aria-label="加载区域限制中" title="区域限制加载中">
              <svg class="animate-spin h-3.5 w-3.5 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </span>
          </div>
          <input
            v-model="phoneNational"
            type="tel"
            inputmode="numeric"
            :maxlength="phoneNationalMaxLen"
            autocomplete="tel-national"
            :placeholder="phoneNationalPlaceholder"
            class="w-40 min-w-0 px-3 py-2 border-0 text-sm focus:outline-none bg-transparent"
            @input="onPhoneNationalInput"
            @paste="onPhoneNationalPaste"
          >
        </div>
        <button
          type="button"
          :disabled="sendingSms || smsCountdown > 0 || !isPhoneValid"
          class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
          @click="sendSms"
        >
          {{ smsCountdown > 0 ? `${smsCountdown} 秒后可重发` : (sendingSms ? '发送中…' : '发送验证码') }}
        </button>
      </div>
      <div class="flex flex-wrap gap-2 items-center">
        <input
          v-model="smsCode"
          type="text"
          inputmode="numeric"
          maxlength="6"
          autocomplete="one-time-code"
          placeholder="6 位短信验证码"
          class="w-40 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          @keyup.enter="verifyCode"
        >
        <button
          type="button"
          :disabled="verifying || smsCode.trim().length !== 6 || !isPhoneValid"
          class="px-4 py-2 rounded-lg bg-green-600 text-white text-sm hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          @click="verifyCode"
        >
          {{ verifying ? '验证中…' : '验证并绑定' }}
        </button>
      </div>
    </div>

    <div v-if="errorMsg" class="text-red-600 text-sm" :data-traceId="errorMsgTraceId || undefined">{{ errorMsg }}</div>
    <div v-if="successMsg" class="text-green-600 text-sm">{{ successMsg }}</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { getCookie } from '../utils/cookieUtils'
import { extractTraceId } from '../utils/traceId.js'
import { useAllowedCountryCodes } from '../composables/auth/useAllowedCountryCodes.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import {
  DEFAULT_PHONE_COUNTRY_PREFIX,
  buildPhoneForApi,
  isValidE164Loose,
  maxNationalDigitsForPrefix,
  parsePasteToPrefixAndNational} from '../utils/phoneInput.js'

const props = defineProps({
  tenantId: { type: String, required: true },
  active: { type: Boolean, default: false },
  // 厂商申请等场景强制要求短信验证（不受支付策略 required 开关影响）
  forceRequired: { type: Boolean, default: false },
  descriptionText: {
    type: String,
    default: '支付前需先完成手机号短信验证，以保障账户安全。',
  },
  // OPT-20260726-014: 父组件预检传入的手机验证状态，避免重复 API 请求
  phoneStatus: {
    type: Object,
    default: () => ({
      has_phone: false,
      phone_masked: '',
      phone_e164: '',
      sms_verified: false,
      required: false})}})

const emit = defineEmits(['verified'])

// 从公开 API 获取管理员设置的允许手机号区域，过滤下拉选项
const { filteredCountryOptions, defaultCountryPrefix, syncPrefixWithAllowedOptions, fetchAllowedCodes, loading: countryLoading, error: countryError } = useAllowedCountryCodes()

// ---- 状态 ----
const loading = ref(false)
const phoneInfo = ref({
  has_phone: false,
  phone_masked: '',
  phone_e164: '',
  sms_verified: false,
  required: false})
// OPT-20260806-027: 默认前缀与后端白名单联动（白名单不含 +86 时取首个选项）
const phonePrefix = ref(defaultCountryPrefix.value)
syncPrefixWithAllowedOptions(phonePrefix)
const phoneNational = ref('')
const smsCode = ref('')
const smsCountdown = ref(0)
let smsTimer = null
const sendingSms = ref(false)
const verifying = ref(false)

const sendSmsGuard = createClickGuard()
const verifyCodeGuard = createClickGuard()
const errorMsg = ref('')
const errorMsgTraceId = ref('')
const successMsg = ref('')

const isRequired = computed(() => props.forceRequired || !!phoneInfo.value.required)
const visible = computed(() => props.active && isRequired.value && !loading.value && !phoneInfo.value.sms_verified)

const hasPhone = computed(() => phoneInfo.value.has_phone)
const phoneMasked = computed(() => phoneInfo.value.phone_masked)

const phoneForApi = computed(() => buildPhoneForApi(phonePrefix.value, phoneNational.value))
const isPhoneValid = computed(() => isValidE164Loose(phoneForApi.value))
const phoneNationalMaxLen = computed(() => maxNationalDigitsForPrefix(phonePrefix.value))
const phoneNationalPlaceholder = computed(() =>
  phonePrefix.value === '+86' || phonePrefix.value === '86' ? '1×××××××××' : '本地号码',
)

// ---- 手机号输入处理 ----
function onPhoneNationalInput(e) {
  const maxL = maxNationalDigitsForPrefix(phonePrefix.value)
  phoneNational.value = String(e?.target?.value ?? phoneNational.value ?? '')
    .replace(/\D/g, '')
    .slice(0, maxL)
}

function onPhoneNationalPaste(e) {
  const text = e.clipboardData?.getData('text') || ''
  const parsed = parsePasteToPrefixAndNational(text)
  if (parsed && parsed.national) {
    e.preventDefault()
    phonePrefix.value = parsed.prefix
    const maxL = maxNationalDigitsForPrefix(parsed.prefix)
    phoneNational.value = parsed.national.slice(0, maxL)
  }
}

// OPT-20260726-014: 初始化手机信息 — 优先使用父组件传入的 phoneStatus prop
const initPhoneInfo = () => {
  const st = props.phoneStatus || {}
  phoneInfo.value = {
    has_phone: !!st.has_phone,
    phone_masked: st.phone_masked || '',
    phone_e164: st.phone_e164 || '',
    sms_verified: !!st.sms_verified,
    required: props.forceRequired || !!st.required,
  }
  // 已验证：直接通知（forceRequired 场景也适用）
  if (phoneInfo.value.sms_verified && (phoneInfo.value.has_phone || phoneInfo.value.phone_e164)) {
    emit('verified', { phone_e164: phoneInfo.value.phone_e164 || '' })
    loading.value = false
    return
  }
  loading.value = false
}

// 激活时从 phoneStatus prop 初始化（替代独立 fetchStatus）
watch(() => props.active, (val) => {
  if (val) {
    loading.value = true
    initPhoneInfo()
  }
})

onMounted(() => {
  fetchAllowedCodes()
  if (props.active) {
    loading.value = true
    initPhoneInfo()
  }
})


const startCountdown = () => {
  smsCountdown.value = 60
  if (smsTimer) clearInterval(smsTimer)
  smsTimer = setInterval(() => {
    smsCountdown.value -= 1
    if (smsCountdown.value <= 0) {
      clearInterval(smsTimer)
      smsTimer = null
    }
  }, 1000)
}

const sendSms = async () => {
  errorMsg.value = ''
  errorMsgTraceId.value = ''
  successMsg.value = ''

  let phoneToSend = ''
  if (phoneInfo.value.has_phone) {
    phoneToSend = phoneInfo.value.phone_e164
    if (!phoneToSend) {
      errorMsg.value = '无法获取绑定手机号，请刷新页面后重试'
      return
    }
  } else {
    if (!isPhoneValid.value) {
      errorMsg.value = '请输入有效手机号（含国家/地区码）'
      return
    }
    phoneToSend = phoneForApi.value
  }

  // OPT-20260819-038: 发送短信验证码是账号写操作，防连点/超时重试双发
  await sendSmsGuard.run(async ({ idempotencyKey }) => {
    sendingSms.value = true
    try {
      const r = await apiFetch('/api/accounts/users/send_verification_code/', {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ phone: phoneToSend })})
      const d = await r.json().catch(() => ({}))
      if (!r.ok) {
        const msg = d.detail || d.error || d.message || '发送失败'
        throw new Error(typeof msg === 'string' ? msg : '发送失败')
      }
      successMsg.value = '验证码已发送'
      startCountdown()
    } catch (e) {
      console.error('发送验证码失败:', e)
      errorMsgTraceId.value = extractTraceId(e) || ''
      errorMsg.value = e.message || '发送失败'
    } finally {
      sendingSms.value = false
    }
  })
}

const verifyCode = async () => {
  errorMsg.value = ''
  errorMsgTraceId.value = ''
  successMsg.value = ''

  if (smsCode.value.trim().length !== 6) {
    errorMsg.value = '请输入 6 位验证码'
    return
  }

  let phoneToVerify = ''
  if (phoneInfo.value.has_phone) {
    phoneToVerify = phoneInfo.value.phone_e164
    if (!phoneToVerify) {
      errorMsg.value = '无法获取绑定手机号，请刷新页面后重试'
      return
    }
  } else {
    if (!isPhoneValid.value) {
      errorMsg.value = '请输入有效手机号'
      return
    }
    phoneToVerify = phoneForApi.value
  }

  // OPT-20260819-038: 验证手机是账号写操作，防连点/超时重试双发
  await verifyCodeGuard.run(async ({ idempotencyKey }) => {
    verifying.value = true
    try {
      const r = await apiFetch(`/api/tenant/${props.tenantId}/billing/verify-phone-code/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ phone: phoneToVerify, code: smsCode.value.trim() })})
      const d = await r.json().catch(() => ({}))
      if (!r.ok) {
        throw new Error(d.error || d.detail || '验证失败')
      }
      phoneInfo.value.sms_verified = true
      phoneInfo.value.phone_masked = d.phone_masked || phoneInfo.value.phone_masked
      phoneInfo.value.phone_e164 = phoneToVerify
      phoneInfo.value.has_phone = true
      successMsg.value = '验证成功'
      smsCode.value = ''
      emit('verified', { phone_e164: phoneToVerify })
    } catch (e) {
      console.error('手机验证失败:', e)
      errorMsgTraceId.value = extractTraceId(e) || ''
      errorMsg.value = e.message || '验证失败'
    } finally {
      verifying.value = false
    }
  })
}

onUnmounted(() => {
  if (smsTimer) {
    clearInterval(smsTimer)
    smsTimer = null
  }
})
</script>
