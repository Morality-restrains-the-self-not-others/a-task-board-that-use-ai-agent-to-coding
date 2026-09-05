<template>
  <div class="phone-register">
    <div>
      <label for="register-phone-national" class="block text-sm font-medium text-text-light mb-2">手机号</label>
      <div class="flex rounded-lg border border-gray-300 focus-within:ring-2 focus-within:ring-primary relative">
        <div class="relative z-30 shrink-0 border-r border-gray-200 bg-gray-50">
          <select
            v-model="registerPhonePrefix"
            class="block w-full min-w-[6.5rem] max-h-48 py-2.5 pl-2 pr-8 text-sm text-gray-700 bg-transparent border-0 appearance-none cursor-pointer"
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
        </div>
        <input
          id="register-phone-national"
          :value="phoneNational"
          type="tel"
          name="phone_national"
          autocomplete="tel-national"
          required
          class="flex-1 min-w-0 px-3 py-2.5 border-0 text-sm focus:outline-none bg-white"
          :placeholder="nationalPlaceholder"
          :maxlength="maxNationalLen"
          inputmode="numeric"
          @input="onNationalInput"
          @paste="onNationalPaste"
        >
      </div>
      <p class="mt-1 text-xs text-text-light">默认区号 +86；可粘贴含 +86 / 86 前缀的完整号码。</p>
      <div id="phone-error" class="text-error text-sm mt-1" :class="{ 'hidden': !errors.phone }" :data-traceId="traceId || undefined">{{ errors.phone }}</div>
    </div>
    
    <!-- 验证码输入区域 -->
    <div class="grid grid-cols-3 gap-4">
      <div class="col-span-2">
        <label for="code" class="block text-sm font-medium text-text-light mb-2">验证码</label>
        <input 
          type="text" 
          id="code" 
          :value="modelValue.code" 
          name="code" 
          placeholder="请输入6位验证码" 
          required 
          class="input w-full"
          @input="handleCodeInput"
        >
        <div id="code-error" class="text-error text-sm mt-1" :class="{ 'hidden': !errors.code }" :data-traceId="traceId || undefined">{{ errors.code }}</div>
      </div>
      <div class="col-span-1 flex flex-col justify-end">
        <p
          v-if="sendCodeError"
          class="text-xs text-red-500 mb-1"
          role="alert"
          :data-traceId="sendCodeErrorTraceId || undefined"
        >{{ sendCodeError }}</p>
        <button
          type="button"
          class="btn-secondary w-full py-3 text-base"
          :disabled="isSendingCode || !canSendCode || countdown > 0"
          @click="sendVerificationCode"
        >
          {{ sendCodeButtonText }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useAllowedCountryCodes } from '../../composables/auth/useAllowedCountryCodes.js'
import {
  DEFAULT_PHONE_COUNTRY_PREFIX,
  buildPhoneForApi,
  isValidE164Loose,
  maxNationalDigitsForPrefix,
  parsePasteToPrefixAndNational,
} from '../../utils/phoneInput.js'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({
      phone: '',
      code: '',
      password: ''
    })
  },
  errors: {
    type: Object,
    default: () => ({
      phone: '',
      code: ''
    })
  },
  csrfToken: {
    type: String,
    default: ''
  },
  /** 本次提交 API 失败 traceId，挂到字段级错误节点供 trace-first 排障（OPT-20260821-025） */
  traceId: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'update:errors', 'send-code'])

const { filteredCountryOptions, defaultCountryPrefix, syncPrefixWithAllowedOptions, fetchAllowedCodes } = useAllowedCountryCodes()
// OPT-20260806-027: 默认前缀与后端白名单联动（白名单不含 +86 时取首个选项）
const registerPhonePrefix = ref(defaultCountryPrefix.value)
syncPrefixWithAllowedOptions(registerPhonePrefix)
const phoneNational = ref('')

const isSendingCode = ref(false)
const countdown = ref(0)
const canSendCode = ref(false)
/** API/网络失败时展示在「获取验证码」按钮上方（非弹窗，对齐登录页 UX） */
const sendCodeError = ref('')
/** 仅请求失败时填充；纯前端校验不得伪造 data-traceId */
const sendCodeErrorTraceId = ref('')
/** 倒计时定时器句柄（防重复发送叠加） */
let sendCodeTimer = null
/** 父组件未在窗口内回报发码结果时的兜底超时，避免 isSendingCode 卡死 */
const SEND_CODE_RESULT_TIMEOUT_MS = 15000

