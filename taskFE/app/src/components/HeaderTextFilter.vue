<template>
  <input
    :value="modelValue"
    type="text"
    :data-alias="dataAlias"
    :aria-label="ariaLabel"
    :placeholder="placeholder"
    :class="inputClass"
    @input="$emit('update:modelValue', $event.target.value)"
  >
</template>

<script setup>
// 表头文本过滤：受控 input，消费方防抖后发请求。
defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '筛选' },
  ariaLabel: { type: String, default: '' },
  dataAlias: { type: String, default: '' },
  inputClass: {
    type: String,
    default:
      'w-full min-w-[6rem] px-2.5 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
  },
})

defineEmits(['update:modelValue'])
</script>
