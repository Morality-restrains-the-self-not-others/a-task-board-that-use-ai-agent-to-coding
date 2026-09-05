<template>
  <div v-if="bound">
    <slot />
  </div>
  <div v-else class="space-y-3" data-testid="referral-mp-follow-gate">
    <p class="text-sm text-text">请先使用微信扫描下方服务号二维码并关注，系统绑定后再填写个人名称与简介。若已关注，请再扫一次本页二维码以完成绑定。</p>
    <p class="text-xs text-text-light">
      若当前账号尚未绑定微信登录，请先绑定；否则关注后无法用 unionId 锁定身份。绑定成功后请回到本页再点「我已关注」。
    </p>
    <p v-if="qrLoading" class="text-xs text-text-light text-center" data-testid="referral-mp-qr-loading">正在生成关注二维码...</p>
    <img
      v-if="qrSrc"
      :src="qrSrc"
      alt="服务号二维码"
      class="w-40 h-40 mx-auto border border-border rounded-lg bg-white p-1"
      data-testid="referral-mp-qr"
    />
    <p class="text-xs text-text-light text-center">关注后点击「我已关注」，无需刷新页面等待。</p>
    <div class="flex flex-wrap gap-3">
      <button
        type="button"
        class="bg-white text-text border border-border px-4 py-2 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
        data-testid="referral-mp-bind-wechat-btn"
        :disabled="bindBusy"
        :aria-busy="bindBusy ? 'true' : 'false'"
        @click="onBindWechat"
      >
        {{ bindBusy ? '跳转中...' : '绑定微信登录' }}
      </button>
      <button
        type="button"
        class="bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
        data-testid="referral-mp-followed-btn"
        :disabled="busy"
        :aria-busy="busy ? 'true' : 'false'"
        @click="onConfirm"
      >
        {{ busy ? '正在确认...' : '我已关注' }}
      </button>
    </div>
    <p
      v-if="bindError"
      class="text-sm text-red-500"
      data-testid="referral-mp-bind-error"
    >{{ bindError }}</p>
    <p
      v-if="qrError"
      class="text-sm text-red-500"
      data-testid="referral-mp-qr-error"
      :data-traceId="qrErrorTraceId || undefined"
    >{{ qrError }}</p>
    <p
      v-if="error"
      class="text-sm text-red-500"
      data-testid="referral-mp-follow-error"
      :data-traceId="errorTraceId || undefined"
    >{{ error }}</p>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { getActiveToken } from '../domain/auth/services/saved_accounts_store.js'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import { messageFromFailedResponse } from '../utils/httpError.js'

const props = defineProps({
  bound: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  error: { type: String, default: '' },
  errorTraceId: { type: String, default: '' },
})

const emit = defineEmits(['confirm'])
const confirmGuard = createClickGuard()
const bindGuard = createClickGuard()
const qrGuard = createClickGuard()
const bindBusy = ref(false)
const bindError = ref('')
const qrSrc = ref('')
const qrLoading = ref(false)
const qrError = ref('')
const qrErrorTraceId = ref('')

const issueQr = async () => {
  if (props.bound) return
  await qrGuard.run(async ({ idempotencyKey }) => {
    qrLoading.value = true
    qrError.value = ''
    qrErrorTraceId.value = ''
    try {
      const response = await apiFetch('/api/auth/wechat/mp/follow-qr/', {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { Accept: 'application/json' },
          idempotencyKey,
        ),
      })
      if (!response.ok) {
        qrError.value = messageFromFailedResponse(response, '无法生成关注二维码，请稍后重试')
        qrErrorTraceId.value = extractTraceId(response)
        return
      }
      const data = await response.json()
      qrSrc.value = String(data && data.qr_src ? data.qr_src : '')
      if (!qrSrc.value) {
        qrError.value = '无法生成关注二维码，请稍后重试'
      }
    } catch (error) {
      qrError.value = '无法生成关注二维码，请稍后重试'
      qrErrorTraceId.value = extractTraceId(error)
    } finally {
      qrLoading.value = false
    }
  })
}

onMounted(() => {
  issueQr()
})

watch(() => props.bound, (next, prev) => {
  if (prev && !next) {
    issueQr()
  }
})

const onConfirm = async () => {
  if (props.busy) return
  await confirmGuard.run(async () => {
    emit('confirm')
  })
}

const onBindWechat = async () => {
  await bindGuard.run(async () => {
    bindBusy.value = true
    bindError.value = ''
    try {
      const token = await getActiveToken()
      if (!token) {
        bindError.value = '登录状态已失效，请重新登录'
        return
      }
      document.cookie = `token=${encodeURIComponent(token)}; path=/api/auth/wechat; max-age=600; SameSite=Lax`
      window.location.href = '/api/auth/wechat/bind/?app=web&next=' + encodeURIComponent('/profile/referral/')
    } catch {
      bindError.value = '发起微信绑定失败，请稍后重试'
    } finally {
      bindBusy.value = false
    }
  })
}
</script>
