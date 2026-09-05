<template>
  <div class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50" data-testid="impersonate-reason-modal">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">以该用户身份登录</h3>
        <button type="button" class="text-gray-500 hover:text-gray-700" aria-label="关闭" @click="$emit('cancel')">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>
      <p class="text-sm text-gray-600 mb-3">{{ IMPERSONATE_REASON_PROMPT }}</p>
      <label class="block text-sm font-medium text-gray-700 mb-1" for="impersonate-reason">理由</label>
      <textarea
        id="impersonate-reason"
        :value="reason"
        rows="4"
        maxlength="500"
        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
        :placeholder="IMPERSONATE_REASON_PLACEHOLDER"
        data-testid="impersonate-reason-input"
        @input="$emit('update:reason', $event.target.value)"
      />
      <p
        v-if="error"
        class="text-sm text-red-600 mt-2"
        data-testid="impersonate-reason-error"
        :data-trace-id="errorTraceId || undefined"
      >{{ error }}</p>
      <div class="flex justify-end space-x-3 mt-6">
        <button type="button" class="px-4 py-2 text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200" :disabled="pending" @click="$emit('cancel')">
          取消
        </button>
        <button
          type="button"
          class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-60"
          :disabled="pending"
          :aria-busy="pending ? 'true' : 'false'"
          data-testid="impersonate-reason-confirm"
          @click="$emit('confirm')"
        >
          {{ pending ? '正在登录...' : '确认登录' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script>
// 与 taskAuth/domain/impersonation.go forbiddenImpersonationReasons 保持同步：
// 后端对逐字重复以下文案的 reason 返回 400，前端据此提前拦截。
export const IMPERSONATE_REASON_PROMPT = '请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。'
export const IMPERSONATE_REASON_PLACEHOLDER = '例如：排查线上工单 T-12345'
export const FORBIDDEN_IMPERSONATION_REASONS = new Set([IMPERSONATE_REASON_PROMPT, IMPERSONATE_REASON_PLACEHOLDER])
</script>

<script setup>
defineProps({
  reason: { type: String, default: '' },
  pending: { type: Boolean, default: false },
  error: { type: String, default: '' },
  errorTraceId: { type: String, default: '' },
})
defineEmits(['update:reason', 'confirm', 'cancel'])
</script>
