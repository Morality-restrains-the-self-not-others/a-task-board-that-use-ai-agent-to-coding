<template>
<!-- 500-line rule exception: WorkspaceSettingsCloudPlatform is 807 lines (307 over). Vue SFC co-locates template+script+style. Split requires sub-component extraction. -->
  <TenantPageAccessEmpty v-if="!accessAllowed" page-key="settings.cloud" />
  <div v-else class="p-8">
    <div class="flex flex-col space-y-6">
      <div class="flex justify-between items-center">
        <div>
          <h2 class="text-2xl font-bold text-text">云平台设置</h2>
          <p class="text-text-light mt-1">管理工作空间的云平台相关设置</p>
        </div>
      </div>

      <!-- 云平台授权管理 -->
      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex justify-between items-center mb-6">
          <h3 class="text-xl font-bold text-text">云平台授权管理</h3>
          <button 
            class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
            @click="addAuthorizationModalVisible = true"
          >
            添加云平台授权
          </button>
        </div>
        
        <!-- 云平台授权列表 -->
        <div v-if="loadingAuthorizations" class="flex justify-center items-center py-12">
          <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary"></div>
          <span class="ml-3 text-gray-600">加载云平台授权列表中...</span>
        </div>
        
        <div v-else>
          <!-- 空状态 -->
          <div v-if="cloudPlatformAuthorizations.length === 0 && oauthTokens.length === 0" class="text-center py-12">
            <svg class="w-12 h-12 text-gray-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            <h4 class="text-lg font-medium text-gray-900 mb-2">暂无云平台授权</h4>
            <p class="text-gray-500 mb-6">系统中还没有任何云平台授权，请点击"添加云平台授权"按钮创建第一个授权。</p>
            <button 
              class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
              @click="addAuthorizationModalVisible = true"
            >
              添加云平台授权
            </button>
          </div>
          
          <!-- 授权列表 -->
          <div v-else class="space-y-4">
            <!-- 显示OAuth Token授权 -->
            <OAuthTokenRow 
              v-for="token in oauthTokens" 
              :key="token.id"
              :authorization="token"
              :cloud-platforms="cloudPlatforms"
              @delete="handleDeleteOAuthToken"
              @toggle-active="handleToggleAuthorizationActive"
            />
            <!-- 显示常规云平台授权 -->
            <CloudPlatformAuthorizationRow 
              v-for="authorization in cloudPlatformAuthorizations" 
              :key="authorization.id"
              :authorization="authorization"
              :cloud-platforms="cloudPlatforms"
              :verifying-authorization-id="verifyingAuthorizationId"
              @edit="openEditAuthorizationModal"
              @delete="handleDeleteAuthorization"
              @toggle-active="handleToggleAuthorizationActive"
              @verify-credentials="handleVerifyCloudCredentials"
            />
            

          </div>
        </div>
      </div>


    </div>

    <AddCloudPlatformAuthorizationModal
      :visible="addAuthorizationModalVisible"
      :form="addAuthorizationForm"
      :cloud-platforms="cloudPlatforms"
      :available-auth-types="availableAuthTypes"
      :adding-authorization="addingAuthorization"
      @close="addAuthorizationModalVisible = false"
      @submit="handleAddAuthorization"
      @oauth-login="handleOAuthLogin"
      @platform-type-change="handleAuthorizationPlatformTypeChange"
    />

    <EditCloudPlatformAuthorizationModal
      :visible="editAuthorizationModalVisible"
      :form="editAuthorizationForm"
      :cloud-platforms="cloudPlatforms"
      :editing-authorization="editingAuthorization"
      @close="editAuthorizationModalVisible = false"
      @submit="handleEditAuthorization"
    />

  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js';
import { safeJson } from '../utils/safeResponseJson.js';
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js';
import { ref, onMounted, computed } from 'vue'
import CloudPlatformAuthorizationRow from '../components/CloudPlatformAuthorizationRow.vue'
import OAuthTokenRow from '../components/OAuthTokenRow.vue'
import AddCloudPlatformAuthorizationModal from '../components/cloud/AddCloudPlatformAuthorizationModal.vue'
import EditCloudPlatformAuthorizationModal from '../components/cloud/EditCloudPlatformAuthorizationModal.vue'
import { getCookie } from '../utils/cookieUtils'
import modalService from '../utils/modalService.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { useTenantPageAccess } from '../composables/useTenantPageAccess.js'
import TenantPageAccessEmpty from '../components/TenantPageAccessEmpty.vue'

