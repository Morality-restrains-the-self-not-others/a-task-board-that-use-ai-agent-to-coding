<template>
  <button @click="$emit('close')">Close</button>
</template>

<script setup>
defineEmits(['close']);
</script>
