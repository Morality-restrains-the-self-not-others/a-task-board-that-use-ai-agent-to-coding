<template>
  <select
    :value="modelValue"
    :data-alias="dataAlias"
    :aria-label="ariaLabel"
    :class="inputClass"
    @change="$emit('update:modelValue', $event.target.value)"
  >
    <option value="">{{ placeholder }}</option>
    <option v-for="opt in options" :key="opt.value" :value="opt.value">
      {{ opt.label }}
    </option>
  </select>
</template>

<script setup>
// OPT-20260807-046 共用表头下拉过滤（分类/类型/支付来源等）。
// 受控 select：:value + update:modelValue（@change）。消费方把值映射到
// 自己的 update:filter(key, value) + apply 语义。
defineProps({
  modelValue: { type: [String, Number], default: '' },
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: '全部' },
  ariaLabel: { type: String, default: '' },
  dataAlias: { type: String, default: '' },
  inputClass: {
    type: String,
    default:
      'w-full px-2.5 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
  },
})

defineEmits(['update:modelValue'])
</script>
