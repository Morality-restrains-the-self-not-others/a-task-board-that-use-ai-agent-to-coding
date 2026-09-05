<template>
  <div class="flex space-x-4">
    <div class="flex items-center">
      <input
        type="checkbox"
        :id="prefix + '-is-tenant'"
        v-model="form.is_tenant"
        :disabled="form.is_tester"
        class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded disabled:opacity-50"
      >
      <label :for="prefix + '-is-tenant'" class="ml-2 block text-sm text-gray-700">租户</label>
    </div>
    <div class="flex items-center">
      <input
        type="checkbox"
        :id="prefix + '-is-tester'"
        v-model="form.is_tester"
        class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded"
        @change="$emit('tester-change')"
      >
      <label :for="prefix + '-is-tester'" class="ml-2 block text-sm text-gray-700">测试</label>
    </div>
  </div>
</template>

<script setup>
defineProps({
  form: { type: Object, required: true },
  prefix: { type: String, required: true },
})
defineEmits(['tester-change'])
</script>
