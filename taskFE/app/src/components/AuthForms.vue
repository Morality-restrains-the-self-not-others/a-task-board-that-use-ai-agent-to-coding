<template>
  <div class="auth-forms">
    <!-- 登录表单 -->
    <div v-if="activeForm === 'login'" class="login-form">
      <h3 class="text-lg font-semibold mb-4">{{ loginTitle || '登录' }}</h3>
      
      <div class="mb-4">
        <label class="block text-gray-700 mb-2">账号（邮箱/手机号/用户名）</label>
        <input 
          v-model="loginForm.username"
          type="text"
          class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="请输入账号"
        >
      </div>
      
      <div class="mb-4">
        <label class="block text-gray-700 mb-2">密码</label>
        <input 
          v-model="loginForm.password"
          type="password"
          class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="请输入密码"
        >
      </div>
      
      <div class="flex justify-between mb-6">
        <div class="flex items-center">
          <input 
            v-model="loginForm.remember"
            type="checkbox"
            class="mr-2"
          >
          <label class="text-gray-700">记住我</label>
        </div>
        <a href="/forgot-password" class="text-primary hover:underline">忘记密码？</a>
      </div>
      
      <div class="flex flex-col gap-3">
        <button
          @click="handleLogin"
          class="bg-primary text-white px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors"
          :disabled="isLoggingIn || loginGuard.isBusy()"
        >
          <span v-if="isLoggingIn">登录中...</span>
          <span v-else>登录</span>
        </button>
        
        <button 
          @click="activeForm = 'register'"
          class="border border-primary text-primary px-6 py-2 rounded-lg hover:bg-primary/5 transition-colors"
        >
          注册新账号
        </button>
      </div>
    </div>
    
    <!-- 注册表单 -->
    <div v-else-if="activeForm === 'register'" class="register-form">
      <h3 class="text-lg font-semibold mb-4">{{ registerTitle || '注册新账号' }}</h3>
      
      <div class="mb-4">
        <label class="block text-gray-700 mb-2">邮箱</label>
        <input 
          v-model="registerForm.email"
          type="email"
          class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="请输入邮箱"
        >
      </div>
      
      <div class="mb-4">
        <label class="block text-gray-700 mb-2">用户名</label>
        <input 
          v-model="registerForm.username"
          type="text"
          class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="请输入用户名"
        >
      </div>
      
      <div class="mb-4">
        <label class="block text-gray-700 mb-2">密码</label>
        <input 
          v-model="registerForm.password"
          type="password"
          class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="请输入密码"
        >
      </div>
      
      <div class="mb-6">
        <label class="block text-gray-700 mb-2">确认密码</label>
        <input 
          v-model="registerForm.confirmPassword"
          type="password"
          class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="请确认密码"
        >
      </div>
      
      <div class="flex gap-3">
        <button
          @click="handleRegister"
          class="bg-primary text-white px-6 py-2 rounded-lg hover:bg-primary/90 transition-colors flex-1"
          :disabled="isRegistering || registerGuard.isBusy()"
        >
          <span v-if="isRegistering">注册中...</span>
          <span v-else>注册</span>
        </button>
        <button 
          @click="activeForm = 'login'"
          class="border border-gray-300 text-gray-700 px-6 py-2 rounded-lg hover:bg-gray-100 transition-colors"
        >
          取消
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch, extractErrorMessage } from '../utils/apiUtils.js';
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js';

import { ref } from 'vue'

// 定义组件属性
const props = defineProps({
  loginTitle: {
    type: String,
    default: '登录'
  },
  registerTitle: {
    type: String,
    default: '注册新账号'
  },
  csrfToken: {
    type: String,
    default: ''
  }
})

// 定义组件事件
const emit = defineEmits(['login-success', 'login-failure', 'register-success', 'register-failure'])

// 响应式状态
const activeForm = ref('login')
const isLoggingIn = ref(false)
const isRegistering = ref(false)
// OPT-20260819-038: 登录/注册是写操作，防连点/超时重试双发 POST
const loginGuard = createClickGuard()
const registerGuard = createClickGuard()

// 登录表单数据
const loginForm = ref({
  username: '',
  password: '',
  remember: false
})

// 注册表单数据
const registerForm = ref({
  email: '',
  username: '',
  password: '',
  confirmPassword: ''
})

const handleLogin = async () => {
  if (!loginForm.value.username || !loginForm.value.password) {
    emit('login-failure', '请输入账号和密码')
    return
  }

  // OPT-20260819-038: 登录是写操作，防连点/超时重试双发 POST
  await loginGuard.run(async ({ idempotencyKey }) => {
    isLoggingIn.value = true

    try {
      // Plain text password, server handles bcrypt verification
      const response = await apiFetch('/api/accounts/users/login/', {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          username: loginForm.value.username,
          password: loginForm.value.password,
          remember: loginForm.value.remember
        })
      })
    
    if (!response.ok) {
      const errorData = await response.json()
      const err = new Error(extractErrorMessage(errorData, response, '登录失败'))
      // 按元规则挂载 traceId 到 Error 对象，供上层展示 data-traceId
      const traceId = errorData.trace_id || errorData.traceId || ''
      if (traceId) err.traceId = traceId
      throw err
    }
    
    const result = await response.json()
    
    // 登录成功，存储 token / userId，并写入多账号槽
    const { setCachedAuthToken } = await import('../utils/apiUtils.js')
    if (result.token && result.user?.id) {
      const { persistLoginAccountSlot } = await import('../domain/auth/services/activate_session_service.js')
      try {
        await persistLoginAccountSlot({
          userId: result.user.id,
          token: result.token,
          username: result.user.username || loginForm.value.username || '',
          avatarUrl: result.user.avatar_url || null,
        })
      } catch (slotErr) {
        setCachedAuthToken(result.token)
        console.warn('账号槽写入失败', slotErr)
      }
      console.log('登录成功，凭据已存储')
    } else if (result.token) {
      setCachedAuthToken(result.token)
      console.log('登录成功，凭据已存储')
    }
    
      // 登录成功
      emit('login-success', result)
    } catch (error) {
      console.error('登录失败:', error, 'traceId:', error.traceId || '')
      emit('login-failure', error.message)
    } finally {
      isLoggingIn.value = false
    }
  })
}

const handleRegister = async () => {
  // 表单验证
  if (!registerForm.value.email || !registerForm.value.username || !registerForm.value.password) {
    emit('register-failure', '请填写所有必填字段')
    return
  }

  if (registerForm.value.password !== registerForm.value.confirmPassword) {
    emit('register-failure', '两次输入的密码不一致')
    return
  }

  // OPT-20260819-038: 注册是写操作，防连点/超时重试双发 POST
  await registerGuard.run(async ({ idempotencyKey }) => {
    isRegistering.value = true

    try {
      // Plain text password, server handles bcrypt hashing
      const response = await apiFetch('/api/accounts/users/email_register/', {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          email: registerForm.value.email,
          username: registerForm.value.username,
          password: registerForm.value.password
        })
      })
    
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || '注册失败')
      }

      const result = await response.json()

      // 注册成功
      emit('register-success', result)
    } catch (error) {
      console.error('注册失败:', error)
      emit('register-failure', error.message)
    } finally {
      isRegistering.value = false
    }
  })
}
</script>

<style scoped>
.auth-forms {
  max-width: 28rem;
  margin: 0 auto;
}
</style>