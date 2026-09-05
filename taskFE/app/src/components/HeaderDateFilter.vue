<template>
  <div class="flex flex-col gap-1.5">
    <input
      :value="startDate"
      type="date"
      :data-alias="startAlias"
      :aria-label="startLabel"
      :class="inputClass"
      @input="$emit('update:startDate', $event.target.value)"
      @change="$emit('commit', { key: 'startDate', value: $event.target.value })"
    >
    <input
      :value="endDate"
      type="date"
      :data-alias="endAlias"
      :aria-label="endLabel"
      :class="inputClass"
      @input="$emit('update:endDate', $event.target.value)"
      @change="$emit('commit', { key: 'endDate', value: $event.target.value })"
    >
  </div>
</template>

<script setup>
// OPT-20260807-046 共用表头日期过滤器。
// - 受控：:value + update:startDate/update:endDate（@input 即时同步父级状态）
// - 提交：@change 聚合 commit({key, value})，供「变更即自动应用」的消费方绑定
//   （如 BillingTransactionsTable 的 apply 语义），无需消费方区分输入/变更事件。
defineProps({
  startDate: { type: String, default: '' },
  endDate: { type: String, default: '' },
  startAlias: { type: String, default: '' },
  endAlias: { type: String, default: '' },
  startLabel: { type: String, default: '开始日期' },
  endLabel: { type: String, default: '结束日期' },
  inputClass: {
    type: String,
    default:
      'px-2.5 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
  },
})

defineEmits(['update:startDate', 'update:endDate', 'commit'])
</script>
