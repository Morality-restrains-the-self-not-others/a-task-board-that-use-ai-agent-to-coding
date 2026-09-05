<template>
<!-- 500-line rule exception: SystemAdminDeliverableSystem is 550 lines (50 over). Vue SFC requires template+script+style co-location. -->
  <div data-alias="view-system-admin-deliverable-system" id="deliverable-system" class="p-6">
    <!-- 顶部固定区域 -->
    <div class="mb-6">
      <div class="flex justify-between items-center">
        <h2 class="text-xl font-semibold text-gray-900">交付物体系管理</h2>
        <!-- 创建交付物体系按钮，点击打开模态框 -->
        <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="createDeliverableSystemModalVisible = true">
          创建交付物体系
        </button>
      </div>
    </div>
    
    <!-- 创建交付物体系模态框 -->
    <div v-if="createDeliverableSystemModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">创建交付物体系</h3>
          <button @click="createDeliverableSystemModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
        
        <!-- 创建交付物体系表单 -->
        <form @submit.prevent="handleCreateDeliverableSystem">
          <div class="space-y-4">
            <div>
              <label for="deliverable-system-name" class="block text-sm font-medium text-gray-700 mb-2">交付物体系名称</label>
              <input type="text" id="deliverable-system-name" v-model="createDeliverableSystemForm.name" 
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                     placeholder="输入交付物体系名称" required>
            </div>
            
            <div>
              <label for="deliverable-system-description" class="block text-sm font-medium text-gray-700 mb-2">交付物体系描述</label>
              <textarea id="deliverable-system-description" v-model="createDeliverableSystemForm.description" 
                        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                        placeholder="描述交付物体系的层级结构" rows="3"></textarea>
            </div>
            
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">层级结构</label>
              <div class="space-y-2">
                <div v-for="(level, index) in createDeliverableSystemForm.level_names" :key="index" class="flex items-center space-x-2">
                  <div class="flex items-center space-x-2 w-full">
                    <input type="text" v-model="createDeliverableSystemForm.level_names[index]" 
                           class="w-full px-4 py-2 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                           :placeholder="`第${index + 1}层（如：${getDefaultLevelName(index + 1)}）`">
                    <div class="flex items-center space-x-1">
                      <button type="button" @click="addLevel(index)" :disabled="createDeliverableSystemForm.level_names.length >= 5" 
                              class="p-1 rounded-full border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
                        </svg>
                      </button>
                      <button type="button" @click="removeLevel(index)" :disabled="createDeliverableSystemForm.level_names.length <= 1" 
                              class="p-1 rounded-full border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18 12H6"></path>
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
                <!-- 空状态处理 -->
                <div v-if="createDeliverableSystemForm.level_names.length === 0" class="flex items-center space-x-2">
                  <button type="button" @click="addLevel(0)" class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors">
                    添加第一层
                  </button>
                </div>
              </div>
            </div>
          </div>
          
          <div class="flex justify-end space-x-3 mt-6">
            <button type="button" @click="createDeliverableSystemModalVisible = false" 
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="submit" :disabled="creatingDeliverableSystem" 
                    class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
              {{ creatingDeliverableSystem ? '创建中...' : '创建交付物体系' }}
            </button>
          </div>
        </form>
      </div>
    </div>
    
    <!-- 编辑交付物体系模态框 -->
    <div v-if="editDeliverableSystemModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">编辑交付物体系</h3>
          <button @click="editDeliverableSystemModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
        
        <!-- 编辑交付物体系表单 -->
        <form @submit.prevent="handleEditDeliverableSystem">
          <input type="hidden" v-model="editDeliverableSystemForm.id">
          <div class="space-y-4">
            <div>
              <label for="edit-deliverable-system-name" class="block text-sm font-medium text-gray-700 mb-2">交付物体系名称</label>
              <input type="text" id="edit-deliverable-system-name" v-model="editDeliverableSystemForm.name" 
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                     placeholder="输入交付物体系名称" required>
            </div>
            
            <div>
              <label for="edit-deliverable-system-description" class="block text-sm font-medium text-gray-700 mb-2">交付物体系描述</label>
              <textarea id="edit-deliverable-system-description" v-model="editDeliverableSystemForm.description" 
                        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                        placeholder="描述交付物体系的层级结构" rows="3"></textarea>
            </div>
            
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">层级结构</label>
              <div class="space-y-2">
                <div v-for="(level, index) in editDeliverableSystemForm.level_names" :key="index" class="flex items-center space-x-2">
                  <div class="flex items-center space-x-2 w-full">
                    <input type="text" v-model="editDeliverableSystemForm.level_names[index]" 
                           class="w-full px-4 py-2 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                           :placeholder="`第${index + 1}层（如：${getDefaultLevelName(index + 1)}）`">
                    <div class="flex items-center space-x-1">
                      <button type="button" @click="addEditLevel(index)" :disabled="editDeliverableSystemForm.level_names.length >= 5" 
                              class="p-1 rounded-full border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
                        </svg>
                      </button>
                      <button type="button" @click="removeEditLevel(index)" :disabled="editDeliverableSystemForm.level_names.length <= 1" 
                              class="p-1 rounded-full border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18 12H6"></path>
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
                <!-- 空状态处理 -->
                <div v-if="editDeliverableSystemForm.level_names.length === 0" class="flex items-center space-x-2">
                  <button type="button" @click="addEditLevel(0)" class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors">
                    添加第一层
                  </button>
                </div>
              </div>
            </div>
          </div>
          
          <div class="flex justify-end space-x-3 mt-6">
            <button type="button" @click="editDeliverableSystemModalVisible = false" 
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="submit" :disabled="editingDeliverableSystem" 
                    class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
              {{ editingDeliverableSystem ? '保存中...' : '保存修改' }}
            </button>
          </div>
        </form>
      </div>
    </div>
    
    <!-- 删除确认模态框 -->
    <div v-if="deleteDeliverableSystemModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">删除交付物体系</h3>
          <button @click="deleteDeliverableSystemModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
        
        <p class="text-gray-600 mb-6">您确定要删除此交付物体系吗？此操作不可恢复。</p>
        
        <div class="flex justify-end space-x-3">
          <button type="button" @click="deleteDeliverableSystemModalVisible = false" 
                  class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
            取消
          </button>
          <button type="button" :disabled="deletingDeliverableSystem" @click="handleDeleteDeliverableSystem" 
                  class="px-4 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-all duration-300">
            {{ deletingDeliverableSystem ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>
    
    <!-- 加载状态 -->
    <div v-if="loadingDeliverableSystems" class="flex justify-center items-center py-20">
      <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
      <span class="ml-3 text-gray-600">加载交付物体系列表中...</span>
    </div>
    
    <!-- 主体内容 -->
    <div v-else>
      <!-- 交付物体系列表表格 -->
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">名称</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">描述</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">层级结构</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <!-- 空状态 -->
            <tr v-if="deliverableSystems.length === 0">
              <td colspan="6" class="px-6 py-12 text-center">
                <div class="flex flex-col items-center justify-center">
                  <svg class="w-16 h-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"></path>
                  </svg>
                  <h3 class="text-lg font-medium text-gray-900 mb-1">暂无交付物体系</h3>
                  <p class="text-gray-500 mb-6">系统中还没有任何交付物体系，请点击"创建交付物体系"按钮创建第一个交付物体系。</p>
                  <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="createDeliverableSystemModalVisible = true">
                    创建交付物体系
                  </button>
                </div>
              </td>
            </tr>
            <!-- 交付物体系列表 -->
            <tr v-for="deliverableSystem in deliverableSystems" :key="deliverableSystem.id">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ deliverableSystem.id }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{{ deliverableSystem.name }}</td>
              <td class="px-6 py-4 text-sm text-gray-500">{{ deliverableSystem.description }}</td>
              <td class="px-6 py-4 text-sm text-gray-500">{{ deliverableSystem.level_structure }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ formatDate(deliverableSystem.created_at) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                <button class="text-primary hover:text-primary/90 mr-3" @click="openEditDeliverableSystemModal(deliverableSystem)">
                  编辑
                </button>
                <button class="text-red-600 hover:text-red-800" @click="openDeleteDeliverableSystemModal(deliverableSystem)">
                  删除
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js';

/* @alias:view-system-admin-deliverable-system */
import { ref, onMounted } from 'vue'
import { getCookie } from '../utils/cookieUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

// 模态框可见性
const createDeliverableSystemModalVisible = ref(false)
const editDeliverableSystemModalVisible = ref(false)
const deleteDeliverableSystemModalVisible = ref(false)

// 加载状态
const loadingDeliverableSystems = ref(false)
const creatingDeliverableSystem = ref(false)

// OPT-20260819-038: 交付物体系创建/编辑/删除均为写操作，防连点/超时重试双发
const createDeliverableSystemGuard = createClickGuard()
const editDeliverableSystemGuard = createClickGuard()
const deleteDeliverableSystemGuard = createClickGuard()
const editingDeliverableSystem = ref(false)
const deletingDeliverableSystem = ref(false)

// 表单数据
const createDeliverableSystemForm = ref({
  name: '',
  description: '',
  level_names: ['价值流', '业务流程', '活动', '工作项 / 交付物', '子工作项 / 子交付物']
})

// 获取默认层级名称
const getDefaultLevelName = (level) => {
  const defaultNames = {
    1: '价值流',
    2: '业务流程',
    3: '活动',
    4: '工作项 / 交付物',
    5: '子工作项 / 子交付物'
  }
  return defaultNames[level] || ''
}

// 添加层级
const addLevel = (index) => {
  if (createDeliverableSystemForm.value.level_names.length < 5) {
    const newLevelIndex = createDeliverableSystemForm.value.level_names.length
    createDeliverableSystemForm.value.level_names.push(getDefaultLevelName(newLevelIndex + 1))
  }
}

// 删除层级
const removeLevel = (index) => {
  if (createDeliverableSystemForm.value.level_names.length > 1) {
    createDeliverableSystemForm.value.level_names.splice(index, 1)
  }
}

// 添加编辑层级
const addEditLevel = (index) => {
  if (editDeliverableSystemForm.value.level_names.length < 5) {
    const newLevelIndex = editDeliverableSystemForm.value.level_names.length
    editDeliverableSystemForm.value.level_names.push(getDefaultLevelName(newLevelIndex + 1))
  }
}

// 删除编辑层级
const removeEditLevel = (index) => {
  if (editDeliverableSystemForm.value.level_names.length > 1) {
    editDeliverableSystemForm.value.level_names.splice(index, 1)
  }
}

const editDeliverableSystemForm = ref({
  id: '',
  name: '',
  description: '',
  level_names: []
})

// 数据列表
const deliverableSystems = ref([])



// 格式化日期
const formatDate = (dateString) => {
  if (!dateString) {
    return '-';
  }
  
  // 先检查是否已经是日期对象
  if (dateString instanceof Date) {
    return dateString.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    });
  }
  
  // 转换为字符串，处理可能的对象类型
  const dateStr = String(dateString);
  
  // 直接尝试解析
  let date = new Date(dateStr);
  
  // 如果解析失败，尝试处理不同的日期格式
  if (isNaN(date.getTime())) {
    // 处理各种常见的日期格式
    let processedDateStr = dateStr;
    
    // 移除时区信息（如 +00:00 或 Z）
    processedDateStr = processedDateStr.replace(/(\+\d{2}:\d{2}|Z)$/, '');
    
    // 替换 T 为空格
    processedDateStr = processedDateStr.replace('T', ' ');
    
    // 处理 Django 默认的日期格式（如 2023-01-01 12:00:00+00:00）
    processedDateStr = processedDateStr.replace(/\.\d+/, ''); // 移除毫秒
    
    // 再次尝试解析
    date = new Date(processedDateStr);
    
    // 如果还是解析失败，尝试使用正则表达式提取日期时间部分
    if (isNaN(date.getTime())) {
      const dateRegex = /(\d{4})[-/](\d{2})[-/](\d{2})\s*(\d{2}):(\d{2}):(\d{2})/;
      const match = processedDateStr.match(dateRegex);
      if (match) {
        const [, year, month, day, hour, minute, second] = match;
        date = new Date(year, month - 1, day, hour, minute, second);
      }
    }
  }
  
  // 如果所有尝试都失败，返回原始字符串
  if (isNaN(date.getTime())) {
    return dateStr;
  }
  
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
}

