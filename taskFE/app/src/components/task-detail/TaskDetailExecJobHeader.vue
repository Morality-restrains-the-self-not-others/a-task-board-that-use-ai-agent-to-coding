<template>
  <div>
    <p class="text-xs text-gray-700 mb-1">
      <span class="font-medium">{{ job?.id }}</span>
      · {{ job?.status || '—' }}
      <span v-if="job?.command" class="text-gray-500">
        · {{ jobCommandHead }}</span>
    </p>
    <p
      v-if="stepsNote"
      class="text-xs text-amber-800 mb-1 whitespace-pre-wrap"
    >
      {{ stepsNote }}
    </p>
  </div>
</template>

<script setup>
defineProps({
  job: {
    type: Object,
    default: null,
  },
  jobCommandHead: { type: String, default: '' },
  stepsNote: { type: String, default: '' },
})
</script>
