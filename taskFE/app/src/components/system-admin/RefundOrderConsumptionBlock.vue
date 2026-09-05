<template>
  <div
    class="mb-4 max-h-64 overflow-y-auto border border-gray-200 rounded-md p-3"
    data-testid="refund-decision-consumption"
  >
    <p v-if="loading" class="text-sm text-gray-500">正在加载资源消耗…</p>
    <p
      v-else-if="error"
      class="text-sm text-red-700"
      :data-traceId="errorTraceId || undefined"
    >
      {{ error }}
    </p>
    <OrderResourceConsumption v-else :consumption="consumption" always-show />
  </div>
</template>

<script setup>
import OrderResourceConsumption from '../OrderResourceConsumption.vue'

defineProps({
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' },
  errorTraceId: { type: String, default: '' },
  consumption: { type: Object, default: null },
})
</script>
