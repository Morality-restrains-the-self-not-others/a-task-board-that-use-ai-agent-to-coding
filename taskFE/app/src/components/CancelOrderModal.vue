<template>
  <div class="app-modal-overlay z-50 flex items-center justify-center bg-black/50 p-4" @click.self="$emit('close')">
    <div class="bg-white rounded-lg max-w-sm w-full p-6 shadow-xl" @keydown.enter="$emit('confirm')">
      <h3 class="text-lg font-semibold mb-4">确认取消订单</h3>
      <p class="text-sm text-gray-600 mb-2">
        订单号：<strong class="break-all">{{ order?.order_number }}</strong>
      </p>
      <p class="text-sm text-gray-600 mb-4">
        金额：<strong>{{ order?.total_yuan }} 元</strong>
      </p>
      <p class="text-sm text-red-500 mb-4">取消后不可恢复，确定要取消该订单吗？</p>
      <div class="flex gap-2">
        <button
          type="button"
          class="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm"
          @click="$emit('close')"
        >
          返回
        </button>
        <button
          type="button"
          class="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 text-sm disabled:opacity-50"
          :disabled="cancelling"
          @click="$emit('confirm')"
        >
          {{ cancelling ? '取消中…' : '确认取消' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  order: { type: Object, default: null },
  cancelling: { type: Boolean, default: false },
})

defineEmits(['close', 'confirm'])
</script>
