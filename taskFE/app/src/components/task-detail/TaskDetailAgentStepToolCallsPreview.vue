<template>
  <div
    v-if="items.length"
    class="mt-1 space-y-1"
    data-testid="layer-agent-step-tool-calls-preview"
  >
    <p class="text-[10px] font-medium text-gray-500">工具调用</p>
    <template v-for="(tc, idx) in visibleItems" :key="`tc-${idx}-${tc.name}`">
      <div
        class="rounded border border-amber-100 bg-amber-50/50 px-2 py-1 text-[10px] text-gray-800 break-words"
      >
        <span class="font-mono text-amber-900">{{ tc.name }}</span>
        <span v-if="tc.hint" class="text-gray-600"> · {{ tc.hint }}</span>
      </div>
    </template>
    <button
      v-if="collapsedCount > 0 && !expanded"
      class="text-[10px] text-blue-600 hover:text-blue-800 hover:underline cursor-pointer border-0 bg-transparent p-0"
      data-testid="tool-calls-expand-more"
      @click="expanded = true"
    >
      +{{ collapsedCount }} 更多
    </button>
    <button
      v-if="expanded && items.length > defaultShow"
      class="text-[10px] text-blue-600 hover:text-blue-800 hover:underline cursor-pointer border-0 bg-transparent p-0"
      data-testid="tool-calls-collapse"
      @click="expanded = false"
    >
      收起
    </button>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  items: {
    type: Array,
    default: () => [],
  },
})

const defaultShow = 3
const expanded = ref(false)

const visibleItems = computed(() => {
  if (expanded.value || props.items.length <= defaultShow) return props.items
  return props.items.slice(0, defaultShow)
})

const collapsedCount = computed(() => {
  if (props.items.length <= defaultShow) return 0
  return props.items.length - defaultShow
})
</script>
