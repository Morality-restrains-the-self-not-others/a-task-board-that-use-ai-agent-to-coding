<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-60">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-2xl p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">{{ formData.vswitch_id ? '编辑交换机' : '创建交换机' }}</h3>
        <button type="button" @click="handleClose" 
                class="text-gray-500 hover:text-gray-700 transition-colors">
          &times;
        </button>
      </div>
      
      <!-- 创建交换机表单 -->
      <form @submit.prevent="handleSubmit">
        <div class="space-y-4">
          <input type="hidden" v-model="formData.authorization_id">
          <input type="hidden" v-model="formData.platform_type">
          <input type="hidden" v-model="formData.vpc_id">
          
          <!-- 当前选择的VPC信息 -->
          <div class="bg-gray-50 border border-gray-200 rounded-lg p-3">
            <div class="text-sm font-medium text-gray-700 mb-1">当前选择的VPC：</div>
            <div class="text-sm text-gray-600">
              <span class="font-medium">{{ props.initialData.vpc_name || '未知' }}</span>
              <span class="ml-2 text-gray-500">({{ props.initialData.vpc_cidr_block || '未知网段' }})</span>
            </div>
          </div>
          
          <!-- 已存在的交换机网段 -->
          <div v-if="existingVswitchCidrs.length > 0" class="bg-gray-50 border border-gray-200 rounded-lg p-3">
            <div class="text-sm font-medium text-gray-700 mb-1">当前VPC已占用的网段：</div>
            <div class="flex flex-wrap gap-2">
              <span 
                v-for="cidr in existingVswitchCidrs" 
                :key="cidr"
                class="px-3 py-1 bg-gray-100 text-gray-800 rounded-full text-sm"
              >
                {{ cidr }}
              </span>
            </div>
          </div>
          
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">交换机名称</label>
            <input 
              type="text" 
              v-model="formData.name" 
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
              placeholder="请输入交换机名称" 
              required
            >
          </div>
          
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">可用区</label>
            <select 
              v-model="formData.zone_id" 
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
              required
            >
              <option value="">请选择可用区</option>
              <option v-for="zone in zones" :key="zone.id" :value="zone.id">{{ zone.name }}</option>
            </select>
          </div>
          
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">IP网段 (IPv4地址段)</label>
            <input 
              type="text" 
              v-model="formData.cidr_block" 
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
              placeholder="例如: 192.168.1.0/24" 
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
                {{ loading ? (formData.vswitch_id ? '保存中...' : '创建中...') : (formData.vswitch_id ? '保存交换机' : '创建交换机') }}
              </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import modalService from '../../utils/modalService.js'

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
      vpc_id: '',
      vpc_name: '',
      region: '',
      vpc_cidr_block: ''
    })
  }
})

// Emits
const emit = defineEmits(['close', 'created'])

// 表单数据
const formData = ref({
  name: '',
  zone_id: '',
  cidr_block: '',
  vswitch_id: '',
  authorization_id: '',
  platform_type: '',
  vpc_id: ''
})

// 加载状态
const loading = ref(false)

// 可用区列表
const zones = ref([])

// 推荐IP网段计算属性
const recommendedCidrBlocks = ref([])

// 已存在的交换机网段列表
const existingVswitchCidrs = ref([])

