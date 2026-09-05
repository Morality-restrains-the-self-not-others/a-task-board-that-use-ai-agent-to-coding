<template>
  <div class="p-8">
    <div class="flex flex-col space-y-6">
      <div class="flex justify-between items-center">
        <div>
          <h2 class="text-2xl font-bold text-text">工作空间设置</h2>
          <p class="text-text-light mt-1">管理工作空间的基本信息和各项配置</p>
        </div>
      </div>

      <!-- 工作空间基本信息 -->
      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">基本信息</h3>
        </div>
        
        <div class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="workspace-name" class="block text-sm font-medium text-text-light mb-1">
                工作空间名称
              </label>
              <input 
                type="text" 
                id="workspace-name" 
                v-model="workspaceName"
                class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
              >
            </div>
            <div>
              <label for="workspace-description" class="block text-sm font-medium text-text-light mb-1">
                工作空间描述
              </label>
              <input 
                type="text" 
                id="workspace-description" 
                v-model="workspaceDescription"
                class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
              >
            </div>
          </div>
          <div class="flex justify-end">
            <button 
              class="btn-primary"
              :disabled="saving || saveBasicInfoGuard.isBusy()"
              @click="saveBasicInfo"
            >
              <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path>
              </svg>
              {{ saving ? '保存中...' : '保存设置' }}
            </button>
          </div>
        </div>
      </div>

      <!-- 通知设置 -->
      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">通知设置</h3>
        </div>
        
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <label class="block text-sm font-medium text-text mb-1">
                邮件通知
              </label>
              <p class="text-sm text-text-light">接收工作空间相关的邮件通知</p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="emailNotifications" class="sr-only peer">
              <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
          <div class="flex items-center justify-between">
            <div>
              <label class="block text-sm font-medium text-text mb-1">
                系统通知
              </label>
              <p class="text-sm text-text-light">接收系统相关的通知消息</p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="systemNotifications" class="sr-only peer">
              <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
          <div class="flex items-center justify-between">
            <div>
              <label class="block text-sm font-medium text-text mb-1">
                任务通知
              </label>
              <p class="text-sm text-text-light">接收任务状态变更的通知</p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="taskNotifications" class="sr-only peer">
              <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
          <div class="flex justify-end">
            <button 
              class="btn-primary"
              :disabled="saving || saveNotificationSettingsGuard.isBusy()"
              @click="saveNotificationSettings"
            >
              <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path>
              </svg>
              {{ saving ? '保存中...' : '保存设置' }}
            </button>
          </div>
        </div>
      </div>

      <!-- 安全设置 -->
      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">安全设置</h3>
        </div>
        
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <label class="block text-sm font-medium text-text mb-1">
                两步验证
              </label>
              <p class="text-sm text-text-light">为工作空间添加额外的安全保护</p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="twoFactorAuth" class="sr-only peer">
              <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
          <div class="flex items-center justify-between">
            <div>
              <label class="block text-sm font-medium text-text mb-1">
                会话超时
              </label>
              <p class="text-sm text-text-light">设置用户会话的自动超时时间</p>
            </div>
            <select 
              v-model="sessionTimeout"
              class="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            >
              <option value="30">30分钟</option>
              <option value="60">1小时</option>
              <option value="120">2小时</option>
              <option value="240">4小时</option>
              <option value="480">8小时</option>
            </select>
          </div>
          <div class="flex justify-end">
            <button 
              class="btn-primary"
              :disabled="saving || saveSecuritySettingsGuard.isBusy()"
              @click="saveSecuritySettings"
            >
              <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path>
              </svg>
              {{ saving ? '保存中...' : '保存设置' }}
            </button>
          </div>
        </div>
      </div>

      <!-- 高级设置 -->
      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">高级设置</h3>
        </div>
        
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <label class="block text-sm font-medium text-text mb-1">
                API 访问
              </label>
              <p class="text-sm text-text-light">启用或禁用工作空间的 API 访问</p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="apiAccess" class="sr-only peer">
              <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
          <div class="flex items-center justify-between">
            <div>
              <label class="block text-sm font-medium text-text mb-1">
                数据导出
              </label>
              <p class="text-sm text-text-light">允许导出工作空间的数据</p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="dataExport" class="sr-only peer">
              <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
          <div class="flex justify-end">
            <button 
              class="btn-primary"
              :disabled="saving || saveAdvancedSettingsGuard.isBusy()"
              @click="saveAdvancedSettings"
            >
              <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path>
              </svg>
              {{ saving ? '保存中...' : '保存设置' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch, extractErrorMessage } from '../utils/apiUtils.js';
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js';

import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getCookie } from '../utils/cookieUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { safeJson, safeResponseJson } from '@/utils/safeResponseJson.js'

const route = useRoute()

// 响应式数据
const workspaceName = ref('')
const workspaceDescription = ref('')
const emailNotifications = ref(true)
const systemNotifications = ref(true)
const taskNotifications = ref(true)
const twoFactorAuth = ref(false)
const sessionTimeout = ref('60')
const apiAccess = ref(true)
const dataExport = ref(true)
const saving = ref(false)
// OPT-20260819-038: 各分项设置保存均为写操作，防连点/超时重试双发 POST
const saveBasicInfoGuard = createClickGuard()
const saveNotificationSettingsGuard = createClickGuard()
const saveSecuritySettingsGuard = createClickGuard()
const saveAdvancedSettingsGuard = createClickGuard()

