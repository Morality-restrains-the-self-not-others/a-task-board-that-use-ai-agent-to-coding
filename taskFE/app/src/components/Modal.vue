<template>
  <div 
    v-if="show" 
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999"
    @click="handleOverlayClick"
  >
    <div 
      class="bg-white rounded-lg shadow-xl w-full max-w-md p-6 z-10000 relative"
      @click.stop
    >
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-xl font-bold text-gray-900">{{ title }}</h3>
        <button 
          v-if="showCloseButton" 
          class="text-gray-500 hover:text-gray-700"
          @click="close"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>
      <div class="space-y-4">
        <p>{{ message }}</p>
      </div>
      <div class="mt-6 flex justify-end space-x-3">
        <button 
          v-if="type === 'confirm'" 
          type="button" 
          class="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
          @click="handleCancel"
        >
          {{ cancelText }}
        </button>
        <button 
          type="button" 
          class="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
          @click="handleConfirm"
        >
          {{ confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, defineProps, defineEmits } from 'vue'

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default: '提示'
  },
  message: {
    type: String,
    default: ''
  },
  type: {
    type: String,
    default: 'alert', // alert or confirm
    validator: (value) => ['alert', 'confirm'].includes(value)
  },
  showCloseButton: {
    type: Boolean,
    default: true
  },
  confirmText: {
    type: String,
    default: '确定'
  },
  cancelText: {
    type: String,
    default: '取消'
  }
})

const emit = defineEmits(['close', 'confirm', 'cancel'])

const close = () => {
  emit('close')
}

const handleConfirm = () => {
  emit('confirm')
  emit('close')
}

const handleCancel = () => {
  emit('cancel')
  emit('close')
}

const handleOverlayClick = () => {
  if (props.type === 'alert') {
    close()
  }
}
</script>