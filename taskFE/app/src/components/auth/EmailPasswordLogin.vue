<template>
  <div class="email-password-login">
    <div>
      <label for="email" class="block text-sm font-medium text-text-light mb-2">邮箱</label>
      <input 
        type="email" 
        id="email" 
        name="email" 
        autocomplete="email" 
        required 
        class="input w-full"
        placeholder="请输入邮箱"
      >
    </div>
    <div>
      <label for="password" class="block text-sm font-medium text-text-light mb-2">密码</label>
      <input 
        type="password" 
        id="password" 
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
// 暴露方法给父组件，直接从 DOM 获取表单数据
defineExpose({
  getFormData: () => {
    const emailInput = document.getElementById('email')
    const passwordInput = document.getElementById('password')
    const rememberMeInput = document.getElementById('remember-me')
    
    return {
      email: emailInput ? emailInput.value : '',
      password: passwordInput ? passwordInput.value : '',
      rememberMe: rememberMeInput ? rememberMeInput.checked : false
    }
  },
  resetForm: () => {
    const emailInput = document.getElementById('email')
    const passwordInput = document.getElementById('password')
    const rememberMeInput = document.getElementById('remember-me')
    
    if (emailInput) emailInput.value = ''
    if (passwordInput) passwordInput.value = ''
    if (rememberMeInput) rememberMeInput.checked = false
  }
})
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>