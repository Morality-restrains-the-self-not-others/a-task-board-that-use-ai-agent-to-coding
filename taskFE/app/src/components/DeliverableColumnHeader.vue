<template>
  <div class="deliverable-column-header mb-3">
    <h3 class="font-semibold text-text flex items-center min-w-0 px-1">
      <span
        class="inline-block w-3 h-3 rounded-full flex-shrink-0"
        :style="{ backgroundColor: column.color }"
      ></span>
      <span class="ml-2 truncate">{{ column.name }}</span>
      <span
        class="ml-2 px-2.5 py-0.5 rounded-full text-xs font-medium flex-shrink-0"
        :style="{
          backgroundColor: `${column.color}20`,
          color: column.color
        }"
      >
        {{ taskCount }}
      </span>
    </h3>
  </div>
</template>

<script setup>
defineProps({
  column: {
    type: Object,
    required: true,
  },
  taskCount: {
    type: Number,
    default: 0,
  },
})
</script>
