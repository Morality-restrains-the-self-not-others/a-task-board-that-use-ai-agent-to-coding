<template>
  <div data-alias="cmp-reset-password-form" class="bg-gradient-to-br from-primary/5 to-background min-h-screen flex flex-col">
    <!-- 重置密码表单 -->
    <main class="flex-grow flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div class="max-w-md w-full space-y-8 bg-white p-10 rounded-2xl shadow-xl">
        <div class="text-center">
          <div class="inline-block w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mb-6">
            <svg class="w-8 h-8 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path>
            </svg>
          </div>
          <h2 class="text-[clamp(1.5rem,3vw,2.5rem)] font-bold text-text mb-2">设置新密码</h2>
          <p class="text-text-light">请输入您的新密码，确保密码安全且符合要求</p>
        </div>
        <!-- Token 验证中 -->
        <div v-if="isCheckingToken" class="text-center py-8">
          <div class="inline-block w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin mb-4"></div>
          <p class="text-text-light">正在验证重置链接...</p>
        </div>

        <!-- Token 已失效提示 -->
        <div v-else-if="token && tokenValid === false" class="text-center py-8">
          <div class="inline-block w-16 h-16 bg-error/10 rounded-full flex items-center justify-center mb-6">
            <svg class="w-8 h-8 text-error" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-text mb-2">链接已失效</h3>
          <p class="text-text-light mb-4" :data-traceId="errorTraceId || undefined">{{ errors.password }}</p>
          <a href="/auth/reset-password/" class="text-primary hover:text-primary-dark transition-colors duration-300 font-medium">重新请求密码重置 →</a>
        </div>

        <form v-else class="space-y-6" @submit.prevent="handleSubmit">
          <input type="hidden" name="uidb64" :value="uidb64">
          <input type="hidden" name="token" :value="token">
          <div>
            <label for="password" class="block text-sm font-medium text-text-light mb-2">新密码</label>
            <input
              type="password"
              id="password"
              v-model="formData.password"
              name="password"
              placeholder="请输入新密码（至少8位，包含字母、数字和特殊字符）"
              required
              class="input w-full"
              :disabled="isCheckingToken"
              @input="checkPasswordStrength"
            >
            <div id="password-error" class="text-error text-sm mt-1" :class="{ 'hidden': !errors.password }" :data-traceId="errorTraceId || undefined">{{ errors.password }}</div>
            
            <!-- 密码强度指示器 -->
            <div id="password-strength" class="mt-2" :class="{ 'hidden': !formData.password }">
              <div class="flex items-center mb-1">
                <span class="text-xs font-medium text-gray-600 mr-2">密码强度：</span>
                <span id="strength-text" class="text-xs font-medium" :class="strengthTextClass">{{ passwordStrength.level }}</span>
              </div>
              <div class="w-full bg-gray-200 rounded-full h-2">
                <div 
                  id="strength-bar" 
                  class="h-2 rounded-full transition-all duration-300" 
                  :class="strengthBarClass"
                  :style="{ width: `${passwordStrength.strength * 20}%` }"
                ></div>
              </div>
            </div>
          </div>
          <div>
            <label for="password_confirm" class="block text-sm font-medium text-text-light mb-2">确认新密码</label>
            <input
              type="password"
              id="password_confirm"
              v-model="formData.passwordConfirm"
              name="password_confirm"
              placeholder="请再次输入新密码"
              required
              class="input w-full"
              :disabled="isCheckingToken"
            >
            <div id="password-confirm-error" class="text-error text-sm mt-1" :class="{ 'hidden': !errors.passwordConfirm }" :data-traceId="errorTraceId || undefined">{{ errors.passwordConfirm }}</div>
          </div>
          <div>
            <button type="submit" class="btn-primary w-full py-3 text-base" :disabled="isSubmitting || isCheckingToken || tokenValid === false">
              {{ isSubmitting ? '设置中...' : '设置新密码' }}
            </button>
          </div>
        </form>
        <div class="text-center">
          <p class="text-sm text-text-light">
            想起密码了？ <a href="/auth/login/" class="font-medium text-primary hover:text-primary-dark transition-colors duration-300">返回登录</a>
          </p>
        </div>
      </div>
    </main>

    <!-- 页脚 -->
    <footer class="bg-gray-800 text-white py-12">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid grid-cols-1 md:grid-cols-4 gap-8">
          <div>
            <h3 class="text-xl font-bold mb-4">SaaS平台</h3>
            <p class="text-gray-400">提供全方位的项目管理解决方案，帮助您的团队更高效地协作和交付。</p>
          </div>
          <div>
            <h3 class="text-lg font-bold mb-4">产品</h3>
            <ul class="space-y-2">
              <li><a href="#" class="text-gray-400 hover:text-white">项目管理</a></li>
              <li><a href="#" class="text-gray-400 hover:text-white">云平台</a></li>
              <li><a href="#" class="text-gray-400 hover:text-white">API文档</a></li>
              <li><a href="#" class="text-gray-400 hover:text-white">定价</a></li>
            </ul>
          </div>
          <div>
            <h3 class="text-lg font-bold mb-4">资源</h3>
            <ul class="space-y-2">
              <li><a href="#" class="text-gray-400 hover:text-white">文档中心</a></li>
              <li><a href="#" class="text-gray-400 hover:text-white">教程</a></li>
              <li><a href="#" class="text-gray-400 hover:text-white">支持</a></li>
              <li><a href="#" class="text-gray-400 hover:text-white">博客</a></li>
            </ul>
          </div>
          <div>
            <h3 class="text-lg font-bold mb-4">联系我们</h3>
            <ul class="space-y-2">
              <li class="text-gray-400">邮箱：contact@daydaymoney.com</li>
              <li class="text-gray-400">电话：400-123-4567</li>
              <li class="text-gray-400">地址：北京市朝阳区</li>
            </ul>
          </div>
        </div>
        <div class="mt-12 pt-8 border-t border-gray-700 text-center text-gray-400">
          <p>&copy; 2025 SaaS平台. 保留所有权利.</p>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js';