// 加载同一个VPC下其他交换机的网段列表
const loadExistingVswitchCidrs = async () => {
  existingVswitchCidrs.value = []
  
  if (!formData.value.platform_type || !formData.value.authorization_id || !formData.value.vpc_id || !props.initialData.region) {
    return
  }
  
  try {
    // 获取当前路由的租户ID
    const tenantId = window.location.pathname.match(/tenant\/(\d+)\//)[1];
    
    const response = await apiFetch(`/api/cloud/server-images/vswitches/tenant_id/${tenantId}/?region_id=${props.initialData.region}&vpc_id=${formData.value.vpc_id}&authorization_id=${formData.value.authorization_id}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (response.ok) {
      const data = await response.json()
      if (data.status !== 'error' && Array.isArray(data)) {
        // 提取已存在的交换机网段
        existingVswitchCidrs.value = data
          .map(vswitch => vswitch.cidr_block || vswitch.cidr)
          .filter(cidr => cidr)
      }
    }
  } catch (error) {
    console.error('加载已存在的交换机网段失败:', error)
  }
}

// 生成推荐的IP网段
const generateRecommendedCidrBlocks = (vpcCidr) => {
  if (!vpcCidr) {
    return []
  }
  
  try {
    // 解析VPC的CIDR块
    const [baseIp, prefix] = vpcCidr.split('/')
    const prefixInt = parseInt(prefix)
    
    // 生成推荐的网段，使用/24前缀
    const recommended = []
    
    // 简单的网段生成逻辑
    if (prefixInt <= 24) {
      // 对于较大的VPC网段，生成多个/24子网
      const ipParts = baseIp.split('.').map(Number)
      
      // 生成前10个可用的/24子网，确保有足够的选择
      for (let i = 1; i <= 10; i++) {
        if (i <= 254) {
          // 生成有效的网络地址（主机位为0）
          // 使用i作为第三个octet，确保每个子网都不同
          const newIp = `${ipParts[0]}.${ipParts[1]}.${i}.0`
          const cidr = `${newIp}/24`
          // 排除已存在的交换机网段
          if (!existingVswitchCidrs.value.includes(cidr)) {
            recommended.push(cidr)
          }
        }
      }
    } else {
      // 对于较小的VPC网段，直接使用
      if (!existingVswitchCidrs.value.includes(vpcCidr)) {
        recommended.push(vpcCidr)
      }
    }
    
    return recommended
  } catch (error) {
    console.error('解析VPC CIDR失败:', error)
    return []
  }
}

// 选择推荐的IP网段
const selectRecommendedCidr = (cidr) => {
  formData.value.cidr_block = cidr
}

// 加载可用区列表
const loadZones = async () => {
  // 重置可用区列表
  zones.value = []
  
  if (!formData.value.platform_type || !formData.value.authorization_id || !props.initialData.region) {
    return
  }
  
  try {
    // 获取当前路由的租户ID
    const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];
    
    // 使用当前设置默认配置表单中选择的地域来加载可用区
    const regionId = props.initialData.region
    
    if (regionId) {
      const response = await apiFetch(`/api/cloud/cloud-platform/${formData.value.authorization_id}/zones/tenant_id/${tenantId}/?region_id=${regionId}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })
      
      if (response.ok) {
        const data = await response.json()
        if (data.status === 'success' && data.zones && Array.isArray(data.zones)) {
          // 处理 API 返回的数据格式，确保每个可用区对象都有 id 和 name 属性
          zones.value = data.zones.map(zone => ({
            id: zone.zone_id || zone.id || zone.value || zone.code,
            name: zone.zone_name || zone.name || zone.label || zone.title || zone.id
          })).filter(zone => zone.id)
        } else if (Array.isArray(data)) {
          // 处理 API 返回的数据格式，确保每个可用区对象都有 id 和 name 属性
          zones.value = data.map(zone => ({
            id: zone.zone_id || zone.id || zone.value || zone.code,
            name: zone.zone_name || zone.name || zone.label || zone.title || zone.id
          })).filter(zone => zone.id)
        }
      }
    }
  } catch (error) {
    console.error('加载可用区列表失败:', error)
  }
}

// 处理关闭模态框
const handleClose = () => {
  emit('close')
}