const { accessAllowed } = useTenantPageAccess('settings.cloud')

// 模态框可见性
const addAuthorizationModalVisible = ref(false)
const editAuthorizationModalVisible = ref(false)
const addImageModalVisible = ref(false)
const editImageModalVisible = ref(false)

// 加载状态
const loadingAuthorizations = ref(false)
const loadingImages = ref(false)
const loadingRegions = ref(false)
const loadingImagesList = ref(false)
const addingAuthorization = ref(false)
const editingAuthorization = ref(false)
const deletingAuthorization = ref(false)
const verifyingAuthorizationId = ref(null)
const addingImage = ref(false)
const editingImage = ref(false)
const deletingImage = ref(false)

// 数据列表
const cloudPlatformAuthorizations = ref([])
const oauthTokens = ref([])
const cloudServerImages = ref([])
const cloudPlatforms = ref([
  { value: 'aliyun', label: '阿里云' },
  { value: 'jdcloud', label: '京东云' },
  { value: 'tencentcloud', label: '腾讯云' },
  { value: 'huaweicloud', label: '华为云' },
  { value: 'ctyun', label: '天翼云' },
  { value: 'cmcc', label: '移动云' },
  { value: 'cucloud', label: '联通云' },
  { value: 'baiducloud', label: '百度智能云' },
  { value: 'aws', label: 'AWS' }
])
const regions = ref([])
const images = ref([])
const selectedImage = ref(null)
const regionError = ref('')

// 表单数据
const addAuthorizationForm = ref({
  platform_type: '',
  authorization_type: 'oauth',
  access_key: '',
  secret_key: '',
  remark: '',
  is_active: true
})

const editAuthorizationForm = ref({
  id: '',
  platform_type: '',
  authorization_type: 'access_key',
  access_key: '',
  secret_key: '',
  remark: '',
  is_active: true
})

// 根据选择的云平台类型动态显示可用的授权方式
const availableAuthTypes = computed(() => {
  const types = []
  const hasSelectedPlatform = addAuthorizationForm.value.platform_type
  
  // 始终返回两个选项，确保布局一致
  // Access Key 方式
  types.push({
    value: 'access_key',
    label: 'Access Key',
    disabled: !hasSelectedPlatform
  })
  
  // OAuth 方式 - 禁用阿里云的OAuth选项和未选择平台时
  types.push({
    value: 'oauth',
    label: 'OAuth 2.0',
    disabled: !hasSelectedPlatform || addAuthorizationForm.value.platform_type === 'aliyun'
  })
  
  return types
})

const addImageForm = ref({
  platform_type: '',
  region: '',
  image_name: '',
  image_id: '',
  os_type: '',
  os_version: '',
  is_active: true
})

const editImageForm = ref({
  id: '',
  platform_type: '',
  image_name: '',
  image_id: '',
  region: '',
  os_type: '',
  os_version: '',
  is_active: true
})

// 获取云平台名称
const getPlatformLabel = (value) => {
  const platform = cloudPlatforms.value.find(p => p.value === value)
  return platform ? platform.label : value
}

/** 兼容 Django 数组与 Go { authorizations: [...] } / 分页 { results: [...] } */
const normalizeCloudPlatformAuthorizationList = (payload) => {
  let rows = payload
  if (payload && !Array.isArray(payload)) {
    if (Array.isArray(payload.authorizations)) {
      rows = payload.authorizations
    } else if (Array.isArray(payload.results)) {
      rows = payload.results
    } else {
      rows = []
    }
  }
  return (rows || []).map((item) => ({
    ...item,
    is_active: item.is_active ?? item.active ?? false,
  }))
}

