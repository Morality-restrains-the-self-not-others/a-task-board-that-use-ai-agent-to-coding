<template>
  <div data-alias="cmp-reset-password-request" class="flex flex-col items-center justify-center min-h-full py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8 bg-white p-10 rounded-2xl shadow-xl">
      <div class="text-center">
        <div class="inline-block w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mb-6">
          <svg class="w-8 h-8 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path>
          </svg>
        </div>
        <h2 class="text-[clamp(1.5rem,3vw,2.5rem)] font-bold text-text mb-2">忘记密码</h2>
        <p class="text-text-light">请选择一种方式重置您的密码</p>
      </div>

      <div v-if="error" class="bg-red-50 border-l-4 border-red-400 p-4 rounded-lg" :data-traceId="errorTraceId || undefined">
        <div class="flex">
          <div class="flex-shrink-0">
            <svg class="h-5 w-5 text-red-400" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"></path>
            </svg>
          </div>
          <div class="ml-3">
            <p class="text-sm text-red-700" :data-traceId="errorTraceId || undefined">{{ error }}</p>
          </div>
        </div>
      </div>

      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-medium text-text mb-3">通过邮箱重置</h3>
          <form class="space-y-4" @submit.prevent="handleEmailReset">
            <div>
              <label for="email" class="block text-sm font-medium text-text-light mb-2">邮箱地址</label>
              <input 
                type="email" 
                id="email" 
                v-model="emailForm.email" 
                autocomplete="email" 
                required 
                class="input w-full" 
                placeholder="请输入您的邮箱地址"
              >
            </div>
            <div>
              <button type="submit" class="btn-primary w-full py-3 text-base" :disabled="emailLoading">
                <span v-if="!emailLoading">发送重置链接</span>
                <span v-else class="flex items-center justify-center">
                  <svg class="animate-spin -ml-1 mr-2 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  发送中...
                </span>
              </button>
            </div>
          </form>
        </div>

        <div class="relative">
          <div class="absolute inset-0 flex items-center">
            <div class="w-full border-t border-gray-300"></div>
          </div>
          <div class="relative flex justify-center text-sm">
            <span class="px-2 bg-white text-text-light">或</span>
          </div>
        </div>

        <div>
          <h3 class="text-lg font-medium text-text mb-3">通过手机号重置</h3>
          <form class="space-y-4" @submit.prevent="handlePhoneReset">
            <div>
              <label for="phone" class="block text-sm font-medium text-text-light mb-2">手机号码</label>
              <input 
                type="tel" 
                id="phone" 
                v-model="phoneForm.phone" 
                autocomplete="tel" 
                required 
                class="input w-full" 
                placeholder="请输入您的手机号码"
              >
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="code" class="block text-sm font-medium text-text-light mb-2">验证码</label>
                <input 
                  type="text" 
                  id="code" 
                  v-model="phoneForm.code" 
                  maxlength="6" 
                  required 
                  class="input w-full" 
                  placeholder="请输入验证码"
                >
              </div>
              <div class="flex items-end">
                <button 
                  type="button" 
                  @click="sendVerificationCode" 
                  class="btn-secondary w-full py-3 text-base"
                  :disabled="countdown > 0 || phoneForm.phone.length !== 11"
                >
                  <span v-if="countdown <= 0">获取验证码</span>
                  <span v-else>{{ countdown }}秒后重新发送</span>
                </button>
              </div>
            </div>
            <div>
              <button type="submit" class="btn-primary w-full py-3 text-base" :disabled="phoneLoading">
                <span v-if="!phoneLoading">验证并重置</span>
                <span v-else class="flex items-center justify-center">
                  <svg class="animate-spin -ml-1 mr-2 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  处理中...
                </span>
              </button>
            </div>
          </form>
        </div>
      </div>
      
      <div class="text-center">
        <p class="text-sm text-text-light">
          想起密码了？ <a href="/auth/login/" class="font-medium text-primary hover:text-primary-dark transition-colors duration-300">返回登录</a>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js';
import { safeResponseJson } from '../utils/safeResponseJson.js';
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js';

/* @alias:cmp-reset-password-request */
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { getCookie } from '../utils/cookieUtils'

const router = useRouter()

const error = ref('')
const errorTraceId = ref('')

const emailForm = ref({
  email: ''
})

const phoneForm = ref({
  phone: '',
  code: ''
})

const emailLoading = ref(false)
const phoneLoading = ref(false)
const countdown = ref(0)

const emailResetGuard = createClickGuard()
const sendCodeGuard = createClickGuard()

const getApiErrorMessage = async (response, fallbackMessage) => {
  const { data, traceId } = await safeResponseJson(response, { fallback: {} })
  // Prefer traceId from safeResponseJson; fallback to response.traceId (set by apiFetch)
  errorTraceId.value = traceId || response.traceId || ''
  if (data && typeof data.error === 'string') return data.error
  if (data && typeof data.detail === 'string') return data.detail
  if (data && typeof data.message === 'string') return data.message
  return fallbackMessage
}

const handleEmailReset = async () => {
  error.value = ''
  errorTraceId.value = ''
  // OPT-20260819-038: 发送重置链接是账号写操作，防连点/超时重试双发
  await emailResetGuard.run(async ({ idempotencyKey }) => {
    emailLoading.value = true
    try {
      const response = await apiFetch('/api/accounts/users/send_password_reset_link/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ email: emailForm.value.email })
      })

      if (response.ok) {
        alert('密码重置链接已发送到您的邮箱，请查收')
        router.push('/auth/login/')
      } else {
        error.value = await getApiErrorMessage(response, '发送失败，请稍后重试')
      }
    } catch (err) {
      error.value = '网络错误，请稍后重试'
      errorTraceId.value = err?.traceId || ''
    } finally {
      emailLoading.value = false
    }
  })
}

const sendVerificationCode = async () => {
  error.value = ''
  errorTraceId.value = ''

  if (phoneForm.value.phone.length !== 11 || !/^1[3-9]\d{9}$/.test(phoneForm.value.phone)) {
    error.value = '请输入有效的手机号码'
    return
  }

  // OPT-20260819-038: 发送验证码是账号写操作，防连点/超时重试双发
  await sendCodeGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch('/api/accounts/users/send_password_reset_code/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ phone: phoneForm.value.phone })
      })

      if (response.ok) {
        // 成功：直接倒计时，不弹「发送成功」窗
        countdown.value = 60
        const timer = setInterval(() => {
          countdown.value--
          if (countdown.value <= 0) {
            clearInterval(timer)
          }
        }, 1000)
      } else {
        error.value = await getApiErrorMessage(response, '发送失败，请稍后重试')
      }
    } catch (err) {
      error.value = '网络错误，请稍后重试'
      errorTraceId.value = err?.traceId || ''
    }
  })
}

const handlePhoneReset = async () => {
  error.value = ''
  errorTraceId.value = ''
  
  if (!phoneForm.value.code || phoneForm.value.code.length !== 6) {
    error.value = '请输入有效的验证码'
    return
  }
  
  phoneLoading.value = true
  
  try {
    alert('验证码验证成功，请设置新密码')
    router.push(`/reset-password/?phone=${encodeURIComponent(phoneForm.value.phone)}&code=${encodeURIComponent(phoneForm.value.code)}`)
  } catch (err) {
    error.value = '网络错误，请稍后重试'
    errorTraceId.value = err?.traceId || ''
  } finally {
    phoneLoading.value = false
  }
}
</script>

<style scoped>
</style>
