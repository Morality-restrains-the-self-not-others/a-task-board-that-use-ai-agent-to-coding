<template>
  <div class="flex flex-col items-center justify-center min-h-full py-12 px-4">
    <div class="max-w-md w-full bg-white p-10 rounded-2xl shadow-xl text-center">
      <h1 class="text-2xl font-bold text-text mb-4">邮件邀请退订</h1>
      <p v-if="resubscribed" class="text-text">
        您已恢复接收邀请邮件。之后他人邀请该邮箱时将继续发送邮件。
      </p>
      <p v-else-if="ok" class="text-text">
        您已退订邮件邀请。之后他人邀请该邮箱时将不再发送邮件，需手动复制邀请链接给您。
      </p>
      <p v-else class="text-text">
        退订链接无效或已过期。如需退订，请使用邀请邮件中的退订按钮。
      </p>

      <!-- OPT-20260829-002: 误点退订后的恢复入口（仅退订成功且带回 token 时展示） -->
      <div v-if="ok && !resubscribed && hasToken" class="mt-6">
        <button
          type="button"
          class="btn btn-primary w-full"
          data-testid="email-resubscribe-btn"
          :disabled="busy"
          @click="resubscribe"
        >
          {{ busy ? '处理中…' : '重新接收邀请邮件' }}
        </button>
        <p v-if="error" class="text-error text-sm mt-2" :data-traceId="errorTraceId || undefined">{{ error }}</p>
      </div>

      <p class="mt-8">
        <a href="/auth/login/" class="text-primary hover:underline">返回登录</a>
      </p>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { createClickGuard } from '../utils/clickGuard'
import { extractTraceId } from '../utils/traceId'

const route = useRoute()
const ok = computed(() => String(route.query.ok || '') === '1')
const token = computed(() => String(route.query.token || '').trim())
const hasToken = computed(() => token.value !== '')

const resubscribed = ref(false)
const busy = ref(false)
const error = ref('')
const errorTraceId = ref('')
const guard = createClickGuard()

const resubscribe = async () => {
  const result = await guard.run(async ({ idempotencyKey }) => {
    busy.value = true
    error.value = ''
    errorTraceId.value = ''
    try {
      const r = await apiFetch(`/api/public/email-resubscribe/?token=${encodeURIComponent(token.value)}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Idempotency-Key': idempotencyKey,
        },
      })
      const data = await r.json().catch(() => ({}))
      if (!r.ok) {
        error.value = data.message || data.error || '操作失败，请稍后重试'
        errorTraceId.value = extractTraceId(r) || extractTraceId(data) || ''
        return false
      }
      resubscribed.value = true
      return true
    } catch (e) {
      error.value = e.message || '操作失败，请稍后重试'
      errorTraceId.value = extractTraceId(e) || ''
      return false
    } finally {
      busy.value = false
    }
  })
  if (result.skipped) return
}
</script>
