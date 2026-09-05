<template>
  <p
    class="mt-2 text-xs flex items-start gap-2 text-amber-700"
    data-testid="comment-layer-ztree-loading-error"
    v-bind="traceBind"
  >
    <span class="shrink-0 text-amber-600 mt-0.5" aria-hidden="true">!</span>
    <span>{{ hint }}</span>
  </p>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  hint: { type: String, required: true },
  traceId: { type: String, default: '' },
})

const traceBind = computed(() => {
  const t = String(props.traceId || '').trim()
  return t ? { 'data-traceId': t } : {}
})
</script>
