<template>
  <!-- 切换任务面板：独立控件 -->
  <div ref="rootEl" data-alias="cmp-workspace-switcher" class="relative inline-block">
    <!-- 任务面板切换按钮 -->
    <button
      id="workspace-toggle"
      type="button"
      class="flex items-center justify-between w-full px-4 py-2 border border-gray-300 rounded-md shadow-sm bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
      @click="toggleWorkspaceMenu"
    >
      <span>{{ currentWorkspaceName }}</span>
      <svg class="w-4 h-4 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
      </svg>
    </button>
    
    <!-- 任务面板下拉菜单 -->
    <div
      v-if="isWorkspaceMenuOpen"
      ref="menuRootEl"
      class="absolute right-0 mt-2 w-64 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 z-50"
    >
      <div class="py-1">
        <!-- 工作空间列表 -->
        <template v-if="isLoading">
          <div class="px-4 py-2 text-sm text-gray-500">加载中...</div>
        </template>
        <template v-else-if="workspaceOptions.length === 0">
          <div class="px-4 py-2 text-sm text-gray-500">无可用工作空间</div>
        </template>
        <template v-else>
          <button
            v-for="option in workspaceOptions"
            :key="option.value"
            type="button"
            class="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 hover:text-gray-900"
            :class="{ 'bg-primary/10 text-primary': selectedWorkspace === option.value }"
            @click="switchToWorkspace(option.value)"
          >
            {{ option.text }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
/* @alias:cmp-workspace-switcher */
import { ref, computed, onMounted, watch } from 'vue'
import { getCookie } from '../utils/cookieUtils'
import { apiFetch } from '../utils/apiUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { safeJson } from '../utils/safeResponseJson.js'
import { useRoute, useRouter } from 'vue-router'
import { useClickOutside } from '../composables/useClickOutside.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

// 定义组件属性
const props = defineProps({
  initialSelectedWorkspace: {
    type: String,
    default: ''
  },
  tenant: {
    type: [Number, String],
    required: true
  },
  refreshTrigger: {
    type: Number,
    default: 0
  }
})

const tenantPath = computed(() => {
  return '/tenant/' + props.tenant
})

// 定义组件事件
// workspace-load-error：工作空间列表加载失败时上抛请求 traceId（apiFetch 已注入 .traceId），
// 供父级 header-title-row 等元素挂 data-traceId（对齐全站「错误元素必须带 traceId」约定）。
// workspace-loaded：加载成功后通知父级清除过期 traceId。
const emit = defineEmits(['workspace-switched', 'workspace-load-error', 'workspace-loaded'])

// 定义路由对象
const router = useRouter()

// 定义响应式数据
const workspaceOptions = ref([])
const selectedWorkspace = ref(props.initialSelectedWorkspace)
const isLoading = ref(true)
const isWorkspaceMenuOpen = ref(false)
const menuRootEl = ref(null)
const rootEl = ref(null)
const switchWorkspaceGuard = createClickGuard()

useClickOutside(
  () => rootEl.value || menuRootEl.value,
  () => {
    isWorkspaceMenuOpen.value = false
  },
  { enabled: isWorkspaceMenuOpen, capture: true },
)

// 计算当前任务面板名称
const currentWorkspaceName = computed(() => {
  if (isLoading.value) {
    return '任务面板'
  }
  
  // 首先尝试根据 selectedWorkspace 查找
  const currentWorkspace = workspaceOptions.value.find(option => option.value === selectedWorkspace.value)
  if (currentWorkspace) {
    return currentWorkspace.text
  }
  
  // 如果没有找到，尝试查找 is_current 为 true 的工作空间（作为备用）
  const defaultWorkspace = workspaceOptions.value.find(option => option.is_current === true)
  
  if (defaultWorkspace) {
    selectedWorkspace.value = defaultWorkspace.value
    return defaultWorkspace.text
  }
  
  // 如果没有找到任何工作空间，返回默认值
  if (workspaceOptions.value.length === 0) {
    return '无可用工作空间'
  }
  
  // 如果有工作空间但没有选中的，默认选择第一个
  selectedWorkspace.value = workspaceOptions.value[0].value
  return workspaceOptions.value[0].text
})



// 切换工作空间菜单显示/隐藏
const toggleWorkspaceMenu = () => {
  isWorkspaceMenuOpen.value = !isWorkspaceMenuOpen.value
}

// 切换到指定工作空间
const switchToWorkspace = async (workspaceId) => {
  isWorkspaceMenuOpen.value = false

  // 查找对应的工作空间名称
  const workspace = workspaceOptions.value.find(option => option.value === workspaceId)
  const workspaceName = workspace ? workspace.text : '未知工作空间'

  // OPT-20260819-038: 切换工作空间是资源写操作，防连点/超时重试双发
  await switchWorkspaceGuard.run(async ({ idempotencyKey }) => {
    // 显示加载状态
    isLoading.value = true

    // 发送AJAX请求切换工作空间
    const tenant = props.tenant || ''
    apiFetch(`/api/projects/switch/tenant_id/${tenant}/`, {
      method: 'POST',
      headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
      body: JSON.stringify({ workspace_id: workspaceId })
    })
    .then(response => {

      if (!response.ok) {
        const err = new Error('Network response was not ok')
        err.traceId = response.traceId || ''
        throw err
      }
      return safeJson(response, {})
    })
    .then(data => {
      if (data.success) {
        // 先发送事件通知父组件，传递工作空间ID和名称
        emit('workspace-switched', { id: workspaceId, name: workspaceName })
        // 切换成功，更新URL中的workspace_id参数并刷新页面
        const url = new URL(window.location.href)
        url.searchParams.set('workspace_id', workspaceId)
        window.location.href = url.toString()
      } else {
        // 切换失败，恢复原始状态
        showRequestError('切换工作空间失败: ' + (data.error || '未知错误'), data)
        isLoading.value = false
      }
    })
    .catch(error => {
      console.warn('[WorkspaceSwitcher] 切换工作空间失败:', error)
      showRequestError('切换工作空间失败，请刷新页面重试', error)
      isLoading.value = false
    })
  })
}

// 加载工作空间列表
const loadWorkspaces = () => {
  // 检查租户ID是否存在
  if (!props.tenant) {
    console.error('缺少租户ID，无法加载工作空间')
    isLoading.value = false
    return
  }
  
  // 发送AJAX请求获取工作空间列表，包含当前页面的workspace_id参数
  const currentUrl = new URL(window.location.href)
  const workspaceIdParam = currentUrl.searchParams.get('workspace_id')
  
  // 构建API URL，包含租户ID
  let apiUrl = `/api/projects/workspaces/tenant_id/${props.tenant}`
  if (workspaceIdParam) {
    apiUrl += `?workspace_id=${workspaceIdParam}`
  }
  
  apiFetch(apiUrl, {
    headers: {
      'Accept': 'application/json'
    }
  })
    .then(response => {

      if (!response.ok) {
        const err = new Error('Network response was not ok')
        err.traceId = response.traceId || ''
        throw err
      }
      return safeJson(response, [])
    })
    .then(data => {
      // 清空工作空间选项
      workspaceOptions.value = []
      
      let currentActiveWorkspace = null
      
      if (data.length > 0) {
        // 填充工作空间选项，保留完整的workspace对象
        data.forEach(workspace => {
          workspaceOptions.value.push({
            value: workspace.id,
            text: workspace.name,
            // 保留is_current字段，用于后续计算
            is_current: workspace.is_current
          })
          
          // 设置当前选中的工作空间
          if (workspace.is_current) {
            selectedWorkspace.value = workspace.id
            currentActiveWorkspace = {
              id: workspace.id,
              name: workspace.name
            }
          }
        })
        
        // 如果没有标记为当前的工作空间，默认选择第一个
        if (!currentActiveWorkspace && workspaceOptions.value.length > 0) {
          selectedWorkspace.value = workspaceOptions.value[0].value
          currentActiveWorkspace = {
            id: workspaceOptions.value[0].value,
            name: workspaceOptions.value[0].text
          }
        }
        
        // 发出事件，通知父组件当前激活的工作空间
        if (currentActiveWorkspace) {
          emit('workspace-switched', currentActiveWorkspace)
        }
      }
      
      isLoading.value = false
      emit('workspace-loaded')
    })
  .catch(error => {
    console.warn('[WorkspaceSwitcher] 加载工作空间失败:', error)
    workspaceOptions.value = []
    isLoading.value = false
    emit('workspace-load-error', error.traceId || '')
  })
}

// 组件挂载后加载工作空间列表
onMounted(() => {
  loadWorkspaces()
})

// 监听刷新触发信号
watch(
  () => props.refreshTrigger,
  (newValue, oldValue) => {
    if (newValue !== oldValue) {
      console.log('触发工作空间列表刷新')
      loadWorkspaces()
    }
  }
)
</script>

<style scoped>
/* 组件样式可以在这里添加 */
</style>