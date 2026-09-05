<template>
  <ModalUi 
    :show="state.show" 
    :title="state.title"
    :message="state.message"
    :type="state.type"
    :confirm-text="state.confirmText"
    :cancel-text="state.cancelText"
    @close="close"
    @confirm="handleConfirm"
    @cancel="handleCancel"
  />
</template>

<script setup>
// 逻辑组件 - 负责业务逻辑处理，允许包含副作用
import { reactive } from 'vue'
import ModalUi from './Modal.ui.vue'

// 响应式状态
const state = reactive({
  show: false,
  title: '提示',
  message: '',
  type: 'alert', // alert or confirm
  confirmText: '确定',
  cancelText: '取消',
  onConfirm: null,
  onCancel: null
})

// 显示提示弹窗
const alert = (message, title = '提示', options = {}) => {
  return new Promise((resolve) => {
    state.show = true
    
    // 处理message为对象的情况
    if (typeof message === 'object' && message !== null) {
      // 如果message对象有message属性，使用它
      if (message.message) {
        state.message = message.message
      } else {
        // 否则将对象转换为字符串
        state.message = JSON.stringify(message)
      }
      // 如果message对象有title属性，使用它
      if (message.title) {
        state.title = message.title
      } else {
        state.title = title
      }
      // 处理其他选项
      if (message.onConfirm) {
        state.onConfirm = message.onConfirm
      } else {
        state.onConfirm = resolve
      }
    } else {
      // 正常情况，message是字符串
      state.title = title
      state.message = message
      state.onConfirm = resolve
    }
    
    state.type = 'alert'
    state.onCancel = null
    
    // 如果有自动关闭选项
    if (options.autoClose) {
      setTimeout(() => {
        close()
        resolve()
      }, options.autoClose)
    }
  })
}

// 显示确认弹窗
const confirm = (message, title = '确认', confirmText = '确定', cancelText = '取消') => {
  return new Promise((resolve, reject) => {
    state.show = true
    state.title = title
    state.message = message
    state.type = 'confirm'
    state.confirmText = confirmText
    state.cancelText = cancelText
    state.onConfirm = resolve
    state.onCancel = reject
  })
}

// 关闭弹窗
const close = () => {
  state.show = false
}

// 处理确认
const handleConfirm = () => {
  if (state.onConfirm) {
    state.onConfirm(true)
  }
  close()
}

// 处理取消
const handleCancel = () => {
  if (state.onCancel) {
    state.onCancel(false)
  }
  close()
}

// 导出方法
defineExpose({
  alert,
  confirm,
  close,
  handleConfirm,
  handleCancel
})
</script>