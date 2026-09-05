<template>
  <div class="p-8">
    <div class="flex flex-col space-y-6">
      <div class="flex justify-between items-center">
        <div>
          <h2 class="text-2xl font-bold text-text">默认自定义进度列管理</h2>
          <p class="text-text-light mt-1">管理系统默认的自定义进度体系</p>
        </div>
      </div>

      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">进度体系管理</h3>
          <button 
            class="btn-primary"
            @click="showAddModal = true"
          >
            <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
            </svg>
            添加进度体系
          </button>
        </div>
        
        <div class="space-y-4">
          <div v-if="loading" class="text-center py-8">
            <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary mx-auto"></div>
            <p class="mt-4 text-gray-600">加载中...</p>
          </div>
          <div v-else-if="progressSystems.length === 0" class="text-center py-8">
            <p class="text-gray-600">暂无进度体系</p>
            <button 
              class="mt-4 btn-primary"
              @click="showAddModal = true"
            >
              添加第一个进度体系
            </button>
          </div>
          <div 
            v-else
            class="space-y-3"
          >
            <div 
              v-for="system in progressSystems" 
              :key="system.id"
              class="border border-gray-200 rounded-lg hover:bg-gray-50"
            >
              <div class="flex items-center justify-between p-3">
                <div class="flex items-center space-x-4">
                  <span class="font-medium">{{ system.name }}</span>
                  <span v-if="system.is_default" class="px-2 py-0.5 bg-primary/10 text-primary rounded text-xs">默认</span>
                </div>
                <div class="flex items-center space-x-2">
                  <button 
                    class="text-gray-500 hover:text-primary"
                    @click="editProgressSystem(system)"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path>
                    </svg>
                  </button>
                  <button 
                    class="text-gray-500 hover:text-primary"
                    @click="setAsDefault(system)"
                    :disabled="system.is_default"
                  >
                    <span class="mr-1">{{ system.is_default ? '已设为默认' : '设为默认' }}</span>
                  </button>
                  <button 
                    class="text-gray-500 hover:text-red-500"
                    @click="deleteProgressSystem(system)"
                    :disabled="system.is_default"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                    </svg>
                  </button>
                </div>
              </div>
              <div class="p-3 border-t border-gray-100">
                <h4 class="font-medium mb-2">进度列配置</h4>
                <div class="grid grid-cols-3 gap-2">
                  <div 
                    v-for="column in system.columns" 
                    :key="column.id || column.name"
                    class="px-2 py-1 bg-gray-50 rounded text-sm"
                  >
                    {{ column.name }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加进度体系模态框 -->
    <div v-if="showAddModal" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-2xl">
        <h4 class="text-lg font-bold mb-4">{{ editingSystem ? '编辑进度体系' : '添加新进度体系' }}</h4>
        <div class="space-y-4">
          <div>
            <label for="system-name" class="block text-sm font-medium text-gray-700 mb-1">进度体系名称</label>
            <input 
              type="text" 
              id="system-name"
              v-model="newSystem.name"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
              placeholder="例如：标准进度体系"
            >
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">进度列配置</label>
            <div class="space-y-2">
              <div 
                v-for="(column, index) in newSystem.columns" 
                :key="index"
                class="flex items-center space-x-2"
              >
                <input 
                  type="text" 
                  v-model="column.name"
                  class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                  placeholder="进度列名称"
                >
                <button 
                  class="text-gray-500 hover:text-red-500"
                  @click="removeColumn(index)"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                  </svg>
                </button>
              </div>
              <button 
                class="flex items-center text-primary hover:text-primary-dark"
                @click="addColumn"
              >
                <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                </svg>
                添加进度列
              </button>
            </div>
          </div>
          <div class="flex justify-end space-x-3 pt-4">
            <button 
              class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
              @click="showAddModal = false; resetNewSystem()"
            >
              取消
            </button>
            <button 
              class="btn-primary"
              @click="saveProgressSystem"
              :disabled="!newSystem.name || newSystem.columns.length === 0"
            >
              保存
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch, extractErrorMessage } from '../utils/apiUtils.js';
import { showRequestError } from '../utils/requestErrorDisplay.js';
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js';

import { ref, onMounted } from 'vue'
import { getCookie } from '../utils/cookieUtils'

const progressSystems = ref([])
const loading = ref(true)
const showAddModal = ref(false)
const editingSystem = ref(null)

