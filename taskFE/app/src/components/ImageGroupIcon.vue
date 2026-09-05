<template>
  <img
    v-if="src"
    data-testid="image-group-icon"
    class="h-10 w-10 rounded-lg object-cover ring-1 ring-gray-200"
    :src="src"
    :alt="alt"
  >
  <span
    v-else
    data-testid="image-group-icon-placeholder"
    class="inline-block h-10 w-10 shrink-0 rounded-lg bg-gray-200 ring-1 ring-gray-200"
    aria-hidden="true"
  />
</template>

<script setup>
/** Anti-Replay-OK: display-only icon / placeholder, no write action. */
defineProps({
  src: { type: String, default: '' },
  alt: { type: String, default: '' },
})
</script>
