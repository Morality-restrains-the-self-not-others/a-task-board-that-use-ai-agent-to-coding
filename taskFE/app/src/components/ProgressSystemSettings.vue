<template>
  <div class="bg-white p-6 rounded-xl shadow">
    <div class="flex justify-between items-center mb-6">
      <h3 class="text-xl font-bold text-text">进度体系设置</h3>
      <button 
        class="text-gray-500 hover:text-gray-700"
        @click="$emit('close')"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
        </svg>
      </button>
    </div>
    
    <div class="space-y-4">
      <div class="flex items-center space-x-4">
        <div class="w-1/3">
          <label for="progress-system-select" class="block text-sm font-medium text-text-light mb-1">
            选择进度体系
          </label>
          <select 
            id="progress-system-select" 
            v-model="selectedProgressSystem"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          >
            <option value="" disabled>加载中...</option>
          </select>
        </div>
        <div class="w-2/3">
          <label class="block text-sm font-medium text-text-light mb-1">
            当前进度体系层级
          </label>
          <div class="flex flex-wrap gap-2">
            <span 
              v-if="currentProgressStatus.length === 0" 
              class="px-3 py-1 bg-gray-100 text-gray-600 rounded-full text-sm"
            >
              未选择进度体系
            </span>
            <span 
              v-for="status in currentProgressStatus" 
              :key="status.id"
              class="px-3 py-1 bg-primary/10 text-primary rounded-full text-sm"
            >
              {{ status.name }}
            </span>
          </div>
        </div>
      </div>
      <div class="flex justify-end">
        <button 
          id="save-progress-system-btn" 
          class="btn-primary"
          :disabled="saving || saveGuard.isBusy()"
          @click="saveProgressSystem"
        >
          <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path>
          </svg>
          {{ saving ? '保存中...' : '保存设置' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch, extractErrorMessage } from '../utils/apiUtils.js';
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js';

import { ref, onMounted, watch } from 'vue'
import { getCookie } from '../utils/cookieUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'

const props = defineProps({
  workspaceId: {
    type: String,
    default: ''
  },
  tenantId: {
    type: [String, Number],
    default: ''
  }
})

const emit = defineEmits(['close'])

const selectedProgressSystem = ref('')
const currentProgressStatus = ref([])
const saving = ref(false)
const loading = ref(true)
const progressSystems = ref([]) // 存储所有进度体系数据（系统+公司）

const getCurrentWorkspaceId = () => {
  return props.workspaceId || ''
}

const loadProgressSystems = async () => {
  loading.value = true
  try {
    const workspaceId = getCurrentWorkspaceId()
    
    // 确保tenantId有值
    if (!props.tenantId) {
      console.error('租户ID为空，无法加载公司进度体系')
      // 使用默认选项
      const progressSystemSelect = document.getElementById('progress-system-select')
      if (progressSystemSelect) {
        progressSystemSelect.innerHTML = ''
        const option = document.createElement('option')
        option.value = ''
        option.textContent = '暂无可用进度体系'
        option.disabled = true
        progressSystemSelect.appendChild(option)
      }
      loading.value = false
      return
    }
    
    console.log('加载进度体系，租户ID:', props.tenantId)
    
    // 获取工作空间信息，用于获取当前进度体系ID
    let workspaceResponse = null
    if (workspaceId) {
      workspaceResponse = await apiFetch(`/api/projects/manage-progress-column/tenant_id/${props.tenantId}?workspace_id=${workspaceId}`, {
        headers: {
        }
      })
    }
    
    // 加载系统进度体系
    let systemProgressSystems = []
    const systemResponse = await apiFetch('/api/system/progress-systems/', {
      headers: {
      }
    })
    
    if (systemResponse.ok) {
      const systemResult = await systemResponse.json()
      if (systemResult.status === 'success') {
        systemProgressSystems = systemResult.project_progress_systems || []
      }
    }
    
    // 加载公司自建进度体系
    let companyProgressSystems = []
    const companyResponse = await apiFetch(`/api/projects/progress-systems/tenant_id/${props.tenantId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (companyResponse.ok) {
      const companyResult = await companyResponse.json()
      // 检查返回数据结构，确保companyProgressSystems是一个数组
      if (Array.isArray(companyResult)) {
        companyProgressSystems = companyResult
      } else if (companyResult.status === 'success' && Array.isArray(companyResult.project_progress_systems)) {
        companyProgressSystems = companyResult.project_progress_systems
      } else {
        console.error('公司进度体系数据格式不正确:', companyResult)
        companyProgressSystems = []
      }
    } else {
      console.error('获取公司进度体系失败:', companyResponse.status)
      // 尝试获取响应文本，看看具体错误
      try {
        const errorText = await companyResponse.text()
        console.error('错误响应:', errorText.substring(0, 200)) // 只显示前200个字符
      } catch (e) {
        console.error('无法获取错误响应:', e)
      }
    }
    
    // 合并系统和公司进度体系
    progressSystems.value = [...systemProgressSystems, ...companyProgressSystems]
    
    // 填充下拉框
    const progressSystemSelect = document.getElementById('progress-system-select')
    if (progressSystemSelect) {
      progressSystemSelect.innerHTML = ''
      
      // 添加系统进度体系选项组
      if (systemProgressSystems.length > 0) {
        const systemOptgroup = document.createElement('optgroup')
        systemOptgroup.label = '系统进度体系'
        
        systemProgressSystems.forEach(system => {
          const option = document.createElement('option')
          option.value = system.id
          option.textContent = `${system.name}${system.is_default ? ' (默认)' : ''}`
          systemOptgroup.appendChild(option)
        })
        
        progressSystemSelect.appendChild(systemOptgroup)
      }
      
      // 添加公司自建进度体系选项组
      if (companyProgressSystems.length > 0) {
        const companyOptgroup = document.createElement('optgroup')
        companyOptgroup.label = '公司自建进度体系'
        
        companyProgressSystems.forEach(system => {
          const option = document.createElement('option')
          option.value = system.id
          option.textContent = system.name
          companyOptgroup.appendChild(option)
        })
        
        progressSystemSelect.appendChild(companyOptgroup)
      }
      
      // 如果没有任何进度体系
      if (progressSystems.value.length === 0) {
        const option = document.createElement('option')
        option.value = ''
        option.textContent = '暂无可用进度体系'
        option.disabled = true
        progressSystemSelect.appendChild(option)
      }
      
      // 获取当前进度体系ID
      if (workspaceResponse && workspaceResponse.ok) {
        const workspaceData = await workspaceResponse.json()
        if (workspaceData.status === 'success' && workspaceData.current_progress_system_id) {
          selectedProgressSystem.value = workspaceData.current_progress_system_id
        }
      }
    }
    
    // 加载当前进度体系的层级信息
    updateCurrentProgressStatus()
  } catch (error) {
    console.error('加载进度体系失败:', error)
  } finally {
    loading.value = false
  }
}

const saveGuard = createClickGuard()

const saveProgressSystem = async () => {
  // OPT-20260819-038: 保存进度体系是写操作，防连点双发 POST
  await saveGuard.run(async ({ idempotencyKey }) => {
    saving.value = true

    try {
      const progressSystemId = selectedProgressSystem.value
      const workspaceId = getCurrentWorkspaceId()

      const form = new URLSearchParams()
      form.append('action', 'update_progress_system')
      form.append('workspace_id', workspaceId)
      form.append('progress_system_id', progressSystemId)

      const response = await apiFetch(`/api/projects/manage-progress-column/tenant_id/${props.tenantId}`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: form
      })

      const result = await response.json()

      if (result.status === 'success') {
        // 保存成功后，重新加载进度体系
        await loadProgressSystems()
      } else {
        const errorMsg = extractErrorMessage(result, response, '未知错误')
        showRequestError('保存进度体系失败: ' + errorMsg, response)
      }
    } catch (error) {
      console.error('保存进度体系失败:', error)
      showRequestError('保存进度体系失败，请重试', error)
    } finally {
      saving.value = false
    }
  })
}

// 更新当前进度体系的层级信息
const updateCurrentProgressStatus = () => {
  if (!selectedProgressSystem.value) {
    currentProgressStatus.value = []
    return
  }
  
  // 从已加载的进度体系数据中找到当前选中的进度体系
  const progressSystem = progressSystems.value.find(system => system.id === selectedProgressSystem.value)
  if (progressSystem) {
    // 将statuses转换为currentProgressStatus需要的格式
    if (progressSystem.statuses && Array.isArray(progressSystem.statuses)) {
      currentProgressStatus.value = progressSystem.statuses.map((status, index) => ({
        id: status.id || index + 1, // 使用实际ID或索引作为临时ID
        name: status.name
      }))
    } else if (progressSystem.columns && Array.isArray(progressSystem.columns)) {
      // 处理系统进度体系的columns格式
      currentProgressStatus.value = progressSystem.columns.map((column, index) => ({
        id: column.id || index + 1, // 使用实际ID或索引作为临时ID
        name: column.name
      }))
    } else {
      currentProgressStatus.value = []
    }
  } else {
    currentProgressStatus.value = []
  }
}

onMounted(() => {
  loadProgressSystems()
})

// 监听workspaceId变化，重新加载进度体系
watch(() => props.workspaceId, () => {
  loadProgressSystems()
})

// 监听tenantId变化，重新加载进度体系
watch(() => props.tenantId, () => {
  loadProgressSystems()
})

// 监听selectedProgressSystem变化，更新当前进度体系层级
watch(selectedProgressSystem, () => {
  updateCurrentProgressStatus()
})
</script>

<style scoped>
</style>