// 新进度体系表单
const newSystem = ref({
  name: '',
  columns: [
    { name: '待办' },
    { name: '进行中' },
    { name: '已完成' }
  ]
})



// 加载进度体系列表
const loadProgressSystems = async () => {
  loading.value = true
  try {
    const response = await apiFetch('/api/system-admin/progress-systems/', {
      headers: {
      }
    })
    
    const result = await response.json()
    
    if (result.status === 'success') {
      progressSystems.value = result.project_progress_systems || []
    } else {
      console.error('加载进度体系失败:', result.message)
    }
  } catch (error) {
    console.error('加载进度体系失败:', error)
  } finally {
    loading.value = false
  }
}

// 添加进度列
const addColumn = () => {
  newSystem.value.columns.push({ name: '' })
}

// 移除进度列
const removeColumn = (index) => {
  newSystem.value.columns.splice(index, 1)
}

// 编辑进度体系
const editProgressSystem = (system) => {
  editingSystem.value = system.id
  newSystem.value = {
    name: system.name,
    columns: [...system.columns]
  }
  showAddModal.value = true
}

const saveProgressSystemGuard = createClickGuard()
const setAsDefaultGuard = createClickGuard()
const deleteProgressSystemGuard = createClickGuard()

// 保存进度体系
const saveProgressSystem = async () => {
  if (!newSystem.value.name || newSystem.value.columns.length === 0) {
    return
  }
  // OPT-20260819-038: 连点/超时重试会双发 POST/PUT — createClickGuard 在途锁 +
  // Idempotency-Key（后端幂等去重）
  await saveProgressSystemGuard.run(async ({ idempotencyKey }) => {
    try {
      const url = editingSystem.value ? `/api/system-admin/progress-systems/${editingSystem.value}/` : '/api/system-admin/progress-systems/'
      const method = editingSystem.value ? 'PUT' : 'POST'

      const response = await apiFetch(url, {
        method,
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify(newSystem.value)
      })

      const result = await response.json()

      if (result.status === 'success') {
        progressSystems.value = result.project_progress_systems || []
        showAddModal.value = false
        resetNewSystem()
      } else {
        const errorMsg = extractErrorMessage(result, response, '未知错误')
        showRequestError('保存进度体系失败: ' + errorMsg, response)
      }
    } catch (error) {
      console.error('保存进度体系失败:', error)
      showRequestError('保存进度体系失败，请重试', error)
    }
  })
}

// 设置为默认
const setAsDefault = async (system) => {
  // OPT-20260819-038: 设置默认是写操作，防连点双发 POST
  await setAsDefaultGuard.run(async ({ idempotencyKey }) => {
    try {
      const response = await apiFetch(`/api/system-admin/progress-systems/${system.id}/set-default/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
      })

      const result = await response.json()

      if (result.status === 'success') {
        progressSystems.value = result.project_progress_systems || []
      } else {
        const errorMsg = extractErrorMessage(result, response, '未知错误')
        showRequestError('设置默认进度体系失败: ' + errorMsg, response)
      }
    } catch (error) {
      console.error('设置默认进度体系失败:', error)
      showRequestError('设置默认进度体系失败，请重试', error)
    }
  })
}

// 删除进度体系
const deleteProgressSystem = async (system) => {
  if (confirm('确定要删除这个进度体系吗？')) {
    // OPT-20260819-038: 删除是破坏性写操作，防连点双发 DELETE
    await deleteProgressSystemGuard.run(async ({ idempotencyKey }) => {
      try {
        const response = await apiFetch(`/api/system-admin/progress-systems/${system.id}/`, {
          method: 'DELETE',
          headers: mergeIdempotencyHeaders({}, idempotencyKey),
        })

        const result = await response.json()

        if (result.status === 'success') {
          progressSystems.value = result.project_progress_systems || []
        } else {
          const errorMsg = extractErrorMessage(result, response, '未知错误')
          showRequestError('删除进度体系失败: ' + errorMsg, response)
        }
      } catch (error) {
        console.error('删除进度体系失败:', error)
        showRequestError('删除进度体系失败，请重试', error)
      }
    })
  }
}

// 重置新进度体系表单
const resetNewSystem = () => {
  newSystem.value = {
    name: '',
    columns: [
      { name: '待办' },
      { name: '进行中' },
      { name: '已完成' }
    ]
  }
  editingSystem.value = null
}

// 页面加载时获取进度体系列表
onMounted(() => {
  loadProgressSystems()
})
</script>

<style scoped>
/* 组件内样式 */
</style>