<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-60">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-2xl p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">{{ formData.vpc_id ? '编辑VPC（专有网络）' : '创建VPC（专有网络）' }}</h3>
        <button type="button" @click="handleClose" 
                class="text-gray-500 hover:text-gray-700 transition-colors">
          &times;
        </button>
      </div>
      
      <!-- 创建VPC表单 -->
      <form @submit.prevent="handleSubmit">
        <div class="space-y-4">
          <input type="hidden" v-model="formData.authorization_id">
          <input type="hidden" v-model="formData.platform_type">
          
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">VPC名称</label>
            <input 
              type="text" 
              v-model="formData.name" 
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
              placeholder="请输入VPC名称" 
              required
            >
          </div>
          
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">地域</label>
            <select 
              v-model="formData.region" 
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
              required
              :disabled="!!formData.region"
            >
              <option value="">请选择地域</option>
              <option v-for="region in regions" :key="region.id" :value="region.id">{{ region.name }}</option>
            </select>
          </div>
          
          <!-- 已被占用的IP网段 -->
          <div v-if="formData.region && occupiedCidrBlocks.length > 0">
            <label class="block text-sm font-medium text-gray-700 mb-2">已被占用的IP网段</label>
            <div class="bg-gray-50 border border-gray-200 rounded-lg p-3">
              <div class="flex flex-wrap gap-2">
                <span v-for="cidr in occupiedCidrBlocks" :key="cidr" class="px-2 py-1 bg-red-100 text-red-800 rounded-full text-xs">
                  {{ cidr }}
                </span>
              </div>
            </div>
          </div>
          
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">IP网段 (IPv4地址段)</label>
            <input 
              type="text" 
              v-model="formData.cidr_block" 
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
              placeholder="例如: 192.168.0.0/16" 
              required
            >
            <!-- 推荐IP网段标签 -->
            <div v-if="recommendedCidrBlocks.length > 0" class="mt-3">
              <div class="text-xs text-gray-500 mb-2">推荐IP网段：</div>
              <div class="flex flex-wrap gap-2">
                <button 
                  v-for="cidr in recommendedCidrBlocks" 
                  :key="cidr"
                  type="button"
                  @click="selectRecommendedCidr(cidr)"
                  class="px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm hover:bg-blue-200 transition-colors"
                >
                  {{ cidr }}
                </button>
              </div>
            </div>
          </div>
        </div>
        
        <div class="flex justify-end space-x-3 mt-6">
          <button 
            type="button" 
            @click="handleClose" 
            class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300"
          >
            取消
          </button>
          <button 
                type="submit" 
                :disabled="loading" 
                class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300"
              >
                {{ loading ? (formData.vpc_id ? '保存中...' : '创建中...') : (formData.vpc_id ? '保存VPC' : '创建VPC') }}
              </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import modalService from '../../utils/modalService.js'

const getRegionDisplayName = (regionId, fallbackName) => {
  return fallbackName || regionId
}

// Props
const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  initialData: {
    type: Object,
    default: () => ({
      authorization_id: '',
      platform_type: '',
      region: ''
    })
  }
})

// Emits
const emit = defineEmits(['close', 'created'])

// 表单数据
const formData = ref({
  name: '',
  region: '',
  cidr_block: '',
  vpc_id: '',
  authorization_id: '',
  platform_type: ''
})

// 加载状态
const loading = ref(false)

// 地域列表
const regions = ref([])

// 已被占用的IP网段
const occupiedCidrBlocks = ref([])

// 推荐IP网段计算属性
const recommendedCidrBlocks = computed(() => {
  // 常见的私有IP网段
  const commonCidrBlocks = [
    '192.168.0.0/16',
    '192.168.1.0/24',
    '192.168.10.0/24',
    '10.0.0.0/16',
    '10.0.1.0/24',
    '10.1.0.0/16',
    '172.16.0.0/16',
    '172.16.1.0/24',
    '172.17.0.0/16'
  ]
  
  // 过滤掉已被占用的网段
  return commonCidrBlocks.filter(cidr => !occupiedCidrBlocks.value.includes(cidr))
})

// 选择推荐的IP网段
const selectRecommendedCidr = (cidr) => {
  formData.value.cidr_block = cidr
}

