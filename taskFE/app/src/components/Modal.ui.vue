<template>
  <div 
    v-if="show" 
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999"
    @click="handleOverlayClick"
  >
    <div
      class="bg-white rounded-lg shadow-xl w-full max-w-md p-6 z-10000 relative"
      v-bind="traceId ? { 'data-traceId': traceId } : {}"
      @click.stop
      @keydown.enter="handleEnterKey"
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
        <!-- 错误文案节点挂 data-traceId，便于 Agent/Playwright 从高亮元素直接取链 -->
        <p v-bind="traceId ? { 'data-traceId': traceId } : {}">{{ message }}</p>
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
          v-for="(action, index) in additionalActions" 
          :key="index"
          type="button" 
          class="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary hover:bg-purple-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
          @click="handleAdditionalAction(action)"
        >
          {{ action.text }}
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
// 界面组件 - 专注于展示逻辑，使用函数式组件实现
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
  },
  additionalActions: {
    type: Array,
    default: () => []
  },
  /** 请求失败时的 traceId；有值时挂到弹层根节点 data-traceId */
  traceId: {
    type: String,
    default: ''
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

const handleAdditionalAction = (action) => {
  if (action.href) {
    if (typeof window !== 'undefined') window.location.assign(action.href)
    return
  }
  if (action.callback) {
    action.callback()
  }
}

const handleEnterKey = (event) => {
  // 文本框内回车应换行，不触发弹窗确认
  const tag = (event.target?.tagName || '').toLowerCase()
  if (tag === 'textarea') return
  handleConfirm()
}

const handleOverlayClick = () => {
  // 门禁类 alert（showCloseButton=false）禁止点遮罩关掉
  if (props.type === 'alert' && props.showCloseButton) {
    close()
  }
}
</script>