const phoneForApi = computed(() =>
  buildPhoneForApi(registerPhonePrefix.value, phoneNational.value),
)

const maxNationalLen = computed(() => maxNationalDigitsForPrefix(registerPhonePrefix.value))

const nationalPlaceholder = computed(() =>
  registerPhonePrefix.value === '+86' || registerPhonePrefix.value === '86'
    ? '1×××××××××'
    : '本地号码（不含区号）',
)

const sendCodeButtonText = computed(() => {
  if (isSendingCode.value) {
    return '发送中...'
  }
  if (countdown.value > 0) {
    return `${countdown.value}秒后重新获取`
  }
  return '获取验证码'
})

function clearSendCodeError() {
  sendCodeError.value = ''
  sendCodeErrorTraceId.value = ''
}

function startCountdown() {
  if (sendCodeTimer) {
    clearInterval(sendCodeTimer)
  }
  countdown.value = 60
  sendCodeTimer = setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0) {
      clearInterval(sendCodeTimer)
      sendCodeTimer = null
    }
  }, 1000)
}

function syncModelPhoneAndValidate() {
  clearSendCodeError()
  const phone = phoneForApi.value
  emit('update:modelValue', { ...props.modelValue, phone })

  const nationalDigits = String(phoneNational.value || '').replace(/\D/g, '')
  if (!nationalDigits) {
    emit('update:errors', { ...props.errors, phone: '' })
    canSendCode.value = false
    return
  }
  if (!isValidE164Loose(phone)) {
    emit('update:errors', { ...props.errors, phone: '请输入有效的手机号（需含国家码）' })
    canSendCode.value = false
    return
  }
  emit('update:errors', { ...props.errors, phone: '' })
  canSendCode.value = true
}

watch(registerPhonePrefix, () => {
  const maxL = maxNationalDigitsForPrefix(registerPhonePrefix.value)
  const d = String(phoneNational.value || '').replace(/\D/g, '')
  if (d.length > maxL) {
    phoneNational.value = d.slice(0, maxL)
  }
})

onMounted(() => {
  fetchAllowedCodes()
})

watch([registerPhonePrefix, phoneNational], () => {
  syncModelPhoneAndValidate()
}, { immediate: true })

function onNationalInput(e) {
  const maxL = maxNationalDigitsForPrefix(registerPhonePrefix.value)
  phoneNational.value = String(e?.target?.value ?? phoneNational.value ?? '').replace(/\D/g, '').slice(0, maxL)
}

function onNationalPaste(e) {
  const text = e.clipboardData?.getData('text') || ''
  const parsed = parsePasteToPrefixAndNational(text)
  if (parsed && parsed.national) {
    e.preventDefault()
    registerPhonePrefix.value = parsed.prefix
    const maxL = maxNationalDigitsForPrefix(parsed.prefix)
    phoneNational.value = parsed.national.slice(0, maxL)
  }
}

const handleCodeInput = (event) => {
  const code = event.target.value
  emit('update:modelValue', { ...props.modelValue, code })
}

const sendVerificationCode = async () => {
  clearSendCodeError()
  const phone = phoneForApi.value
  if (!isValidE164Loose(phone)) {
    emit('update:errors', { ...props.errors, phone: '请输入有效的手机号（需含国家码）' })
    return
  }

  isSendingCode.value = true
  let timeoutId = null
  try {
    // 等父组件回报发码结果后再决定倒计时/展示错误；resolve 由父组件在成功/失败分支调用。
    const outcome = await Promise.race([
      new Promise((resolve) => {
        emit('send-code', { phone, resolve })
      }),
      new Promise((resolve) => {
        timeoutId = setTimeout(
          () => resolve({ ok: false, message: '发送验证码超时，请稍后重试' }),
          SEND_CODE_RESULT_TIMEOUT_MS,
        )
      }),
    ])
    if (timeoutId) {
      clearTimeout(timeoutId)
    }
    if (outcome && outcome.ok) {
      startCountdown()
    } else {
      sendCodeError.value = (outcome && outcome.message) || '发送验证码失败'
      sendCodeErrorTraceId.value = (outcome && outcome.traceId) || ''
    }
  } catch (error) {
    console.error('发送验证码失败:', error)
    sendCodeError.value = '发送验证码失败，请稍后重试'
  } finally {
    isSendingCode.value = false
  }
}
</script>
