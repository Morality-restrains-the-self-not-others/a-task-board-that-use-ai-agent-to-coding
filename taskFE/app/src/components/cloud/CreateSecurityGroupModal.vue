<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-60">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-2xl p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">{{ formData.security_group_id ? '编辑安全组' : '创建安全组' }}</h3>
        <button type="button" @click="handleClose" 
                class="text-gray-500 hover:text-gray-700 transition-colors">
          &times;
        </button>
      </div>
      
      <!-- 创建安全组表单 -->
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
            </div>
          </div>
          
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">安全组名称</label>
            <input 
              type="text" 
              v-model="formData.name" 
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
              placeholder="请输入安全组名称" 
              required
            >
            <p
              v-if="hasFullInboundOpen"
              class="mt-1.5 text-xs text-amber-800 bg-amber-50 border border-amber-100 rounded-md px-2 py-1.5"
              role="status"
            >
              已包含「开放所有入口」规则，名称须带后缀「{{ riskNameSuffix }}」。
            </p>
          </div>
          
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">描述</label>
            <input 
              type="text" 
              v-model="formData.description" 
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
              placeholder="请输入安全组描述"
            >
          </div>
          
          <!-- 入方向规则 -->
          <div>
            <div class="flex justify-between items-center mb-2 flex-wrap gap-2">
              <label class="block text-sm font-medium text-gray-700">入方向规则</label>
              <div class="flex flex-wrap items-center gap-2">
                <button 
                  type="button" 
                  @click="addInboundRule" 
                  class="text-sm text-blue-600 hover:text-blue-800"
                >
                  添加规则
                </button>
                <button
                  type="button"
                  @click="addInboundFullOpen"
                  class="text-sm text-amber-800 hover:text-amber-950 underline decoration-dotted"
                >
                  开放所有入口
                </button>
              </div>
            </div>
            <div class="space-y-3">
              <div 
                v-for="(rule, index) in formData.inbound_rules" 
                :key="`inbound-${index}`" 
                class="flex items-center space-x-2 p-3 border border-gray-200 rounded-lg"
              >
                <select v-model="rule.ip_protocol" class="px-2 py-1 border border-gray-300 rounded">
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="icmp">ICMP</option>
                  <option value="all">ALL</option>
                </select>
                <input 
                  v-model="rule.port_range" 
                  placeholder="端口范围，如 22/22 或 80/443" 
                  class="px-2 py-1 border border-gray-300 rounded flex-1"
                >
                <input 
                  v-model="rule.source_cidr_ip" 
                  placeholder="源IP，如 0.0.0.0/0" 
                  class="px-2 py-1 border border-gray-300 rounded flex-1"
                >
                <button 
                  type="button" 
                  @click="removeInboundRule(index)" 
                  class="text-red-600 hover:text-red-800"
                >
                  删除
                </button>
              </div>
              <div v-if="formData.inbound_rules.length === 0" class="text-sm text-gray-500 italic">
                暂无入方向规则，点击"添加规则"按钮添加
              </div>
            </div>
          </div>
          
          <!-- 出方向规则 -->
          <div>
            <div class="flex justify-between items-center mb-2">
              <label class="block text-sm font-medium text-gray-700">出方向规则</label>
              <button 
                type="button" 
                @click="addOutboundRule" 
                class="text-sm text-blue-600 hover:text-blue-800"
              >
                添加规则
              </button>
            </div>
            <div class="space-y-3">
              <div 
                v-for="(rule, index) in formData.outbound_rules" 
                :key="`outbound-${index}`" 
                class="flex items-center space-x-2 p-3 border border-gray-200 rounded-lg"
              >
                <select v-model="rule.ip_protocol" class="px-2 py-1 border border-gray-300 rounded">
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="icmp">ICMP</option>
                  <option value="all">ALL</option>
                </select>
                <input 
                  v-model="rule.port_range" 
                  placeholder="端口范围，如 22/22 或 80/443" 
                  class="px-2 py-1 border border-gray-300 rounded flex-1"
                >
                <input 
                  v-model="rule.destination_cidr_ip" 
                  placeholder="目标IP，如 0.0.0.0/0" 
                  class="px-2 py-1 border border-gray-300 rounded flex-1"
                >
                <button 
                  type="button" 
                  @click="removeOutboundRule(index)" 
                  class="text-red-600 hover:text-red-800"
                >
                  删除
                </button>
              </div>
              <div v-if="formData.outbound_rules.length === 0" class="text-sm text-gray-500 italic">
                暂无出方向规则，点击"添加规则"按钮添加
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
                {{ loading ? (formData.security_group_id ? '保存中...' : '创建中...') : (formData.security_group_id ? '保存安全组' : '创建安全组') }}
              </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import modalService from '../../utils/modalService.js'

/** 存在「全入口」入方向规则时，强制附加在安全组名称上的后缀（与任务页自动创建安全组策略一致） */
const riskNameSuffix = '端口全开有风险'

function isFullOpenInboundRule(rule) {
  if (!rule) return false
  const proto = String(rule.ip_protocol || '').toLowerCase().trim()
  const port = String(rule.port_range || '').trim()
  const src = String(rule.source_cidr_ip || '').trim()
  return proto === 'all' && port === '-1/-1' && (src === '0.0.0.0/0' || src === '::/0')
}

function stripRiskSuffixFromName(name) {
  const s = String(name || '').trim()
  if (s.endsWith(riskNameSuffix)) {
    let base = s.slice(0, -riskNameSuffix.length).replace(/-+$/, '')
    return base.trim()
  }
  return s
}

