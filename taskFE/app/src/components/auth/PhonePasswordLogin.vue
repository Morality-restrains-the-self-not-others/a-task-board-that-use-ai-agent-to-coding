<template>
  <div class="phone-password-login">
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
    <div>
      <label for="password" class="block text-sm font-medium text-text-light mb-2">密码</label>
      <input 
        type="password" 
        id="password" 
        v-model="password" 
        name="password" 
        autocomplete="current-password" 
        required 
        class="input w-full"
        placeholder="请输入密码"
      >
    </div>
    
    <!-- 记住我选项和忘记密码链接 -->
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
      <div class="text-sm">
        <a href="/auth/reset-password-request/" class="font-medium text-primary hover:text-primary-dark transition-colors duration-300">忘记密码？</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

// 本地状态存储输入值
const phone = ref('')
const password = ref('')
const rememberMe = ref(false)



// 暴露方法给父组件
defineExpose({
  getFormData: () => ({
    phone: phone.value,
    password: password.value,
    rememberMe: rememberMe.value
  }),
  resetForm: () => {
    phone.value = ''
    password.value = ''
    rememberMe.value = false
  }
})
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>