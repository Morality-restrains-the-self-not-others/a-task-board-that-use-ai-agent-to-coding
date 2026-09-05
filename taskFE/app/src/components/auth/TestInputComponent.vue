<template>
  <div class="test-input-component">
    <h2>测试输入组件</h2>
    <div>
      <label for="test-email">邮箱</label>
      <input 
        type="email" 
        id="test-email" 
        :value="email" 
        @input="handleEmailInput"
        placeholder="请输入邮箱"
      >
      <p>当前邮箱: {{ email }}</p>
    </div>
    <div>
      <label for="test-password">密码</label>
      <input 
        type="password" 
        id="test-password" 
        :value="password" 
        @input="handlePasswordInput"
        placeholder="请输入密码"
      >
      <p>当前密码: {{ password }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const email = ref('')
const password = ref('')

const handleEmailInput = (event) => {
  console.log('Email input:', event.target.value)
  email.value = event.target.value
}

const handlePasswordInput = (event) => {
  console.log('Password input:', event.target.value)
  password.value = event.target.value
}
</script>

<style scoped>
.test-input-component {
  padding: 20px;
  border: 1px solid #ddd;
  border-radius: 8px;
  background: #f9f9f9;
}

h2 {
  margin-top: 0;
  margin-bottom: 20px;
}

div {
  margin-bottom: 15px;
}

label {
  display: block;
  margin-bottom: 5px;
  font-weight: 500;
}

input {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

p {
  margin-top: 5px;
  font-size: 14px;
  color: #666;
}
</style>