<template>
  <div data-alias="cmp-create-workspace" class="min-h-screen bg-background py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-3xl mx-auto">
      <div class="bg-white rounded-2xl shadow-xl p-8">
        <h1 class="text-3xl font-bold text-text-dark mb-6">创建工作空间</h1>
        <p class="text-text-light mb-8">创建一个命名工作空间，用于管理您的项目和任务</p>
        
        <form class="space-y-6" @submit.prevent="handleSubmit">
          <!-- 工作空间名称 -->
          <div>
            <label for="name" class="block text-sm font-medium text-text-light mb-2">工作空间名称 *</label>
            <input 
              type="text" 
              id="name" 
              v-model="formData.name" 
              name="name" 
              required 
              placeholder="请输入工作空间名称" 
              class="input w-full"
            >
            <p v-if="errors.name" class="mt-1 text-sm text-error">{{ errors.name }}</p>
          </div>
          
          <!-- 工作空间描述 -->
          <div>
            <label for="description" class="block text-sm font-medium text-text-light mb-2">工作空间描述</label>
            <textarea 
              id="description" 
              v-model="formData.description" 
              name="description" 
              rows="3" 
              placeholder="请输入工作空间描述（可选）" 
              class="input w-full resize-none"
            ></textarea>
            <p v-if="errors.description" class="mt-1 text-sm text-error">{{ errors.description }}</p>
          </div>
          
          <!-- 提交按钮 -->
          <div class="flex items-center justify-end space-x-4">
            <button 
              type="button" 
              @click="cancel" 
              class="btn-secondary py-2 px-6 text-base"
            >
              取消
            </button>
            <button 
              type="submit"
              :disabled="isLoading || submitGuard.isBusy()"
              class="btn-primary py-2 px-6 text-base"
            >
              <span v-if="isLoading" class="flex items-center">
                <svg class="animate-spin -ml-1 mr-2 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                创建中...
              </span>
              <span v-else>创建工作空间</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
/* @alias:cmp-create-workspace */
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getCookie } from '../utils/cookieUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

// 定义响应式状态
const isLoading = ref(false)
const formData = ref({
  name: '',
  description: ''
})
const errors = ref({})

const router = useRouter()
const route = useRoute()

// 获取当前租户信息
const tenant = computed(() => route.params.tenant || '')

// 表单验证
const validateForm = () => {
  const newErrors = {}
  
  // 验证名称是否为空
  if (!formData.value.name.trim()) {
    newErrors.name = '工作空间名称不能为空'
  }
  
  // 验证名称长度
  if (formData.value.name.length > 100) {
    newErrors.name = '工作空间名称不能超过100个字符'
  }
  
  // 验证描述长度
  if (formData.value.description.length > 500) {
    newErrors.description = '工作空间描述不能超过500个字符'
  }
  
  errors.value = newErrors
  return Object.keys(newErrors).length === 0
}

// 处理表单提交
const submitGuard = createClickGuard()

const handleSubmit = async () => {
  // OPT-20260819-038: 创建工作空间是写操作，防连点双发 POST
  await submitGuard.run(async ({ idempotencyKey }) => {
    // 表单验证
    if (!validateForm()) {
      return
    }

    try {
      isLoading.value = true
      errors.value = {}

      // 发送创建工作空间请求
      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenant.value}`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
          },
          idempotencyKey,
        ),
        credentials: 'include',
        body: JSON.stringify({
          name: formData.value.name,
          description: formData.value.description,
          is_default: true // 新创建的工作空间设为默认
        })
      })

      if (response.ok) {
        // 创建成功，跳转到项目列表页面
        const workspaceData = await response.json()
        console.log('工作空间创建成功:', workspaceData)

        // 跳转到项目列表页面
        await router.push({
          name: 'projects',
          params: { tenant: tenant.value }
        })
      } else {
        // 处理错误响应
        const errorData = await response.json().catch(() => ({}))
        const errorMessage = errorData.error || errorData.detail || '创建工作空间失败'

        // 如果是字段错误，提取字段错误信息
        if (errorData.name) {
          errors.value.name = errorData.name
        }
        if (errorData.description) {
          errors.value.description = errorData.description
        }

        // 如果是其他错误，显示通用错误信息
        if (!Object.keys(errors.value).length) {
          showRequestError('创建失败：' + errorMessage, { traceId: response.traceId, ...errorData })
        }
      }
    } catch (error) {
        console.error('创建工作空间时发生网络错误:', error)
        showRequestError('创建失败：网络错误，请稍后重试', error)
      } finally {
        isLoading.value = false
      }
    })
}

// 取消操作
const cancel = () => {
  // 跳转到项目列表页面
  router.push({ 
    name: 'projects',
    params: { tenant: tenant.value }
  })
}
</script>

<style scoped>
/* 组件内样式 */
</style>
