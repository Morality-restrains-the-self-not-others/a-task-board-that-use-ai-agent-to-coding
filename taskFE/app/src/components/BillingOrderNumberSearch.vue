<template>
  <div class="mb-6" data-testid="billing-order-number-search">
    <p class="text-xs text-gray-500 mb-2">
      可粘贴微信支付交易单号或商户订单号，精确查找本租户订单。
    </p>
    <div class="flex flex-wrap items-end gap-3">
      <label class="block text-sm text-gray-700">
        <span class="mb-1 block font-medium">交易单号</span>
        <input
          v-model="txnInput"
          type="text"
          class="w-64 border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
          placeholder="微信支付交易单号"
          data-testid="billing-order-txn-input"
          @keydown.enter="submitSearch"
        >
      </label>
      <label class="block text-sm text-gray-700">
        <span class="mb-1 block font-medium">商户单号</span>
        <input
          v-model="outTradeInput"
          type="text"
          class="w-64 border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
          placeholder="微信商户订单号"
          data-testid="billing-order-out-trade-input"
          @keydown.enter="submitSearch"
        >
      </label>
      <!-- Anti-Replay-OK: 只读列表查询；同步门闩防连点，无 Idempotency-Key -->
      <button
        type="button"
        class="px-4 py-2 bg-primary text-white text-sm rounded-lg hover:bg-primary/90 disabled:opacity-50"
        :disabled="searching"
        :aria-busy="searching ? 'true' : 'false'"
        data-testid="billing-order-number-search-btn"
        @click="submitSearch"
      >
        {{ searching ? '查询中...' : '查询' }}
      </button>
      <!-- Anti-Replay-OK: 只读清空筛选 -->
      <button
        type="button"
        class="px-4 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
        :disabled="searching"
        data-testid="billing-order-number-clear-btn"
        @click="submitClear"
      >
        清空
      </button>
    </div>
    <p
      v-if="inputError"
      class="mt-2 text-sm text-red-700"
      data-testid="billing-order-number-search-error"
    >{{ inputError }}</p>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { createClickGuard } from '../utils/clickGuard.js'

const props = defineProps({
  searching: { type: Boolean, default: false },
  // OPT-20260829-006: URL 深链预填的查询串；变化时回填输入框
  searchQuery: { type: String, default: '' },
})

const emit = defineEmits(['search', 'clear'])

const txnInput = ref('')
const outTradeInput = ref('')
const inputError = ref('')
const searchGuard = createClickGuard()
const clearGuard = createClickGuard()

watch(
  () => props.searchQuery,
  (q) => {
    const normalized = String(q || '').trim()
    const current = normalizeTradeNoInput(txnInput.value) || normalizeTradeNoInput(outTradeInput.value)
    if (!normalized || normalized === current) return
    txnInput.value = normalized
    outTradeInput.value = ''
  },
)

function normalizeTradeNoInput(raw) {
  return String(raw || '')
    .trim()
    .replace(/(?:^[\s`'"]+|[\s`'"]+$)/g, '')
    .trim()
}

const submitSearch = async () => {
  await searchGuard.run(async () => {
    const txn = normalizeTradeNoInput(txnInput.value)
    const outNo = normalizeTradeNoInput(outTradeInput.value)
    inputError.value = ''
    const q = txn || outNo
    if (!q) {
      inputError.value = '请输入交易单号或商户单号'
      return
    }
    emit('search', q)
  })
}

const submitClear = async () => {
  await clearGuard.run(async () => {
    txnInput.value = ''
    outTradeInput.value = ''
    inputError.value = ''
    emit('clear')
  })
}
</script>