// 刷新交付物体系列表
const refreshDeliverableSystems = async () => {
  loadingDeliverableSystems.value = true
  try {
    const response = await apiFetch('/api/system-admin/deliverable-systems/', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (response.ok) {
      deliverableSystems.value = await response.json()
    } else {
      const err = new Error('获取交付物体系列表失败')
      err.traceId = response.traceId || ''
      throw err
    }
  } catch (error) {
    console.error('刷新交付物体系列表失败:', error)
    showRequestError('刷新交付物体系列表失败', error)
  } finally {
    loadingDeliverableSystems.value = false
  }
}

// 打开编辑交付物体系模态框
const openEditDeliverableSystemModal = (deliverableSystem) => {
  // 填充表单数据
  editDeliverableSystemForm.value = {
    id: deliverableSystem.id,
    name: deliverableSystem.name,
    description: deliverableSystem.description,
    level_names: deliverableSystem.level_names && deliverableSystem.level_names.length > 0 ? deliverableSystem.level_names : ['价值流', '业务流程', '活动', '工作项 / 交付物', '子工作项 / 子交付物']
  }
  editDeliverableSystemModalVisible.value = true
}

// 打开删除交付物体系模态框
const openDeleteDeliverableSystemModal = (deliverableSystem) => {
  editDeliverableSystemForm.value.id = deliverableSystem.id
  deleteDeliverableSystemModalVisible.value = true
}

// 创建交付物体系
const handleCreateDeliverableSystem = async () => {
  // OPT-20260819-038: 创建交付物体系是写操作，防连点双发 POST
  await createDeliverableSystemGuard.run(async ({ idempotencyKey }) => {
    creatingDeliverableSystem.value = true
    try {
      const response = await apiFetch('/api/system-admin/deliverable-systems/', {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          idempotencyKey,
        ),
        credentials: 'include',
        body: JSON.stringify(createDeliverableSystemForm.value)
      })

      if (response.ok) {
        // 关闭模态框
        createDeliverableSystemModalVisible.value = false
        // 重置表单
        createDeliverableSystemForm.value = {
          name: '',
          description: '',
          level_names: ['价值流', '业务流程', '活动', '工作项 / 交付物', '子工作项 / 子交付物']
        }
        // 刷新交付物体系列表
        await refreshDeliverableSystems()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || '创建交付物体系失败')
        err.traceId = response.traceId || errorData._traceId || ''
        throw err
      }
    } catch (error) {
      console.error('创建交付物体系失败:', error)
      showRequestError('创建交付物体系失败: ' + error.message, error)
    } finally {
      creatingDeliverableSystem.value = false
    }
  })
}

