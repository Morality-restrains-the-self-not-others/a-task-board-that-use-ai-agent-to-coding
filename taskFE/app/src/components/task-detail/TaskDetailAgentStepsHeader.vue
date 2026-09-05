<template>
  <summary
    class="cursor-pointer select-none flex items-center gap-2 text-xs font-medium text-gray-600 list-none [&::-webkit-details-marker]:hidden"
  >
    <span
      class="inline-block text-gray-500 transition-transform duration-200 group-open/agent-steps:rotate-90"
      aria-hidden="true"
    >▶</span>
    <span>代理步骤</span>
    <span
      :class="[
        'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium leading-none',
        running ? 'bg-amber-100 text-amber-900' : 'bg-gray-200/80 text-gray-700',
      ]"
    >
      {{ running ? '执行中' : '已结束' }}
    </span>
  </summary>
</template>

<script setup>
defineProps({
  running: { type: Boolean, default: false },
})
</script>
