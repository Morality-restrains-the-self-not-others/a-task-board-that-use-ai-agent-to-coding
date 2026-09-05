<template>
  <div
    v-if="open"
    class="app-modal-overlay z-50 flex items-center justify-center bg-black/50 p-4"
    data-testid="refund-reject-modal"
    @click.self="emit('cancel')"
  >
    <div class="bg-white rounded-lg max-w-2xl w-full p-6 shadow-xl max-h-[90vh] overflow-y-auto">
      <h3 class="text-lg font-semibold text-gray-900 mb-2">拒绝退款</h3>
      <p class="text-sm text-gray-600 mb-4">
        申请 ID：<span class="font-mono font-medium">{{ target?.id }}</span>
      </p>
      <RefundOrderConsumptionBlock
        :loading="loading"
        :error="error"
        :error-trace-id="errorTraceId"
        :consumption="consumption"
      />
      <label class="block text-sm font-medium text-gray-700 mb-1">拒绝原因（可选）</label>
      <textarea
        :value="note"
        rows="3"
        data-testid="refund-reject-note-input"
        class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-4"
        placeholder="填写拒绝原因（可为空）"
        @input="emit('update:note', $event.target.value.trim())"
      />
      <div
        v-if="actionError"
        class="mb-4 p-3 bg-red-50 text-red-800 rounded-lg text-sm"
        data-testid="refund-reject-action-error"
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
          data-testid="refund-reject-confirm-btn"
          class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
          :disabled="busy"
          :aria-busy="busy ? 'true' : 'false'"
          @click="emit('confirm')"
        >
          {{ busy ? '处理中…' : '确认拒绝' }}
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
  note: { type: String, default: '' },
  busy: { type: Boolean, default: false },
  actionError: { type: String, default: '' },
  actionErrorTraceId: { type: String, default: '' },
})

const emit = defineEmits(['cancel', 'confirm', 'update:note'])

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
