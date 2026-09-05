<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6">
    <h2 class="text-lg font-semibold text-text mb-4">身份绑定（手机号）</h2>
    <p class="text-sm text-text-light mb-4">
      用于登录、安全校验与支付验证等。更换号码时，请先填写<strong class="font-medium text-text">新的手机号</strong>并完成短信验证；成功后新号码绑定到本账号，原号码自动解绑。号码若已在其他账号上，验证码通过后可确认转移到本账号。
    </p>
    <div v-if="hasPhone" class="space-y-4">
      <p class="text-sm text-text">
        当前绑定：<span class="font-medium">{{ phoneMasked || '—' }}</span>
      </p>
      <div class="rounded-lg border border-border bg-gray-50/80 p-4 space-y-3">
        <p class="text-sm font-medium text-text">更换手机号</p>
        <div class="flex flex-wrap gap-2 items-center">
          <div class="flex rounded-lg border border-border focus-within:ring-2 focus-within:ring-primary relative bg-white">
            <div class="relative z-30 shrink-0 border-r border-border bg-gray-50" :title="countryError ? '区域限制暂不可用，显示全部地区' : ''">
              <select
                v-model="replacePhonePrefix"
                class="block w-full min-w-[6.5rem] max-h-48 py-2 pl-2 pr-8 text-sm text-text bg-transparent border-0 appearance-none cursor-pointer"
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
              v-model="replacePhoneNational"
              type="tel"
              inputmode="numeric"
              :maxlength="replaceNationalMaxLen"
              autocomplete="tel-national"
              :placeholder="replaceNationalPlaceholder"
              class="w-40 min-w-0 px-3 py-2 border-0 text-sm focus:outline-none bg-transparent"
              @input="onReplacePhoneNationalInput"
              @paste="onReplacePhoneNationalPaste"
            >
          </div>
          <button
            type="button"
            :disabled="isSendingReplaceSms || replaceSmsCountdown > 0 || !isReplacePhoneValid"
            class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            @click="sendReplaceSms"
          >
            {{ replaceSmsCountdown > 0 ? `${replaceSmsCountdown} 秒后可重发` : (isSendingReplaceSms ? '发送中…' : '发送验证码') }}
          </button>
        </div>
        <div class="flex flex-wrap gap-2 items-center">
          <input
            v-model="replaceSmsCode"
            type="text"
            inputmode="numeric"
            maxlength="6"
            autocomplete="one-time-code"
            placeholder="6 位短信验证码"
            class="w-40 px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary bg-white"
            @keyup.enter="submitReplacePhone(false)"
          >
          <button
            type="button"
            :disabled="isReplacingPhone || replaceSmsCode.trim().length !== 6 || !isReplacePhoneValid"
            class="px-4 py-2 rounded-lg border border-primary text-primary text-sm hover:bg-primary/10 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            :aria-busy="isReplacingPhone ? 'true' : 'false'"
            @click="submitReplacePhone(false)"
          >
            {{ isReplacingPhone ? '提交中…' : '验证并更换' }}
          </button>
        </div>
        <p v-if="replaceInlineMessage" class="text-sm text-text-light">{{ replaceInlineMessage }}</p>
        <p v-if="replaceInlineError" class="text-sm text-danger" :data-traceId="replaceInlineErrorTraceId || undefined">{{ replaceInlineError }}</p>
        <button
          v-if="replaceReclaimAvailable"
          type="button"
          :disabled="isReplacingPhone"
          class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:opacity-50"
          :aria-busy="isReplacingPhone ? 'true' : 'false'"
          @click="submitReplacePhone(true)"
        >
          {{ isReplacingPhone ? '转移中…' : '确认将号码转移到本账号' }}
        </button>
      </div>
    </div>
    <div v-else class="space-y-4">
      <p class="text-sm text-text-light">当前未绑定手机号，请填写手机号完成短信验证以绑定。</p>
      <div class="rounded-lg border border-border bg-gray-50/80 p-4 space-y-3">
        <p class="text-sm font-medium text-text">绑定手机号</p>
        <div class="flex flex-wrap gap-2 items-center">
          <div class="flex rounded-lg border border-border focus-within:ring-2 focus-within:ring-primary relative bg-white">
            <div class="relative z-30 shrink-0 border-r border-border bg-gray-50" :title="countryError ? '区域限制暂不可用，显示全部地区' : ''">
              <select
                v-model="bindPhonePrefix"
                class="block w-full min-w-[6.5rem] max-h-48 py-2 pl-2 pr-8 text-sm text-text bg-transparent border-0 appearance-none cursor-pointer"
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
              v-model="bindPhoneNational"
              type="tel"
              inputmode="numeric"
              :maxlength="bindNationalMaxLen"
              autocomplete="tel-national"
              :placeholder="bindNationalPlaceholder"
              class="w-40 min-w-0 px-3 py-2 border-0 text-sm focus:outline-none bg-transparent"
              @input="onBindPhoneNationalInput"
              @paste="onBindPhoneNationalPaste"
            >
          </div>
          <button
            type="button"
            :disabled="isSendingBindSms || bindSmsCountdown > 0 || !isBindPhoneValid"
            class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            @click="sendBindSms"
          >
            {{ bindSmsCountdown > 0 ? `${bindSmsCountdown} 秒后可重发` : (isSendingBindSms ? '发送中…' : '发送验证码') }}
          </button>
        </div>
        <div class="flex flex-wrap gap-2 items-center">
          <input
            v-model="bindSmsCode"
            type="text"
            inputmode="numeric"
            maxlength="6"
            autocomplete="one-time-code"
            placeholder="6 位短信验证码"
            class="w-40 px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary bg-white"
            @keyup.enter="submitBindPhone(false)"
          >
          <button
            type="button"
            :disabled="isBindingPhone || bindSmsCode.trim().length !== 6 || !isBindPhoneValid"
            class="px-4 py-2 rounded-lg border border-primary text-primary text-sm hover:bg-primary/10 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            :aria-busy="isBindingPhone ? 'true' : 'false'"
            @click="submitBindPhone(false)"
          >
            {{ isBindingPhone ? '提交中…' : '验证并绑定' }}
          </button>
        </div>
        <p v-if="bindInlineMessage" class="text-sm text-text-light">{{ bindInlineMessage }}</p>
        <p v-if="bindInlineError" class="text-sm text-danger" :data-traceId="bindInlineErrorTraceId || undefined">{{ bindInlineError }}</p>
        <button
          v-if="bindReclaimAvailable"
          type="button"
          :disabled="isBindingPhone"
          class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:opacity-50"
          :aria-busy="isBindingPhone ? 'true' : 'false'"
          @click="submitBindPhone(true)"
        >
          {{ isBindingPhone ? '转移中…' : '确认将号码转移到本账号' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useUserProfilePhoneBindingPanel } from '../composables/auth/useUserProfilePhoneBindingPanel.js'

defineProps({
  hasPhone: { type: Boolean, default: false },
  phoneMasked: { type: String, default: '' },
})

const emit = defineEmits(['profile-updated', 'message', 'error'])

const {
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
} = useUserProfilePhoneBindingPanel(emit)
</script>