// 验证CIDR块格式
const validateCidrBlock = (cidr) => {
  // 简单的CIDR块格式验证
  const cidrRegex = /^([0-9]{1,3}\.){3}[0-9]{1,3}\/([0-9]|[1-2][0-9]|3[0-2])$/;
  if (!cidrRegex.test(cidr)) {
    return false;
  }
  
  // 验证网络地址是否有效（主机位为0）
  const parts = cidr.split('/');
  const ipParts = parts[0].split('.').map(Number);
  const prefix = parseInt(parts[1]);
  
  // 对于/24前缀，最后一个octet应该是0
  if (prefix === 24 && ipParts[3] !== 0) {
    return false;
  }
  
  return true;
}

// 处理提交表单
const handleSubmit = async () => {
  if (!formData.value.name || !formData.value.zone_id || !formData.value.cidr_block) {
    modalService.alert('请填写完整的交换机信息')
    return
  }
  
  // 验证CIDR块格式
  if (!validateCidrBlock(formData.value.cidr_block)) {
    modalService.alert('请输入有效的CIDR块格式，例如: 192.168.1.0/24')
    return
  }
  
  const isEditing = !!formData.value.vswitch_id
  loading.value = true
  try {
    // 获取当前路由的租户ID
    const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];
    
    // 构建请求数据，包含region参数
    const requestData = {
      ...formData.value,
      region: props.initialData.region
    }
    
    const endpoint = isEditing
      ? `/api/cloud/server-images/update-vswitch/tenant_id/${tenantId}/`
      : `/api/cloud/server-images/create-vswitch/tenant_id/${tenantId}/`
    
    const response = await apiFetch(endpoint, {
      method: isEditing ? 'PUT' : 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include',
      body: JSON.stringify(requestData)
    })
    
    if (response.ok) {
      const data = await response.json()
      if (data.status === 'success') {
        modalService.alert(isEditing ? '交换机编辑成功' : '交换机创建成功')
        emit('created', {
          vswitch_id: String(data.vswitch_id || formData.value.vswitch_id || '').trim(),
        })
        emit('close')
      } else {
        throw new Error(data.message || (isEditing ? '交换机编辑失败' : '交换机创建失败'))
      }
    } else {
      throw new Error(isEditing ? '交换机编辑失败' : '交换机创建失败')
    }
  } catch (error) {
    console.error(isEditing ? '编辑交换机失败:' : '创建交换机失败:', error)
    modalService.alert((isEditing ? '编辑交换机失败: ' : '创建交换机失败: ') + error.message, '错误', { traceId: error.traceId })
  } finally {
    loading.value = false
  }
}

// 当props.visible变化时，重置表单并加载数据
onMounted(async () => {
  if (props.visible) {
    formData.value = {
      name: props.initialData.vswitch_name || '',
      zone_id: props.initialData.zone_id || '',
      cidr_block: props.initialData.cidr_block || '',
      vswitch_id: props.initialData.vswitch_id || '',
      authorization_id: props.initialData.authorization_id || '',
      platform_type: props.initialData.platform_type || '',
      vpc_id: props.initialData.vpc_id || ''
    }
    
    // 加载可用区列表
    await loadZones()
    
    // 加载已存在的交换机网段
    await loadExistingVswitchCidrs()
    
    // 生成推荐的IP网段
    recommendedCidrBlocks.value = generateRecommendedCidrBlocks(props.initialData.vpc_cidr_block)
  }
})

// 监听visible变化
watch(() => props.visible, async (newValue) => {
  if (newValue) {
    formData.value = {
      name: props.initialData.vswitch_name || '',
      zone_id: props.initialData.zone_id || '',
      cidr_block: props.initialData.cidr_block || '',
      vswitch_id: props.initialData.vswitch_id || '',
      authorization_id: props.initialData.authorization_id || '',
      platform_type: props.initialData.platform_type || '',
      vpc_id: props.initialData.vpc_id || ''
    }
    
    // 加载可用区列表
    await loadZones()
    
    // 加载已存在的交换机网段
    await loadExistingVswitchCidrs()
    
    // 生成推荐的IP网段
    recommendedCidrBlocks.value = generateRecommendedCidrBlocks(props.initialData.vpc_cidr_block)
  }
})
</script>