// 编辑交付物体系
const handleEditDeliverableSystem = async () => {
  // OPT-20260819-038: 编辑交付物体系是写操作，防连点双发 PUT
  await editDeliverableSystemGuard.run(async ({ idempotencyKey }) => {
    editingDeliverableSystem.value = true
    try {
      const response = await apiFetch(`/api/system-admin/deliverable-systems/${editDeliverableSystemForm.value.id}/`, {
        method: 'PUT',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          idempotencyKey,
        ),
        credentials: 'include',
        body: JSON.stringify(editDeliverableSystemForm.value)
      })

      if (response.ok) {
        // 关闭模态框
        editDeliverableSystemModalVisible.value = false
        // 刷新交付物体系列表
        await refreshDeliverableSystems()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || '编辑交付物体系失败')
        err.traceId = response.traceId || errorData._traceId || ''
        throw err
      }
    } catch (error) {
      console.error('编辑交付物体系失败:', error)
      showRequestError('编辑交付物体系失败: ' + error.message, error)
    } finally {
      editingDeliverableSystem.value = false
    }
  })
}

// 删除交付物体系
const handleDeleteDeliverableSystem = async () => {
  // OPT-20260819-038: 删除交付物体系是写操作，防连点双发 DELETE
  await deleteDeliverableSystemGuard.run(async ({ idempotencyKey }) => {
    deletingDeliverableSystem.value = true
    try {
      const response = await apiFetch(`/api/system-admin/deliverable-systems/${editDeliverableSystemForm.value.id}/`, {
        method: 'DELETE',
        headers: mergeIdempotencyHeaders(
          {
            'X-Requested-With': 'XMLHttpRequest'
          },
          idempotencyKey,
        ),
        credentials: 'include'
      })

      if (response.ok) {
        // 关闭模态框
        deleteDeliverableSystemModalVisible.value = false
        // 刷新交付物体系列表
        await refreshDeliverableSystems()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || '删除交付物体系失败')
        err.traceId = response.traceId || errorData._traceId || ''
        throw err
      }
    } catch (error) {
      console.error('删除交付物体系失败:', error)
      showRequestError('删除交付物体系失败: ' + error.message, error)
    } finally {
      deletingDeliverableSystem.value = false
    }
  })
}

// 页面加载时获取数据
onMounted(async () => {
  await refreshDeliverableSystems()
})
</script>
<style scoped>
/* 组件内样式可以在这里添加 */
</style>