// 刷新云平台授权列表
const refreshCloudPlatformAuthorizations = async () => {
  loadingAuthorizations.value = true
  try {
    // 获取当前路由的租户ID
    const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];
    
    const response = await apiFetch(`/api/cloud/cloud-platform-authorizations/tenant_id/${tenantId}/`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (response.ok) {
      const payload = await safeJson(response, [])
      cloudPlatformAuthorizations.value = normalizeCloudPlatformAuthorizationList(payload)
    } else {
      const err = new Error('获取云平台授权列表失败')
      err.traceId = response.traceId || ''
      throw err
    }
  } catch (error) {
    console.error('刷新云平台授权列表失败:', error)
    showRequestError('刷新云平台授权列表失败', error)
  } finally {
    loadingAuthorizations.value = false
  }
}

// 刷新OAuth Token列表
const refreshOAuthTokens = async () => {
  try {
    // 获取当前路由的租户ID
    const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];
    
    const response = await apiFetch(`/api/cloud/oauth-tokens/tenant_id/${tenantId}/`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (response.ok) {
      oauthTokens.value = await safeJson(response, [])
    } else {
      const err = new Error('获取OAuth Token列表失败')
      err.traceId = response.traceId || ''
      throw err
    }
  } catch (error) {
    console.error('刷新OAuth Token列表失败:', error)
    // 不显示错误提示，因为OAuth可能未配置
  }
}

// 刷新云平台服务器镜像列表
const refreshCloudServerImages = async () => {
  loadingImages.value = true
  try {
    const response = await apiFetch('/api/cloud/server-images/', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (response.ok) {
      cloudServerImages.value = await safeJson(response, [])
    } else {
      const err = new Error('获取服务器镜像列表失败')
      err.traceId = response.traceId || ''
      throw err
    }
  } catch (error) {
    console.error('刷新服务器镜像列表失败:', error)
    showRequestError('刷新服务器镜像列表失败', error)
  } finally {
    loadingImages.value = false
  }
}

// 加载地域列表
const loadRegions = async (platformType) => {
  if (!platformType) {
    regions.value = []
    images.value = []
    regionError.value = ''
    return
  }
  
  loadingRegions.value = true
  regionError.value = ''
  try {
    const response = await apiFetch(`/api/cloud/regions/?platform_type=${platformType}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (response.ok) {
      const data = await safeJson(response, [])
      // 检查返回数据是否为错误格式
      if (data && data.status === 'error') {
        const err = new Error(data.message || '获取地域列表失败')
        err.traceId = response.traceId || data._traceId || ''
        throw err
      }
      regions.value = data
      images.value = []
    } else {
      // 尝试解析错误响应
      try {
        const errorData = await safeJson(response, {})
        throw new Error(errorData?.message || `获取地域列表失败: ${response.status}`)
      } catch (e) {
        const err = new Error(`获取地域列表失败: ${e.message || response.status}`)
        err.traceId = response.traceId || errorData?._traceId || ''
        throw err
      }
    }
  } catch (error) {
    console.error('加载地域列表失败:', error)
    regions.value = []
    images.value = []
    // 优化错误信息，使其更加友好和详细
    if (error.message.includes('请先激活云平台授权')) {
      regionError.value = `请先在云平台授权管理中激活 ${platformType} 的授权`
    } else {
      regionError.value = `${error.message}`
    }
  } finally {
    loadingRegions.value = false
  }
}

// 加载镜像列表
const loadImages = async (platformType, regionId) => {
  if (!platformType || !regionId) {
    images.value = []
    return
  }
  
  loadingImagesList.value = true
  try {
    const response = await apiFetch(`/api/cloud/images/?platform_type=${platformType}&region_id=${regionId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      },
      credentials: 'include'
    })
    
    if (response.ok) {
      images.value = await safeJson(response, [])
    } else {
      images.value = []
      throw new Error('获取镜像列表失败')
    }
  } catch (error) {
    console.error('加载镜像列表失败:', error)
    images.value = []
  } finally {
    loadingImagesList.value = false
  }
}

// 云平台类型变化处理（镜像表单）
const handlePlatformTypeChange = async () => {
  addImageForm.value.region = ''
  addImageForm.value.image_name = ''
  addImageForm.value.image_id = ''
  addImageForm.value.os_type = ''
  addImageForm.value.os_version = ''
  selectedImage.value = null
  regionError.value = ''
  await loadRegions(addImageForm.value.platform_type)
}

