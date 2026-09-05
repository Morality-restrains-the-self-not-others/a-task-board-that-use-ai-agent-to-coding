<template>
  <div
    class="safe-md text-[11px] leading-snug text-gray-800 border border-gray-100 rounded bg-white/90 p-2 max-h-48 overflow-auto"
    v-html="html"
  />
</template>

<script setup>
import { computed } from 'vue'
import { renderSafeMarkdown } from '../../utils/safeMarkdown.js'
import 'highlight.js/styles/github.min.css'

const props = defineProps({
  source: { type: String, default: '' },
})

const html = computed(() => renderSafeMarkdown(props.source))
</script>

<style scoped>
.safe-md :deep(pre) {
  margin: 0.35em 0;
  padding: 0.5em;
  border-radius: 4px;
  overflow: auto;
  font-size: 10px;
  background: #f6f8fa;
}
.safe-md :deep(code) {
  font-size: 10px;
}
.safe-md :deep(p) {
  margin: 0.25em 0;
}
.safe-md :deep(ul),
.safe-md :deep(ol) {
  margin: 0.25em 0;
  padding-left: 1.25em;
}
.safe-md :deep(table) {
  font-size: 10px;
}
</style>
