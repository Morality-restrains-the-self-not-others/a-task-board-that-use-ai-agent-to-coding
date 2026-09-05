<template>
  <div>
    <button @click="$emit('close')">Close</button>
  </div>
</template>

<script>
export default {
  name: 'SimpleComponent',
  emits: ['close']
}
</script>
