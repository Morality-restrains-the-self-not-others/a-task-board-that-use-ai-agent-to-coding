<template>
  <div
    class="app-modal-overlay z-50 flex items-center justify-center bg-black/50 p-4"
    data-testid="billing-refund-confirm-modal"
    @click.self="onClose"
  >
    <div class="bg-white rounded-lg max-w-md w-full p-6 shadow-xl">
      <h3 class="text-lg font-semibold text-gray-900 mb-2">申请退款</h3>
      <p class="text-sm text-gray-600 mb-3">
        订单号：<span class="font-mono font-medium">{{ order?.order_number }}</span>
        &nbsp;|&nbsp;金额：<span class="font-medium">{{ order?.total_yuan }} 元</span>
      </p>
      <p
        class="text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded-md px-3 py-2 mb-3"
        data-testid="billing-refund-consumed-notice"
      >
        {{ consumedNotice }}
      </p>
      <OrderResourceConsumption
        v-if="consumption"
        class="mb-3"
        :consumption="consumption"
      />
      <p class="text-sm text-gray-500 mb-4">
        提交后将按该订单实付金额进入审批；平台通过后按原支付渠道（微信/PayPal）退回。
      </p>
      <label class="block text-sm font-medium text-gray-700 mb-1">
        退款原因 <span class="text-red-500">*</span>
      </label>
      <textarea
        v-model.trim="refundReason"
        rows="3"
        class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-1"
        :class="refundReasonError ? 'border-red-400' : ''"
        placeholder="请说明申请退款的原因"
      />
      <p v-if="refundReasonError" class="text-xs text-red-500 mb-3">{{ refundReasonError }}</p>
      <p v-else class="text-xs text-gray-400 mb-4">退款原因不能为空</p>
      <div
        v-if="applyError"
        class="mb-4 p-3 bg-red-50 text-red-800 rounded-lg text-sm"
        :data-traceId="applyErrorTraceId || undefined"
      >
        {{ applyError }}
      </div>
      <div class="flex justify-end gap-3">
        <button
          type="button"
          class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg"
          :disabled="busy"
          @click="onClose"
        >
          取消
        </button>
        <button
          type="button"
          data-testid="billing-refund-confirm-btn"
          class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
          :disabled="!refundReason || busy"
          :aria-busy="busy ? 'true' : 'false'"
          @click="onConfirm"
        >
          {{ busy ? '提交中…' : '确认申请' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { createClickGuard } from '../utils/clickGuard.js'
import { BILLING_REFUND_CONSUMED_NOTICE } from '../utils/billingRefundCopy.js'
import OrderResourceConsumption from './OrderResourceConsumption.vue'

const props = defineProps({
  order: { type: Object, default: null },
  consumption: { type: Object, default: null },
  applyError: { type: String, default: '' },
  applyErrorTraceId: { type: String, default: '' },
  applying: { type: Boolean, default: false },
  submit: { type: Function, required: true },
})

const emit = defineEmits(['close'])
const refundReason = ref('')
const refundReasonError = ref('')
const confirmGuard = createClickGuard()
const consumedNotice = BILLING_REFUND_CONSUMED_NOTICE
const busy = computed(() => props.applying || confirmGuard.isBusy())

const onClose = () => {
  if (busy.value) return
  emit('close')
}

const onConfirm = async () => {
  if (!refundReason.value) {
    refundReasonError.value = '请填写退款原因'
    return
  }
  refundReasonError.value = ''
  await confirmGuard.run(async ({ idempotencyKey }) => {
    const ok = await props.submit(refundReason.value, idempotencyKey)
    if (ok) emit('close')
  })
}
</script>
