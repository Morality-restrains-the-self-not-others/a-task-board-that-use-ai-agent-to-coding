<template>
  <div class="bg-white border border-gray-200 rounded-lg p-6 mb-6" data-testid="order-number-paste-box">
    <label class="block text-sm font-medium text-gray-700 mb-1" for="order-number-paste">交易单号查询</label>
    <p class="text-xs text-gray-500 mb-3">
      可粘贴展示订单号、订单主键、微信支付单号或商户订单号（微信商户后台导出），精确查找，无需先选租户。
    </p>
    <div class="flex gap-2">
      <input
        id="order-number-paste"
        v-model="orderNumberInput"
        type="text"
        class="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
        placeholder="例如 WX878981209491800064 或 4500000359202608221274536815"
        data-testid="order-number-paste-input"
        @keydown.enter="submitSearch"
      >
      <!-- Anti-Replay-OK: 只读列表查询；同步门闩防连点，无 Idempotency-Key -->
      <button
        type="button"
        class="px-4 py-2 bg-gray-800 text-white text-sm rounded-lg hover:bg-gray-700 disabled:opacity-50"
        :disabled="searching"
        :aria-busy="searching ? 'true' : 'false'"
        data-testid="order-number-paste-jump"
        @click="submitSearch"
      >
        {{ searching ? '查询中...' : '查询' }}
      </button>
    </div>
    <p
      v-if="orderNumberError"
      class="mt-2 text-sm text-red-700"
      data-testid="order-number-parse-error"
    >{{ orderNumberError }}</p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { createClickGuard } from '../../utils/clickGuard.js'

defineProps({
  searching: { type: Boolean, default: false },
})

const emit = defineEmits(['search'])

const orderNumberInput = ref('')
const orderNumberError = ref('')
const searchGuard = createClickGuard()

function normalizeTradeNoInput(raw) {
  return String(raw || '')
    .trim()
    .replace(/(?:^[\s`'"]+|[\s`'"]+$)/g, '')
    .trim()
}

const submitSearch = async () => {
  await searchGuard.run(async () => {
    const raw = normalizeTradeNoInput(orderNumberInput.value)
    orderNumberError.value = ''
    if (!raw) {
      orderNumberError.value = '请输入交易单号'
      return
    }
    emit('search', raw)
  })
}
</script>