// 云平台类型变化处理（授权表单）
const handleAuthorizationPlatformTypeChange = () => {
  // 重置授权方式
  addAuthorizationForm.value.authorization_type = 'access_key'
  // 重置其他字段
  addAuthorizationForm.value.access_key = ''
  addAuthorizationForm.value.secret_key = ''
}




// OPT-20260819-038: 云平台授权/凭据属资源路径，写操作统一 createClickGuard 防连点双发。
const addAuthorizationGuard = createClickGuard()
const editAuthorizationGuard = createClickGuard()
const verifyCredentialsGuard = createClickGuard()
const deleteAuthorizationGuard = createClickGuard()
const deleteOAuthTokenGuard = createClickGuard()
const toggleActiveGuard = createClickGuard()

// 添加云平台授权
const handleAddAuthorization = async () => {
  // 验证是否选择了云平台类型
  if (!addAuthorizationForm.value.platform_type) {
    alert('请选择云平台类型')
    return
  }

  await addAuthorizationGuard.run(async ({ idempotencyKey }) => {
    addingAuthorization.value = true
    try {
      // 获取当前路由的租户ID
      const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];

      // 准备发送到后端的数据，将access_key映射为secret_id
      const formData = {
        ...addAuthorizationForm.value,
        secret_id: addAuthorizationForm.value.access_key,
        secret_key: addAuthorizationForm.value.secret_key,
        tenant_id: tenantId
      }
      // 删除不需要的字段
      delete formData.access_key

      const response = await apiFetch(`/api/cloud/cloud-platform-authorizations/tenant_id/${tenantId}/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        }, idempotencyKey),
        credentials: 'include',
        body: JSON.stringify(formData)
      })

      if (response.ok) {
        addAuthorizationModalVisible.value = false
        addAuthorizationForm.value = {
          platform_type: '',
          access_key: '',
          secret_key: '',
          remark: '',
          is_active: true
        }
        await refreshCloudPlatformAuthorizations()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || '添加云平台授权失败')
        err.traceId = response.traceId || errorData._traceId || ''
        throw err
      }
    } catch (error) {
      console.error('添加云平台授权失败:', error)
      showRequestError('添加云平台授权失败: ' + error.message, error)
    } finally {
      addingAuthorization.value = false
    }
  })
}

// 打开编辑云平台授权模态框
const openEditAuthorizationModal = (authorization) => {
  editAuthorizationForm.value = {
    id: authorization.id,
    platform_type: authorization.platform_type,
    access_key: authorization.secret_id, // 后端返回的是secret_id，映射为前端的access_key
    secret_key: authorization.secret_key,
    remark: authorization.remark || '',
    is_active: authorization.is_active
  }
  editAuthorizationModalVisible.value = true
}

// 编辑云平台授权
const handleEditAuthorization = async () => {
  await editAuthorizationGuard.run(async ({ idempotencyKey }) => {
    editingAuthorization.value = true
    try {
      // 获取当前路由的租户ID
      const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];

      // 准备发送到后端的数据，将access_key映射为secret_id
      const formData = {
        ...editAuthorizationForm.value,
        secret_id: editAuthorizationForm.value.access_key,
        secret_key: editAuthorizationForm.value.secret_key,
        tenant_id: tenantId
      }
      // 删除不需要的字段
      delete formData.access_key

      const response = await apiFetch(`/api/cloud/cloud-platform-authorizations/${editAuthorizationForm.value.id}/tenant_id/${tenantId}/`, {
        method: 'PUT',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        }, idempotencyKey),
        credentials: 'include',
        body: JSON.stringify(formData)
      })

      if (response.ok) {
        editAuthorizationModalVisible.value = false
        await refreshCloudPlatformAuthorizations()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || '编辑云平台授权失败')
        err.traceId = response.traceId || errorData._traceId || ''
        throw err
      }
    } catch (error) {
      console.error('编辑云平台授权失败:', error)
      showRequestError('编辑云平台授权失败: ' + error.message, error)
    } finally {
      editingAuthorization.value = false
    }
  })
}

const CALLER_IDENTITY_LABELS = {
  account_id: '账号 ID',
  arn: 'ARN',
  identity_type: '身份类型',
  principal_id: '主体 ID',
  user_id: '用户 ID',
  role_id: '角色 ID',
  request_id: '请求 ID',
}

/** 从 DRF / 业务 JSON 错误体中提取可读说明（支持 detail 为字符串或数组） */
const pickApiErrorDetail = (data) => {
  if (!data || typeof data !== 'object') {
    return ''
  }
  const d = data.detail
  if (typeof d === 'string') {
    return d
  }
  if (Array.isArray(d) && d.length) {
    return d.map((x) => String(x)).join(' ')
  }
  if (data.message != null) {
    return String(data.message)
  }
  return ''
}

// 使用云厂商 SDK（如阿里云 STS GetCallerIdentity）测试 AccessKey / SecretKey
const handleVerifyCloudCredentials = async (authorization) => {
  if (authorization.authorization_type !== 'access_key') {
    return
  }
  const pathMatch = window.location.pathname.match(/\/tenant\/(\d+)\//)
  if (!pathMatch) {
    await modalService.alert('无法解析租户 ID')
    return
  }
  const tenantId = pathMatch[1]
  await verifyCredentialsGuard.run(async ({ idempotencyKey }) => {
    verifyingAuthorizationId.value = authorization.id
    try {
      const response = await apiFetch(
        `/api/cloud/cloud-platform-authorizations/${authorization.id}/verify-credentials/tenant_id/${tenantId}/`,
        {
          method: 'POST',
          headers: mergeIdempotencyHeaders({
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
          }, idempotencyKey),
          credentials: 'include',
          body: '{}',
        },
      )
      const data = await response.json().catch(() => ({}))
      if (response.ok && data.success) {
        const ci = data.caller_identity || {}
        const lines = Object.entries(ci).map(
          ([k, v]) => `${CALLER_IDENTITY_LABELS[k] || k}: ${v}`,
        )
        await modalService.alert('凭据有效。\n' + lines.join('\n'))
      } else {
        const msg =
          pickApiErrorDetail(data) ||
          (Array.isArray(data.non_field_errors) && data.non_field_errors[0]) ||
          `HTTP ${response.status}`
        await modalService.alert('凭据校验失败：' + msg, '错误', {
          traceId: response.traceId || data._traceId || '',
        })
      }
    } catch (error) {
      console.error('凭据校验失败:', error)
      showRequestError('凭据校验请求失败', error)
    } finally {
      verifyingAuthorizationId.value = null
    }
  })
}

// 删除云平台授权
const handleDeleteAuthorization = async (id, platformType) => {
  if (id == null || id === '' || String(id) === 'undefined') {
    await modalService.alert('无法删除：授权 ID 无效，请刷新页面后重试')
    return
  }

  await deleteAuthorizationGuard.run(async ({ idempotencyKey }) => {
    console.log('开始删除云平台授权:', {
      authorizationId: id,
      platformType: platformType,
      platformLabel: getPlatformLabel(platformType),
      timestamp: new Date().toISOString()
    });

    try {
      await modalService.confirm(`确定要删除 ${getPlatformLabel(platformType)} 的授权吗？`)

      // 获取当前路由的租户ID
      const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];

      console.log('执行删除云平台授权请求:', {
        authorizationId: id,
        platformType: platformType,
        tenantId: tenantId,
        url: `/api/cloud/cloud-platform-authorizations/${id}/tenant_id/${tenantId}/`
      });

      deletingAuthorization.value = true
      const response = await apiFetch(`/api/cloud/cloud-platform-authorizations/${id}/tenant_id/${tenantId}/`, {
        method: 'DELETE',
        headers: mergeIdempotencyHeaders({
          'X-Requested-With': 'XMLHttpRequest'
        }, idempotencyKey),
        credentials: 'include'
      })

      if (response.ok) {
        console.log('删除云平台授权成功:', {
          authorizationId: id,
          platformType: platformType,
          responseStatus: response.status
        });
        await refreshCloudPlatformAuthorizations()
      } else {
        const errorData = await response.json().catch(() => ({}))
        console.error('删除云平台授权失败:', {
          authorizationId: id,
          platformType: platformType,
          responseStatus: response.status,
          errorMessage: errorData.message || '删除云平台授权失败'
        });
        const err = new Error(errorData.message || '删除云平台授权失败')
        err.traceId = response.traceId || errorData._traceId || ''
        throw err
      }
    } catch (error) {
      // 当用户取消确认时，不显示错误提示
      if (error) {
        console.log('删除云平台授权操作被取消或失败:', {
          authorizationId: id,
          platformType: platformType,
          error: error.message || error,
          isCancel: error === 'cancel' // 假设取消时的错误是 'cancel'
        });

        // 只有非取消操作的错误才显示提示
        if (error !== 'cancel' && error.message !== 'cancel') {
          showRequestError('删除云平台授权失败', error)
        }
      }
    } finally {
      deletingAuthorization.value = false
      console.log('删除云平台授权操作完成:', {
        authorizationId: id,
        platformType: platformType,
        timestamp: new Date().toISOString()
      });
    }
  })
}

// 删除OAuth Token
const handleDeleteOAuthToken = async (id, platformType) => {
  await deleteOAuthTokenGuard.run(async ({ idempotencyKey }) => {
    try {
      await modalService.confirm(`确定要删除 ${getPlatformLabel(platformType)} 的OAuth授权吗？`)

      // 获取当前路由的租户ID
      const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];

      deletingAuthorization.value = true
      const response = await apiFetch(`/api/cloud/oauth-tokens/${id}/tenant_id/${tenantId}/`, {
        method: 'DELETE',
        headers: mergeIdempotencyHeaders({
          'X-Requested-With': 'XMLHttpRequest'
        }, idempotencyKey),
        credentials: 'include'
      })

      if (response.ok) {
        await refreshOAuthTokens()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || '删除OAuth授权失败')
        err.traceId = response.traceId || errorData._traceId || ''
        throw err
      }
    } catch (error) {
      if (error) {
        console.error('删除OAuth授权失败:', error)
        showRequestError('删除OAuth授权失败', error)
      }
    } finally {
      deletingAuthorization.value = false
    }
  })
}

// 处理OAuth登录
const handleOAuthLogin = () => {
  // 获取当前路由的租户ID
  const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];
  
  // 构建OAuth登录URL，添加租户ID作为查询参数
  const oauthLoginUrl = `/api/cloud/oauth/aliyun/login/?tenant_id=${tenantId}`
  
  // 打开新窗口进行OAuth授权
  window.open(oauthLoginUrl, '_blank', 'width=800,height=600')
  
  // 关闭当前模态框
  addAuthorizationModalVisible.value = false
  
  // 刷新授权列表
  setTimeout(() => {
    Promise.all([
      refreshCloudPlatformAuthorizations(),
      refreshOAuthTokens()
    ])
  }, 3000)
}

// 处理云平台授权激活状态切换
const handleToggleAuthorizationActive = async (id, newActiveStatus) => {
  await toggleActiveGuard.run(async ({ idempotencyKey }) => {
    try {
      // 获取当前路由的租户ID
      const tenantId = window.location.pathname.match(/\/tenant\/(\d+)\//)[1];

      // 发送请求到toggle-active端点
      const response = await apiFetch(`/api/cloud/toggle-active/tenant_id/${tenantId}/`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        }, idempotencyKey),
        credentials: 'include',
        body: JSON.stringify({ id, is_active: newActiveStatus })
      })

      if (response.ok) {
        // 刷新授权列表
        await Promise.all([
          refreshCloudPlatformAuthorizations(),
          refreshOAuthTokens()
        ])
      } else {
        const errorData = await response.json().catch(() => ({}))
        const err = new Error(errorData.message || '切换云平台授权状态失败')
        // 保留 response.traceId（apiUtils 注入）或 body._traceId，确保 data-traceId 可挂载
        err.traceId = response.traceId || errorData._traceId || ''
        throw err
      }
    } catch (error) {
      console.error('切换云平台授权状态失败:', error)
      showRequestError('切换云平台授权状态失败', error)
    }
  })
}

// 页面加载时获取数据
onMounted(async () => {
  await Promise.all([
    refreshCloudPlatformAuthorizations(),
    refreshOAuthTokens()
  ])
})
</script>
<style scoped>
/* 组件内样式 */
</style>
