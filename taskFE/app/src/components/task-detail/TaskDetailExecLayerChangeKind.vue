<template>
  <span class="shrink-0 text-gray-500">{{ text }}</span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  kind: { type: String, default: '' },
})

const text = computed(() => {
  if (props.kind === 'modified') return '修改'
  if (props.kind === 'added') return '新增'
  if (props.kind === 'removed') return '删除'
  return props.kind || '未知'
})
</script>
