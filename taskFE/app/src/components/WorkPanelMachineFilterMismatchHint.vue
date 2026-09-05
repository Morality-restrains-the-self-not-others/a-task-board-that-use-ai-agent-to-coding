<template>
  <div
    v-if="visible"
    data-alias="machine-filter-mismatch-hint"
    class="px-6 py-2 bg-amber-50 border-b border-amber-100 text-sm text-amber-900"
  >
    摘要显示有已启动/闲置机器，但当前看板没有对应任务（任务列表可能未加载，或机器绑定的任务不在列表中）。
    <button
      type="button"
      class="ml-2 underline text-amber-950 hover:text-amber-800"
      data-alias="machine-filter-mismatch-clear"
      @click="$emit('clear')"
    >清除机器过滤</button>
    <button
      type="button"
      class="ml-2 underline text-amber-950 hover:text-amber-800"
      data-alias="machine-filter-mismatch-reload"
      @click="$emit('reload')"
    >重新加载任务</button>
  </div>
</template>

<script setup>
defineProps({
  visible: { type: Boolean, default: false },
})
defineEmits(['clear', 'reload'])
</script>
