<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6">
    <h2 class="text-lg font-semibold text-text mb-4">身份绑定（邮箱）</h2>
    <p class="text-sm text-text-light mb-4">
      用于厂商门户等身份校验场景。绑定真实邮箱后，即可申请成为厂商门户、使用镜像市场的厂商入口。
    </p>
    <div v-if="hasEmail" class="space-y-4">
      <p class="text-sm text-text">
        当前绑定：<span class="font-medium">{{ email || '—' }}</span>
      </p>
      <div class="rounded-lg border border-border bg-gray-50/80 p-4 space-y-3">
        <p class="text-sm font-medium text-text">更换邮箱</p>
        <div class="flex flex-wrap gap-2 items-center">
          <input
            v-model.trim="replaceEmail"
            type="email"
            maxlength="255"
            autocomplete="email"
            placeholder="请输入新的邮箱地址"
            class="w-64 px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary bg-white"
            @input="clearInlineFeedback"
          >
          <button
            type="button"
            :disabled="isSendingCode || codeCountdown > 0"
            class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            @click="sendCode(replaceEmail)"
          >
            {{ codeCountdown > 0 ? `${codeCountdown} 秒后可重发` : (isSendingCode ? '发送中…' : '发送验证码') }}
          </button>
        </div>
        <div class="flex flex-wrap gap-2 items-center">
          <input
            v-model.trim="replaceCode"
            type="text"
            inputmode="numeric"
            maxlength="6"
            autocomplete="one-time-code"
            placeholder="6 位邮箱验证码"
            class="w-40 px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary bg-white"
            @keyup.enter="submitBind(replaceEmail, replaceCode, '已更换绑定邮箱')"
          >
          <button
            type="button"
            :disabled="isBinding || replaceCode.trim().length !== 6"
            class="px-4 py-2 rounded-lg border border-primary text-primary text-sm hover:bg-primary/10 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            @click="submitBind(replaceEmail, replaceCode, '已更换绑定邮箱')"
          >
            {{ isBinding ? '提交中…' : '验证并更换' }}
          </button>
        </div>
        <p v-if="inlineMessage" class="text-sm text-text-light">{{ inlineMessage }}</p>
        <p v-if="inlineError" class="text-sm text-danger" :data-traceId="inlineErrorTraceId || undefined">{{ inlineError }}</p>
      </div>
    </div>
    <div v-else class="space-y-4">
      <p class="text-sm text-text-light">当前未绑定邮箱，请填写邮箱并完成邮箱验证码验证以绑定。</p>
      <div class="rounded-lg border border-border bg-gray-50/80 p-4 space-y-3">
        <p class="text-sm font-medium text-text">绑定邮箱</p>
        <div class="flex flex-wrap gap-2 items-center">
          <input
            v-model.trim="bindEmail"
            type="email"
            maxlength="255"
            autocomplete="email"
            placeholder="请输入邮箱地址"
            class="w-64 px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary bg-white"
            @input="clearInlineFeedback"
          >
          <button
            type="button"
            :disabled="isSendingCode || codeCountdown > 0"
            class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            @click="sendCode(bindEmail)"
          >
            {{ codeCountdown > 0 ? `${codeCountdown} 秒后可重发` : (isSendingCode ? '发送中…' : '发送验证码') }}
          </button>
        </div>
        <div class="flex flex-wrap gap-2 items-center">
          <input
            v-model.trim="bindCode"
            type="text"
            inputmode="numeric"
            maxlength="6"
            autocomplete="one-time-code"
            placeholder="6 位邮箱验证码"
            class="w-40 px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary bg-white"
            @keyup.enter="submitBind(bindEmail, bindCode, '已成功绑定邮箱')"
          >
          <button
            type="button"
            :disabled="isBinding || bindCode.trim().length !== 6"
            class="px-4 py-2 rounded-lg border border-primary text-primary text-sm hover:bg-primary/10 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            @click="submitBind(bindEmail, bindCode, '已成功绑定邮箱')"
          >
            {{ isBinding ? '提交中…' : '验证并绑定' }}
          </button>
        </div>
        <p v-if="inlineMessage" class="text-sm text-text-light">{{ inlineMessage }}</p>
        <p v-if="inlineError" class="text-sm text-danger" :data-traceId="inlineErrorTraceId || undefined">{{ inlineError }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onUnmounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import { humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

defineProps({
  hasEmail: { type: Boolean, default: false },
  email: { type: String, default: '' },
})

// 绑定成功仅通知父组件（父组件自行重新拉取 profile，避免局部 payload 覆盖整页状态）
const emit = defineEmits(['profile-updated', 'message', 'error'])

const bindEmail = ref('')
const bindCode = ref('')
const replaceEmail = ref('')
const replaceCode = ref('')
const codeCountdown = ref(0)
let codeTimer = null
const isSendingCode = ref(false)
const isBinding = ref(false)
const inlineError = ref('')
const inlineErrorTraceId = ref('')
const inlineMessage = ref('')

const sendCodeGuard = createClickGuard()
const submitBindGuard = createClickGuard()

// 简单邮箱校验：须含 @ 与点号域；拒绝合成邮箱（sso-<id>@sso.invalid，RFC 2606 保留域）
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const isEmailValid = (value) =>
  EMAIL_RE.test(String(value || '').toLowerCase()) &&
  !String(value || '').toLowerCase().endsWith('@sso.invalid')

const clearInlineFeedback = () => {
  inlineError.value = ''
  inlineErrorTraceId.value = ''
  inlineMessage.value = ''
}

function startCodeCountdown() {
  codeCountdown.value = 60
  if (codeTimer) clearInterval(codeTimer)
  codeTimer = setInterval(() => {
    codeCountdown.value -= 1
    if (codeCountdown.value <= 0) {
      clearInterval(codeTimer)
      codeTimer = null
    }
  }, 1000)
}

const sendCode = async (emailValue) => {
  clearInlineFeedback()
  const email = String(emailValue || '').trim().toLowerCase()
  if (!isEmailValid(email)) {
    inlineError.value = '请输入有效邮箱地址'
    return
  }
  // OPT-20260819-038: 发送验证码是账号写操作，防连点/超时重试双发
  await sendCodeGuard.run(async ({ idempotencyKey }) => {
    isSendingCode.value = true
    try {
      const response = await apiFetch('/api/accounts/users/send_verification_code/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ email }),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        const raw = data?.detail || data?.error || data?.message
        inlineErrorTraceId.value = traceId || ''
        inlineError.value = humanizeRequestErrorMessage(
          typeof raw === 'string' ? raw : '发送失败',
        )
        return
      }
      inlineMessage.value = '验证码已发送，请查收邮箱'
      startCodeCountdown()
    } catch (error) {
      console.error('发送邮箱验证码失败:', error)
      inlineErrorTraceId.value = ''
      inlineError.value = humanizeRequestErrorMessage(error.message || '发送失败')
    } finally {
      isSendingCode.value = false
    }
  })
}

const submitBind = async (emailValue, codeValue, successMessage) => {
  clearInlineFeedback()
  emit('error', '')
  emit('message', '')
  const email = String(emailValue || '').trim().toLowerCase()
  if (!isEmailValid(email)) {
    inlineError.value = '请输入有效邮箱地址'
    return
  }
  if (String(codeValue || '').trim().length !== 6) {
    inlineError.value = '请输入 6 位验证码'
    return
  }
  // OPT-20260819-038: 绑定/更换邮箱是账号写操作，防连点/超时重试双发
  await submitBindGuard.run(async ({ idempotencyKey }) => {
    isBinding.value = true
    try {
      const response = await apiFetch('/api/accounts/users/bind_email/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ email, code: String(codeValue).trim() }),
      })
      const { data, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        const raw = data?.detail || data?.error || data?.message
        inlineErrorTraceId.value = traceId || ''
        inlineError.value = humanizeRequestErrorMessage(
          typeof raw === 'string' ? raw : '绑定失败',
        )
        return
      }
      bindEmail.value = ''
      bindCode.value = ''
      replaceEmail.value = ''
      replaceCode.value = ''
      emit('profile-updated', data)
      emit('message', successMessage)
    } catch (error) {
      console.error('绑定邮箱失败:', error)
      inlineErrorTraceId.value = ''
      inlineError.value = humanizeRequestErrorMessage(error.message || '绑定失败')
    } finally {
      isBinding.value = false
    }
  })
}

onUnmounted(() => {
  if (codeTimer) clearInterval(codeTimer)
})
</script>
