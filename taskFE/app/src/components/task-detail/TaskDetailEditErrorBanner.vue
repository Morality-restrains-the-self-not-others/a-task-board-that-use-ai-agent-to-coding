<template>
  <div
    v-if="message"
    ref="bannerRef"
    class="p-3 bg-red-50 border border-red-200 rounded-lg"
    role="alert"
    aria-live="assertive"
    data-testid="task-edit-error-banner"
  >
    <p class="text-sm font-medium text-red-800 mb-1">保存失败</p>
    <p class="text-sm text-red-700 whitespace-pre-wrap leading-snug">{{ message }}</p>
  </div>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'

const props = defineProps({
  message: { type: String, default: '' },
})

const bannerRef = ref(null)

watch(
  () => props.message,
  async (message) => {
    if (!message) return
    await nextTick()
    bannerRef.value?.scrollIntoView?.({ behavior: 'smooth', block: 'nearest' })
  },
)
</script>
