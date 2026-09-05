<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <h3 class="text-xl font-bold text-text">进度体系设置</h3>
      <button 
        class="text-gray-500 hover:text-gray-700"
        @click="$emit('close')"
      >
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
        </svg>
      </button>
    </div>

    <div class="space-y-4">
      <div v-if="loading" class="text-center py-8">
        <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary mx-auto"></div>
        <p class="mt-4 text-gray-600">加载中...</p>
      </div>
      <div v-else>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">选择进度体系</label>
          <select 
            v-model="selectedProgressSystemId"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @change="handleProgressSystemChange"
          >
            <option value="">请选择</option>
            <optgroup label="系统管理员创建的">
              <option 
                v-for="system in adminProgressSystems" 
                :key="system.id"
                :value="system.id"
              >
                {{ system.name }}{{ system.is_default ? ' (默认)' : '' }}
              </option>
            </optgroup>
            <optgroup label="租户创建的">
              <option 
                v-for="system in tenantProgressSystems" 
                :key="system.id"
                :value="system.id"
              >
                {{ system.name }}
              </option>
            </optgroup>
          </select>
        </div>

        <div v-if="selectedProgressSystemId" class="mt-6">
          <h4 class="font-medium mb-2">进度列配置预览</h4>
          <div class="grid grid-cols-3 gap-2">
            <div 
              v-for="(column, index) in selectedProgressSystemColumns" 
              :key="index"
              class="px-2 py-1 bg-gray-50 rounded text-sm"
            >
              {{ column.name }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch, extractErrorMessage } from '../utils/apiUtils.js';
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js';

import { ref, onMounted, computed } from 'vue'
import { getCookie } from '../utils/cookieUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'

const props = defineProps({
  workspaceId: {
    type: String,
    required: true
  },
  tenantId: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['close'])

// 数据
const adminProgressSystems = ref([])
const tenantProgressSystems = ref([])
const selectedProgressSystemId = ref('')
const loading = ref(true)



// 加载进度体系列表
const loadProgressSystems = async () => {
  loading.value = true
  try {
    // 加载管理员创建的进度体系
    const adminResponse = await apiFetch('/api/system/progress-systems/', {
      headers: {
      }
    })
    
    if (adminResponse.ok) {
      const adminResult = await adminResponse.json()
      if (adminResult.status === 'success') {
        adminProgressSystems.value = adminResult.project_progress_systems || []
      }
    }
    
    // 加载租户创建的进度体系
    const tenantResponse = await apiFetch(`/api/projects/progress-systems/tenant_id/${props.tenantId}`, {
      headers: {
      }
    })
    
    if (tenantResponse.ok) {
      const tenantResult = await tenantResponse.json()
      if (tenantResult.status === 'success') {
        tenantProgressSystems.value = tenantResult.project_progress_systems || []
      }
    }
    
    // 加载当前工作空间的进度体系设置
    await loadCurrentWorkspaceProgressSystem()
  } catch (error) {
    console.error('加载进度体系失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载当前工作空间的进度体系设置
const loadCurrentWorkspaceProgressSystem = async () => {
  try {
    const response = await apiFetch(`/api/projects/workspaces/tenant_id/${props.tenantId}/${props.workspaceId}/progress-system/`, {
      headers: {
      }
    })
    
    if (response.ok) {
      const result = await response.json()
      if (result.status === 'success' && result.progress_system_id) {
        selectedProgressSystemId.value = result.progress_system_id
      }
    }
  } catch (error) {
    console.error('加载当前工作空间进度体系设置失败:', error)
  }
}

// 处理进度体系变更
const progressChangeGuard = createClickGuard()

const handleProgressSystemChange = async () => {
  // OPT-20260819-038: 切换工作空间进度体系是写操作，防连点双发 POST
  await progressChangeGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${props.tenantId}/${props.workspaceId}/progress-system/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          progress_system_id: selectedProgressSystemId.value
        })
      })
    
      const result = await response.json()

      if (result.status !== 'success') {
        const errorMsg = extractErrorMessage(result, response, '未知错误')
        showRequestError('设置进度体系失败: ' + errorMsg, response)
      }
    } catch (error) {
      console.error('设置进度体系失败:', error)
      showRequestError('设置进度体系失败，请重试', error)
    }
  })
}

// 获取选中的进度体系的进度列配置
const selectedProgressSystemColumns = computed(() => {
  if (!selectedProgressSystemId.value) {
    return []
  }
  
  // 先在管理员进度体系中查找
  const adminSystem = adminProgressSystems.value.find(system => system.id === selectedProgressSystemId.value)
  if (adminSystem) {
    return adminSystem.columns
  }
  
  // 再在租户进度体系中查找
  const tenantSystem = tenantProgressSystems.value.find(system => system.id === selectedProgressSystemId.value)
  if (tenantSystem) {
    return tenantSystem.columns
  }
  
  return []
})

// 页面加载时获取数据
onMounted(() => {
  loadProgressSystems()
})
</script>

<style scoped>
/* 组件内样式 */
</style>