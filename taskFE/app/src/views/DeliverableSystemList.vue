<template>
<!-- 500-line rule exception: DeliverableSystemList is 645 lines (145 over). Vue SFC co-locates template+script+style. Split requires sub-component extraction. -->
  <TenantPageAccessEmpty v-if="!accessAllowed" page-key="settings.deliverable" />
  <div v-else class="p-8">
    <div class="flex flex-col space-y-6">
      <div class="flex justify-between items-center">
        <div>
          <h2 class="text-2xl font-bold text-text">交付物体系列表</h2>
          <p class="text-text-light mt-1">管理交付物体系</p>
        </div>
        <button 
          class="btn-primary"
          @click="handleCreateDeliverableSystem"
        >
          <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
          </svg>
          创建交付物体系
        </button>
      </div>

      <!-- 默认交付物体系设置 -->
      <div class="bg-white p-6 rounded-xl shadow">
        <h3 class="text-xl font-bold text-text mb-6">默认交付物体系设置</h3>
        <div class="flex items-center space-x-4">
          <div class="w-1/3">
            <label class="block text-sm font-medium text-text-light mb-1">
              选择默认交付物体系
            </label>
            <select 
              v-model="selectedDefaultDeliverableSystem"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
              @change="handleDefaultDeliverableSystemChange"
            >
              <option value="" disabled>请选择交付物体系</option>
              <option 
                v-for="deliverableSystem in deliverableSystems" 
                :key="deliverableSystem.id"
                :value="deliverableSystem.id"
              >
                {{ deliverableSystem.name }}{{ deliverableSystem.is_default ? ' (当前默认)' : '' }}
              </option>
            </select>
          </div>
          <div class="w-2/3">
            <label class="block text-sm font-medium text-text-light mb-1">
              当前默认交付物体系
            </label>
            <div class="flex items-center">
              <span v-if="currentDefaultDeliverableSystem" class="font-medium">
                {{ currentDefaultDeliverableSystem.name }}
              </span>
              <span v-else class="text-gray-500">
                暂无默认交付物体系
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 交付物体系列表 -->
      <div class="bg-white p-6 rounded-xl shadow">
        <h3 class="text-xl font-bold text-text mb-6">交付物体系管理</h3>
        
        <div class="space-y-4">
          <div v-if="loadingDeliverableSystems" class="text-center py-8">
            <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary mx-auto"></div>
            <p class="mt-4 text-gray-600">加载中...</p>
          </div>
          <div v-else-if="deliverableSystems.length === 0" class="text-center py-8">
            <p class="text-gray-600">暂无交付物体系</p>
            <button 
              class="mt-4 btn-primary"
              @click="handleCreateDeliverableSystem"
            >
              创建第一个交付物体系
            </button>
          </div>
          <div 
            v-else
            class="space-y-3"
          >
            <div 
              v-for="deliverableSystem in deliverableSystems" 
              :key="deliverableSystem.id"
              class="border border-gray-200 rounded-lg hover:bg-gray-50"
            >
              <div class="flex items-center justify-between p-4">
                <div>
                  <div class="flex items-center">
                    <h4 class="font-medium">{{ deliverableSystem.name }}</h4>
                    <span v-if="deliverableSystem.is_default" class="px-2 py-0.5 bg-primary/10 text-primary rounded text-xs ml-2">默认</span>
                  </div>
                  <p v-if="deliverableSystem.description" class="text-sm text-text-light mt-1">{{ deliverableSystem.description }}</p>
                  <div class="flex flex-wrap gap-2 mt-2">
                    <span 
                      v-for="(level, index) in deliverableSystem.level_names" 
                      :key="index"
                      class="px-2 py-1 bg-primary/10 text-primary rounded text-xs"
                    >
                      {{ level }}
                    </span>
                  </div>
                </div>
                <div class="flex items-center space-x-2">
                  <button 
                    v-if="!deliverableSystem.is_system" 
                    class="text-gray-500 hover:text-primary"
                    @click="handleEditDeliverableSystem(deliverableSystem)"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"></path>
                    </svg>
                  </button>
                  <button 
                    v-if="!deliverableSystem.is_system" 
                    class="text-gray-500 hover:text-red-500"
                    @click="handleDeleteDeliverableSystem(deliverableSystem)"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建交付物体系模态框 -->
    <div v-if="showCreateDeliverableSystemModal" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6 z-10000 relative">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">{{ editingDeliverableSystem ? '编辑交付物体系' : '创建交付物体系' }}</h3>
          <button @click="closeCreateDeliverableSystemModal" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
        
        <!-- 创建交付物体系表单 -->
        <form @submit.prevent="handleCreateDeliverableSystemSubmit">
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
                <div v-for="i in createDeliverableSystemForm.levelCount" :key="i" class="flex items-center space-x-2">
                  <div class="flex items-center space-x-2 w-full">
                    <input type="text" v-model="createDeliverableSystemForm.level_names[i-1]" 
                           class="w-full px-4 py-2 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                           :placeholder="`第${i}层（如：${getDefaultLevelName(i)}）`">
                    <div class="flex items-center space-x-1">
                      <button type="button" @click="addLevel(i)" :disabled="createDeliverableSystemForm.levelCount >= 5" 
                              class="p-1 rounded-full border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
                        </svg>
                      </button>
                      <button type="button" @click="removeLevel(i)" :disabled="createDeliverableSystemForm.levelCount <= 1" 
                              class="p-1 rounded-full border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18 12H6"></path>
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <div class="flex justify-end space-x-3 mt-6">
            <button type="button" @click="closeCreateDeliverableSystemModal" 
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="submit" :disabled="creatingDeliverableSystem" 
                    class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
              {{ creatingDeliverableSystem ? '保存中...' : (editingDeliverableSystem ? '保存' : '创建交付物体系') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js';

import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getCookie } from '../utils/cookieUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { useTenantPageAccess } from '../composables/useTenantPageAccess.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import TenantPageAccessEmpty from '../components/TenantPageAccessEmpty.vue'

const route = useRoute()
const router = useRouter()
const { accessAllowed } = useTenantPageAccess('settings.deliverable')

// 交付物体系相关
const deliverableSystems = ref([])
const loadingDeliverableSystems = ref(false)
const editingDeliverableSystem = ref(null)
const showCreateDeliverableSystemModal = ref(false)
const creatingDeliverableSystem = ref(false)

// OPT-20260819-038: 创建/删除/设默认交付物体系均为写操作，防连点/超时重试双发
const createDeliverableSystemGuard = createClickGuard()
const deleteDeliverableSystemGuard = createClickGuard()
const setDefaultDeliverableSystemGuard = createClickGuard()

// 默认交付物体系设置相关
const selectedDefaultDeliverableSystem = ref('')
const currentDefaultDeliverableSystem = ref(null)

// 创建交付物体系表单数据
const createDeliverableSystemForm = ref({
  name: '',
  description: '',
  levelCount: 5,
  level_names: ['价值流', '业务流程', '活动', '工作项 / 交付物', '子工作项 / 子交付物']
})



// 获取当前租户ID
const getCurrentTenantId = () => {
  return route.params.tenant || ''
}

// 获取交付物体系列表
const fetchDeliverableSystems = async () => {
  const tenantId = getCurrentTenantId()
  if (!tenantId) return
  
  loadingDeliverableSystems.value = true
  try {
    // 并行获取公司交付物体系与默认交付物体系设置。
    // 注：后端 handleListDeliverableSystems 的列表已含系统级体系（company_id=? OR is_system=1），
    // 无需再发无 tenant 的 /api/projects/deliverable-systems/（必 400 tenant_id required，OPT-20260808-005）。
    const [companyDeliverableSystemsResponse, defaultDeliverableSystemResponse] = await Promise.all([
      // 获取公司交付物体系（含系统级体系，is_system 由后端标注）
      apiFetch(`/api/projects/deliverable-systems/tenant_id/${tenantId}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      }),
      // 获取默认交付物体系设置
      apiFetch(`/api/projects/default-deliverable-system/tenant_id/${tenantId}`, {
        method: 'GET',
        headers: {
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })
    ])

    let companyDeliverableSystems = []
    if (companyDeliverableSystemsResponse.ok) {
      companyDeliverableSystems = await companyDeliverableSystemsResponse.json()
    }

    // 处理公司交付物体系数据，确保数据结构一致。
    // 注意：保留后端列表响应中的 is_default 标注（OPT-20260808-004 修复前此处
    // 强制覆盖为 false，而 default-deliverable-system 端点 404，导致页面恒显示
    // 「暂无默认交付物体系」）。下方 default-deliverable-system 覆盖仍作为权威来源。
    const mergedDeliverableSystems = companyDeliverableSystems.map(system => ({
      ...system
    }))
    
    // 处理默认交付物体系设置
    if (defaultDeliverableSystemResponse.ok) {
      const defaultDeliverableSystemData = await defaultDeliverableSystemResponse.json()
      if (defaultDeliverableSystemData.status === 'success' && defaultDeliverableSystemData.default_deliverable_system_id) {
        const defaultId = defaultDeliverableSystemData.default_deliverable_system_id
        mergedDeliverableSystems.forEach(system => {
          system.is_default = system.id === defaultId
        })
      }
    }
    
    // 先按是否为公司级别排序（公司级别在上），再按名称排序
    mergedDeliverableSystems.sort((a, b) => {
      // 公司级别的交付物体系（!is_system）排在前面
      if (!a.is_system && b.is_system) return -1
      if (a.is_system && !b.is_system) return 1
      // 同级别的按名称排序
      return a.name.localeCompare(b.name)
    })
    
    deliverableSystems.value = mergedDeliverableSystems
    
    // 更新当前默认交付物体系
    updateCurrentDefaultDeliverableSystem()
  } catch (error) {
    console.error('获取交付物体系列表失败:', error)
  } finally {
    loadingDeliverableSystems.value = false
  }
}

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
  if (createDeliverableSystemForm.value.levelCount < 5) {
    createDeliverableSystemForm.value.levelCount++
    // 如果新增的层级没有默认值，设置默认值
    const newLevel = createDeliverableSystemForm.value.levelCount
    if (!createDeliverableSystemForm.value.level_names[newLevel-1]) {
      createDeliverableSystemForm.value.level_names[newLevel-1] = getDefaultLevelName(newLevel)
    }
  }
}

// 删除层级
const removeLevel = (index) => {
  if (createDeliverableSystemForm.value.levelCount > 1) {
    createDeliverableSystemForm.value.levelCount--
  }
}

// 处理创建交付物体系
const handleCreateDeliverableSystem = () => {
  showCreateDeliverableSystemModal.value = true
  // 阻止背景滚动
  document.body.style.overflow = 'hidden'
}

// 关闭创建交付物体系模态框
const closeCreateDeliverableSystemModal = () => {
  showCreateDeliverableSystemModal.value = false
  // 重置编辑状态
  editingDeliverableSystem.value = null
  // 恢复背景滚动
  document.body.style.overflow = ''
}

// 处理编辑交付物体系
const handleEditDeliverableSystem = (deliverableSystem) => {
  // 检查是否为系统级交付物体系
  if (deliverableSystem.is_system) {
    return
  }
  
  // 填充编辑表单数据
  editingDeliverableSystem.value = { ...deliverableSystem }
  // 初始化层级数据
  const formData = {
    name: deliverableSystem.name,
    description: deliverableSystem.description || '',
    levelCount: deliverableSystem.level_names?.length || 5,
    level_names: [...(deliverableSystem.level_names || [])]
  }
  // 为未填写的层级设置默认值
  for (let i = deliverableSystem.level_names?.length || 0; i < 5; i++) {
    if (!formData.level_names[i]) {
      formData.level_names[i] = getDefaultLevelName(i + 1)
    }
  }
  createDeliverableSystemForm.value = formData
  showCreateDeliverableSystemModal.value = true
  // 阻止背景滚动
  document.body.style.overflow = 'hidden'
}

// 处理删除交付物体系
const handleDeleteDeliverableSystem = async (deliverableSystem) => {
  // 检查是否为系统级交付物体系
  if (deliverableSystem.is_system) {
    return
  }

  // OPT-20260819-038: 删除是写操作，防连点/超时重试双发 DELETE
  await deleteDeliverableSystemGuard.run(async ({ idempotencyKey }) => {
    let response
    try {
      if (confirm('确定要删除这个交付物体系吗？')) {
        const tenantId = getCurrentTenantId()
        response = await apiFetch(`/api/projects/deliverable-systems/tenant_id/${tenantId}/${deliverableSystem.id}/`, {
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
          // 刷新交付物体系列表
          await fetchDeliverableSystems()
        } else {
          throw new Error('删除交付物体系失败')
        }
      }
    } catch (error) {
      console.error('删除交付物体系失败:', error)
      showRequestError('删除交付物体系失败: ' + error.message, response || error)
    }
  })
}

// 处理创建交付物体系表单提交
const handleCreateDeliverableSystemSubmit = async () => {
  const tenantId = getCurrentTenantId()
  if (!tenantId) return

  // OPT-20260819-038: 创建/编辑交付物体系是写操作，防连点/超时重试双发 POST/PUT
  await createDeliverableSystemGuard.run(async ({ idempotencyKey }) => {
    creatingDeliverableSystem.value = true
    let response
    try {
      const url = editingDeliverableSystem.value ? `/api/projects/deliverable-systems/tenant_id/${tenantId}/${editingDeliverableSystem.value.id}/` : `/api/projects/deliverable-systems/tenant_id/${tenantId}`
      const method = editingDeliverableSystem.value ? 'PUT' : 'POST'

      // 构建请求体，使用level_names数组
      const requestBody = {
        name: createDeliverableSystemForm.value.name,
        description: createDeliverableSystemForm.value.description,
        levelCount: createDeliverableSystemForm.value.levelCount,
        level_names: createDeliverableSystemForm.value.level_names.slice(0, createDeliverableSystemForm.value.levelCount)
      }

      response = await apiFetch(url, {
        method: method,
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          idempotencyKey,
        ),
        credentials: 'include',
        body: JSON.stringify(requestBody)
      })

      if (response.ok) {
        // 关闭模态框
        closeCreateDeliverableSystemModal()
        // 重置表单和编辑状态
        createDeliverableSystemForm.value = {
          name: '',
          description: '',
          levelCount: 5,
          level_names: ['价值流', '业务流程', '活动', '工作项 / 交付物', '子工作项 / 子交付物']
        }
        editingDeliverableSystem.value = null
        // 刷新交付物体系列表
        await fetchDeliverableSystems()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || '保存交付物体系失败')
        if (response.traceId) err.traceId = response.traceId
        throw err
      }
    } catch (error) {
      console.error('保存交付物体系失败:', error)
      showRequestError('保存交付物体系失败: ' + error.message, response || error)
    } finally {
      creatingDeliverableSystem.value = false
    }
  })
}

// 安全解析 JSON 响应：非 ok 响应提取 _errorData 错误信息，traceId 从 response 继承
const _safeParseSetDefaultResult = async (response) => {
  if (!response.ok) {
    let errorMsg = `HTTP ${response.status}`
    const ed = response._errorData
    if (ed) {
      if (ed._rawErrorText) {
        const raw = String(ed._rawErrorText).trim()
        errorMsg = raw.length > 200 ? raw.substring(0, 200) + '...' : raw
      } else if (ed.message) {
        errorMsg = ed.message
      } else if (ed.detail) {
        errorMsg = ed.detail
      }
    }
    const err = new Error(errorMsg)
    if (response.traceId) err.traceId = response.traceId
    err.response = response
    throw err
  }
  try {
    return await response.json()
  } catch (jsonError) {
    const err = new Error(jsonError.message || '响应不是有效的 JSON')
    if (response.traceId) err.traceId = response.traceId
    err.response = response
    err.cause = jsonError
    throw err
  }
}

// 设置默认交付物体系 — 共享核心：成功返回 true，失败/无租户返回 false。
// OPT-20260807-053 收尾：此前 setAsDefault 仅经 defineExpose 供测试驱动（无模板入口），
// handleDefaultDeliverableSystemChange 重复了整套 URL 构建 + POST 逻辑；现收敛为单一实现。
const setAsDefault = async (deliverableSystem) => {
  const tenantId = getCurrentTenantId()
  if (!tenantId) return false

  // OPT-20260819-038: 设置默认交付物体系是写操作，防连点/超时重试双发 POST
  const outcome = await setDefaultDeliverableSystemGuard.run(async ({ idempotencyKey }) => {
    let response
    try {
      // 构建API URL — 系统交付物与公司交付物均走真实 id（曾硬编码 ${1}，
      // 而系统交付物体系 id 为 ds_xxx 格式 → 后端 404「交付物体系不存在」，OPT-20260807-053 根因）
      let url
      if (deliverableSystem.is_system) {
        url = `/api/projects/deliverable-systems/${deliverableSystem.id}/set-default/tenant_id/${tenantId}/`
      } else {
        url = `/api/projects/deliverable-systems/tenant_id/${tenantId}/${deliverableSystem.id}/set-default`
      }

      // 发送POST请求
      response = await apiFetch(url, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          idempotencyKey,
        ),
        credentials: 'include'
      })

      const result = await _safeParseSetDefaultResult(response)

      if (result.status === 'success') {
        // 后端返回合并后的 systems（已带 is_default 标注）；company_/system_ 旧字段名为历史兼容
        const systems =
          Array.isArray(result.systems) && result.systems.length > 0
            ? result.systems
            : result.company_deliverable_systems || []
        if (systems.length > 0) {
          deliverableSystems.value = systems
        }
        return true
      } else {
        const err = new Error(result.message || '设置默认交付物体系失败')
        if (response.traceId) err.traceId = response.traceId
        throw err
      }
    } catch (error) {
      console.error('设置默认交付物体系失败:', error)
      // 优先用 response（带 traceId）作为 trace 源；回退到 error 自身
      showRequestError('设置默认交付物体系失败: ' + error.message, response || error)
      return false
    }
  })
  return outcome.skipped ? false : outcome.result
}

// 处理默认交付物体系变更（select @change 唯一入口）— 委托 setAsDefault，成功后刷新当前默认
const handleDefaultDeliverableSystemChange = async () => {
  const tenantId = getCurrentTenantId()
  if (!tenantId || !selectedDefaultDeliverableSystem.value) return

  // 找到选中的交付物体系
  const selectedSystem = deliverableSystems.value.find(
    system => system.id === selectedDefaultDeliverableSystem.value
  )

  if (!selectedSystem) return

  const ok = await setAsDefault(selectedSystem)
  if (ok) {
    // 更新当前默认交付物体系
    updateCurrentDefaultDeliverableSystem()
  }
}

// 更新当前默认交付物体系
const updateCurrentDefaultDeliverableSystem = () => {
  const defaultSystem = deliverableSystems.value.find(system => system.is_default)
  currentDefaultDeliverableSystem.value = defaultSystem
  if (defaultSystem) {
    selectedDefaultDeliverableSystem.value = defaultSystem.id
  }
}

// 页面加载时获取交付物体系列表
onMounted(() => {
  fetchDeliverableSystems()
})

// 暴露给测试（单测直接驱动 set-default 请求；生产无影响）
defineExpose({
  setAsDefault,
  handleDefaultDeliverableSystemChange,
  fetchDeliverableSystems,
  deliverableSystems,
})
</script>
<style scoped>
/* 组件内样式 */
</style>