import { safeResponseJson } from '../utils/safeResponseJson.js';

/* @alias:cmp-reset-password-form */
import { ref, computed, onMounted } from 'vue'
import { getCookie } from '../utils/cookieUtils'
import modalService from '../utils/modalService.js'

// 定义组件属性
const props = defineProps({
  user: {
    type: Object,
    default: () => ({
      isAuthenticated: false,
      isSuperuser: false,
      username: ''
    })
  }
})

// 定义组件事件
const emit = defineEmits(['reset-success', 'reset-failure'])

// 从URL参数获取uidb64、token、phone和code
const getUrlParams = () => {
  const params = new URLSearchParams(window.location.search)
  return {
    uidb64: params.get('uidb64') || '',
    token: params.get('token') || '',
    phone: params.get('phone') || '',
    code: params.get('code') || ''
  }
}

// 响应式状态
const isSubmitting = ref(false)
const isCheckingToken = ref(false)
const tokenValid = ref(null) // null=未检查, true=有效, false=已失效
const uidb64 = ref(getUrlParams().uidb64)
const token = ref(getUrlParams().token)
const phone = ref(getUrlParams().phone)
const code = ref(getUrlParams().code)

// 表单数据
const formData = ref({
  password: '',
  passwordConfirm: ''
})

// 错误信息
const errors = ref({
  password: '',
  passwordConfirm: ''
})

// API error traceId for data-traceId binding
const errorTraceId = ref('')

// 密码强度
const passwordStrength = ref({
  strength: 0,
  level: '',
  requirements: []
})

// 计算属性：密码强度文本类
const strengthTextClass = computed(() => {
  switch (passwordStrength.value.level) {
    case '弱':
      return 'text-red-600'
    case '中':
      return 'text-yellow-600'
    case '强':
      return 'text-green-600'
    default:
      return ''
  }
})

// 计算属性：密码强度条类
const strengthBarClass = computed(() => {
  switch (passwordStrength.value.level) {
    case '弱':
      return 'bg-red-500'
    case '中':
      return 'bg-yellow-500'
    case '强':
      return 'bg-green-500'
    default:
      return ''
  }
})

// 检查密码强度
const checkPasswordStrength = () => {
  const password = formData.value.password
  if (!password) {
    passwordStrength.value = {
      strength: 0,
      level: '',
      requirements: []
    }
    return
  }
  
  let strength = 0
  const requirements = []
  
  // 检查长度
  if (password.length >= 8) {
    strength += 1
    requirements.push({ text: '至少8位字符', status: '✅' })
  } else {
    requirements.push({ text: '至少8位字符', status: '❌' })
  }
  
  // 检查包含小写字母
  if (/[a-z]/.test(password)) {
    strength += 1
    requirements.push({ text: '包含小写字母', status: '✅' })
  } else {
    requirements.push({ text: '包含小写字母', status: '❌' })
  }
  
  // 检查包含大写字母
  if (/[A-Z]/.test(password)) {
    strength += 1
    requirements.push({ text: '包含大写字母', status: '✅' })
  } else {
    requirements.push({ text: '包含大写字母', status: '❌' })
  }
  
  // 检查包含数字
  if (/[0-9]/.test(password)) {
    strength += 1
    requirements.push({ text: '包含数字', status: '✅' })
  } else {
    requirements.push({ text: '包含数字', status: '❌' })
  }
  
  // 检查包含特殊字符
  if (/[^A-Za-z0-9]/.test(password)) {
    strength += 1
    requirements.push({ text: '包含特殊字符', status: '✅' })
  } else {
    requirements.push({ text: '包含特殊字符', status: '❌' })
  }
  
  let level = ''
  if (strength <= 2) {
    level = '弱'
  } else if (strength <= 4) {
    level = '中'
  } else {
    level = '强'
  }
  
  passwordStrength.value = {
    strength,
    level,
    requirements
  }
}

