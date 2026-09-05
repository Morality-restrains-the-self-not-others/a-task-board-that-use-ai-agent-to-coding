<template>
  <div
    v-if="open"
    class="app-modal-overlay z-50 flex items-center justify-center bg-black/50 p-4"
    data-testid="refund-approve-modal"
    @click.self="emit('cancel')"
  >
    <div class="bg-white rounded-lg max-w-2xl w-full p-6 shadow-xl max-h-[90vh] overflow-y-auto">
      <h3 class="text-lg font-semibold text-gray-900 mb-2">批准退款</h3>
      <p class="text-sm text-gray-600 mb-4">
        申请 ID：<span class="font-mono font-medium">{{ target?.id }}</span>
      </p>
      <div class="mb-4">
        <label class="block text-sm font-medium text-gray-700 mb-1">租户申请原因</label>
        <p
          class="text-sm text-gray-800 bg-gray-50 border border-gray-200 rounded-md px-3 py-2 whitespace-pre-wrap break-words"
          data-testid="refund-approve-application-reason"
        >
          {{ target?.reason?.trim() ? target.reason : '—' }}
        </p>
      </div>
      <RefundOrderConsumptionBlock
        :loading="loading"
        :error="error"
        :error-trace-id="errorTraceId"
        :consumption="consumption"
      />
      <label class="block text-sm font-medium text-gray-700 mb-1">
        退款原因 <span class="text-red-500">*</span>
      </label>
      <textarea
        :value="reason"
        rows="3"
        data-testid="refund-approve-reason-input"
        class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-1"
        :class="reasonError ? 'border-red-400' : ''"
        placeholder="审批通过后写入账单的退款原因"
        @input="emit('update:reason', $event.target.value.trim())"
      />
      <p v-if="reasonError" class="text-xs text-red-500 mb-3">{{ reasonError }}</p>
      <p v-else class="text-xs text-gray-400 mb-4">可修改为其他退款原因，默认「订单退款」</p>
      <div
        v-if="actionError"
        class="mb-4 p-3 bg-red-50 text-red-800 rounded-lg text-sm"
        data-testid="refund-approve-action-error"
        v-bind="actionErrorTraceId ? { 'data-traceId': actionErrorTraceId } : {}"
      >
        {{ actionError }}
      </div>
      <div class="flex justify-end gap-3">
        <button
          type="button"
          class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg"
          :disabled="busy"
          @click="emit('cancel')"
        >
          取消
        </button>
        <button
          type="button"
          data-testid="refund-approve-confirm-btn"
          class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
          :disabled="!reason || busy"
          :aria-busy="busy ? 'true' : 'false'"
          @click="emit('confirm')"
        >
          {{ busy ? '处理中…' : '确认批准' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { watch } from 'vue'
import { useAdminRefundOrderConsumption } from '../../composables/useAdminRefundOrderConsumption.js'
import RefundOrderConsumptionBlock from './RefundOrderConsumptionBlock.vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  target: { type: Object, default: null },
  reason: { type: String, default: '' },
  reasonError: { type: String, default: '' },
  busy: { type: Boolean, default: false },
  actionError: { type: String, default: '' },
  actionErrorTraceId: { type: String, default: '' },
})

const emit = defineEmits(['cancel', 'confirm', 'update:reason'])

const { consumption, loading, error, errorTraceId, loadFor, clear } = useAdminRefundOrderConsumption()

watch(
  () => [props.open, props.target],
  ([open, target]) => {
    if (open && target) {
      loadFor(target)
      return
    }
    clear()
  },
  { immediate: true },
)
</script>