// 加载地域列表
const loadRegions = async () => {
  if (!formData.value.platform_type || !formData.value.authorization_id) {
    regions.value = []
    return
  }
  
  try {
    // 获取当前路由的租户ID
    const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];
    
    const response = await apiFetch(`/api/cloud/regions/tenant_id/${tenantId}/?platform_type=${encodeURIComponent(formData.value.platform_type)}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (response.ok) {
      const data = await response.json()
      
      // 处理 API 返回的数据格式
if (Array.isArray(data)) {
        // handleCloudRegions 返回 formatRegionsIDName: [{id, name}]
        regions.value = data.map(region => {
            const regionId = region.region_id || region.id || region.value || region.code
            const regionName = region.region_name || region.name || region.label || region.title
            return {
              id: regionId,
              name: getRegionDisplayName(regionId, regionName)
            }
          }).filter(region => region.id && region.name)
      } else if (data.status === 'success') {
        if (data.regions && Array.isArray(data.regions)) {
          // 数据在 regions 字段中
          regions.value = data.regions.map(region => {
            const regionId = region.region_id || region.id || region.value || region.code
            const regionName = region.region_name || region.name || region.label || region.title
            return {
              id: regionId,
              name: getRegionDisplayName(regionId, regionName)
            }
          }).filter(region => region.id && region.name)
        } else if (data.data && Array.isArray(data.data)) {
          // 数据在 data 字段中
          regions.value = data.data.map(region => {
            const regionId = region.region_id || region.id || region.value || region.code
            const regionName = region.region_name || region.name || region.label || region.title
            return {
              id: regionId,
              name: getRegionDisplayName(regionId, regionName)
            }
          }).filter(region => region.id && region.name)
        } else {
          regions.value = []
        }
      } else if (data.status === 'error') {
        // 如果是错误响应，重置所有数据
        console.log('API返回错误:', data.message)
        regions.value = []
      } else if (data.data && Array.isArray(data.data)) {
        // fallback: 数据在 data 字段中
        regions.value = data.data.map(region => {
            const regionId = region.region_id || region.id || region.value || region.code
            const regionName = region.region_name || region.name || region.label || region.title
            return {
              id: regionId,
              name: getRegionDisplayName(regionId, regionName)
            }
          }).filter(region => region.id && region.name)
      } else {
        regions.value = []
      }
    } else {
      regions.value = []
    }
  } catch (error) {
    regions.value = []
  }
}

// 拉取已被占用的IP网段
const loadOccupiedCidrBlocks = async () => {
  if (formData.value.platform_type && formData.value.authorization_id && formData.value.region) {
    try {
      // 获取当前路由的租户ID
      const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];
      
      // 构建查询参数
      const params = new URLSearchParams({
        platform_type: formData.value.platform_type,
        authorization_id: formData.value.authorization_id,
        region: formData.value.region
      })
      
      const response = await apiFetch(`/api/cloud/occupied-cidr-blocks/tenant_id/${tenantId}/?${params.toString()}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })
      
      if (response.ok) {
        const data = await response.json()
        if (data.status !== 'error' && data.occupied_cidr_blocks && Array.isArray(data.occupied_cidr_blocks)) {
          occupiedCidrBlocks.value = data.occupied_cidr_blocks
        } else {
          occupiedCidrBlocks.value = []
        }
      } else {
        occupiedCidrBlocks.value = []
      }
    } catch (error) {
      occupiedCidrBlocks.value = []
    }
  } else {
    occupiedCidrBlocks.value = []
  }
}

// 处理地域变化
const handleRegionChange = async () => {
  // 地域变化时，重置可用区和VPC相关数据
  
  // 拉取已被占用的IP网段
  await loadOccupiedCidrBlocks()
}

// 处理关闭模态框
const handleClose = () => {
  emit('close')
}

// 处理提交表单
const handleSubmit = async () => {
  if (!formData.value.name || !formData.value.region || !formData.value.cidr_block) {
    modalService.alert('请填写完整的VPC信息')
    return
  }
  
  const isEditing = !!formData.value.vpc_id
  loading.value = true
  try {
    // 获取当前路由的租户ID
    const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];
    
    const endpoint = isEditing 
      ? `/api/cloud/update-vpc/tenant_id/${tenantId}/` 
      : `/api/cloud/create-vpc/tenant_id/${tenantId}/`
    
    const response = await apiFetch(endpoint, {
      method: isEditing ? 'PUT' : 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include',
      body: JSON.stringify(formData.value)
    })
    
    if (response.ok) {
      const data = await response.json()
      if (data.status === 'success') {
        modalService.alert(isEditing ? 'VPC编辑成功' : 'VPC创建成功')
        emit('created', {
          vpc_id: String(data.vpc_id || formData.value.vpc_id || '').trim(),
          cidr_block: String(formData.value.cidr_block || '').trim(),
          name: String(formData.value.name || '').trim(),
        })
        emit('close')
      } else {
        throw new Error(data.message || (isEditing ? 'VPC编辑失败' : 'VPC创建失败'))
      }
    } else {
      throw new Error(isEditing ? 'VPC编辑失败' : 'VPC创建失败')
    }
  } catch (error) {
    console.error(isEditing ? '编辑VPC失败:' : '创建VPC失败:', error)
    modalService.alert((isEditing ? '编辑VPC失败: ' : '创建VPC失败: ') + error.message, '错误', { traceId: error.traceId })
  } finally {
    loading.value = false
  }
}

// 当props.visible变化时，重置表单并加载数据
onMounted(() => {
  if (props.visible) {
    formData.value = {
      name: props.initialData.vpc_name || '',
      region: props.initialData.region || '',
      cidr_block: props.initialData.cidr_block || '',
      vpc_id: props.initialData.vpc_id || '',
      authorization_id: props.initialData.authorization_id || '',
      platform_type: props.initialData.platform_type || ''
    }
    
    // 加载地域列表
    loadRegions()
    
    // 加载已被占用的IP网段
    loadOccupiedCidrBlocks()
  }
})

// 监听visible变化
watch(() => props.visible, (newValue) => {
  if (newValue) {
    formData.value = {
      name: props.initialData.vpc_name || '',
      region: props.initialData.region || '',
      cidr_block: props.initialData.cidr_block || '',
      vpc_id: props.initialData.vpc_id || '',
      authorization_id: props.initialData.authorization_id || '',
      platform_type: props.initialData.platform_type || ''
    }
    
    // 加载地域列表
    loadRegions()
    
    // 加载已被占用的IP网段
    loadOccupiedCidrBlocks()
  }
})
</script>