// 获取用户标识符（邮箱或手机号）
const getUserIdFromToken = async (resetToken) => {
  try {
    const response = await apiFetch(`/api/accounts/users/get-reset-user-info/${resetToken}/`, {
      method: 'GET',
      headers: {
      }
    })

    if (response.ok) {
      const { data } = await safeResponseJson(response, { fallback: {} })
      return data?.identifier || null
    }
    const { traceId } = await safeResponseJson(response, { fallback: {} })
    errorTraceId.value = traceId || response.traceId || ''
    return null
  } catch (error) {
    console.error('获取用户信息失败:', error)
    errorTraceId.value = error?.traceId || ''
    return null
  }
}

// 表单提交处理
const handleSubmit = async () => {
  try {
    isSubmitting.value = true
    errors.value = {
      password: '',
      passwordConfirm: ''
    }
    
    // 验证密码匹配
    if (formData.value.password !== formData.value.passwordConfirm) {
      errors.value.passwordConfirm = '两次输入的密码不一致'
      isSubmitting.value = false
      return
    }
    
    // 验证密码强度
    if (passwordStrength.value.strength < 3) {
      errors.value.password = '密码强度不足，请确保密码包含字母、数字和特殊字符'
      isSubmitting.value = false
      return
    }
    
    let response = null

    // 处理两种重置方式
    if (token.value) {
      // 方式一：使用链接重置密码（token）
      const identifier = await getUserIdFromToken(token.value)
      if (!identifier) {
        errors.value.password = '获取用户信息失败，请重新请求密码重置'
        isSubmitting.value = false
        return
      }

      // 请求重置密码链接API — plain text password, server handles bcrypt
      response = await apiFetch(`/api/accounts/users/reset-password-with-link/${token.value}/`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          new_password: formData.value.password
        })
      })
    } else if (phone.value && code.value) {
      // 方式二：使用验证码重置密码（phone + code）
      // Plain text password, server handles bcrypt
      response = await apiFetch('/api/accounts/users/reset_password_with_code/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          phone: phone.value,
          code: code.value,
          new_password: formData.value.password
        })
      })
    } else {
      errors.value.password = '无效的重置请求，请重新开始'
      isSubmitting.value = false
      return
    }
    
    const { data, traceId } = await safeResponseJson(response, { fallback: {} })

    if (response.ok) {
      emit('reset-success', data)
      modalService.alert('密码重置成功，请使用新密码登录')
      window.location.href = '/auth/login/'
    } else {
      errorTraceId.value = traceId || response.traceId || ''
      emit('reset-failure', data)
      if (data && typeof data.error === 'string') {
        errors.value.password = data.error
      } else if (data && typeof data.detail === 'string') {
        errors.value.password = data.detail
      } else if (data && typeof data.message === 'string') {
        errors.value.password = data.message
      } else {
        errors.value.password = '重置失败，请稍后重试'
      }
    }
  } catch (error) {
    console.error('Reset password error:', error)
    errorTraceId.value = error?.traceId || ''
    emit('reset-failure', { error: error.message })
    errors.value.password = '网络错误，请稍后重试'
  } finally {
    isSubmitting.value = false
  }
}

// 组件挂载时检查URL参数
onMounted(async () => {
  // 从URL参数获取错误信息
  const params = new URLSearchParams(window.location.search)
  const errorParam = params.get('error')

  if (errorParam) {
    // 显示错误信息在表单中，而不是URL上
    errors.value.password = decodeURIComponent(errorParam)
    // 清除URL中的错误参数
    window.history.replaceState({}, document.title, window.location.pathname + window.location.search.replace(/[?&]error=[^&]*/g, '').replace(/\?$/, ''))
  }

  // 检查重置参数是否存在
  if (!token.value && !phone.value && !code.value) {
    errors.value.password = '无效的重置密码请求，请重新开始'
    return
  }

  // 如果有 token，立即验证其有效性，避免用户填完表单后才提示链接已失效
  if (token.value) {
    isCheckingToken.value = true
    const identifier = await getUserIdFromToken(token.value)
    isCheckingToken.value = false
    if (!identifier) {
      tokenValid.value = false
      errors.value.password = '密码重置链接已失效，请重新请求密码重置'
    } else {
      tokenValid.value = true
    }
  }
})
</script>

<style scoped>
/* 组件内样式 */
</style>
