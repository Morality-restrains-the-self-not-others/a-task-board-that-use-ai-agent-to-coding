<template>
  <div class="bg-white p-6 rounded-xl shadow">
    <div class="flex justify-between items-center mb-6">
      <h3 class="text-xl font-bold text-text">交付物体系设置</h3>
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
          <label for="deliverable-system-select" class="block text-sm font-medium text-text-light mb-1">
            选择交付物体系
          </label>
          <select 
            id="deliverable-system-select" 
            v-model="selectedDeliverableSystem"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          >
            <option value="" disabled>加载中...</option>
          </select>
        </div>
        <div class="w-2/3">
          <label class="block text-sm font-medium text-text-light mb-1">
            当前交付物体系层级
          </label>
          <div class="flex flex-wrap gap-2">
            <span 
              v-if="currentDeliverableTypes.length === 0" 
              class="px-3 py-1 bg-gray-100 text-gray-600 rounded-full text-sm"
            >
              暂无交付物体系层级
            </span>
            <span 
              v-for="deliverableType in currentDeliverableTypes" 
              :key="deliverableType.id"
              class="px-3 py-1 bg-primary/10 text-primary rounded-full text-sm"
            >
              {{ deliverableType.name }}
            </span>
          </div>
        </div>
      </div>
      <div class="flex justify-end">
        <button 
          id="save-deliverable-system-btn" 
          class="btn-primary"
          :disabled="saving"
          @click="saveDeliverableSystem"
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
import { apiFetch } from '../utils/apiUtils.js';
import toastService from '../utils/toastService'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

import { ref, onMounted, watch } from 'vue'
import { getCookie } from '../utils/cookieUtils'

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

const selectedDeliverableSystem = ref('')
const currentDeliverableTypes = ref([])
const saving = ref(false)
const loading = ref(true)
const deliverableSystems = ref([]) // 存储所有交付物体系数据

const getCurrentWorkspaceId = () => {
  return props.workspaceId || ''
}

