<template>
<!-- 500-line rule exception: WorkspaceSettingsStatus is 554 lines (54 over, OPT-20260808-006 移除死代码后). Vue SFC co-locates template+script+style. Split requires sub-component extraction. -->
  <TenantPageAccessEmpty v-if="!accessAllowed" page-key="settings.status" />
  <div v-else class="p-8">
    <div class="flex flex-col space-y-6">
      <div class="flex justify-between items-center">
        <div>
          <h2 class="text-2xl font-bold text-text">工作空间设置</h2>
          <p class="text-text-light mt-1">管理工作空间的任务状态和进度体系</p>
        </div>
      </div>

      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">默认进度体系设置</h3>
          <div v-if="currentDefaultProgressSystem" class="px-3 py-1 bg-green-100 text-green-600 rounded-full text-sm">
            当前: {{ currentDefaultProgressSystem.name }}
          </div>
        </div>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">选择默认进度体系</label>
            <select 
              v-model="selectedProgressSystem"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
              @change="handleProgressSystemChange"
            >
              <option value="">请选择</option>
              <optgroup label="租户创建的">
                <option 
                  v-for="system in tenantProgressSystems" 
                  :key="'tenant-' + system.id"
                  :value="system.id"
                >
                  {{ system.name }}
                </option>
              </optgroup>
              <optgroup label="系统管理员创建的">
                <option 
                  v-for="system in adminProgressSystems" 
                  :key="'admin-' + system.id"
                  :value="system.id"
                >
                  {{ system.name }}{{ system.is_default ? ' (默认)' : '' }}
                </option>
              </optgroup>
            </select>

          </div>
        </div>
      </div>

      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">进度体系管理</h3>
          <button 
            id="add-status-btn"
            class="btn-primary"
            @click="showCreateTenantProgressSystemModal = true"
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
          <div v-else-if="tenantProgressSystems.length === 0 && adminProgressSystems.length === 0" class="text-center py-8">
            <p class="text-gray-600">暂无进度体系</p>
            <button 
              class="mt-4 btn-primary"
              @click="showCreateTenantProgressSystemModal = true"
            >
              添加第一个进度体系
            </button>
          </div>
          <div 
            v-else
            class="space-y-3"
            id="column-system-list"
          >
            <!-- 租户进度体系 -->
            <div 
              v-for="system in tenantProgressSystems" 
              :key="'tenant-' + system.id"
              class="flex items-center justify-between p-3 border border-gray-200 rounded-lg hover:bg-gray-50"
            >
              <div class="flex items-center space-x-4">
                <span class="font-medium">{{ system.name }}</span>
                <span class="px-2 py-0.5 bg-gray-100 text-gray-600 rounded text-xs">{{ system.columns.length }} 进度列</span>
                <span class="px-2 py-0.5 bg-green-100 text-green-600 rounded text-xs">租户</span>
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
                  class="text-gray-500 hover:text-red-500"
                  @click="deleteProgressSystem(system)"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                  </svg>
                </button>
              </div>
            </div>
            
            <!-- 系统进度体系 -->
            <div 
              v-for="system in adminProgressSystems" 
              :key="'admin-' + system.id"
              class="flex items-center justify-between p-3 border border-gray-200 rounded-lg bg-gray-50"
            >
              <div class="flex items-center space-x-4">
                <span class="font-medium">{{ system.name }}</span>
                <span class="px-2 py-0.5 bg-gray-100 text-gray-600 rounded text-xs">{{ system.columns.length }} 进度列</span>
                <span class="px-2 py-0.5 bg-blue-100 text-blue-600 rounded text-xs">系统</span>
              </div>
              <div class="flex items-center space-x-2">
                <span class="text-gray-400 text-xs">不可编辑</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建租户进度体系模态框 -->
    <div v-if="showCreateTenantProgressSystemModal" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-2xl">
        <h4 class="text-lg font-bold mb-4">创建租户进度体系</h4>
        <div class="space-y-4">
          <div>
            <label for="tenant-system-name" class="block text-sm font-medium text-gray-700 mb-1">进度体系名称</label>
            <input 
              type="text" 
              id="tenant-system-name"
              v-model="newTenantProgressSystem.name"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
              placeholder="例如：租户标准进度体系"
            >
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">列配置</label>
            <div class="space-y-2">
              <div 
                v-for="(column, index) in newTenantProgressSystem.columns" 
                :key="index"
                class="flex items-center space-x-2"
              >
                <input 
                  type="text" 
                  v-model="column.name"
                  class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                  placeholder="列名称"
                >
                <button 
                  class="text-gray-500 hover:text-red-500"
                  @click="removeTenantColumn(index)"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                  </svg>
                </button>
              </div>
              <button 
                class="flex items-center text-primary hover:text-primary-dark"
                @click="addTenantColumn"
              >
                <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                </svg>
                添加列
              </button>
            </div>
          </div>
          <div class="flex justify-end space-x-3 pt-4">
            <button 
              class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
              @click="showCreateTenantProgressSystemModal = false; resetNewTenantProgressSystem()"
            >
              取消
            </button>
            <button 
              class="btn-primary"
              @click="createTenantProgressSystem"
              :disabled="!newTenantProgressSystem.name || newTenantProgressSystem.columns.length === 0"
            >
              保存
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 编辑租户进度体系模态框 -->
    <div v-if="showEditProgressSystemModal" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-2xl">
        <h4 class="text-lg font-bold mb-4">编辑租户进度体系</h4>
        <div class="space-y-4">
          <div>
            <label for="edit-tenant-system-name" class="block text-sm font-medium text-gray-700 mb-1">进度体系名称</label>
            <input 
              type="text" 
              id="edit-tenant-system-name"
              v-model="editingProgressSystem.name"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
              placeholder="例如：租户标准进度体系"
            >
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">列配置</label>
            <div class="space-y-2">
              <div 
                v-for="(column, index) in editingProgressSystem.columns" 
                :key="index"
                class="flex items-center space-x-2"
              >
                <input 
                  type="text" 
                  v-model="column.name"
                  class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                  placeholder="列名称"
                >
                <button 
                  class="text-gray-500 hover:text-red-500"
                  @click="removeEditingColumn(index)"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                  </svg>
                </button>
              </div>
              <button 
                class="flex items-center text-primary hover:text-primary-dark"
                @click="addEditingColumn"
              >
                <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                </svg>
                添加列
              </button>
            </div>
          </div>
          <div class="flex justify-end space-x-3 pt-4">
            <button 
              class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
              @click="showEditProgressSystemModal = false"
            >
              取消
            </button>
            <button 
              class="btn-primary"
              @click="updateProgressSystem"
              :disabled="!editingProgressSystem.name || editingProgressSystem.columns.length === 0"
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
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js';