const getCurrentWorkspaceId = () => {
  return route.params.tenant || ''
}

// 加载工作空间信息
const loadWorkspaceInfo = async () => {
  try {
    const workspaceId = getCurrentWorkspaceId()
    let url = `/api/workspace/info/`
    if (workspaceId) {
      url += `?workspace_id=${workspaceId}`
    }
    
    const response = await apiFetch(url, {
      headers: {
      }
    })
    
    const result = await safeJson(response, {})
    
    if (result.status === 'success') {
      workspaceName.value = result.workspace.name || ''
      workspaceDescription.value = result.workspace.description || ''
      // 加载其他设置
    }
  } catch (error) {
    console.error('加载工作空间信息失败:', error)
  }
}

// 保存基本信息
const saveBasicInfo = async () => {
  // OPT-20260819-038: 保存基本信息是写操作，防连点/超时重试双发 POST
  await saveBasicInfoGuard.run(async ({ idempotencyKey }) => {
    saving.value = true

    try {
      const workspaceId = getCurrentWorkspaceId()

      const form = new URLSearchParams()
      form.append('action', 'update_basic_info')
      form.append('workspace_id', workspaceId)
      form.append('name', workspaceName.value)
      form.append('description', workspaceDescription.value)

      const response = await apiFetch('/api/workspace/settings/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: form
      })

      const result = await safeJson(response, {})

      if (result.status === 'success') {
        alert('保存成功！')
      } else {
        showRequestError('保存失败: ' + (extractErrorMessage(result, response, '未知错误')), response)
      }
    } catch (error) {
      console.error('保存基本信息失败:', error)
      showRequestError('保存失败，请重试', error)
    } finally {
      saving.value = false
    }
  })
}

// 保存通知设置
const saveNotificationSettings = async () => {
  // OPT-20260819-038: 保存通知设置是写操作，防连点/超时重试双发 POST
  await saveNotificationSettingsGuard.run(async ({ idempotencyKey }) => {
    saving.value = true

    try {
      const workspaceId = getCurrentWorkspaceId()

      const form = new URLSearchParams()
      form.append('action', 'update_notification_settings')
      form.append('workspace_id', workspaceId)
      form.append('email_notifications', emailNotifications.value)
      form.append('system_notifications', systemNotifications.value)
      form.append('task_notifications', taskNotifications.value)

      const response = await apiFetch('/api/workspace/settings/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: form
      })

      const result = await safeJson(response, {})

      if (result.status === 'success') {
        alert('保存成功！')
      } else {
        showRequestError('保存失败: ' + (extractErrorMessage(result, response, '未知错误')), response)
      }
    } catch (error) {
      console.error('保存通知设置失败:', error)
      showRequestError('保存失败，请重试', error)
    } finally {
      saving.value = false
    }
  })
}

// 保存安全设置
const saveSecuritySettings = async () => {
  // OPT-20260819-038: 保存安全设置是写操作，防连点/超时重试双发 POST
  await saveSecuritySettingsGuard.run(async ({ idempotencyKey }) => {
    saving.value = true

    try {
      const workspaceId = getCurrentWorkspaceId()

      const form = new URLSearchParams()
      form.append('action', 'update_security_settings')
      form.append('workspace_id', workspaceId)
      form.append('two_factor_auth', twoFactorAuth.value)
      form.append('session_timeout', sessionTimeout.value)

      const response = await apiFetch('/api/workspace/settings/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: form
      })

      const result = await safeJson(response, {})

      if (result.status === 'success') {
        alert('保存成功！')
      } else {
        showRequestError('保存失败: ' + (extractErrorMessage(result, response, '未知错误')), response)
      }
    } catch (error) {
      console.error('保存安全设置失败:', error)
      showRequestError('保存失败，请重试', error)
    } finally {
      saving.value = false
    }
  })
}

// 保存高级设置
const saveAdvancedSettings = async () => {
  // OPT-20260819-038: 保存高级设置是写操作，防连点/超时重试双发 POST
  await saveAdvancedSettingsGuard.run(async ({ idempotencyKey }) => {
    saving.value = true

    try {
      const workspaceId = getCurrentWorkspaceId()

      const form = new URLSearchParams()
      form.append('action', 'update_advanced_settings')
      form.append('workspace_id', workspaceId)
      form.append('api_access', apiAccess.value)
      form.append('data_export', dataExport.value)

      const response = await apiFetch('/api/workspace/settings/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: form
      })

      const result = await safeJson(response, {})

      if (result.status === 'success') {
        alert('保存成功！')
      } else {
        showRequestError('保存失败: ' + (extractErrorMessage(result, response, '未知错误')), response)
      }
    } catch (error) {
      console.error('保存高级设置失败:', error)
      showRequestError('保存失败，请重试', error)
    } finally {
      saving.value = false
    }
  })
}

onMounted(() => {
  loadWorkspaceInfo()
})
</script>

<style scoped>
</style>
