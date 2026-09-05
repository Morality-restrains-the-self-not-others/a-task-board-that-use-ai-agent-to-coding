<template>
  <div>
    <button @click="$emit('close')">Close</button>
  </div>
</template>

<script setup>
const emit = defineEmits(['close'])
</script>