import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getCookie } from '../utils/cookieUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { useTenantPageAccess } from '../composables/useTenantPageAccess.js'
import TenantPageAccessEmpty from '../components/TenantPageAccessEmpty.vue'

const route = useRoute()
const { accessAllowed } = useTenantPageAccess('settings.status')

const loading = ref(true)

// 进度体系相关
const adminProgressSystems = ref([])
const tenantProgressSystems = ref([])
const selectedProgressSystem = ref('')
const currentDefaultProgressSystem = ref(null)
const showCreateTenantProgressSystemModal = ref(false)
// OPT-20260819-038: 进度体系 设默认/创建/删除/更新 均为写操作，防连点双发
const setDefaultProgressSystemGuard = createClickGuard()
const createProgressSystemGuard = createClickGuard()
const deleteProgressSystemGuard = createClickGuard()
const updateProgressSystemGuard = createClickGuard()

// 新租户进度体系表单
const newTenantProgressSystem = ref({
  name: '',
  columns: [
    { name: '待办' },
    { name: '进行中' },
    { name: '已完成' }
  ]
})



// 获取当前租户ID
const getCurrentTenantId = () => {
  return route.params.tenant || ''
}

// 加载进度体系
const loadProgressSystems = async () => {
  try {
    const tenantId = getCurrentTenantId()
    
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
    const tenantResponse = await apiFetch(`/api/projects/progress-systems/tenant_id/${tenantId}`, {
      headers: {
      }
    })
    
    if (tenantResponse.ok) {
      const tenantResult = await tenantResponse.json()
      if (tenantResult.status === 'success') {
        tenantProgressSystems.value = tenantResult.project_progress_systems || []
      }
    }
    
    // 加载当前默认进度体系
    await loadCurrentDefaultProgressSystem()
  } catch (error) {
    console.error('加载进度体系失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载当前默认进度体系
const loadCurrentDefaultProgressSystem = async () => {
  try {
    const tenantId = getCurrentTenantId()
    const response = await apiFetch(`/api/projects/settings/default-progress-system/tenant_id/${tenantId}/`, {
      headers: {
      }
    })
    
    if (response.ok) {
      const result = await response.json()
      if (result.status === 'success' && result.progress_system) {
        selectedProgressSystem.value = result.progress_system.id
        currentDefaultProgressSystem.value = result.progress_system
      }
    }
  } catch (error) {
    console.error('加载当前默认进度体系失败:', error)
  }
}

// 处理进度体系变更
const handleProgressSystemChange = async () => {
  // OPT-20260819-038: 设默认进度体系是写操作，防连点/超时重试双发 POST
  await setDefaultProgressSystemGuard.run(async ({ idempotencyKey }) => {
    try {
      const tenantId = getCurrentTenantId()
      const form = new URLSearchParams()
      form.append('system_id', selectedProgressSystem.value)

      const response = await apiFetch(`/api/projects/settings/default-progress-system/tenant_id/${tenantId}/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: form
      })

      const result = await response.json()

      if (result.status === 'success') {
        if (result.progress_system) {
          currentDefaultProgressSystem.value = result.progress_system
        }
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

// 添加租户列
const addTenantColumn = () => {
  newTenantProgressSystem.value.columns.push({ name: '' })
}

// 移除租户列
const removeTenantColumn = (index) => {
  newTenantProgressSystem.value.columns.splice(index, 1)
}

// 创建租户进度体系
const createTenantProgressSystem = async () => {
  if (!newTenantProgressSystem.value.name || newTenantProgressSystem.value.columns.length === 0) {
    return
  }

  // OPT-20260819-038: 创建进度体系是写操作，防连点/超时重试双发 POST
  await createProgressSystemGuard.run(async ({ idempotencyKey }) => {
    try {
      const tenantId = getCurrentTenantId()
      const response = await apiFetch(`/api/projects/progress-systems/tenant_id/${tenantId}`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify(newTenantProgressSystem.value)
      })

      const result = await response.json()

      if (result.status === 'success') {
        tenantProgressSystems.value = result.project_progress_systems || []
        showCreateTenantProgressSystemModal.value = false
        resetNewTenantProgressSystem()
      } else {
        const errorMsg = extractErrorMessage(result, response, '未知错误')
        showRequestError('创建租户进度体系失败: ' + errorMsg, response)
      }
    } catch (error) {
      console.error('创建租户进度体系失败:', error)
      showRequestError('创建租户进度体系失败，请重试', error)
    }
  })
}

// 重置新租户进度体系表单
const resetNewTenantProgressSystem = () => {
  newTenantProgressSystem.value = {
    name: '',
    columns: [
      { name: '待办' },
      { name: '进行中' },
      { name: '已完成' }
    ]
  }
}

// 编辑进度体系
const editingProgressSystem = ref({
  id: '',
  name: '',
  columns: []
})
const showEditProgressSystemModal = ref(false)

const editProgressSystem = (system) => {
  editingProgressSystem.value = {
    id: system.id,
    name: system.name,
    columns: [...system.columns]
  }
  showEditProgressSystemModal.value = true
}

// 删除进度体系
const deleteProgressSystem = async (system) => {
  if (confirm('确定要删除这个进度体系吗？')) {
    // OPT-20260819-038: 删除进度体系是写操作，防连点/超时重试双发 DELETE
    await deleteProgressSystemGuard.run(async ({ idempotencyKey }) => {
      try {
        const tenantId = getCurrentTenantId()
        const response = await apiFetch(`/api/projects/progress-systems/tenant_id/${tenantId}/${system.id}/`, {
          method: 'DELETE',
          headers: mergeIdempotencyHeaders({}, idempotencyKey),
        })

        const result = await response.json()

        if (result.status === 'success') {
          // 重新加载进度体系列表
          await loadProgressSystems()
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

// 更新进度体系
const updateProgressSystem = async () => {
  if (!editingProgressSystem.value.id || !editingProgressSystem.value.name || editingProgressSystem.value.columns.length === 0) {
    return
  }

  // OPT-20260819-038: 更新进度体系是写操作，防连点/超时重试双发 PUT
  await updateProgressSystemGuard.run(async ({ idempotencyKey }) => {
    try {
      const tenantId = getCurrentTenantId()
      const response = await apiFetch(`/api/projects/progress-systems/tenant_id/${tenantId}/${editingProgressSystem.value.id}/`, {
        method: 'PUT',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          name: editingProgressSystem.value.name,
          columns: editingProgressSystem.value.columns
        })
      })

      const result = await response.json()

      if (result.status === 'success') {
        tenantProgressSystems.value = result.project_progress_systems || []
        showEditProgressSystemModal.value = false
      } else {
        const errorMsg = extractErrorMessage(result, response, '未知错误')
        showRequestError('更新进度体系失败: ' + errorMsg, response)
      }
    } catch (error) {
      console.error('更新进度体系失败:', error)
      showRequestError('更新进度体系失败，请重试', error)
    }
  })
}

// 添加编辑列
const addEditingColumn = () => {
  editingProgressSystem.value.columns.push({ name: '' })
}

// 移除编辑列
const removeEditingColumn = (index) => {
  editingProgressSystem.value.columns.splice(index, 1)
}

// 页面加载时获取数据
onMounted(() => {
  // 进度体系（系统级 + 租户级）由 loadProgressSystems 统一加载；
  // 工作空间级进度列在 Go 架构下由 /api/projects/workspaces/.../progress-system/ 提供，
  // 由 ColumnSystemSettings 组件消费，本页不再调用 manage-progress-column。
  loadProgressSystems()
})
</script>
<style scoped>
/* 组件内样式 */
</style>
