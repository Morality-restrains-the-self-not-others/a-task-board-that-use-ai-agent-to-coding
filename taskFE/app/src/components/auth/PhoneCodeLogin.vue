<template>
  <div class="phone-code-login">
    <div>
      <label for="phone" class="block text-sm font-medium text-text-light mb-2">手机号</label>
      <input 
        type="tel" 
        id="phone" 
        v-model="phone" 
        name="phone" 
        autocomplete="tel" 
        required 
        class="input w-full"
        placeholder="请输入手机号"
        maxlength="11"
      >
    </div>
    <div class="relative">
      <label for="code" class="block text-sm font-medium text-text-light mb-2">验证码</label>
      <div class="flex gap-2">
        <input 
          type="text" 
          id="code" 
          v-model="code" 
          name="code" 
          autocomplete="one-time-code" 
          required 
          class="input flex-1"
          placeholder="请输入验证码"
          maxlength="6"
        >
        <button 
          type="button" 
          @click="sendVerificationCode"
          :disabled="isSendingCode || countdown > 0"
          class="btn-primary whitespace-nowrap px-4 py-3 text-sm"
        >
          {{ countdown > 0 ? `${countdown}s后重发` : (isSendingCode ? '发送中...' : '获取验证码') }}
        </button>
      </div>
    </div>
    
    <!-- 记住我选项 -->
    <div class="flex items-center justify-between">
      <div class="flex items-center">
        <input 
          id="remember-me" 
          name="remember-me" 
          type="checkbox" 
          v-model="rememberMe"
          class="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded"
        >
        <label for="remember-me" class="ml-2 block text-sm text-text">记住我</label>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import modalService from '../../utils/modalService.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'

const emit = defineEmits(['send-code'])

// 本地状态
const isSendingCode = ref(false)
const countdown = ref(0)
let countdownTimer = null

// 本地状态存储输入值
const phone = ref('')
const code = ref('')
const rememberMe = ref(false)



// 发送验证码
const sendVerificationCode = async () => {
  if (!phone.value || phone.value.length !== 11) {
    modalService.alert('请输入有效的11位手机号码')
    return
  }
  
  try {
    isSendingCode.value = true
    
    // 触发发送验证码事件
    emit('send-code', { phone: phone.value })
    
    // 开始倒计时
    countdown.value = 60
    if (countdownTimer) {
      clearInterval(countdownTimer)
    }
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
    }, 1000)
  } catch (error) {
    console.error('Send verification code error:', error)
    showRequestError('发送验证码失败：网络错误，请稍后重试', error)
  } finally {
    isSendingCode.value = false
  }
}

// 暴露方法给父组件
defineExpose({
  getFormData: () => ({
    phone: phone.value,
    code: code.value,
    rememberMe: rememberMe.value
  }),
  resetForm: () => {
    phone.value = ''
    code.value = ''
    rememberMe.value = false
    countdown.value = 0
    if (countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
  }
})
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>