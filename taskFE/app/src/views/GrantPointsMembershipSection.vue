<template>
  <div class="border border-gray-200 rounded-lg p-4 bg-gray-50/50">
    <label class="block text-sm font-medium text-gray-700 mb-1">VIP 等级</label>
    <p class="text-xs text-gray-500 mb-2">
      不修改则保持租户当前等级。管理员设定后会锁定，累计消费满 100 元也不会自动改等级。
    </p>
    <p v-if="currentLabel" class="text-sm text-gray-700 mb-2" data-testid="grant-membership-current">
      当前：<span class="font-medium">{{ currentLabel }}</span>
      <span v-if="locked" class="ml-2 text-xs text-amber-800">已锁定</span>
    </p>
    <select
      :value="modelValue"
      data-testid="grant-membership-tier"
      class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option value="">不修改</option>
      <option value="normal">普通会员</option>
      <option value="vip1">VIP1</option>
    </select>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  currentTier: { type: String, default: '' },
  locked: { type: Boolean, default: false },
})

defineEmits(['update:modelValue'])

const labels = { normal: '普通会员', vip1: 'VIP1' }
const currentLabel = computed(() => labels[props.currentTier] || '')
</script>