function syncRiskSuffixInName() {
  if (hasFullInboundOpen.value) {
    const current = String(formData.value.name || '').trim()
    if (current.endsWith(riskNameSuffix)) {
      return
    }
    const base = stripRiskSuffixFromName(current)
    formData.value.name = base ? `${base}-${riskNameSuffix}` : riskNameSuffix
  } else {
    const current = String(formData.value.name || '').trim()
    if (current.endsWith(riskNameSuffix)) {
      formData.value.name = stripRiskSuffixFromName(current)
    }
  }
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
      vpc_id: '',
      vpc_name: '',
      region: ''
    })
  }
})

// Emits
const emit = defineEmits(['close', 'created'])

// 表单数据
const formData = ref({
  name: '',
  description: '',
  security_group_id: '',
  authorization_id: '',
  platform_type: '',
  vpc_id: '',
  inbound_rules: [],
  outbound_rules: []
})

// 加载状态
const loading = ref(false)

const hasFullInboundOpen = computed(() =>
  (formData.value.inbound_rules || []).some(isFullOpenInboundRule)
)

// 添加入方向规则
const addInboundRule = () => {
  // 使用新数组替换旧数组，确保响应式更新
  formData.value.inbound_rules = [...formData.value.inbound_rules, {
    ip_protocol: 'tcp',
    port_range: '22/22',
    source_cidr_ip: '0.0.0.0/0'
  }]
}

/** 添加「全入口」规则：ALL + -1/-1 + 0.0.0.0/0（与阿里云放通全部入站一致） */
const addInboundFullOpen = () => {
  if ((formData.value.inbound_rules || []).some(isFullOpenInboundRule)) {
    modalService.alert('已存在「开放所有入口」规则，无需重复添加')
    return
  }
  formData.value.inbound_rules = [
    ...(formData.value.inbound_rules || []),
    {
      ip_protocol: 'all',
      port_range: '-1/-1',
      source_cidr_ip: '0.0.0.0/0'
    }
  ]
  syncRiskSuffixInName()
}

// 移除入方向规则
const removeInboundRule = (index) => {
  // 使用新数组替换旧数组，确保响应式更新
  formData.value.inbound_rules = formData.value.inbound_rules.filter((_, i) => i !== index)
}

// 添加出方向规则
const addOutboundRule = () => {
  // 使用新数组替换旧数组，确保响应式更新
  formData.value.outbound_rules = [...formData.value.outbound_rules, {
    ip_protocol: 'tcp',
    port_range: '80/443',
    destination_cidr_ip: '0.0.0.0/0'
  }]
}

// 移除出方向规则
const removeOutboundRule = (index) => {
  // 使用新数组替换旧数组，确保响应式更新
  formData.value.outbound_rules = formData.value.outbound_rules.filter((_, i) => i !== index)
}

// 处理关闭模态框
const handleClose = () => {
  emit('close')
}

// 处理提交表单
const handleSubmit = async () => {
  syncRiskSuffixInName()
  if (!formData.value.name) {
    modalService.alert('请填写完整的安全组信息')
    return
  }
  if (hasFullInboundOpen.value && !String(formData.value.name).trim().endsWith(riskNameSuffix)) {
    modalService.alert(`入方向已放通全部入口，安全组名称必须以「${riskNameSuffix}」结尾`)
    return
  }
  
  const isEditing = !!formData.value.security_group_id
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
      ? `/api/cloud/server-images/update-security-group/tenant_id/${tenantId}/`
      : `/api/cloud/server-images/create-security-group/tenant_id/${tenantId}/`
    
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
        modalService.alert(isEditing ? '安全组编辑成功' : '安全组创建成功')
        emit('created')
        emit('close')
      } else {
        throw new Error(data.message || (isEditing ? '安全组编辑失败' : '安全组创建失败'))
      }
    } else {
      throw new Error(isEditing ? '安全组编辑失败' : '安全组创建失败')
    }
  } catch (error) {
    console.error(isEditing ? '编辑安全组失败:' : '创建安全组失败:', error)
    modalService.alert((isEditing ? '编辑安全组失败: ' : '创建安全组失败: ') + error.message, '错误', { traceId: error.traceId })
  } finally {
    loading.value = false
  }
}

// 处理规则中的ip_protocol，转换为小写
const processRules = (rules) => {
  return rules.map(rule => ({
    ...rule,
    ip_protocol: rule.ip_protocol ? rule.ip_protocol.toLowerCase() : 'tcp'
  }))
}

function resetFormFromProps() {
  formData.value = {
    name: props.initialData.security_group_name || '',
    description: props.initialData.description || '',
    security_group_id: props.initialData.security_group_id || '',
    authorization_id: props.initialData.authorization_id || '',
    platform_type: props.initialData.platform_type || '',
    vpc_id: props.initialData.vpc_id || '',
    inbound_rules: processRules(props.initialData.inbound_rules || []),
    outbound_rules: processRules(props.initialData.outbound_rules || [])
  }
}

// 当props.visible变化时，重置表单并加载数据
onMounted(async () => {
  if (props.visible) {
    resetFormFromProps()
    await nextTick()
    syncRiskSuffixInName()
  }
})

// 监听visible变化
watch(() => props.visible, async (newValue) => {
  if (newValue) {
    resetFormFromProps()
    await nextTick()
    syncRiskSuffixInName()
  }
})

watch(
  () => formData.value.inbound_rules,
  () => {
    syncRiskSuffixInName()
  },
  { deep: true }
)
</script>