const loadDeliverableSystems = async () => {
  loading.value = true
  try {
    const workspaceId = getCurrentWorkspaceId()
    
    // 确保tenantId有值
    if (!props.tenantId) {
      console.error('租户ID为空，无法加载公司交付物体系')
      // 使用默认选项
      const deliverableSystemSelect = document.getElementById('deliverable-system-select')
      if (deliverableSystemSelect) {
        deliverableSystemSelect.innerHTML = ''
        const option = document.createElement('option')
        option.value = ''
        option.textContent = '暂无可用交付物体系'
        option.disabled = true
        deliverableSystemSelect.appendChild(option)
      }
      loading.value = false
      return
    }
    
    console.log('加载交付物体系，租户ID:', props.tenantId)
    
    // 并行获取公司交付物体系和项目交付物体系，与WorkPanel.vue保持一致
    const [companyDeliverableSystemsResponse, projectDeliverableSystemsResponse, workspaceResponse] = await Promise.all([
      // 获取公司交付物体系
      apiFetch(`/api/projects/deliverable-systems/tenant_id/${props.tenantId}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      }),
      // 获取项目交付物体系
      apiFetch('/api/projects/deliverable-systems/', {
        method: 'GET',
        headers: {
          'Accept': 'application/json'
        },
        credentials: 'include'
      }),
      // 获取工作空间信息，用于获取当前交付物体系ID
      workspaceId ? apiFetch(`/api/projects/manage-deliverable-system/tenant_id/${props.tenantId}?workspace_id=${workspaceId}`, {
        headers: {
        }
      }) : Promise.resolve({ ok: false })
    ])
    
    console.log('公司交付物体系响应状态:', companyDeliverableSystemsResponse.status)
    console.log('项目交付物体系响应状态:', projectDeliverableSystemsResponse.status)
    
    let companyDeliverableSystems = []
    if (companyDeliverableSystemsResponse.ok) {
      companyDeliverableSystems = await companyDeliverableSystemsResponse.json()
      console.log('公司交付物体系数据:', companyDeliverableSystems)
    } else {
      console.error('获取公司交付物体系失败:', companyDeliverableSystemsResponse.status)
      // 尝试获取响应文本，看看具体错误
      try {
        const errorText = await companyDeliverableSystemsResponse.text()
        console.error('错误响应:', errorText.substring(0, 200)) // 只显示前200个字符
      } catch (e) {
        console.error('无法获取错误响应:', e)
      }
    }
    
    let projectDeliverableSystems = []
    if (projectDeliverableSystemsResponse.ok) {
      projectDeliverableSystems = await projectDeliverableSystemsResponse.json()
      console.log('项目交付物体系数据:', projectDeliverableSystems)
    } else {
      console.error('获取项目交付物体系失败:', projectDeliverableSystemsResponse.status)
      // 尝试获取响应文本，看看具体错误
      try {
        const errorText = await projectDeliverableSystemsResponse.text()
        console.error('错误响应:', errorText.substring(0, 200)) // 只显示前200个字符
      } catch (e) {
        console.error('无法获取错误响应:', e)
      }
    }
    
    // 处理项目交付物体系数据，确保数据结构一致
    const processedProjectDeliverableSystems = projectDeliverableSystems.map(system => ({
      id: system.id,
      name: system.name,
      description: system.description || '',
      is_system: true, // 标记为系统交付物体系
      level_names: system.level_names || []
    }))
    
    // 合并两个列表，去重（基于ID）
    const mergedDeliverableSystems = [...companyDeliverableSystems]
    const existingIds = new Set(companyDeliverableSystems.map(system => system.id))
    
    processedProjectDeliverableSystems.forEach(system => {
      if (!existingIds.has(system.id)) {
        mergedDeliverableSystems.push(system)
        existingIds.add(system.id)
      }
    })
    
    // 先按是否为公司级别排序（公司级别在上），再按名称排序
    mergedDeliverableSystems.sort((a, b) => {
      // 公司级别的交付物体系（!is_system）排在前面
      if (!a.is_system && b.is_system) return -1
      if (a.is_system && !b.is_system) return 1
      // 同级别的按名称排序
      return a.name.localeCompare(b.name)
    })
    
    // 存储所有交付物体系数据
    deliverableSystems.value = mergedDeliverableSystems
    
    // 填充下拉框
    const deliverableSystemSelect = document.getElementById('deliverable-system-select')
    if (deliverableSystemSelect) {
      deliverableSystemSelect.innerHTML = ''
      
      if (mergedDeliverableSystems.length > 0) {
        mergedDeliverableSystems.forEach(deliverable_system => {
          const option = document.createElement('option')
          option.value = deliverable_system.id
          option.textContent = deliverable_system.name
          deliverableSystemSelect.appendChild(option)
        })
        
        // 获取当前交付物体系ID
        if (workspaceResponse.ok) {
          const workspaceData = await workspaceResponse.json()
          if (workspaceData.status === 'success' && workspaceData.current_deliverable_system_id) {
            selectedDeliverableSystem.value = workspaceData.current_deliverable_system_id
          }
        }
      } else {
        const option = document.createElement('option')
        option.value = ''
        option.textContent = '暂无可用交付物体系'
        option.disabled = true
        deliverableSystemSelect.appendChild(option)
      }
    }
    
    // 加载当前交付物体系的层级信息
    updateCurrentDeliverableTypes()
  } catch (error) {
    console.error('加载交付物体系失败:', error)
  } finally {
    loading.value = false
  }
}

const saveDeliverableSystemGuard = createClickGuard()

const saveDeliverableSystem = async () => {
  // OPT-20260819-038: 连点/超时重试会双发 POST — createClickGuard 在途锁 + Idempotency-Key
  await saveDeliverableSystemGuard.run(async ({ idempotencyKey }) => {
    saving.value = true

    try {
      const deliverableSystemId = selectedDeliverableSystem.value
      const workspaceId = getCurrentWorkspaceId()

      const form = new URLSearchParams()
      form.append('action', 'update_deliverable_system')
      form.append('workspace_id', workspaceId)
      form.append('deliverable_system_id', deliverableSystemId)

      const response = await apiFetch(`/api/projects/manage-deliverable-system/tenant_id/${props.tenantId}`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
        body: form
      })

      const result = await response.json()

      if (result.status === 'success') {
        // 保存成功后，重新加载交付物体系
        await loadDeliverableSystems()
        toastService.success('交付物体系设置保存成功')
      } else {
        toastService.error('保存交付物体系失败: ' + result.message)
      }
    } catch (error) {
      console.error('保存交付物体系失败:', error)
      toastService.error('保存交付物体系失败，请重试')
    } finally {
      saving.value = false
    }
  })
}

// 更新当前交付物体系的层级信息
const updateCurrentDeliverableTypes = () => {
  if (!selectedDeliverableSystem.value) {
    currentDeliverableTypes.value = []
    return
  }
  
  // 从已加载的交付物体系数据中找到当前选中的交付物体系
  const deliverableSystem = deliverableSystems.value.find(system => system.id === selectedDeliverableSystem.value)
  if (deliverableSystem) {
    // 将level_names转换为currentDeliverableTypes需要的格式
    currentDeliverableTypes.value = deliverableSystem.level_names.map((level, index) => ({
      id: index + 1, // 使用索引作为临时ID
      name: level
    }))
  } else {
    currentDeliverableTypes.value = []
  }
}

onMounted(() => {
  // Only load deliverable systems if workspaceId is provided
  if (props.workspaceId) {
    loadDeliverableSystems()
  }
})

// 监听workspaceId变化，重新加载交付物体系
watch(() => props.workspaceId, () => {
  loadDeliverableSystems()
})

// 监听tenantId变化，重新加载交付物体系
watch(() => props.tenantId, () => {
  loadDeliverableSystems()
})

// 监听selectedDeliverableSystem变化，更新当前交付物体系层级
watch(selectedDeliverableSystem, () => {
  updateCurrentDeliverableTypes()
})
</script>

<style scoped>
</style>