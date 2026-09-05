<template>
  <div v-if="show" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
      <ProgressSystemSettings 
        :workspaceId="workspaceId"
        :tenantId="tenantId"
        @close="handleClose"
      />
    </div>
  </div>
</template>

<script setup>
import ProgressSystemSettings from './ProgressSystemSettings.vue'

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  workspaceId: {
    type: String,
    default: ''
  },
  tenantId: {
    type: [String, Number],
    default: ''
  }
})

const emit = defineEmits(['close'])

const handleClose = () => {
  emit('close')
}
</script>

<style scoped>
</style>